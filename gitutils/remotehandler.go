package gitutils

import (
	"github.com/exlskills/eocsutil/config"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"time"
)

func CloneRepo(cloneURL string, targetDir string) (err error) {
	_, err = git.PlainClone(targetDir, false, &git.CloneOptions{
		URL:               cloneURL,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})
	return err
}

func CommitAndPush(repoPath string, commitMsg string) (err error) {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		Log.Error("Local Git Repo Open Issue", err)
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		Log.Error("Local Git Repo Worktree Issue", err)
		return err
	}

	commit, err := w.Commit("auto#gen", &git.CommitOptions{
		Author: &object.Signature{
			Name:  config.Cfg().GitUser,
			Email: config.Cfg().GitUserEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		Log.Error("Local Git Repo Commit Issue", err)
		return err
	}

	Log.Debug("Generated Commit ", commit)
	err = r.Push(&git.PushOptions{Auth: &http.BasicAuth{
		Username: "abc123", // anything except an empty string
		Password: config.Cfg().GitUserToken,
	},})

	return err
}
