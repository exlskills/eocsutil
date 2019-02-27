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

	courseUpdatedAt := time.Time{}

	for _, chapter := range course.GetChapters() {
		chapUpdatedAt := time.Time{}

		for _, sequential := range chapter.GetSequentials() {
			seqUpdatedAt := time.Time{}

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

							// Uncomment this to continue looping and get createdAt from the 1st commit
							break
						}
					}
				}  // On Blocks of the Vertical
				Log.Debugf("UpdatedAt %s", updatedAt)

				if !createdAt.IsZero() {
					// Future, see the comment above
				}

				if !updatedAt.IsZero() {
					vert.SetUpdatedAt(updatedAt)
					if seqUpdatedAt.IsZero() || seqUpdatedAt.Before(updatedAt) {
						seqUpdatedAt = updatedAt
					}
				}
			}  // On Verticals of the Sequential

			Log.Debugf("seqUpdatedAt %s", seqUpdatedAt)
			if !seqUpdatedAt.IsZero() {
				sequential.SetUpdatedAt(seqUpdatedAt)
				if chapUpdatedAt.IsZero() || chapUpdatedAt.Before(seqUpdatedAt) {
					chapUpdatedAt = seqUpdatedAt
				}
			}
		} // On Sequentials of the Chapter

		Log.Debugf("chapUpdatedAt %s", chapUpdatedAt)
		if !chapUpdatedAt.IsZero() {
			chapter.SetUpdatedAt(chapUpdatedAt)
			if courseUpdatedAt.IsZero() || courseUpdatedAt.Before(chapUpdatedAt) {
				courseUpdatedAt = chapUpdatedAt
			}
		}
	} // On Chapters of the Course

	Log.Debugf("courseUpdatedAt %s", courseUpdatedAt)
	if !courseUpdatedAt.IsZero() {
		course.SetContentUpdatedAt(courseUpdatedAt)
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
		Log.Info("No changes to Local Git Repo content")
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
