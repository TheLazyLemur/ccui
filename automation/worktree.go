package automation

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// CreateWorktree creates a git worktree for an automation run.
// Returns the worktree directory path.
func CreateWorktree(repoDir, runID string) (string, error) {
	wtDir := filepath.Join(repoDir, ".ccui-worktrees", runID)

	cmd := exec.Command("git", "worktree", "add", "--detach", wtDir)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git worktree add: %s: %w", string(out), err)
	}
	return wtDir, nil
}

// RemoveWorktree removes a git worktree created by CreateWorktree
func RemoveWorktree(repoDir, wtDir string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", wtDir)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		// best-effort: worktree may already be gone
		_ = out
	}
	return nil
}
