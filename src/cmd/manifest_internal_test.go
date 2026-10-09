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
				Dir     string `json:"intf_t"`
				MsgType string `json:"msg_t"`
			} `json:"interfaces"`
		} `json:"services"`
	}
	require.NoError(t, json.Unmarshal(b, &m))

	declared := map[string]string{}
	for _, s := range m.Services {
		for _, i := range s.Interfaces {
			declared[i.MsgType] = i.Dir
		}
	}

	for msgType, dir := range map[string]string{
		debug.CmdLogGetLevel:              "in",
		debug.CmdLogSetLevel:              "in",
		debug.EvtLogLevelReport:           "out",
		cliffAdapter.CmdNetworkGetNode:    "in",
		cliffAdapter.EvtNetworkNodeReport: "out",
	} {
		assert.Equal(t, dir, declared[msgType], msgType)
	}
}
