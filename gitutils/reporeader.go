package gitutils

import (
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/ir"
	"gopkg.in/src-d/go-git.v4"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var Log = config.Cfg().GetLogger()

func SetCourseComponentsTimestamps(repoPath string, course ir.Course) (err error) {
	Log.Debug("In SetCourseComponentsTimestamps")
	Log.Debug("In SetCourseComponentsTimestamps repoPath ", repoPath)

	repoFS, err := git.PlainOpen(repoPath)
	if err != nil {
		Log.Error("Local Git Repo Issue %v", err)
		return err
	}
	headRef, err := repoFS.Head()
	if err != nil {
		Log.Error("Local Git Repo Head Ref Issue %v", err)
		return err
	}
	Log.Debug("Head Ref Name ", headRef.Name())
	list, err := repoFS.Remotes()
	if err != nil {
		Log.Error("Local Git Repo Remotes Issue %v", err)
		return err
	}

	r := repoFS
	if len(list) > 0 {
		repoMem, err := CloneRepoMem(list[0].Config().URLs[0], headRef.Name())
		if err != nil {
			Log.Error("Local Git Repo Clone Remote into Memory Issue %v", err)
			return err
		}
		r = repoMem
	}

	courseUpdatedAt := time.Time{}
	start := time.Now()

	for _, chapter := range course.GetChapters() {
		chapUpdatedAt := time.Time{}

		for _, sequential := range chapter.GetSequentials() {
			seqUpdatedAt := time.Time{}

			for _, vert := range sequential.GetVerticals() {
				Log.Debug("Vertical URL ", vert.GetURLName())
				createdAt := time.Time{}
				updatedAt := time.Time{}

				var blockDirPath = ""
				for _, b := range vert.GetBlocks() {
					// Loop in case need to distinguish by block type. Currently, evaluate the dir based on the first block
					blockFileName := b.GetFSPath()
					Log.Debugf("Block's Git File %s", blockFileName)
					blockDirPath, _ = filepath.Split(blockFileName)
					Log.Debugf("Block Dir Path %s", blockDirPath)
					// See the comment above
					break
				}
				if len(blockDirPath) <= 0 {
					continue;
				}
				err = filepath.Walk(repoPath + string(filepath.Separator) + blockDirPath, func(path string, info os.FileInfo, err error) error {
					Log.Debugf("Checking Local FS path %s", path)
					if info.IsDir() {
						Log.Debug("Skipping as dir")
						return nil
					}
					_, fileNameNoPath := filepath.Split(path)
					if fileNameNoPath == "index.yaml" {
						Log.Debug("Skipping as index.yaml")
						return nil
					}
					fileNameInGit := strings.Replace(path, repoPath + string(filepath.Separator), "", 1)
					Log.Debugf("Getting Commit Log for File %s", fileNameInGit)
					commitsIter, err := r.Log(&git.LogOptions{FileName: &fileNameInGit, Order: git.LogOrderCommitterTime,})
					if err != nil {
						Log.Errorf("Local Git Commits Issue for %s %v", fileNameInGit, err)
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
						Log.Debugf("File Last Committed At %v", updatedAt)

						// Uncomment this to continue looping and get createdAt from the 1st commit
						break
					}

					return nil
				})

				if err != nil {
					continue
				}

				Log.Debugf("Vertical UpdatedAt %s", updatedAt)

				if !createdAt.IsZero() {
					// Future, see the comment above
				}

				if !updatedAt.IsZero() {
					vert.SetUpdatedAt(updatedAt)
					if seqUpdatedAt.IsZero() || seqUpdatedAt.Before(updatedAt) {
						seqUpdatedAt = updatedAt
					}
				}
			} // On Verticals of the Sequential

			Log.Debugf("Sequential UpdatedAt %s", seqUpdatedAt)
			if !seqUpdatedAt.IsZero() {
				sequential.SetUpdatedAt(seqUpdatedAt)
				if chapUpdatedAt.IsZero() || chapUpdatedAt.Before(seqUpdatedAt) {
					chapUpdatedAt = seqUpdatedAt
				}
			}
		} // On Sequentials of the Chapter

		Log.Debugf("Chapter UpdatedAt %s", chapUpdatedAt)
		if !chapUpdatedAt.IsZero() {
			chapter.SetUpdatedAt(chapUpdatedAt)
			if courseUpdatedAt.IsZero() || courseUpdatedAt.Before(chapUpdatedAt) {
				courseUpdatedAt = chapUpdatedAt
			}
		}
	} // On Chapters of the Course

	Log.Debugf("Course UpdatedAt %s", courseUpdatedAt)
	if !courseUpdatedAt.IsZero() {
		course.SetContentUpdatedAt(courseUpdatedAt)
	}

	elapsed := time.Since(start)
	Log.Infof("Git Commits Loop Process took %s", elapsed)
	r = nil

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
