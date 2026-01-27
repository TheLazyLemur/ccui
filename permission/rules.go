package permission

// Decision represents the outcome of a permission check
type Decision int

const (
	Allow Decision = iota // tool can proceed
	Ask                   // must request user permission
	Deny                  // reject immediately
)

// RuleSet determines permissions for tool calls
type RuleSet struct {
	rules map[string]Decision
}

// Check returns the decision for a given tool
func (r *RuleSet) Check(tool, input string) Decision {
	if d, ok := r.rules[tool]; ok {
		return d
	}
	return Deny
}

// DefaultRules returns standard permission rules
func DefaultRules() *RuleSet {
	return &RuleSet{
		rules: map[string]Decision{
			// Safe tools - auto-allow
			"Read":      Allow,
			"Glob":      Allow,
			"Grep":      Allow,
			"WebSearch": Allow,
			"WebFetch":  Allow,
			// Write tools - ask
			"Write":        Ask,
			"Edit":         Ask,
			"NotebookEdit": Ask,
			"Bash":         Ask,
		},
	}
}
