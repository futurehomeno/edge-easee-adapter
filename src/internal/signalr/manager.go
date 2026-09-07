package signalr

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/root"
	"github.com/futurehomeno/cliffhanger/telemetry"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

type DisconnectionReason string

const (
	ChargerNotRegistered DisconnectionReason = "charger is not registered in a manager"
	ChargerNotSubscribed DisconnectionReason = "charger is not subscribed for SignalR observations"
	ChargerOffline       DisconnectionReason = "charger is offline"

	// Per-charger dispatch queue depth. Deep enough to absorb the initial batch a subscribe
	// triggers while a command holds the chargepoint service mutex.
	observationQueueSize = 100
)

// Manager owns the single signalR connection and the chargers subscribed on it.
type Manager interface {
	root.Service

	// Connected reports whether the charger is reachable, and why not when it is not.
	Connected(chargerID string) (bool, DisconnectionReason)
	Register(chargerID string, handler Handler)
	Unregister(chargerID string) error
}

type manager struct {
	mu sync.RWMutex

	running bool
	done    chan struct{}
	cfg     *config.Service

	subscriptions chan string

	// Bumped on every connect, so a subscribe that spans a reconnect can tell its result
	// belongs to a connection that is already gone.
	epoch uint64

	client   Client
	tel      telemetry.Telemetry
	chargers map[string]*charger
}

func NewManager(cfg *config.Service, client Client, tel telemetry.Telemetry) Manager {
	return &manager{
		cfg:           cfg,
		client:        client,
		tel:           tel,
		chargers:      make(map[string]*charger),
		subscriptions: make(chan string, 32),
	}
}

func (m *manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	m.done = make(chan struct{})

	go m.run()

	m.running = true

	return nil
}

func (m *manager) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()

		return nil
	}

	m.running = false
	done := m.done

	// Every dispatcher is stopped here, not left to Unregister: Stop does not unregister, so
	// watching only dispatchDone leaked one goroutine per registered charger on every stop.
	// stopDispatch guards the close, since Unregister closes the same channel.
	for _, c := range m.chargers {
		c.stopDispatch()
	}

	m.mu.Unlock()

	err := m.client.Close()

	close(done)

	return err
}

func (m *manager) Register(chargerID string, handler Handler) {
	m.mu.Lock()

	if _, ok := m.chargers[chargerID]; ok {
		m.mu.Unlock()
		log.Warnf("Charger '%s' is already registered", chargerID)

		return
	}

	c := &charger{
		handler:      handler,
		isSubscribed: false,
		backoff:      m.cfg.SignalRBackoffStateful(),
		observations: make(chan model.Observation, observationQueueSize),
		dispatchDone: make(chan struct{}),
	}
	m.chargers[chargerID] = c

	go m.dispatchObservations(chargerID, c)

	m.client.Start()

	connected := m.client.Connected()
	m.mu.Unlock()

	// The charger is already in m.chargers, so a connect still in flight will pick it up
	// via the handleClientState sweep once it lands. Enqueuing here too would just add a
	// guaranteed "client is not running" failure while the handshake is still running.
	if connected {
		m.enqueueSubscription(chargerID)
	}
}

// enqueueSubscription hands a charger ID to the run loop for (re)subscription.
// The send never happens under m.mu and always has a cancellation escape via done.
func (m *manager) enqueueSubscription(chargerID string) {
	m.mu.RLock()
	done := m.done
	m.mu.RUnlock()

	select {
	case m.subscriptions <- chargerID:
	case <-done:
	}
}

func (m *manager) Unregister(chargerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	charger, ok := m.chargers[chargerID]
	if !ok {
		return nil
	}

	delete(m.chargers, chargerID)

	// Stops this charger's observation dispatcher; anything still queued is abandoned along
	// with the charger it belonged to. Nil for a charger assembled directly in a test rather
	// than through Register.
	charger.stopDispatch()

	var errs error

	if err := m.client.UnsubscribeCharger(chargerID); err != nil {
		errs = errors.Join(errs, err)
	}

	if len(m.chargers) == 0 {
		if err := m.client.Close(); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (m *manager) Connected(chargerID string) (bool, DisconnectionReason) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	charger, ok := m.chargers[chargerID]
	if !ok {
		return false, ChargerNotRegistered
	}

	// An in-flight subscribe counts as connected: the server streams the initial batch as soon
	// as the invoke is made, so gating on isSubscribed alone failed the follow-up report of
	// every observation in that batch and left the app on stale values until the next one.
	if !charger.isSubscribed && !charger.subscribing {
		return false, ChargerNotSubscribed
	}

	if !charger.handler.IsOnline() {
		return false, ChargerOffline
	}

	return true, ""
}

func (m *manager) run() {
	defer telemetry.RecoverAndEmit(m.tel, "manager.run", true)

	m.mu.RLock()
	done := m.done
	m.mu.RUnlock()

	states := m.client.StateC()
	observations := m.client.ObservationC()

	for {
		select {
		case <-done:
			return

		case chargerID := <-m.subscriptions:
			// Off the loop: SubscribeCharger blocks for up to SignalRInvokeTimeout, and this
			// goroutine is the only drainer of the observation channel. Subscribing inline
			// stalled that drain while the server streamed the initial batch the subscribe
			// itself asked for, overflowing the buffer.
			go func() {
				defer telemetry.RecoverAndEmit(m.tel, "manager.subscribe", true)

				if err := m.handleSubscription(chargerID); err != nil {
					log.Warnf("Handle subscription chargerID=%s err: %v", chargerID, err)
				}
			}()

		case state := <-states:
			m.handleClientState(state)

		case observation := <-observations:
			m.handleObservation(observation)
		}
	}
}

func (m *manager) handleSubscription(chargerID string) error {
	m.mu.Lock()

	charger, ok := m.chargers[chargerID]
	if !ok {
		m.mu.Unlock()

		return fmt.Errorf("unknown charger")
	}

	// A retry armed before a disconnect can fire after the reconnect sweep already
	// re-subscribed this charger; don't subscribe twice on the same connection. Subscribes
	// run concurrently now, so an in-flight one has to block a second the same way.
	if charger.isSubscribed || charger.subscribing {
		m.mu.Unlock()

		return nil
	}

	charger.subscribing = true
	epoch := m.epoch

	m.mu.Unlock()

	// Invoked unlocked: SubscribeCharger blocks for up to SignalRInvokeTimeout. Holding m.mu
	// across it stalled Connected(), Register() and every observation lookup for the whole
	// invocation.
	err := m.client.SubscribeCharger(chargerID)

	m.mu.Lock()
	defer m.mu.Unlock()

	// A reconnect retired this connection mid-invoke: the sweep already reset the charger and
	// enqueued a fresh subscribe, so this result would only corrupt that attempt. Nothing to
	// unsubscribe - whatever it established belonged to the connection now gone.
	if epoch != m.epoch {
		return nil
	}

	charger.subscribing = false

	// Unregister can drop this charger - and a later Register re-add a different struct under
	// the same ID - while the invoke ran unlocked, so identity is the test, not presence.
	if current, ok := m.chargers[chargerID]; !ok || current != charger {
		// A Subscribe that wins this race establishes a cloud subscription after
		// Unregister's own Unsubscribe already ran, orphaning it with no local charger
		// left to receive its observations until the next reconnect. Compensated whatever
		// the invoke reported: a local timeout does not cancel the server-side operation,
		// so an error is no proof the subscription was not established. Unsubscribing one
		// that never existed is a no-op, which is the cheaper way to be wrong.
		if unsubErr := m.client.UnsubscribeCharger(chargerID); unsubErr != nil {
			log.Warnf("signalR: cleanup after unregister race chargerID=%s err: %v", chargerID, unsubErr)
		}

		return nil
	}

	if err != nil {
		// A sustained outage (e.g. not logged in) retries forever; warn on the first
		// failure of a streak only, so it does not accumulate thousands of lines.
		if charger.subscribeFailed {
			log.Debugf("Failed to subscribe charger '%s'", chargerID)
		} else {
			log.Warnf("Failed to subscribe charger '%s': %v", chargerID, err)
			charger.subscribeFailed = true
		}

		// One retry chain per charger. Every reconnect enqueues a subscription for every
		// charger, so without this a charger that keeps failing accumulated one more
		// self-perpetuating chain - and one more blocking invoke per cycle - per reconnect.
		if !charger.retryArmed {
			charger.retryArmed = true

			go m.addChargerSubscription(chargerID, charger)
		}

		return nil
	}

	charger.backoff.Reset()
	charger.isSubscribed = true
	charger.subscribeFailed = false

	log.Debugf("signalR: subscribed charger '%s'", chargerID)
	return nil
}

func (m *manager) addChargerSubscription(chargerID string, charger *charger) {
	m.mu.Lock()
	timer := time.NewTimer(charger.backoff.Next())
	done := m.done
	epoch := m.epoch
	m.mu.Unlock()

	defer timer.Stop()

	// Disarmed on every exit path - a charger left armed can never schedule another retry,
	// which strands it worse than the duplicate chains the flag exists to prevent - but never
	// via defer: enqueueSubscription returns as soon as the run loop takes the ID, and that
	// same loop may already have armed the next chain by then. A deferred disarm would clear
	// that fresh flag, so the failure after it would arm a duplicate.
	select {
	case <-done:
		m.disarmRetry(charger, epoch)
	case <-timer.C:
		m.disarmRetry(charger, epoch)
		m.enqueueSubscription(chargerID)
	}
}

// disarmRetry clears the flag only for a chain still on the current connection: a retired
// timer firing later would otherwise disarm the fresh chain armed since.
func (m *manager) disarmRetry(charger *charger, epoch uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if epoch != m.epoch {
		return
	}

	charger.retryArmed = false
}

func (m *manager) handleClientState(state model.ClientState) {
	switch state {
	case model.ClientStateConnected:
		log.Info("signalR: client connected")

		m.mu.Lock()
		chargersIDs := make([]string, 0, len(m.chargers))

		// A subscribe still in flight belongs to the previous connection: its result must not
		// mark this one subscribed, and it must not block the re-subscribe enqueued below.
		m.epoch++

		for chargerID, charger := range m.chargers {
			// A subscription belongs to the connection that made it. Clearing the flag here
			// rather than trusting the disconnect notice makes this sweep the authority: a
			// missed notice would otherwise leave the charger permanently unsubscribed.
			charger.isSubscribed = false
			charger.subscribing = false
			// A chain armed on the previous connection sleeps on a backoff of up to ten minutes,
			// which reconnecting resets without waking it. Left set, the flag blocked the sweep's
			// own failure from arming a fresh chain for the rest of that stale delay.
			charger.retryArmed = false
			chargersIDs = append(chargersIDs, chargerID)
			// A new connection starts a new failure streak, so its first failure has to warn
			// again - otherwise the only visible sign of a charger that never comes back is a
			// debug line the hub does not log. Cleared here rather than on disconnect: a retry
			// armed before the disconnect would otherwise fail during the outage and take the
			// warning with it.
			charger.subscribeFailed = false
		}

		m.mu.Unlock()

		// Offload the re-subscription sends: this runs on the run() goroutine, which is
		// also the only drainer of m.subscriptions, so sending inline could self-block.
		go func() {
			for _, chargerID := range chargersIDs {
				m.enqueueSubscription(chargerID)
			}
		}()

	case model.ClientStateDisconnected:
		log.Warn("signalR: client disconnected")

		m.mu.Lock()

		// A subscribe that returns after this point describes the connection just lost, so it
		// must not mark the charger subscribed on it.
		m.epoch++

		for _, charger := range m.chargers {
			charger.backoff.Reset()
			charger.isSubscribed = false
			// Connected() counts an in-flight subscribe as connected; left set, it would keep
			// reporting a dead connection healthy until the invoke times out.
			charger.subscribing = false
			// Retiring the epoch above also makes disarmRetry a no-op for every chain armed on
			// the lost connection, so clear the flag here or nothing ever arms a retry again.
			charger.retryArmed = false
		}

		m.mu.Unlock()

	default:
		log.Warnf("Unknown client state %v", state)
	}
}

// handleObservation hands the observation to the charger's own dispatcher rather than running
// the handler here: this is the sole drainer of the client's observation channel, and a handler
// blocks on the chargepoint service mutex whenever a command is mid-flight - which is how a
// successful cmd.charge.start came back reported as failed, the command timing out waiting for
// the very observation stuck behind it.
func (m *manager) handleObservation(observation model.Observation) {
	if !observation.ID.Supported() {
		return
	}

	m.mu.RLock()
	charger, ok := m.chargers[observation.ChargerID]
	done := m.done

	var dispatchDone chan struct{}
	if ok {
		dispatchDone = charger.dispatchDone
	}

	m.mu.RUnlock()

	if !ok {
		log.Warnf("Handle observation chargerID=%s err: no handler", observation.ChargerID)

		return
	}

	// Dropped rather than blocking the loop when a charger falls far enough behind to fill its
	// queue: waiting here would stall every other charger's stream, which is the failure this
	// dispatch exists to prevent. Safe for the state observations, which carry absolute values,
	// so the next one for the same ID supersedes whatever was dropped.
	//
	// The session edges are the exception: their handlers persist a session record, so a
	// dropped one is a charging session missing from the history for good - there is no later
	// observation carrying the same event. Those block until the queue drains, or the manager
	// stops. A charger far enough behind to fill 100 slots is already degraded; losing its
	// billing history to keep the loop moving is the worse trade.
	//
	// dispatchDone escapes the third case: an Unregister landing after the RUnlock above stops
	// the dispatcher, and a blocking send on a queue nothing drains any more would park the run
	// loop for good - stalling every other charger, the failure this dispatch exists to prevent.
	if observation.ID == model.ChargingSessionStart || observation.ID == model.ChargingSessionStop {
		select {
		case charger.observations <- observation:
		case <-dispatchDone:
		case <-done:
		}

		return
	}

	select {
	case charger.observations <- observation:
	case <-done:
	default:
		log.Warnf("Drop observation chargerID=%s obs='%s': dispatch queue full",
			observation.ChargerID, observation.ID.Str())
	}
}

// dispatchObservations runs one charger's handler, in arrival order, until the charger is
// unregistered or the manager stops.
func (m *manager) dispatchObservations(chargerID string, c *charger) {
	defer telemetry.RecoverAndEmit(m.tel, "manager.dispatch", true)

	for {
		select {
		// Two exits, because the charger can go away either way round: dispatchDone for its own
		// Unregister, done for a Stop that takes the whole manager with it. Watching only the
		// first leaked a goroutine per registered charger on every Stop, since Stop never
		// unregisters. Closing dispatchDone from Stop instead would race Unregister's close.
		case <-c.dispatchDone:
			return
		case observation := <-c.observations:
			if err := c.handler.HandleObservation(observation); err != nil {
				log.Warnf("Handle observation chargerID=%s obs='%s' err: %v",
					chargerID, observation.ID.Str(), err)
			}
		}
	}
}

type charger struct {
	handler         Handler
	isSubscribed    bool
	subscribing     bool
	subscribeFailed bool
	retryArmed      bool
	backoff         backoff.Stateful

	// Observations for this charger are handled on their own goroutine, off the single run
	// loop, so a handler that blocks - every Send* goes through a service mutex a command may
	// already hold - stops draining only its own charger rather than every charger's stream.
	// One goroutine per charger, not per observation: the handler advances cached state, so
	// its observations have to stay in the order they arrived.
	observations chan model.Observation
	dispatchDone chan struct{}
	dispatchOnce sync.Once
}

// stopDispatch ends this charger's observation dispatcher. Both Unregister and Stop reach it -
// a charger can go away on its own or with the whole manager - so the close is guarded.
// dispatchDone is nil for a charger a test assembled directly rather than through Register.
func (c *charger) stopDispatch() {
	if c.dispatchDone == nil {
		return
	}

	c.dispatchOnce.Do(func() { close(c.dispatchDone) })
}
