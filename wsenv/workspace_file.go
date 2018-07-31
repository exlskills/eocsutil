package wsenv

import (
	"encoding/json"
	"errors"
	"exlgit.com/nva/ecmd/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var ErrInvalidWorkspaceFileName = errors.New("Invalid file name.")
var ErrRecursiveWorkspaceFileStructure = errors.New("Workspace file structure is too deep. Remove at least one directory level before trying again.")
var ErrInvalidWorkspaceFileStructure = errors.New("Invalid file structure. A non-directory may not contain children.")
var ErrCannotMarshalHiddenFileToJSON = errors.New("Cannot marshal hidden file to JSON.")

type WorkspaceFile struct {
	Name        string                    `json:"name"`
	IsDir       bool                      `json:"isDir"`
	IsTmplFile  bool                      `json:"isTmplFile"`
	IsImmutable bool                      `json:"isImmutable"`
	IsHidden    bool                      `json:"isHidden"`
	Contents    string                    `json:"contents,omitempty"`
	Children    map[string]*WorkspaceFile `json:"children,omitempty"`
}

func (wf *WorkspaceFile) MarshalJSON() ([]byte, error) {
	if wf.IsHidden {
		return nil, ErrCannotMarshalHiddenFileToJSON
	}
	return json.Marshal(&struct {
		Name        string                    `json:"name"`
		IsDir       bool                      `json:"isDir"`
		IsTmplFile  bool                      `json:"isTmplFile"`
		IsImmutable bool                      `json:"isImmutable"`
		Contents    string                    `json:"contents,omitempty"`
		Children    map[string]*WorkspaceFile `json:"children,omitempty"`
	}{
		Name:        wf.Name,
		IsDir:       wf.IsDir,
		IsTmplFile:  wf.IsTmplFile,
		IsImmutable: wf.IsImmutable,
		Contents:    wf.Contents,
		Children:    wf.Children,
	})
}

func (wf *WorkspaceFile) Initialize(rootDir, key string, curDepth int64) error {
	if curDepth > 14 {
		return ErrRecursiveWorkspaceFileStructure
	}
	wf.Name = key
	if err := wf.ValidateNameAndStructure(); err != nil {
		return err
	}
	// This is a single file
	if !wf.IsDir {
		err := ioutil.WriteFile(filepath.Join(rootDir, wf.Name), []byte(wf.Contents), 0664)
		return err
	}
	// This is a directory
	curPath := filepath.Join(rootDir, wf.Name)
	err := os.Mkdir(curPath, 0775)
	if err != nil && !strings.HasSuffix(err.Error(), "file exists") {
		return err
	}
	for k, c := range wf.Children {
		err := c.Initialize(curPath, k, curDepth+1)
		if err != nil {
			return err
		}
	}
	return nil
}

func (wf *WorkspaceFile) UpdateContents(isSudoMode bool, fullPath, contents string) error {
	if err := wf.ValidateNameAndStructure(); err != nil {
		return err
	}
	if !isSudoMode && (wf.IsHidden || wf.IsImmutable) {
		return errors.New("Invalid file")
	}
	err := ioutil.WriteFile(fullPath, []byte(contents), 0664)
	if err != nil {
		return err
	}
	wf.Contents = contents
	return nil
}

func (wf *WorkspaceFile) UpdateOrCreateDirectory(isSudoMode bool, fullPath string) error {
	if err := wf.ValidateNameAndStructure(); err != nil {
		return err
	}
	if !isSudoMode && (wf.IsHidden || wf.IsImmutable) {
		return errors.New("Invalid file")
	}
	err := os.Mkdir(fullPath, 0775)
	if err != nil && !strings.HasSuffix(err.Error(), "file exists") {
		return err
	}
	return nil
}

func (wf *WorkspaceFile) RemoveFromSystemAndDrop(isSudoMode bool, wspc *Workspace, pathParts []string, fullPath string) error {
	if err := wf.ValidateNameAndStructure(); err != nil {
		return err
	}
	if !isSudoMode && (wf.IsHidden || wf.IsImmutable) {
		return errors.New("Invalid file")
	}
	depth := len(pathParts)
	curFileArr := wspc.Files
	for outerInd := 0; outerInd < depth; outerInd++ {
		foundPart := false
		atInd := ""
		for innerInd, curFile := range curFileArr {
			if curFile.Name == pathParts[outerInd] {
				if depth == outerInd+1 {
					delete(curFileArr, curFile.Name)
					if curFile.IsDir {
						return utils.RemoveDirectoryContentsRecursively(fullPath)
					} else {
						return os.Remove(fullPath)
					}
				}
				foundPart = true
				atInd = innerInd
				break
			}
		}
		if !foundPart || (foundPart && !curFileArr[atInd].IsDir) {
			return errors.New("No valid file found.")
		}
		curFileArr = curFileArr[atInd].Children
	}
	return os.Remove(fullPath)
}

func (wf *WorkspaceFile) WriteToSystemAndAdd(isSudoMode bool, wspc *Workspace, pathParts []string, fullPath string) error {
	if err := wf.ValidateNameAndStructure(); err != nil {
		return err
	}
	if !isSudoMode && (wf.IsHidden || wf.IsImmutable) {
		return errors.New("Invalid file")
	}
	depth := len(pathParts) - 1
	if depth < 1 {
		wspc.Files[wf.Name] = wf
		if wf.IsDir {
			return wf.UpdateOrCreateDirectory(isSudoMode, fullPath)
		} else {
			return wf.UpdateContents(isSudoMode, fullPath, wf.Contents)
		}
	}
	curFileArr := wspc.Files
	for outerInd := 0; outerInd < depth; outerInd++ {
		foundPart := false
		atInd := ""
		for innerInd, curFile := range curFileArr {
			if curFile.Name == pathParts[outerInd] {
				if depth == outerInd+1 && curFile.IsDir {
					if curFileArr[curFile.Name].Children == nil {
						curFileArr[curFile.Name].Children = make(map[string]*WorkspaceFile)
					}
					curFileArr[curFile.Name].Children[wf.Name] = wf
					if wf.IsDir {
						return wf.UpdateOrCreateDirectory(isSudoMode, fullPath)
					} else {
						return wf.UpdateContents(isSudoMode, fullPath, wf.Contents)
					}
				} else if depth == outerInd+1 {
					return errors.New("The destination file's parent is not a directory")
				}
				foundPart = true
				atInd = innerInd
				break
			}
		}
		if !foundPart || (foundPart && !curFileArr[atInd].IsDir) {
			return errors.New("No valid file found.")
		}
		curFileArr = curFileArr[atInd].Children
	}
	return nil
}

func (wf *WorkspaceFile) ValidateNameAndStructure() error {
	if err := basicFileNameCheck(wf.Name); err != nil {
		return err
	}
	if !wf.IsDir && (strings.Contains(wf.Name, string(os.PathListSeparator)) || strings.Contains(wf.Name, string(os.PathSeparator))) {
		return ErrInvalidWorkspaceFileName
	}
	if !wf.IsDir && len(wf.Children) > 0 {
		return ErrInvalidWorkspaceFileStructure
	}
	return nil
}

func basicFileNameCheck(pathToCheck string) error {
	if pathToCheck == "" || pathToCheck == "." || strings.Contains(pathToCheck, "..") || strings.Contains(pathToCheck, "...") {
		return ErrInvalidWorkspaceFileName
	}
	return nil
}
