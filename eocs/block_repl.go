package eocs

import "github.com/exlskills/eocsutil/wsenv"

type BlockREPL struct {
	APIVersion     int                             `yaml:"api_version"`
	EnvironmentKey string                          `yaml:"environment"`
	SourcePath     string                          `yaml:"src_path"`
	Explanation    string                          `yaml:"explanation,omitempty"`
	TmplPath       string                          `yaml:"tpml_path,omitempty"`
	TestPath       string                          `yaml:"test_path,omitempty"`
	Display        *BlockREPLDisplay               `yaml:"display,omitempty"`
	SrcFiles       map[string]*wsenv.WorkspaceFile `yaml:"-"`
	TmplFiles      map[string]*wsenv.WorkspaceFile `yaml:"-"`
	TestFiles      map[string]*wsenv.WorkspaceFile `yaml:"-"`
}

type BlockREPLDisplay struct {
	Height string `yaml:"height"`
}
