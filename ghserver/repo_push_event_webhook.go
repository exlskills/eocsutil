package ghserver

import (
	"github.com/exlinc/golang-utils/jsonhttp"
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/eocs"
	"github.com/exlskills/eocsutil/ghmodels"
	"github.com/exlskills/eocsutil/gitutils"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
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
	go repoPushEventWebhookProcessor(reqObj)
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
	message, err := repoPushEventWebhookProcessor(reqObj)
	if err != nil {
		jsonhttp.JSONInternalError(w, message, "")
	} else {
		jsonhttp.JSONSuccess(w, nil, message)
	}
	return
}

func repoPushEventWebhookProcessor(reqObj ghmodels.RepoPushEventRequest) (message string, err error){
	Log.Infof("From %s; Head Commit %s ", reqObj.Repository.Name, reqObj.HeadCommit.ID)
	if reqObj.Ref != "refs/heads/master" {
		Log.Info("Skipping push on ref: ", reqObj.Ref)
		return "No-op, must be master branch to sync", nil
	}

	if len(reqObj.Commits) < 1 {
		Log.Info("Skipping. No commits")
		return"No-op, must be commit-based", nil
	}

	hasRealCommits := false
	commitAuthor := ghmodels.CommitAuthor{}
	for _, commit := range reqObj.Commits {
		if !strings.Contains(commit.Message, config.Cfg().GitAutoGenCommitMsg) {
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
		return "An error occurred creating the temp directory", err
	}
	defer os.RemoveAll(rootDir)

	err = gitutils.CloneRepo(reqObj.Repository.CloneURL, rootDir)
	if err != nil {
		Log.Error("An error occurred cloning repo: ", err)
		return"An error occurred cloning repo", err
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
		return "An error occurred importing the course", err
	}

	repoChanged, err := gitutils.IsRepoContentUpdated(rootDir)
	if err != nil {
		Log.Errorf("Course push failed: %s", err.Error())
		return "An error occurred checking local repo for changes", err
	}
	if repoChanged {
		err = gitutils.CommitAndPush(rootDir, commitAuthor)
		if err != nil {
			Log.Error("An error occurred committing and pushing repo changes: ", err)
			return "An error occurred committing and pushing repo changes", err
		}
	}

	return"Successfully imported the course", nil
}
