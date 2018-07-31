package eocs

import "github.com/exlskills/eocsutil/wsenv"

type BlockREPL struct {
	APIVersion     int                             `yaml:"api_version"`
	EnvironmentKey string                          `yaml:"environment"`
	SourcePath     string                          `yaml:"src_path"`
	TestPath       string                          `yaml:"test_path,omitempty"`
	Display        *BlockREPLDisplay               `yaml:"display,omitempty"`
	Question       *BlockREPLQuestion              `yaml:"question,omitempty"`
	SrcFiles       map[string]*wsenv.WorkspaceFile `yaml:"-"`
	TestFiles      map[string]*wsenv.WorkspaceFile `yaml:"-"`
}

type BlockREPLDisplay struct {
	Height string `yaml:"height"`
}

type BlockREPLQuestion struct {
	Type       string `yaml:"type"` // free_response,multiple_choice,multiple_select
	Complexity int    `yaml:"complexity"`
	Points     int    `yaml:"points"`
	EstSeconds int    `yaml:"est_seconds"`
	Hint       string `yaml:"hint,omitempty"`
}
