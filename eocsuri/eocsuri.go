package eocsuri

import (
	"fmt"
	"strings"
	"github.com/pkg/errors"
	"path/filepath"
)

func VerifyAndClean(uri string) (clean string, err error) {
	// TODO support more URI formats in the future
	if !strings.HasPrefix(uri, "file://") {
		uri = fmt.Sprintf("file://%s", uri)
	}
	return StrictVerifyAndClean(uri)
}

func StrictVerifyAndClean(uri string) (clean string, err error) {
	// TODO support more URI formats in the future
	if !strings.HasPrefix(uri, "file://") {
		return "", errors.New("invalid URI protocol")
	}
	// TODO validate/clean the URI
	return uri, nil
}

func GetAbsolutePathFromFileURI(uri string) (string, error) {
	if strings.HasPrefix(uri, "file://") {
		uri = strings.Replace(uri, "file://", "", 1)
	}
	return filepath.Abs(uri)
}
