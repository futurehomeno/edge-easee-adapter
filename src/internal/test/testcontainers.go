package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupMQTTContainer creates a new MQTT container and returns its address.
func SetupMQTTContainer(t *testing.T) string {
	t.Helper()

	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "eclipse-mosquitto:1.6.8", //nolint:misspell
			ExposedPorts: []string{"1883/tcp"},
			WaitingFor:   wait.ForLog("Opening ipv4 listen socket on port 1883"),
		},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, container.Terminate(ctx))
	})

	addr, err := container.Endpoint(ctx, "tcp")
	require.NoError(t, err)

	return addr
}
