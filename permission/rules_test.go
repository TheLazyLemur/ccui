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

func TestPermissionRules_ReadOnly(t *testing.T) {
	a := assert.New(t)
	rules := ReadOnlyRules()

	// read tools allowed
	for _, tool := range []string{"Read", "Glob", "Grep", "WebSearch", "WebFetch"} {
		a.Equal(Allow, rules.Check(tool, ""), "tool %s should be allowed", tool)
	}
	// write tools denied
	for _, tool := range []string{"Write", "Edit", "NotebookEdit", "Bash"} {
		a.Equal(Deny, rules.Check(tool, ""), "tool %s should be denied", tool)
	}
	// unknown denied
	a.Equal(Deny, rules.Check("Unknown", ""))
}

func TestPermissionRules_WorkspaceWrite(t *testing.T) {
	a := assert.New(t)
	rules := WorkspaceWriteRules()

	// read tools allowed
	for _, tool := range []string{"Read", "Glob", "Grep", "WebSearch", "WebFetch"} {
		a.Equal(Allow, rules.Check(tool, ""), "tool %s should be allowed", tool)
	}
	// write/edit allowed
	for _, tool := range []string{"Write", "Edit", "NotebookEdit"} {
		a.Equal(Allow, rules.Check(tool, ""), "tool %s should be allowed", tool)
	}
	// bash denied
	a.Equal(Deny, rules.Check("Bash", ""))
	// unknown denied
	a.Equal(Deny, rules.Check("Unknown", ""))
}
