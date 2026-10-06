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
	bin := t.TempDir()
	calls := filepath.Join(bin, "calls")

	for name, body := range map[string]string{
		"getent":   "exit 0",
		"usermod":  `echo "usermod $*" >> ` + calls,
		"adduser":  `echo "adduser $*" >> ` + calls,
		"addgroup": `echo "addgroup $*" >> ` + calls,
	} {
		require.NoError(t, os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755)) //nolint:gosec // test stub must be executable
	}

	// Only the functions: the dispatch at the bottom would run the whole configure as root.
	cmd := exec.CommandContext(t.Context(), "sh", "-c", `eval "$(sed '/^case /,$d' "$0")"; add_user_and_group`, postinstScript)
	cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	got, err := os.ReadFile(calls) //nolint:gosec
	require.NoError(t, err, "nothing was called")
	assert.Equal(t, "usermod -g futurehome easee\n", string(got))
}
