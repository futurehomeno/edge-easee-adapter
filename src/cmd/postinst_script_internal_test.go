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

	out, calls := runAddUserAndGroup(t, "")

	assert.Equal(t, "usermod -g futurehome easee\n", calls)
	assert.NotContains(t, out, "Warning")
}

func TestPostinst_FailedGroupMoveWarns(t *testing.T) {
	t.Parallel()

	out, calls := runAddUserAndGroup(t, "exit 1")

	assert.Equal(t, "usermod -g futurehome easee\n", calls)
	assert.Contains(t, out, "Warning: could not move easee to the futurehome group")
}

func runAddUserAndGroup(t *testing.T, usermodExit string) (out, calls string) {
	t.Helper()

	bin := t.TempDir()
	callsFile := filepath.Join(bin, "calls")

	for name, body := range map[string]string{
		"getent":   "exit 0",
		"usermod":  `echo "usermod $*" >> ` + callsFile + "\n" + usermodExit,
		"adduser":  `echo "adduser $*" >> ` + callsFile,
		"addgroup": `echo "addgroup $*" >> ` + callsFile,
	} {
		require.NoError(t, os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755)) //nolint:gosec // test stub must be executable
	}

	// Only the functions: the dispatch at the bottom would run the whole configure as root.
	cmd := exec.CommandContext(t.Context(), "sh", "-c", `eval "$(sed '/^case /,$d' "$0")"; add_user_and_group`, postinstScript)
	cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	b, err := cmd.CombinedOutput()
	require.NoError(t, err, string(b))

	got, err := os.ReadFile(callsFile) //nolint:gosec
	require.NoError(t, err, "nothing was called")

	return string(b), string(got)
}
