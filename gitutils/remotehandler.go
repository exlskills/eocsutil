package gitutils

import (
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/ghmodels"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"time"
)

func CloneRepo(cloneURL string, cloneRef plumbing.ReferenceName, targetDir string) (err error) {
	Log.Infof("Cloning repo %s ref %s into %s", cloneURL, cloneRef, targetDir)
	_, err = git.PlainClone(targetDir, false, &git.CloneOptions{
		URL:               cloneURL,
		ReferenceName:     cloneRef,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		SingleBranch:  true,
	})
	return err
}

func CloneRepoMem(cloneURL string, cloneRef plumbing.ReferenceName) (r *git.Repository, err error) {
	Log.Infof("Cloning repo %s ref %s into memory", cloneURL, cloneRef)
	r, err = git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:               cloneURL,
		ReferenceName:     cloneRef,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		SingleBranch:  true,
	})
	return r, err
}

func CommitAndPush(repoPath string, author ghmodels.CommitAuthor, triggerCommit string) (err error) {
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

	autoGenCommitMsg := config.Cfg().GHAutoGenCommitMsg + "_" + SubstringFirstN(triggerCommit,7)

	commit, err := w.Commit(autoGenCommitMsg, &git.CommitOptions{
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

	Log.Info("Generated auto Commit ", commit)
	err = r.Push(&git.PushOptions{Auth: &http.BasicAuth{
		Username: "abc123", // anything except an empty string
		Password: config.Cfg().GHUserToken,
	},})

	return err
}

func SubstringFirstN(s string, n int) string {
	i := 0
	for j := range s {
		if i == n {
			return s[:j]
		}
		i++
	}
	return s
}