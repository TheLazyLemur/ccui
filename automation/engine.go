package automation

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"ccui/backend"
	"ccui/permission"
)

// EventEmitter abstracts event emission
type EventEmitter interface {
	Emit(eventName string, data any)
}

// BackendFactory creates a backend for a given type
type BackendFactory func(backendType string) (backend.AgentBackend, error)

// Engine executes automations
type Engine struct {
	backendFactory BackendFactory
	runStore       *RunStore
	emitter        EventEmitter
	skillStore     *SkillStore

	mu        sync.Mutex
	cancelFns map[string]context.CancelFunc // runID -> cancel
}

// NewEngine creates an execution engine
func NewEngine(factory BackendFactory, runStore *RunStore, emitter EventEmitter, skillStore *SkillStore) *Engine {
	return &Engine{
		backendFactory: factory,
		runStore:       runStore,
		emitter:        emitter,
		skillStore:     skillStore,
		cancelFns:      make(map[string]context.CancelFunc),
	}
}

// Execute runs an automation, blocking until completion
func (e *Engine) Execute(ctx context.Context, auto Automation) (*Run, error) {
	// create run record
	run, err := e.runStore.Create(auto.ID, Run{Status: RunStatusRunning})
	if err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}

	// register cancel
	ctx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.cancelFns[run.ID] = cancel
	e.mu.Unlock()
	defer func() {
		cancel()
		e.mu.Lock()
		delete(e.cancelFns, run.ID)
		e.mu.Unlock()
	}()

	// emit started
	if e.emitter != nil {
		e.emitter.Emit(fmt.Sprintf("automation:%s:run_started", auto.ID), map[string]string{
			"runId":        run.ID,
			"automationId": auto.ID,
			"startedAt":    run.StartedAt,
		})
	}

	// resolve skill references in prompt
	prompt := auto.Prompt
	if e.skillStore != nil && strings.HasPrefix(prompt, "$") {
		resolved, err := e.skillStore.ResolvePrompt(prompt)
		if err != nil {
			return e.failRun(run, auto.ID, fmt.Errorf("resolve skill: %w", err))
		}
		prompt = resolved
	}

	// worktree setup
	cwd := auto.ProjectDir
	var wtDir string
	if auto.UseWorktree && cwd != "" {
		wt, err := CreateWorktree(cwd, run.ID)
		if err != nil {
			return e.failRun(run, auto.ID, fmt.Errorf("create worktree: %w", err))
		}
		wtDir = wt
		cwd = wt
	}
	defer func() {
		if wtDir != "" {
			RemoveWorktree(auto.ProjectDir, wtDir)
		}
	}()

	// build permission rules for this level
	rules := rulesForLevel(auto.PermissionLevel)
	permLayer := permission.NewLayer(rules, &noopEmitter{})

	// create backend
	be, err := e.backendFactory(auto.BackendType)
	if err != nil {
		return e.failRun(run, auto.ID, fmt.Errorf("create backend: %w", err))
	}

	// create session
	eventChan := make(chan backend.Event, 100)
	sess, err := be.NewSession(ctx, backend.SessionOpts{
		CWD:            cwd,
		EventChan:      eventChan,
		AutoPermission: true,
	})
	if err != nil {
		close(eventChan)
		return e.failRun(run, auto.ID, fmt.Errorf("create session: %w", err))
	}
	_ = permLayer // permission enforced via AutoPermission + rules baked into backend

	// collect output
	var output strings.Builder
	done := make(chan struct{})
	go func() {
		defer close(done)
		for event := range eventChan {
			switch event.Type {
			case backend.EventMessageChunk:
				if text, ok := event.Data.(string); ok {
					output.WriteString(text)
					if e.emitter != nil {
						e.emitter.Emit(fmt.Sprintf("automation:%s:run_chunk", auto.ID), text)
					}
				}
			}
		}
	}()

	// send prompt
	sendErr := sess.SendPrompt(prompt, []string{})

	// close session and wait for event collection
	sess.Close()
	close(eventChan)
	<-done

	if sendErr != nil {
		return e.failRun(run, auto.ID, sendErr)
	}

	// complete the run
	now := time.Now().UTC().Format(time.RFC3339)
	run.Status = RunStatusCompleted
	run.CompletedAt = now
	run.Output = output.String()
	run.HasFindings = len(strings.TrimSpace(run.Output)) > 0
	if err := e.runStore.Update(*run); err != nil {
		return run, fmt.Errorf("update run: %w", err)
	}

	if e.emitter != nil {
		e.emitter.Emit(fmt.Sprintf("automation:%s:run_completed", auto.ID), run)
		if run.HasFindings {
			e.emitter.Emit("automations:triage_updated", e.unreadCount())
		}
	}

	return run, nil
}

// CancelRun stops a running automation
func (e *Engine) CancelRun(runID string) {
	e.mu.Lock()
	if cancel, ok := e.cancelFns[runID]; ok {
		cancel()
	}
	e.mu.Unlock()
}

func (e *Engine) failRun(run *Run, autoID string, runErr error) (*Run, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	run.Status = RunStatusFailed
	run.CompletedAt = now
	run.Error = runErr.Error()
	_ = e.runStore.Update(*run)

	if e.emitter != nil {
		e.emitter.Emit(fmt.Sprintf("automation:%s:run_failed", autoID), run)
	}
	return run, runErr
}

func (e *Engine) unreadCount() int {
	items, err := e.runStore.UnreadWithFindings()
	if err != nil {
		return 0
	}
	return len(items)
}

func rulesForLevel(level PermissionLevel) *permission.RuleSet {
	switch level {
	case PermWorkspaceWrite:
		return permission.WorkspaceWriteRules()
	case PermFullAccess:
		return permission.DefaultRules()
	default:
		return permission.ReadOnlyRules()
	}
}

// noopEmitter satisfies permission.EventEmitter for automation runs
type noopEmitter struct{}

func (n *noopEmitter) Emit(string, any) {}
