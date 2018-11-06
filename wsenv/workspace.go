package wsenv

import (
	"errors"
	"github.com/exlskills/eocsutil/config"
	"strings"
)

var Log = config.Cfg().GetLogger()

type Workspace struct {
	Id             string                    `json:"id"`
	Name           string                    `json:"name"`
	EnvironmentKey string                    `json:"environmentKey,omitempty"`
	VersionId      string                    `json:"versionId,omitempty"`
	Files          map[string]*WorkspaceFile `json:"files"`
}

func (wfs *Workspace) SetupFileSystem(rootPath string) error {
	for key := range wfs.Files {
		err := wfs.Files[key].Initialize(rootPath, key, 0)
		if err != nil {
			return err
		}
	}
	return nil
}

// Returns: indices to find file from root filesystem map, pointer to WorkspaceFile, relative path to file, error
func (wfs *Workspace) GetFileRef(isSudoMode bool, relPath string) ([]string, *WorkspaceFile, string, error) {
	Log.Info("---------------> in GetFileRef  ")
	inds := make([]string, 0)
	if err := basicFileNameCheck(relPath); err != nil {
		return inds, nil, "", err
	}
	if strings.HasPrefix(relPath, "/") {
		relPath = relPath[1:]
	} else if strings.HasPrefix(relPath, "./") {
		relPath = relPath[2:]
	}
	pathParts := strings.Split(relPath, "/")
	depth := len(pathParts)
	curFileArr := wfs.Files
	for outerInd := 0; outerInd < depth; outerInd++ {
		foundPart := false
		atInd := ""
		// TODO can this be streamlined to just read from the map?
		/*
			curFileArr[pathParts[outerInd]] ?
		*/
		for innerInd, curFile := range curFileArr {
			if curFile.Name == pathParts[outerInd] {
				inds = append(inds, innerInd)
				if depth == outerInd+1 {
					return inds, curFile, relPath, nil
				}
				if !isSudoMode && curFile.IsHidden {
					return inds, nil, relPath, errors.New("No valid file found.")
				}
				foundPart = true
				atInd = innerInd
				break
			}
		}
		if !foundPart || (foundPart && !curFileArr[atInd].IsDir) {
			return inds, nil, relPath, errors.New("No valid file found.")
		}
		curFileArr = curFileArr[atInd].Children
	}
	return inds, nil, relPath, errors.New("File not found.")
}
