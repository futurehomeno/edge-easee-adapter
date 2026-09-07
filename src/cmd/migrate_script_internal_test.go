package cmd

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const migrateScript = "../../package/debian/usr/lib/futurehome/easee/migrate.sh"

// legacyTree stands in for the four packaged paths migrate.sh moves state between.
type legacyTree struct {
	oldData, oldLogs, newData, newLogs string
}

func newLegacyTree(t *testing.T) legacyTree {
	t.Helper()

	if os.Getuid() == 0 {
		t.Skip("migrate.sh refuses to run as root")
	}

	root := t.TempDir()
	tree := legacyTree{
		oldData: filepath.Join(root, "opt/thingsplex/easee"),
		oldLogs: filepath.Join(root, "var/log/thingsplex/easee"),
		newData: filepath.Join(root, "var/lib/futurehome/easee"),
		newLogs: filepath.Join(root, "var/log/futurehome/easee"),
	}

	for _, dir := range []string{tree.oldData, tree.oldLogs, tree.newData, tree.newLogs} {
		require.NoError(t, os.MkdirAll(dir, 0o750))
	}

	return tree
}

func (l legacyTree) run(t *testing.T) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "sh", migrateScript)
	cmd.Env = append(os.Environ(), "OLD_DATA="+l.oldData, "OLD_LOGS="+l.oldLogs, "NEW_DATA="+l.newData, "NEW_LOGS="+l.newLogs)

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func (l legacyTree) legacyConfig() string {
	return `{"work_dir":"` + l.oldData + `","log_file":"` + l.oldLogs + `/easee.log","accessToken":"a"}`
}

func (l legacyTree) migratedConfig() string {
	return `{"work_dir":"` + l.newData + `","log_file":"` + l.newLogs + `/easee.log","accessToken":"a"}`
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	body, err := os.ReadFile(path) //nolint:gosec
	require.NoError(t, err)

	return string(body)
}

func TestMigrateScript_copiesLegacyState(t *testing.T) {
	t.Parallel()

	tree := newLegacyTree(t)
	writeFile(t, filepath.Join(tree.oldData, "data/config.json"), tree.legacyConfig())
	writeFile(t, filepath.Join(tree.oldData, "data/config.json.bak"), tree.legacyConfig())
	writeFile(t, filepath.Join(tree.oldData, "data/adapter.json"), `{"things":{}}`)
	writeFile(t, filepath.Join(tree.oldData, "data.db"), "sessions")
	writeFile(t, filepath.Join(tree.oldLogs, "easee.log"), "old log")
	// postinst pre-touches an empty log before the script runs.
	writeFile(t, filepath.Join(tree.newLogs, "easee.log"), "")

	tree.run(t)

	assert.Equal(t, tree.migratedConfig(), readFile(t, filepath.Join(tree.newData, "data/config.json")))
	assert.Equal(t, tree.migratedConfig(), readFile(t, filepath.Join(tree.newData, "data/config.json.bak")), "the .bak fallback is rewritten too")
	assert.Equal(t, `{"things":{}}`, readFile(t, filepath.Join(tree.newData, "data/adapter.json")))
	assert.Equal(t, "sessions", readFile(t, filepath.Join(tree.newData, "data.db")))
	assert.Equal(t, "old log", readFile(t, filepath.Join(tree.newLogs, "easee.log")))
	assert.Equal(t, tree.legacyConfig(), readFile(t, filepath.Join(tree.oldData, "data/config.json")), "the legacy tree stays for a downgrade")
	assert.NoDirExists(t, filepath.Join(tree.newData, ".data.tmp"))

	require.NoError(t, filepath.WalkDir(tree.newData, func(path string, _ fs.DirEntry, err error) error {
		require.NoError(t, err)

		info, err := os.Lstat(path)
		require.NoError(t, err)
		assert.Zero(t, info.Mode().Perm()&0o007, "%s must be closed to others", path)

		return nil
	}))

	tree.run(t)
	assert.Equal(t, tree.migratedConfig(), readFile(t, filepath.Join(tree.newData, "data/config.json")), "a re-run is idempotent")
}

// Once data/, data.db or a written log exist in the new tree the service owns them: a retried
// postinst must not roll the hub back to the legacy copies.
func TestMigrateScript_neverOverwritesExistingState(t *testing.T) {
	t.Parallel()

	tree := newLegacyTree(t)
	writeFile(t, filepath.Join(tree.oldData, "data/config.json"), tree.legacyConfig())
	writeFile(t, filepath.Join(tree.oldData, "data.db"), "legacy sessions")
	writeFile(t, filepath.Join(tree.oldLogs, "easee.log"), "old log")
	writeFile(t, filepath.Join(tree.newData, "data/config.json"), `{"work_dir":"`+tree.oldData+`"}`)
	writeFile(t, filepath.Join(tree.newData, "data.db"), "current sessions")
	writeFile(t, filepath.Join(tree.newLogs, "easee.log"), "current log")

	tree.run(t)

	assert.Equal(t, `{"work_dir":"`+tree.newData+`"}`, readFile(t, filepath.Join(tree.newData, "data/config.json")), "kept, only paths rewritten")
	assert.Equal(t, "current sessions", readFile(t, filepath.Join(tree.newData, "data.db")))
	assert.Equal(t, "current log", readFile(t, filepath.Join(tree.newLogs, "easee.log")))
}

// dpkg removes an emptied legacy data/ on purge while data.db outlives it at the work dir root.
func TestMigrateScript_sessionsWithoutDataDir(t *testing.T) {
	t.Parallel()

	tree := newLegacyTree(t)
	writeFile(t, filepath.Join(tree.oldData, "data.db"), "sessions")

	tree.run(t)

	assert.Equal(t, "sessions", readFile(t, filepath.Join(tree.newData, "data.db")))
	assert.NoDirExists(t, filepath.Join(tree.newData, "data"), "postinst creates data/ afterwards")
}

func TestMigrateScript_skipsSymlinkedSessionsAndLog(t *testing.T) {
	t.Parallel()

	tree := newLegacyTree(t)
	writeFile(t, filepath.Join(tree.oldData, "data/config.json"), tree.legacyConfig())
	writeFile(t, filepath.Join(tree.oldData, "elsewhere"), "planted")
	require.NoError(t, os.Symlink(filepath.Join(tree.oldData, "elsewhere"), filepath.Join(tree.oldData, "data.db")))
	require.NoError(t, os.Symlink(filepath.Join(tree.oldData, "elsewhere"), filepath.Join(tree.oldLogs, "easee.log")))

	tree.run(t)

	assert.NoFileExists(t, filepath.Join(tree.newData, "data.db"))
	assert.NoFileExists(t, filepath.Join(tree.newLogs, "easee.log"))
	assert.False(t, strings.Contains(readFile(t, filepath.Join(tree.newData, "data/config.json")), tree.oldData))
}
