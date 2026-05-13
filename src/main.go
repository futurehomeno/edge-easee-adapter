package main

import (
	"regexp"
	"runtime/pprof"
	"strings"

	"github.com/futurehomeno/edge-easee-adapter/cmd"
	log "github.com/sirupsen/logrus"
)

var (
	Version     string
	PackageName string
)

func main() {
	err := cmd.Execute(PackageName, Version)
	s := strings.Builder{}
	if err := pprof.Lookup("goroutine").WriteTo(&s, 2); err == nil {
		log.Infof("%s\n", filterGoroutinesByKeywords(s.String()))
	}

	if err != nil {
		log.Fatal(err)
	}
}

func filterGoroutinesByKeywords(input string) string {
	keywordRE := regexp.MustCompile(`(?i)\b(mutex|semaphore|panic|lock)\b`)

	lines := strings.Split(input, "\n")

	var (
		block    []string
		matched  bool
		out      []string
		inGorout bool
	)

	flush := func() {
		if inGorout && matched && len(block) > 0 {
			out = append(out, block...)
			out = append(out, "")
		}
		block = block[:0]
		matched = false
		inGorout = false
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "goroutine ") {
			flush()
			inGorout = true
			block = append(block, line)
			continue
		}

		if !inGorout {
			continue
		}

		block = append(block, line)

		if keywordRE.MatchString(line) {
			matched = true
		}
	}

	flush()

	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}

	return strings.Join(out, "\n")
}
