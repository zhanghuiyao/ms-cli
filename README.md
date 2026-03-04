# ms-cli

MindSpore CLI — an AI infrastructure agent with a terminal UI.

## Prerequisites

- Go 1.24.2+ (see `go.mod`)

## Quick Start

Build:

```bash
go build -o ms-cli ./app
```

Run demo mode:

```bash
go run ./app --demo
# or
./ms-cli --demo
```

Run real mode:

```bash
go run ./app
# or
./ms-cli
```

## Commands

In TUI input, use slash commands:

### Session Management
- `/new` - Start a new session (clears context, preserves system prompt)
- `/clear` - Clear current session messages
- `/save [file]` - Save session to file (default: session.json)
- `/load [file]` - Load session from file
- `/history` - Show session history

### Model Configuration
- `/model` - Show current model info
- `/model list` - List available models
- `/model use <name>` - Switch to a different model
- `/temperature <n>` - Set temperature (0.0-2.0)
- `/tokens <n>` - Set max tokens

### Context Management
- `/compact` - Manually compress context
- `/roadmap status [path]` - Show project roadmap progress
- `/weekly status [path]` - Show weekly update

### Help
- `/tools` - List available tools
- `/help` - Show all commands

Any non-slash input is treated as a normal task prompt and routed to the engine.

## Features

### Animated Thinking Indicator
When the AI is processing, you'll see an animated "⣽ Thinking..." indicator with smooth spinner animation. When complete, it shows "✓ Done" in the top bar.

### Slash Command Auto-completion
Type `/` to see available commands with vertical selection:
- Use `↑/↓` to navigate through suggestions
- Press `Tab` or `Enter` to accept the selected command
- Commands are filtered as you type

### Stream-Ready Architecture
The TUI supports streaming responses (typing effect) when connected to a streaming LLM provider.

### Step Progress Indicator
Multi-step tasks show progress in the top bar (e.g., "step: 2/5").

## Available Commands

### Session Management
- `/new` - Start a new session (clears context, preserves system prompt)
- `/clear` - Clear current session messages
- `/save [file]` - Save session to file (default: session.json)
- `/load <file>` - Load session from file
- `/history` - Show session history

### Model Configuration
- `/model` - Show current model info
- `/model list` - List available models
- `/model use <name>` - Switch to a different model
- `/temperature <n>` - Set temperature (0.0-2.0)
- `/tokens <n>` - Set max tokens

### Context Management
- `/compact` - Manually compress context
- `/search <keyword>` - Search through message history
- `/theme <name>` - Switch theme (dark/light)

### Project Commands
- `/roadmap status [path]` - Show project roadmap progress
- `/weekly status [path]` - Show weekly update

### Help
- `/tools` - List available tools
- `/help` - Show all commands

## Keybindings

| Key | Action |
|-----|--------|
| `Enter` | Send message / Accept suggestion |
| `Tab` | Accept auto-complete suggestion |
| `↑/↓` | Navigate suggestions (when shown) or scroll chat |
| `PgUp/PgDn` | Scroll chat |
| `Home/End` | Jump to top/bottom |
| `Ctrl+C` | Quit |

## Keybindings

| Key | Action |
|-----|--------|
| `enter` | Send input |
| `pgup` / `pgdn` | Scroll chat |
| `up` / `down` | Scroll chat |
| `home` / `end` | Jump to top / bottom |
| `/` | Start a slash command |
| `ctrl+c` | Quit |

## Project Status Data

Roadmap status engine:

- `internal/project/roadmap.go`
- Parses roadmap YAML, validates schema, and computes phase + overall progress.

Weekly update parser (Markdown + YAML front matter):

- `internal/project/weekly.go`
- Template: `docs/updates/WEEKLY_TEMPLATE.md`

Public roadmap page:

- `docs/roadmap/ROADMAP.md`

Project reports:

- `docs/updates/` (see latest `*-report.md`)

## Repository Structure

```text
ms-cli/
├── app/                        # entry point + wiring
│   ├── main.go
│   ├── bootstrap.go
│   ├── wire.go
│   ├── run.go
│   └── commands.go
├── agent/
│   ├── loop/                   # engine, task/event types, permissions
│   ├── context/                # budget, compaction, context manager
│   └── memory/                 # policy, store, retrieve
├── executor/
│   └── runner.go               # pluggable task executor
├── integrations/
│   ├── domain/                 # external domain client + schema
│   └── skills/                 # skill invocation + repo
├── internal/
│   └── project/
│       ├── roadmap.go
│       └── weekly.go
├── tools/
│   ├── fs/                     # filesystem operations
│   └── shell/                  # shell command runner
├── trace/
│   └── writer.go               # execution trace logging
├── report/
│   └── summary.go              # report generation
├── ui/
│   ├── app.go                  # root Bubble Tea model
│   ├── model/model.go          # shared state types
│   ├── components/             # spinner, textinput, viewport
│   └── panels/                 # topbar, chat, hintbar
├── docs/
│   ├── roadmap/ROADMAP.md
│   └── updates/
├── go.mod
└── README.md
```

## Known Limitations

- The real-mode engine flow is still minimal/stub-oriented.
- Running Bubble Tea in non-interactive shells may fail with `/dev/tty` errors.

## Architecture Rule

UI listens to events; agent loop emits events; executor/tools do not depend on UI.
