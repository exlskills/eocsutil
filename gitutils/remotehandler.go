package gitutils

import (
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/ghmodels"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"time"
)

func CloneRepo(cloneURL string, targetDir string) (err error) {
	Log.Infof("Cloning %s into %s", cloneURL, targetDir)
	_, err = git.PlainClone(targetDir, false, &git.CloneOptions{
		URL:               cloneURL,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})
	return err
}

func CommitAndPush(repoPath string, author ghmodels.CommitAuthor) (err error) {
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

	commit, err := w.Commit(config.Cfg().GHAutoGenCommitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  author.Name,
			Email: author.Email,
			When:  time.Now(),
		},
	})
	if err != nil {
		Log.Error("Local Git Repo Commit Issue", err)
		return err
	}

	Log.Info("Generated auto Commit ")
	Log.Debug("Commit ", commit)
	err = r.Push(&git.PushOptions{Auth: &http.BasicAuth{
		Username: "abc123", // anything except an empty string
		Password: config.Cfg().GHUserToken,
	},})

	return err
}
