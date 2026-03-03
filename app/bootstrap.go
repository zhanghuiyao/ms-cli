package main

import (
	"os"
	"path/filepath"

	"github.com/vigo999/ms-cli/agent/loop"
	"github.com/vigo999/ms-cli/executor"
	"github.com/vigo999/ms-cli/internal/config"
	"github.com/vigo999/ms-cli/ui/model"
)

// Bootstrap wires top-level dependencies.
func Bootstrap(demo bool) (*Application, error) {
	loop.SetExecutorRun(executor.Run)

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

	return &Application{
		Engine:  loop.NewEngine(),
		EventCh: make(chan model.Event, 64),
		Demo:    demo,
		WorkDir: workDir,
		RepoURL: "github.com/vigo999/ms-cli",
		Config:  cfg,
	}, nil
}
