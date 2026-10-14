package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	htpasswd "github.com/krabt/go-htpasswd"
	"github.com/stretchr/testify/require"
)

func TestCreateUpdateVerifyAndDelete(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "users")
	var stdout, stderr bytes.Buffer

	require.NoError(t, run([]string{"-c", "-b", "-B", "-C", "4", filename, "alice", "first"}, strings.NewReader(""), &stdout, &stderr))
	info, err := os.Stat(filename)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	assertPassword(t, filename, "alice", "first")

	require.NoError(t, run([]string{"-b", "-s", filename, "bob", "second"}, strings.NewReader(""), &stdout, &stderr))
	require.NoError(t, run([]string{"-b", filename, "alice", "changed"}, strings.NewReader(""), &stdout, &stderr))
	assertPassword(t, filename, "alice", "changed")
	assertPassword(t, filename, "bob", "second")

	stdout.Reset()
	require.NoError(t, run([]string{"-v", "-b", filename, "alice", "changed"}, strings.NewReader(""), &stdout, &stderr))
	require.Contains(t, stdout.String(), "correct")
	require.Error(t, run([]string{"-v", "-b", filename, "alice", "wrong"}, strings.NewReader(""), &stdout, &stderr))

	require.NoError(t, run([]string{"-D", filename, "alice"}, strings.NewReader(""), &stdout, &stderr))
	file, err := htpasswd.New(filename, htpasswd.DefaultSystems, nil)
	require.NoError(t, err)
	require.False(t, file.Exists("alice"))
	require.True(t, file.Exists("bob"))
}

func TestStdoutAndStdinModes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	require.NoError(t, run([]string{"-n", "-i", "-5", "alice"}, strings.NewReader("secret\n"), &stdout, &stderr))
	line := strings.TrimSpace(stdout.String())
	require.True(t, strings.HasPrefix(line, "alice:$6$"))

	file, err := htpasswd.NewFromReader(strings.NewReader(line+"\n"), htpasswd.DefaultSystems, nil)
	require.NoError(t, err)
	require.True(t, file.Match("alice", "secret"))
}

func TestInteractivePasswordConfirmation(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "users")
	var stdout, stderr bytes.Buffer
	require.NoError(t, run([]string{"-c", filename, "alice"}, strings.NewReader("secret\nsecret\n"), &stdout, &stderr))
	assertPassword(t, filename, "alice", "secret")
	require.ErrorContains(t, run([]string{"-c", filename, "alice"}, strings.NewReader("one\ntwo\n"), &stdout, &stderr), "do not match")
}

func TestRejectsInvalidArguments(t *testing.T) {
	tests := [][]string{
		{"-n", "-D", "alice"},
		{"-b", "-i", "file", "alice", "password"},
		{"-B", "-C", "3", "file", "alice"},
		{"-m", "-s", "file", "alice"},
		{"-n", "bad:user"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			require.Error(t, run(args, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}))
		})
	}
}

func assertPassword(t *testing.T, filename, username, password string) {
	t.Helper()
	file, err := htpasswd.New(filename, htpasswd.DefaultSystems, nil)
	require.NoError(t, err)
	require.True(t, file.Match(username, password))
}
