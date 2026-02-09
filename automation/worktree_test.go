package automation

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}

	// Create a file and initial commit
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644))
	cmds = append(cmds,
		[]string{"git", "add", "."},
		[]string{"git", "commit", "-m", "init"},
	)

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, string(out))
	}

	return dir
}

func TestWorktree_CreateAndRemove(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	repoDir := setupGitRepo(t)

	// when
	wtDir, err := CreateWorktree(repoDir, "test-run-1")
	r.NoError(err)

	// then
	a.DirExists(wtDir)
	a.FileExists(filepath.Join(wtDir, "README.md"))

	// cleanup
	err = RemoveWorktree(repoDir, wtDir)
	r.NoError(err)
	a.NoDirExists(wtDir)
}

func TestWorktree_CreateInNonGitDir(t *testing.T) {
	dir := t.TempDir()
	_, err := CreateWorktree(dir, "run-1")
	assert.Error(t, err)
}

func TestWorktree_RemoveNonexistent(t *testing.T) {
	repoDir := setupGitRepo(t)
	err := RemoveWorktree(repoDir, "/nonexistent/path")
	// should not error fatally, just log
	assert.NoError(t, err)
}
