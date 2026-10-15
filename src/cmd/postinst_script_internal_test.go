package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const postinstScript = "../../package/debian/DEBIAN/postinst"

func TestPostinst_UpgradedUserMovesToFuturehomeGroup(t *testing.T) {
	t.Parallel()

	_, calls := runAddUserAndGroup(t, "")

	assert.Equal(t, "usermod -g futurehome easee\n", calls)
}

func TestPostinst_FailedGroupMoveWarns(t *testing.T) {
	t.Parallel()

	out, calls := runAddUserAndGroup(t, "exit 1")

	assert.Equal(t, "usermod -g futurehome easee\n", calls)
	assert.Contains(t, out, "Warning: could not move easee to the futurehome group")
}

func TestPostinst_DataModesGiveMigratedFilesFuturehomeGroup(t *testing.T) {
	t.Parallel()

	_, calls := runPostinst(t, "set_data_modes", map[string]string{
		"chown": `echo "chown $*" >> "$CALLS"`,
		"find":  "exit 0",
		"rm":    "exit 0",
	})

	assert.Equal(t, "chown -R easee:futurehome /var/lib/futurehome/easee\n", calls)
}

func TestPostinst_FailedDataChownWarns(t *testing.T) {
	t.Parallel()

	out, _ := runPostinst(t, "set_data_modes", map[string]string{
		"chown": `echo "chown $*" >> "$CALLS"` + "\nexit 1",
		"find":  "exit 0",
		"rm":    "exit 0",
	})

	assert.Contains(t, out, "Warning: could not give the migrated data group futurehome")
}

func runAddUserAndGroup(t *testing.T, usermodExit string) (out, calls string) {
	t.Helper()

	return runPostinst(t, "add_user_and_group", map[string]string{
		"getent":   "exit 0",
		"usermod":  `echo "usermod $*" >> "$CALLS"` + "\n" + usermodExit,
		"adduser":  `echo "adduser $*" >> "$CALLS"`,
		"addgroup": `echo "addgroup $*" >> "$CALLS"`,
	})
}

func runPostinst(t *testing.T, fn string, stubs map[string]string) (out, calls string) {
	t.Helper()

	bin := t.TempDir()
	callsFile := filepath.Join(bin, "calls")

	for name, body := range stubs {
		require.NoError(t, os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755)) //nolint:gosec // test stub must be executable
	}

	// Only the functions: the dispatch at the bottom would run the whole configure as root.
	cmd := exec.CommandContext(t.Context(), "sh", "-c", `eval "$(sed '/^case /,$d' "$0")"; "$1"`, postinstScript, fn) //nolint:gosec // fn names a postinst function from the tests
	cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "CALLS="+callsFile)

	b, err := cmd.CombinedOutput()
	require.NoError(t, err, string(b))

	got, err := os.ReadFile(callsFile) //nolint:gosec
	require.NoError(t, err, "nothing was called")

	return string(b), string(got)
}
