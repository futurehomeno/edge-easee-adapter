package signalr

import (
	"fmt"
	"reflect"

	"github.com/philippseith/signalr"
	log "github.com/sirupsen/logrus"
)

type logger struct{}

// NewLogger returns SignalR logging adapter for Logrus.
func NewLogger() signalr.StructuredLogger {
	return logger{}
}

func (l logger) Log(keyVals ...interface{}) error {
	if len(keyVals)%2 != 0 {
		log.Warn("signalr logger: args count is odd, skipping...")

		return nil
	}

	e := log.NewEntry(log.StandardLogger())

	for i := 0; i < len(keyVals); i += 2 {
		key, ok := keyVals[i].(string)
		if !ok {
			continue
		}

		if key == "level" {
			continue
		}

		e = e.WithField(key, keyVals[i+1])
	}

	level := l.detectLevel(keyVals)

	e.Log(level, "signalR client log")

	return nil
}

func (l logger) detectLevel(args []interface{}) log.Level {
	for i, arg := range args {
		a, ok := arg.(string)
		if !ok {
			continue
		}

		if a != "level" {
			continue
		}

		level, ok := args[i+1].(fmt.Stringer)
		if !ok {
			log.Error("log level not a string: ", reflect.TypeOf(args[i+1]))

			continue
		}

		l, err := log.ParseLevel(level.String())
		if err != nil {
			return log.DebugLevel
		}

		return l
	}

	return log.DebugLevel
}
