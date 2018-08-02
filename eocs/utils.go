package eocs

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

func getIndexYAML(indexDir string) (contents []byte, err error) {
	possiblePaths := []string{
		filepath.Join(indexDir, "index.yaml"),
		filepath.Join(indexDir, "index.yml"),
	}
	for _, path := range possiblePaths {
		contents, err = ioutil.ReadFile(path)
		if contents != nil && err == nil {
			return
		}
	}
	return nil, errors.New("unable to read index configuration, check that the index file exists with at least read permissions")
}

func writeIndexYAML(indexDir string, object interface{}) (err error) {
	outYAML, err := yaml.Marshal(object)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(indexDir, "index.yaml"), outYAML, 0755)
	if err != nil {
		return err
	}
	return
}

func concatDirName(index int, dispName string) string {
	return fmt.Sprintf("%02d_%s", index, dispName)
}

func indexAndNameFromConcatenated(concated string) (idx int, name string, err error) {
	if !strings.Contains(concated, "_") {
		return 0, "", errors.New("eocs: dir/file name must take the form of 02_File Name where 02 is the index and 'File Name' is the name of the chapter/sequential/vertical/block")
	}
	splitStr := strings.SplitN(concated, "_", 2)
	if len(splitStr) != 2 {
		return 0, "", errors.New("eocs: dir/file name must take the form of 02_File Name where 02 is the index and 'File Name' is the name of the chapter/sequential/vertical/block")
	}
	idx, err = strconv.Atoi(splitStr[0])
	if err != nil {
		return 0, "", err
	}
	return idx, splitStr[1], nil
}

var ignoredDirs = map[string]struct{}{
	".git": {},
	".hg":  {},
	".bzr": {},
	".":    {},
}

func isIgnoredDir(name string) bool {
	if _, exists := ignoredDirs[name]; exists {
		return true
	}
	return false
}