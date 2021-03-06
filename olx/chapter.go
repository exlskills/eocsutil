package olx

import (
	"encoding/xml"
	"fmt"
	"github.com/exlskills/eocsutil/ir"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

func chaptersToIRChapters(chaps []*Chapter) []ir.Chapter {
	irChaps := make([]ir.Chapter, 0, len(chaps))
	for _, c := range chaps {
		irChaps = append(irChaps, c)
	}
	return irChaps
}

func appendIRChaptersToCourse(course *Course, chaps []ir.Chapter) (err error) {
	course.Chapters = make([]*Chapter, 0, len(chaps))
	for _, c := range chaps {
		newC := &Chapter{
			URLName:     c.GetURLName(),
			DisplayName: c.GetDisplayName(),
			ExtraAttrs:  mapToXMLAttrs(c.GetExtraAttributes()),
		}
		err = appendIRSequentialsToChapter(newC, c.GetSequentials())
		if err != nil {
			return err
		}
		course.Chapters = append(course.Chapters, newC)
	}
	return nil
}

type Chapter struct {
	XMLName     xml.Name      `xml:"chapter"`
	URLName     string        `xml:"url_name,attr"`
	DisplayName string        `xml:"display_name,attr"`
	Sequentials []*Sequential `xml:"sequential"`
	ExtraAttrs  []xml.Attr    `xml:",any,attr"`
	UpdatedAt   time.Time
}

func (chap *Chapter) resolveRecursive(rootDir string) (err error) {
	if _, err := os.Stat(filepath.Join(rootDir, chapterDirName, urlNameToXMLFileName(chap.URLName))); err == nil {
		fullChapXML, err := ioutil.ReadFile(filepath.Join(rootDir, chapterDirName, urlNameToXMLFileName(chap.URLName)))
		if err != nil {
			return err
		}
		fullChap := &Chapter{}
		err = xml.Unmarshal(fullChapXML, fullChap)
		if err != nil {
			return err
		}
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

func (chap *Chapter) SetUpdatedAt(updatedAt time.Time) {
	chap.UpdatedAt = updatedAt
}