package test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/futurehomeno/cliffhanger/manifest"
)

func LoadManifest(t *testing.T) *manifest.Manifest {
	t.Helper()

	f, err := os.ReadFile("./../../../testdata/defaults/app-manifest.json")
	if err != nil {
		t.Fatalf("failed to load manifest from file: %+v", err)
	}

	mf := manifest.New()

	err = json.Unmarshal(f, mf)
	if err != nil {
		t.Fatalf("failed to unmarshal manifest: %+v", err)
	}

	return mf
}
