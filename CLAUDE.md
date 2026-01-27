# CCUI - Agent Coding Interface

A desktop AI coding assistant application built with Wails (Go + Svelte/TypeScript). CCUI provides a unified interface for interacting with AI agents through ACP (Agent Client Protocol) or directly via the Anthropic API.

## Project Overview

CCUI is a multi-session AI coding assistant with the following key features:

- **Multi-session chat**: Create and switch between multiple concurrent AI sessions
- **Real-time tool execution**: Visualize file reads, edits, bash commands, and other tools as they execute
- **Permission system**: Configurable permission layer for tool execution (auto-allow reads, ask for writes)
- **Review workflow**: Review and comment on AI-generated file changes before applying
- **Integrated terminal**: Built-in terminal with PTY support
- **Session modes**: Support for different agent modes (architect, code, debug, etc.)
- **Split-pane UI**: Configurable panel layout with chat, review, and terminal views

## Technology Stack

### Backend
- **Language**: Go 1.23+
- **Framework**: [Wails v2](https://wails.io/) - Desktop app framework
- **Key Dependencies**:
  - `github.com/creack/pty` - PTY support for terminal
  - `github.com/mark3labs/mcp-go` - MCP (Model Context Protocol) server/client
  - `github.com/wailsapp/wails/v2` - Wails runtime

### Frontend
- **Language**: TypeScript
- **Framework**: Svelte 3.x
- **Build Tool**: Vite 3.x
- **Styling**: Tailwind CSS v4 with custom "paper" theme
- **Testing**: Vitest with jsdom
- **Key Dependencies**:
  - `xterm` - Terminal emulator
  - `marked` - Markdown parsing
  - `diff` - Diff generation

## Project Structure

```
.
├── main.go                    # Application entry point
├── app.go                     # Main app logic, Wails bindings, session management
├── mcpserver.go               # MCP server for user question tool
├── pty.go                     # PTY/terminal session management
├── go.mod                     # Go module definition
├── wails.json                 # Wails configuration
│
├── backend/                   # Backend packages
│   ├── interface.go           # AgentBackend and Session interfaces
│   ├── types.go               # Shared types (ToolState, FileChange, etc.)
│   ├── acp/                   # ACP (Agent Client Protocol) implementation
│   │   ├── client.go          # ACP client for claude-code-acp
│   │   ├── transport.go       # JSON-RPC over stdio transport
│   │   ├── adapters.go        # Tool event adapters (claude-code, opencode)
│   │   └── types.go           # ACP protocol types
│   ├── anthropic/             # Direct Anthropic API backend
│   │   ├── backend.go         # AnthropicBackend implementation
│   │   ├── session.go         # Session management for direct API
│   │   ├── stream.go          # SSE streaming for API responses
│   │   └── tools.go           # Tool definitions for Anthropic
│   └── tools/                 # Tool executor for direct API backend
│       ├── executor.go        # Tool registry and execution interface
│       ├── read.go            # Read tool implementation
│       ├── write.go           # Write tool implementation
│       ├── edit.go            # Edit tool implementation
│       ├── bash.go            # Bash tool implementation
│       ├── grep.go            # Grep tool implementation
│       └── glob.go            # Glob tool implementation
│
├── permission/                # Permission layer
│   ├── layer.go               # Permission layer with request/response flow
│   └── rules.go               # Permission rules (Allow/Ask/Deny)
│
├── frontend/                  # Frontend application
│   ├── package.json           # Node.js dependencies (use bun, not npm)
│   ├── vite.config.ts         # Vite configuration
│   ├── tsconfig.json          # TypeScript configuration
│   └── src/
│       ├── main.ts            # Entry point
│       ├── App.svelte         # Main application component
│       ├── style.css          # Global styles with Tailwind
│       ├── lib/
│       │   ├── shared.ts      # Shared types and utilities
│       │   ├── ChatContent.svelte      # Chat message display
│       │   ├── ToolCard.svelte         # Tool execution visualization
│       │   ├── ReviewPanel.svelte      # Code review interface
│       │   ├── Terminal.svelte         # Terminal component
│       │   ├── SplitPane.svelte        # Resizable split pane
│       │   ├── CommandPalette.svelte   # Panel selector
│       │   ├── SessionSelector.svelte  # Session tabs
│       │   └── ModeSelector.svelte     # Agent mode selector
│       └── assets/            # Static assets
│
├── specs/                     # Architecture specifications
│   └── unified-backend/       # Backend architecture design docs
│
├── acp/                       # ACP protocol documentation
├── build/                     # Build assets and configurations
└── agent-os/                  # Agent OS related files
```

## Build Commands

### Development
```bash
# Run in live development mode with hot reload
wails dev

# Frontend development server only (accessible at http://localhost:34115)
cd frontend && bun run dev
```

### Build
```bash
# Build production binary
wails build

# Build for specific platform
wails build -platform darwin/universal
wails build -platform windows/amd64
```

### Testing

#### Go Tests
```bash
# Run all Go tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./backend/acp/...
go test ./backend/tools/...
go test ./permission/...
```

#### Frontend Tests
```bash
cd frontend

# Run tests once
bun run test

# Run tests in watch mode
bun run test:watch

# Run with coverage
bunx vitest run --coverage
```

## Development Guidelines

### Code Style

#### Go
- Follow standard Go conventions (`go fmt`)
- Use meaningful variable names, avoid single-letter except for common cases (i, err, ctx)
- Error handling: wrap errors with context using `fmt.Errorf("...: %w", err)`
- Interface naming: prefer `-er` suffix (e.g., `EventEmitter`, `ToolExecutor`)
- Use `sync.RWMutex` for concurrent access to shared state

#### TypeScript/Svelte
- Use TypeScript strict mode
- Prefer `const` and `let` over `var`
- Use Svelte's reactive statements (`$:`) judiciously
- Event naming: use camelCase with descriptive names

### Frontend Package Management
**Important**: Use `bun` for all frontend operations, not npm/npx.

```bash
# Correct
bun install
bun run dev
bunx vitest

# Incorrect
cd frontend && npm install  # Don't use npm
```

## Architecture

### Backend Architecture (Unified Backend)

The backend follows a layered architecture designed to support multiple AI backends:

```
Frontend (Svelte)
    │
    ▼ Wails Events
Permission Layer
    │
Tool Executor (fs, bash, grep, etc.)  ────┐
    │                                      │
AgentBackend Interface                    │
    │                                      │
    ├── ACP Backend (claude-code-acp) ────┤ (external execution)
    │
    └── Direct API Backend (Anthropic) ───┘ (local execution)
```

### Key Components

1. **Permission Layer** (`permission/`)
   - Deterministic rules: `Read/Glob/Grep` = Allow, `Write/Edit/Bash` = Ask
   - Blocks on user permission requests
   - Emits events to frontend for user interaction

2. **Tool Executor** (`backend/tools/`)
   - Local execution for direct API backend
   - Registry pattern for tool registration
   - Results include content, diffs, and file changes

3. **ACP Client** (`backend/acp/`)
   - Communicates with `claude-code-acp` via stdio JSON-RPC
   - Handles tool events with adapters for different ACP implementations
   - File change tracking for review workflow

4. **Session Management** (`app.go`)
   - Multi-session support with session switching
   - Event bridging between backend events and Wails events
   - Session-scoped event channels (`session:{id}:{event}`)

### Event Flow

1. User sends message → `EventsEmit('send_message', text)`
2. `app.go` routes to active session's ACP client
3. ACP client streams responses via `eventChan`
4. `bridgeEvents` forwards to Wails events with session prefix
5. Frontend subscribes to `session:{id}:chat_chunk`, `session:{id}:tool_state`, etc.

### Backend Selection

Set environment variable to select backend:
```bash
# ACP backend (default)
CCUI_BACKEND=acp ./ccui

# Anthropic direct API backend (planned)
CCUI_BACKEND=anthropic ./ccui
```

## Testing Strategy

### Unit Tests
- **Go**: Table-driven tests for tools, permission layer, and ACP adapters
- **Frontend**: Component tests using Vitest and Testing Library

### Test Files
- Go: `*_test.go` alongside source files
- Frontend: `*.test.ts` in `src/__tests__/` or alongside components

### Running Tests
```bash
# Go tests (run from project root)
go test ./... -v

# Frontend tests (run from frontend/)
cd frontend && bun run test
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ANTHROPIC_API_KEY` | API key for Anthropic | Required for direct API |
| `CCUI_BACKEND` | Backend type (`acp` or `anthropic`) | `acp` |
| `SHELL` | Shell for PTY sessions | `/bin/bash` |

## External Dependencies

The application requires `claude-code-acp` to be installed and available in PATH for the ACP backend:
```bash
# Install claude-code-acp
npm install -g @anthropics/claude-code-acp
```

## Security Considerations

1. **Permission System**: All write operations and bash commands require explicit user permission
2. **Auto-allow list**: Only read operations are auto-allowed (`Read`, `Glob`, `Grep`, `WebSearch`)
3. **API Key Handling**: API keys are read from environment, never stored in code
4. **MCP Server**: Local-only SSE server binding to `127.0.0.1:0` (random port)
5. **Bash Timeout**: Commands have configurable timeout (default 2min, max 10min)

## Common Tasks

### Adding a New Tool
1. Implement `Tool` interface in `backend/tools/`
2. Add to registry in executor setup
3. Add permission rule in `permission/rules.go` if needed

### Adding a New Backend
1. Implement `AgentBackend` and `Session` interfaces from `backend/interface.go`
2. Add backend type constant in `app.go`
3. Initialize in `NewApp()` based on `CCUI_BACKEND` env var

### Frontend Event Handling
1. Backend emits via `runtime.EventsEmit(ctx, "event_name", data)`
2. Frontend subscribes via `EventsOn('event_name', callback)`
3. For session-scoped events: use `session:{sessionId}:{event}` prefix

## References

- [Wails Documentation](https://wails.io/docs/)
- [Svelte Documentation](https://svelte.dev/docs)
- [ACP Protocol](acp/acp-protocol.md)
- [Unified Backend Specs](specs/unified-backend/)
## Frontend Development

Use bun/bunx not npm/npx

## Events

The Wails runtime provides a unified events system, where events can be emitted or received by either Go or JavaScript. Optionally, data may be passed with the events. Listeners will receive the data in the local data types.
EventsOn

This method sets up a listener for the given event name. When an event of type eventName is emitted, the callback is triggered. Any additional data sent with the emitted event will be passed to the callback. It returns a function to cancel the listener.

Go: EventsOn(ctx context.Context, eventName string, callback func(optionalData ...interface{})) func()
JS: EventsOn(eventName string, callback function(optionalData?: any)): () => void
EventsOff

This method unregisters the listener for the given event name, optionally multiple listeners can be unregistered via additionalEventNames.

Go: EventsOff(ctx context.Context, eventName string, additionalEventNames ...string)
JS: EventsOff(eventName string, ...additionalEventNames)
EventsOnce

This method sets up a listener for the given event name, but will only trigger once. It returns a function to cancel the listener.

Go: EventsOnce(ctx context.Context, eventName string, callback func(optionalData ...interface{})) func()
JS: EventsOnce(eventName string, callback function(optionalData?: any)): () => void
EventsOnMultiple

This method sets up a listener for the given event name, but will only trigger a maximum of counter times. It returns a function to cancel the listener.

Go: EventsOnMultiple(ctx context.Context, eventName string, callback func(optionalData ...interface{}), counter int) func()
JS: EventsOnMultiple(eventName string, callback function(optionalData?: any), counter int): () => void
EventsEmit

This method emits the given event. Optional data may be passed with the event. This will trigger any event listeners.

Go: EventsEmit(ctx context.Context, eventName string, optionalData ...interface{})
JS: EventsEmit(eventName: string, ...optionalData: any)

## Dialog

This part of the runtime provides access to native dialogs, such as File Selectors and Message boxes.
JavaScript

Dialog is currently unsupported in the JS runtime.
OpenDirectoryDialog

Opens a dialog that prompts the user to select a directory. Can be customised using OpenDialogOptions.

Go: OpenDirectoryDialog(ctx context.Context, dialogOptions OpenDialogOptions) (string, error)

Returns: Selected directory (blank if the user cancelled) or an error
OpenFileDialog

Opens a dialog that prompts the user to select a file. Can be customised using OpenDialogOptions.

Go: OpenFileDialog(ctx context.Context, dialogOptions OpenDialogOptions) (string, error)

Returns: Selected file (blank if the user cancelled) or an error
OpenMultipleFilesDialog

Opens a dialog that prompts the user to select multiple files. Can be customised using OpenDialogOptions.

Go: OpenMultipleFilesDialog(ctx context.Context, dialogOptions OpenDialogOptions) ([]string, error)

Returns: Selected files (nil if the user cancelled) or an error
SaveFileDialog

Opens a dialog that prompts the user to select a filename for the purposes of saving. Can be customised using SaveDialogOptions.

Go: SaveFileDialog(ctx context.Context, dialogOptions SaveDialogOptions) (string, error)

Returns: The selected file (blank if the user cancelled) or an error
MessageDialog

Displays a message using a message dialog. Can be customised using MessageDialogOptions.

Go: MessageDialog(ctx context.Context, dialogOptions MessageDialogOptions) (string, error)

Returns: The text of the selected button or an error
Options
OpenDialogOptions

```go
type OpenDialogOptions struct {
	DefaultDirectory           string
	DefaultFilename            string
	Title                      string
	Filters                    []FileFilter
	ShowHiddenFiles            bool
	CanCreateDirectories       bool
	ResolvesAliases            bool
	TreatPackagesAsDirectories bool
}
```

## Screen

These methods provide information about the currently connected screens.
ScreenGetAll

Returns a list of currently connected screens.

Go: ScreenGetAll(ctx context.Context) []screen
JS: ScreenGetAll()
Screen

Go struct:
```go
type Screen struct {
	IsCurrent bool
	IsPrimary bool
	Width     int
	Height    int
}
```

Typescript interface:

```ts
interface Screen {
    isCurrent: boolean;
    isPrimary: boolean;
    width : number
    height : number
}
```

## Menu

These methods are related to the application menu.
JavaScript

Menu is currently unsupported in the JS runtime.
MenuSetApplicationMenu

Sets the application menu to the given menu.

Go: MenuSetApplicationMenu(ctx context.Context, menu *menu.Menu)
MenuUpdateApplicationMenu

Updates the application menu, picking up any changes to the menu passed to MenuSetApplicationMenu.

Go: MenuUpdateApplicationMenu(ctx context.Context)

