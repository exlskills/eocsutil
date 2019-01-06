package gitutils

import (
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/ir"
	"gopkg.in/src-d/go-git.v4"
	"time"
)

var Log = config.Cfg().GetLogger()

func SetCourseComponentsTimestamps(repoPath string, course ir.Course) (err error) {
	Log.Debug("In SetCourseComponentsTimestamps")
	r, err := git.PlainOpen(repoPath)

	if err != nil {
		Log.Error("Local Git Repo Issue %v", err)
		return err
	}

	for _, chapter := range course.GetChapters() {
		for _, sequential := range chapter.GetSequentials() {
			for _, vert := range sequential.GetVerticals() {
				createdAt := time.Time{}
				updatedAt := time.Time{}
				for _, b := range vert.GetBlocks() {
					if b.GetBlockType() == "html" {
						fileName := b.GetFSPath()
						Log.Debugf("Local Git File %s", fileName)
						commitsIter, err := r.Log(&git.LogOptions{FileName: &fileName, Order: git.LogOrderCommitterTime,})
						if err != nil {
							Log.Errorf("Local Git Commits Issue for %s %v", fileName, err)
							return err
						}
						for {
							commit, err := commitsIter.Next()
							if err != nil {
								break
							}
							if createdAt.IsZero() || createdAt.After(commit.Author.When) {
								createdAt = commit.Author.When
							}
							if updatedAt.IsZero() || updatedAt.Before(commit.Author.When) {
								updatedAt = commit.Author.When
							}

							// Uncomment this to get createdAt
							break
						}
					}
				}
				Log.Debugf("UpdatedAt %s", updatedAt)

				if (!createdAt.IsZero()) {
					// Future, see the comment above
				}
				if (!updatedAt.IsZero()) {
					vert.SetUpdatedAt(updatedAt)
				}

			}
		}
	}

	return
}

func IsRepoContentUpdated(repoPath string) (contentChanged bool, err error) {
	Log.Debug("In IsRepoContentUpdated")

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		Log.Error("Local Git Repo Open Issue", err)
		return false, err
	}

	w, err := r.Worktree()
	if err != nil {
		Log.Error("Local Git Repo Worktree Issue", err)
		return false, err
	}

	status, err := w.Status()
	if err != nil {
		Log.Error("Local Git Repo Worktree Status Issue", err)
		return false, err
	}

	if status.IsClean() {
		Log.Info("No changes to Git Repo content")
		return false, nil
	}

	// Add All Files
	for k, _ := range status {
		Log.Debug("Adding File ", k)
		_, err = w.Add(k)
		if err != nil {
			Log.Error("Local Git Repo Add File Issue", err)
			return true, err
		}
	}

	return true, nil
}
