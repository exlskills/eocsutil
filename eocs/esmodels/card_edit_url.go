package esmodels

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
)

func GenerateCardEditURL(courseRepoUrl string, cardFSPath string) (string, error) {
	parsedRepoUrl, err := url.Parse(courseRepoUrl)
	if err != nil {
		return "", err
	}
	cardFSPathPartsEncoded := strings.Split(cardFSPath, "/")
	for i := range cardFSPathPartsEncoded {
		cardFSPathPartsEncoded[i] = url.PathEscape(cardFSPathPartsEncoded[i])
	}
	switch parsedRepoUrl.Host {
	case "github.com":
		return fmt.Sprintf("https://github.com%s", path.Join(parsedRepoUrl.Path, "edit/master", strings.Join(cardFSPathPartsEncoded, "/"))), nil
	default:
		return "", errors.New("unsupported course repo url host")
	}
}
