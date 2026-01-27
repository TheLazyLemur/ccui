package permission

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPermissionRules_ReadAllowed(t *testing.T) {
	a := assert.New(t)

	// given
	rules := DefaultRules()

	// when/then - safe tools should be allowed without asking
	safeTools := []string{"Read", "Glob", "Grep", "WebSearch", "WebFetch"}
	for _, tool := range safeTools {
		decision := rules.Check(tool, "any input")
		a.Equal(Allow, decision, "tool %s should be allowed", tool)
	}
}

func TestPermissionRules_WriteAsks(t *testing.T) {
	a := assert.New(t)

	// given
	rules := DefaultRules()

	// when/then - write tools should ask for permission
	writeTools := []string{"Write", "Edit", "NotebookEdit"}
	for _, tool := range writeTools {
		decision := rules.Check(tool, "any input")
		a.Equal(Ask, decision, "tool %s should ask", tool)
	}
}

func TestPermissionRules_BashAllowsSafe(t *testing.T) {
	a := assert.New(t)

	// given
	rules := DefaultRules()

	// when/then - bash should ask by default
	decision := rules.Check("Bash", "git status")
	a.Equal(Ask, decision, "bash should ask by default")

	decision = rules.Check("Bash", "rm -rf /")
	a.Equal(Ask, decision, "bash should ask for dangerous commands")
}

func TestPermissionRules_UnknownToolDenied(t *testing.T) {
	a := assert.New(t)

	// given
	rules := DefaultRules()

	// when/then - unknown tools should be denied
	decision := rules.Check("UnknownTool", "any input")
	a.Equal(Deny, decision, "unknown tools should be denied")
}
