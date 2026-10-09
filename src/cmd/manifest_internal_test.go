package cmd

import (
	"encoding/json"
	"os"
	"testing"

	cliffAdapter "github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/debug"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackagedManifest_DeclaresRoutedLogAndNodeInterfaces(t *testing.T) {
	t.Parallel()

	b, err := os.ReadFile("../../package/debian/usr/share/futurehome/easee/defaults/app-manifest.json")
	require.NoError(t, err)

	var m struct {
		Services []struct {
			Interfaces []struct {
				MsgType string `json:"msg_t"`
			} `json:"interfaces"`
		} `json:"services"`
	}
	require.NoError(t, json.Unmarshal(b, &m))

	var declared []string
	for _, s := range m.Services {
		for _, i := range s.Interfaces {
			declared = append(declared, i.MsgType)
		}
	}

	for _, msgType := range []string{
		debug.CmdLogSetLevel, debug.EvtLogLevelReport,
		cliffAdapter.CmdNetworkGetNode, cliffAdapter.EvtNetworkNodeReport,
	} {
		assert.Contains(t, declared, msgType)
	}
}
