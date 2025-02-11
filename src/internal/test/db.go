package test

import (
	"os"
	"testing"

	"github.com/futurehomeno/cliffhanger/database"
)

func NewDatabase(t *testing.T, cleanup bool) database.Database {
	t.Helper()

	if cleanup {
		_ = os.RemoveAll("../testdata/database")
	}

	db, err := database.NewDatabase(
		"../testdata/database",
		database.WithFilename("test"),
		database.WithCompactionPercentage(50),
		database.WithCompactionSize(100*1024),
	)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	err = db.Start()
	if err != nil {
		t.Fatalf("failed to start database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Stop()

		_ = os.RemoveAll("../testdata/database")
	})

	return db
}
