package ghserver

import (
	"github.com/exlinc/golang-utils/jsonhttp"
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/eocs"
	"github.com/exlskills/eocsutil/ghmodels"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

func repoPushEventWebhook(w http.ResponseWriter, r *http.Request) {
	reqObj := ghmodels.RepoPushEventRequest{}
	err := ghmodels.GHSecureJSONDecodeAndCatchForAPI(w, r, config.Cfg().GHWebhookSecret, &reqObj)
	if err != nil {
		return
	}
	if reqObj.Ref != "refs/heads/master" {
		Log.Info("Skipping push on ref: ", reqObj.Ref)
		jsonhttp.JSONSuccess(w, nil, "No-op, must be master branch to sync")
		return
	}
	rootDir, err := ioutil.TempDir("", "eocsutil-repo-dl-")
	if err != nil {
		Log.Error("An error occurred creating the temp directory: ", err)
		jsonhttp.JSONInternalError(w, "An error occurred creating the temp directory", "")
		return
	}
	defer os.RemoveAll(rootDir)
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
	err = eocs.NewEOCSFormat().Push(courseDir, config.Cfg().GHServerMongoURI)
	if err != nil {
		Log.Errorf("Course mongodb push failed: %s", err.Error())
		jsonhttp.JSONInternalError(w, "An error occurred importing the course", "")
		return
	}
	jsonhttp.JSONSuccess(w, nil, "Successfully imported the course")
	return
}
