package cmd

import (
	"encoding/json"
	"os"
	"testing"

	cliffAdapter "github.com/futurehomeno/cliffhanger/adapter"
	cliffApp "github.com/futurehomeno/cliffhanger/app"
	cliffConfig "github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/debug"
	"github.com/futurehomeno/cliffhanger/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackagedManifest_DeclaresExactlyTheRoutedInterfaces(t *testing.T) {
	t.Parallel()

	b, err := os.ReadFile("../../package/debian/usr/share/futurehome/easee/defaults/app-manifest.json")
	require.NoError(t, err)

	var m manifest.Manifest
	require.NoError(t, json.Unmarshal(b, &m))
	require.Len(t, m.Services, 1)

	declared := map[string]string{}
	for _, i := range m.Services[0].Interfaces {
		declared[i.MsgType] = i.Type + " " + string(i.ValueType)
	}

	require.Len(t, m.Services[0].Interfaces, len(declared), "a message is declared twice")
	assert.Equal(t, map[string]string{
		// bootstrap.DefaultRoute: the config report and debug.Route
		cliffConfig.CmdConfigGetReport:  "in null",
		cliffConfig.EvtConfigReport:     "out object",
		debug.CmdLogGetLevel:            "in null",
		debug.CmdLogSetLevel:            "in string",
		debug.EvtLogLevelReport:         "out string",
		debug.CmdLogGetFormat:           "in null",
		debug.CmdLogSetFormat:           "in string",
		debug.EvtLogFormatReport:        "out string",
		debug.CmdLogGetFile:             "in null",
		debug.CmdLogSetFile:             "in string",
		debug.EvtLogFileReport:          "out string",
		debug.CmdLogGetRevertTimeout:    "in null",
		debug.CmdLogSetRevertTimeout:    "in string",
		debug.EvtLogRevertTimeoutReport: "out string",
		// app.RouteApp with Login and Logout
		cliffApp.CmdAppGetState:             "in null",
		cliffApp.EvtAppStateReport:          "out object",
		cliffApp.CmdConfigGetExtendedReport: "in null",
		cliffApp.EvtConfigExtendedReport:    "out object",
		cliffApp.CmdAppGetManifest:          "in string",
		cliffApp.EvtAppManifestReport:       "out object",
		cliffApp.CmdConfigExtendedSet:       "in object",
		cliffApp.EvtAppConfigReport:         "out object",
		cliffApp.CmdAppUninstall:            "in null",
		cliffApp.EvtAppUninstallReport:      "out object",
		cliffApp.CmdAppDiagGetReport:        "in null",
		cliffApp.EvtAppDiagReport:           "out object",
		cliffApp.CmdAuthLogin:               "in object",
		cliffApp.CmdAuthLogout:              "in null",
		cliffApp.EvtAuthStatusReport:        "out object",
		// adapter.RouteAdapter
		cliffAdapter.CmdThingGetInclusionReport: "in string",
		cliffAdapter.EvtThingInclusionReport:    "out object",
		cliffAdapter.CmdThingDelete:             "in str_map",
		cliffAdapter.EvtThingExclusionReport:    "out object",
		cliffAdapter.CmdNetworkReset:            "in null",
		cliffAdapter.EvtNetworkResetDone:        "out null",
		cliffAdapter.CmdNetworkGetNode:          "in string",
		cliffAdapter.EvtNetworkNodeReport:       "out object",
		cliffAdapter.CmdNetworkGetAllNodes:      "in null",
		cliffAdapter.EvtNetworkAllNodesReport:   "out object",
		cliffAdapter.CmdPingSend:                "in string",
		cliffAdapter.EvtPingReport:              "out object",
	}, declared)
}
