package main

import (
	"os"
	"path/filepath"

	"github.com/vigo999/ms-cli/agent/context"
	"github.com/vigo999/ms-cli/agent/session"
	"github.com/vigo999/ms-cli/executor"
	"github.com/vigo999/ms-cli/internal/config"
	"github.com/vigo999/ms-cli/ui/model"
)

// Bootstrap wires top-level dependencies.
func Bootstrap(demo bool) (*Application, error) {
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	workDir, _ = filepath.Abs(workDir)

	// Load configuration
	cfg, err := config.FindConfig()
	if err != nil {
		// Log warning but continue with defaults
		cfg = config.DefaultConfig()
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Initialize context manager
	ctxManager := context.NewManager(cfg.Context.MaxTokens)
	ctxManager.SetReserveTokens(cfg.Context.ReserveTokens)
	ctxManager.SetThreshold(cfg.Context.CompactionThreshold)

	// Initialize session manager
	sessionManager, err := session.NewManager("")
	if err != nil {
		return nil, err
	}

	// Initialize runner
	var runner *executor.Runner
	if cfg.Model.APIKey != "" {
		runner = executor.NewSmartRunnerWithProvider(
			cfg.Model.Provider,
			cfg.Model.APIKey,
			cfg.Model.Endpoint,
		)
	} else {
		runner = executor.NewRunnerWithWorkDir(workDir)
	}

	return &Application{
		EventCh:        make(chan model.Event, 64),
		Demo:           demo,
		WorkDir:        workDir,
		RepoURL:        "github.com/vigo999/ms-cli",
		Config:         cfg,
		ContextManager: ctxManager,
		SessionManager: sessionManager,
		Runner:         runner,
	}, nil
}
