package main

import (
	"os"
	"runtime/pprof"
	"strings"

	"github.com/futurehomeno/cliffhanger/utils"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/cmd"
)

var (
	PackageName string
	Version     string
)

func main() {
	err := cmd.Execute(PackageName, Version)
	s := strings.Builder{}
	if dumpErr := pprof.Lookup("goroutine").WriteTo(&s, 2); dumpErr == nil {
		log.Infof("%s\n", utils.FilterGoroutinesByKeywords(s.String(), []string{"mutex", "semaphore", "panic", "lock"}))
	}

	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
