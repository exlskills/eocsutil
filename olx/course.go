package olx

import (
	"encoding/xml"
	"github.com/exlskills/eocsutil/ir"
	"io/ioutil"
	"os"
	"path/filepath"
)

func resolveCourseRecursive(rootDir string) (*Course, error) {
	rootCourseXML, err := ioutil.ReadFile(filepath.Join(rootDir, "course.xml"))
	if err != nil {
		return nil, err
	}
	c := &Course{}
	err = xml.Unmarshal(rootCourseXML, c)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(rootDir, courseDirName, urlNameToXMLFileName(c.URLName))); err == nil {
		fullCourseXML, err := ioutil.ReadFile(filepath.Join(rootDir, courseDirName, urlNameToXMLFileName(c.URLName)))
		if err != nil {
			return nil, err
		}
		fullCourseData := &Course{}
		err = xml.Unmarshal(fullCourseXML, fullCourseData)
		if err != nil {
			return nil, err
		}
		fullCourseData.URLName = c.URLName
		if c.Org != "" {
			fullCourseData.Org = c.Org
		}
		if c.Course != "" {
			fullCourseData.Course = c.Course
		}
		c = fullCourseData
	}
	for i := range c.Chapters {
		err = c.Chapters[i].resolveRecursive(rootDir)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

type Course struct {
	XMLName     xml.Name   `xml:"course"`
	URLName     string     `xml:"url_name,attr"`
	DisplayName string     `xml:"display_name,attr"`
	Org         string     `xml:"org,attr"`
	Course      string     `xml:"course,attr"`
	CourseImage string     `xml:"course_image,attr"`
	Language    string     `xml:"language,attr"`
	ExtraAttrs  []xml.Attr `xml:",any,attr"`
	Chapters    []*Chapter `xml:"chapter"`
}

func (course *Course) GetDisplayName() string {
	return course.DisplayName
}

func (course *Course) GetURLName() string {
	return course.URLName
}

func (course *Course) GetOrgName() string {
	return course.Org
}

func (course *Course) GetCourseImage() string {
	return course.CourseImage
}

func (course *Course) GetLanguage() string {
	return course.Language
}

func (course *Course) GetExtraAttributes() map[string]string {
	return xmlAttrsToMap(course.ExtraAttrs)
}

func (course *Course) GetChapters() []ir.Chapter {
	return chaptersToIRChapters(course.Chapters)
}
