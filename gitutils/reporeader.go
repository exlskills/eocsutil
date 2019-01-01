package gitutils

import (
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/ir"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"time"
)

var Log = config.Cfg().GetLogger()

func SetCourseComponentsTimestamps(repoPath string, course ir.Course) (err error) {

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
						Log.Debugf("Local Git File " + fileName)
						commitsIter, err := r.Log(&git.LogOptions{FileName: &fileName, Order: git.LogOrderDFSPost,})
						if err != nil {
							Log.Error("Local Git Commits Issue for %s %v", fileName, err)
							return err
						}
						err = commitsIter.ForEach(func(c *object.Commit) error {
							if createdAt.IsZero() || createdAt.After(c.Author.When) {
								createdAt = c.Author.When
							}
							if updatedAt.IsZero() || updatedAt.Before(c.Author.When) {
								updatedAt = c.Author.When
							}
							return nil
						})
					}
				}
				if (!createdAt.IsZero()) {
					// Future
				}
				if (!updatedAt.IsZero()) {
					vert.SetUpdatedAt(updatedAt)
				}

			}
		}
	}

	return
}
