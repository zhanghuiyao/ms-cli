package main

import (
	"github.com/vigo999/ms-cli/agent/loop"
	"github.com/vigo999/ms-cli/internal/config"
	"github.com/vigo999/ms-cli/tools"
	"github.com/vigo999/ms-cli/ui/model"
)

const Version = "ms-cli v0.2.0"

// Application is the top-level composition container.
type Application struct {
	Engine   *loop.Engine
	EventCh  chan model.Event
	Registry *tools.Registry
	Demo     bool
	WorkDir  string
	RepoURL  string
	Config   *config.Config
}

// SetEngine sets the engine (used by Wire).
func (a *Application) SetEngine(engine *loop.Engine) {
	a.Engine = engine
}

// SetRegistry sets the tool registry (used by Wire).
func (a *Application) SetRegistry(registry *tools.Registry) {
	a.Registry = registry
}
