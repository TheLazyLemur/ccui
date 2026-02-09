package automation

// PermissionLevel controls what tools an automation can use
type PermissionLevel string

const (
	PermReadOnly       PermissionLevel = "read_only"
	PermWorkspaceWrite PermissionLevel = "workspace_write"
	PermFullAccess     PermissionLevel = "full_access"
)

// Automation defines a recurring agent task
type Automation struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Prompt          string          `json:"prompt"`
	Schedule        string          `json:"schedule"`
	ProjectDir      string          `json:"projectDir"`
	BackendType     string          `json:"backendType"`
	PermissionLevel PermissionLevel `json:"permissionLevel"`
	Enabled         bool            `json:"enabled"`
	UseWorktree     bool            `json:"useWorktree"`
	CreatedAt       string          `json:"createdAt"`
	UpdatedAt       string          `json:"updatedAt"`
}

// RunStatus tracks execution state
type RunStatus string

const (
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
)

// Run records a single automation execution
type Run struct {
	ID           string    `json:"id"`
	AutomationID string    `json:"automationId"`
	Status       RunStatus `json:"status"`
	StartedAt    string    `json:"startedAt"`
	CompletedAt  string    `json:"completedAt,omitempty"`
	Output       string    `json:"output,omitempty"`
	Error        string    `json:"error,omitempty"`
	HasFindings  bool      `json:"hasFindings"`
	Read         bool      `json:"read"`
}
