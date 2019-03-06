package ghserver

import (
	"fmt"
	"github.com/exlinc/golang-utils/jsonhttp"
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/eocs"
	"github.com/exlskills/eocsutil/ghmodels"
	"github.com/exlskills/eocsutil/gitutils"
	"github.com/exlskills/eocsutil/smtputils"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const (
	asyncMode     = 0
	syncMode      = 1
	failedSubject = "Course Load Failed"
	okSubject     = "Course Load Processed Successfully"
)

func repoPushEventWebhookLauncher(w http.ResponseWriter, r *http.Request) {
	Log.Debug("In repoPushEventWebhookLauncher")
	reqObj := ghmodels.RepoPushEventRequest{}
	err := ghmodels.GHSecureJSONDecodeAndCatchForAPI(w, r, config.Cfg().GHWebhookSecret, &reqObj)
	if err != nil {
		Log.Error("Issue while reading request ", err)
		jsonhttp.JSONInternalError(w, "Invalid Request", "")
		return
	}
	go repoPushEventWebhookProcessor(reqObj, asyncMode)
	jsonhttp.JSONSuccess(w, nil, "Ack Receipt")
	return
}

func repoPushEventWebhook(w http.ResponseWriter, r *http.Request) {
	Log.Debug("In repoPushEventWebhook")
	reqObj := ghmodels.RepoPushEventRequest{}
	err := ghmodels.GHSecureJSONDecodeAndCatchForAPI(w, r, config.Cfg().GHWebhookSecret, &reqObj)
	if err != nil {
		Log.Error("Issue while reading request ", err)
		jsonhttp.JSONInternalError(w, "Invalid Request", "")
		return
	}
	message, err := repoPushEventWebhookProcessor(reqObj, syncMode)
	if err != nil {
		jsonhttp.JSONInternalError(w, message, "")
	} else {
		jsonhttp.JSONSuccess(w, nil, message)
	}
	return
}

func repoPushEventWebhookProcessor(reqObj ghmodels.RepoPushEventRequest, mode int) (message string, err error) {
	loadHeaderString := fmt.Sprintf("Course Repository %s; Head Commit %s ", reqObj.Repository.Name, reqObj.HeadCommit.ID)
	Log.Info(loadHeaderString)
	if len(reqObj.Ref) < 11 || !strings.HasPrefix(reqObj.Ref, "refs/heads/") {
		Log.Info("Skipping push on empty or invalid ref: ", reqObj.Ref)
		return "No-op, must be a valid push", nil
	}

	branchInHook := reqObj.Ref[11:len(reqObj.Ref)]  // cut the leading 11 chars out: refs/heads/
	// Note: config.Cfg().GHWebhookBranch is prevalidated at server start
	branchesInConfig := strings.Split(config.Cfg().GHWebhookBranch, ",")
	validBranch := false
	for _, a := range branchesInConfig {
		if a == branchInHook {
			validBranch = true
			break
		}
	}

	if !validBranch  {
		Log.Infof("Branch %s is not in the configured Branch List. Skipping push on ref %s ", branchInHook, reqObj.Ref)
		return "No-op, must be in the branch list to sync", nil
	}

	// Note: on a new branch push, only the head_commit is present
	commits := reqObj.Commits
	if len(reqObj.Commits) < 1 {
		commits = append(commits, reqObj.HeadCommit)
	}

	if len(commits) < 1 {
		Log.Info("Skipping. No commits")
		return "No-op, must be commit-based", nil
	}

	hasRealCommits := false
	commitAuthor := ghmodels.CommitAuthor{}
	for _, commit := range commits {
		if !strings.Contains(commit.Message, config.Cfg().GHAutoGenCommitMsg) {
			hasRealCommits = true
			commitAuthor = commit.Author
			break
		}
	}
	if !hasRealCommits {
		Log.Info("Skipping. Auto-gen commits only")
		return "No-op, auto#gen commit", nil
	}

	rootDir, err := ioutil.TempDir("", "eocsutil-repo-dl-")
	if err != nil {
		Log.Error("An error occurred creating the temp directory: ", err)
		if mode == asyncMode {
			smtputils.SendEmail(reqObj.HeadCommit.Author.Email, failedSubject, loadHeaderString+"<br>Internal Error")
		}
		return "An error occurred creating the temp directory", err
	}
	defer os.RemoveAll(rootDir)

	err = gitutils.CloneRepo(reqObj.Repository.CloneURL, plumbing.NewBranchReferenceName(branchInHook), rootDir)
	if err != nil {
		Log.Error("An error occurred cloning repo/branch: ", err)
		if mode == asyncMode {
			smtputils.SendEmail(reqObj.HeadCommit.Author.Email, failedSubject, loadHeaderString+"<br>Repo Clone Failed")
		}
		return "An error occurred cloning repo/branch", err
	}

	/*
	archiveFileName, err := downloadFromUrl(rootDir, strings.Replace(strings.Replace(reqObj.Repository.ArchiveURL, "{/ref}", "/master", 1), "{archive_format}", "zipball", 1))
	if err != nil {
		Log.Error("An error occurred downloading the repo archive: ", err)
		jsonhttp.JSONInternalError(w, "An error occurred downloading the repo archive", "")
		return
	}
	unzippedFilePaths, err := unzip(archiveFileName, rootDir)
	if err != nil {
		Log.Error("An error occurred extracting the repo archive: ", err)
		jsonhttp.JSONInternalError(w, "An error occurred extracting the repo archive", "")
		return
	}
	if len(unzippedFilePaths) < 1 {
		Log.Error("An error occurred extracting the repo archive: EMPTY OUTPUT LIST")
		jsonhttp.JSONInternalError(w, "An error occurred extracting the repo archive", "")
		return
	}
	courseDir := unzippedFilePaths[0]
	*/

	err = eocs.NewEOCSFormat().Push(rootDir, config.Cfg().GHServerMongoURI)
	if err != nil {
		Log.Errorf("Course push failed: %s", err.Error())
		if mode == asyncMode {
			errEmailText := fmt.Sprintf(loadHeaderString+"<br> Error Converting Course<br>%v", err)
			smtputils.SendEmail(reqObj.HeadCommit.Author.Email, failedSubject, errEmailText)
		}
		return "An error occurred importing the course", err
	}

	repoChanged, err := gitutils.IsRepoContentUpdated(rootDir)
	if err != nil {
		Log.Errorf("Local repo content check for updates failed: %s", err.Error())
		if mode == asyncMode {
			smtputils.SendEmail(reqObj.HeadCommit.Author.Email, failedSubject, loadHeaderString+"<br>Local repo content check for updates failed")
		}
		return "An error occurred checking local repo for changes", err
	}

	if repoChanged {
		err = gitutils.CommitAndPush(rootDir, commitAuthor, reqObj.HeadCommit.ID)
		if err != nil {
			Log.Error("An error occurred committing and pushing local repo changes: ", err)
			if mode == asyncMode {
				smtputils.SendEmail(reqObj.HeadCommit.Author.Email, failedSubject, loadHeaderString+"<br>Local repo changes commit and push failed")
			}
			return "An error occurred committing and pushing repo changes", err
		}
	}

	msg := "Successfully imported the course"
	if repoChanged {
		msg = msg + ". Remote Git Repository updated! Run GIT PULL"
	}
	if mode == asyncMode {
		smtputils.SendEmail(reqObj.HeadCommit.Author.Email, okSubject, loadHeaderString +"<br>" + msg)
	}
	return msg, nil
}
