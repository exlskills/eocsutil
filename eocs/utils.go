package eocs

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
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
