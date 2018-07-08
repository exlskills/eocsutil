package olx

import (
	"encoding/xml"
	"fmt"
	"github.com/exlskills/eocsutil/ir"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

func chaptersToIRChapters(chaps []*Chapter) []ir.Chapter {
	irChaps := make([]ir.Chapter, 0, len(chaps))
	for _, c := range chaps {
		irChaps = append(irChaps, c)
	}
	return irChaps
}

type Chapter struct {
	XMLName     xml.Name      `xml:"chapter"`
	URLName     string        `xml:"url_name,attr"`
	DisplayName string        `xml:"display_name,attr"`
	Sequentials []*Sequential `xml:"sequential"`
	ExtraAttrs  []xml.Attr    `xml:",any,attr"`
}

func (chap *Chapter) resolveRecursive(rootDir string) (err error) {
	if _, err := os.Stat(filepath.Join(rootDir, chapterDirName, urlNameToXMLFileName(chap.URLName))); err == nil {
		fullChapXML, err := ioutil.ReadFile(filepath.Join(rootDir, chapterDirName, urlNameToXMLFileName(chap.URLName)))
		if err != nil {
			return err
		}
		fullChap := &Chapter{}
		err = xml.Unmarshal(fullChapXML, fullChap)
		chap.DisplayName = fullChap.DisplayName
		chap.Sequentials = fullChap.Sequentials
	}
	if chap.DisplayName == "" {
		return errors.New(fmt.Sprintf("invalid chapter: %s", chap.URLName))
	}
	for i := range chap.Sequentials {
		err = chap.Sequentials[i].resolveRecursive(rootDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func (chap *Chapter) GetDisplayName() string {
	return chap.DisplayName
}

func (chap *Chapter) GetURLName() string {
	return chap.URLName
}

func (chap *Chapter) GetExtraAttributes() map[string]string {
	return xmlAttrsToMap(chap.ExtraAttrs)
}

func (chap *Chapter) GetSequentials() []ir.Sequential {
	return sequentialsToIRSequentials(chap.Sequentials)
}
