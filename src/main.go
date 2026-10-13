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
	if err == nil {
		return
	}

	// Dumped only on a failed exit, where a goroutine stuck on a lock is the likely cause.
	s := strings.Builder{}
	if dumpErr := pprof.Lookup("goroutine").WriteTo(&s, 2); dumpErr == nil {
		log.Infof("%s\n", utils.FilterGoroutinesByKeywords(s.String(), []string{"mutex", "semaphore", "panic", "lock"}))
	}

	log.Error(err)
	os.Exit(1)
}
