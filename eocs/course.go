package eocs

import (
	"errors"
	"fmt"
	"github.com/exlskills/eocsutil/ir"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

func resolveCourseRecursive(rootDir string) (*Course, error) {
	rootCourseYAML, err := getIndexYAML(rootDir)
	if err != nil {
		return nil, err
	}
	c := &Course{}
	err = yaml.Unmarshal(rootCourseYAML, c)
	if err != nil {
		return nil, err
	}
	// TODO implement reading the rest of the course
	//if _, err := os.Stat(filepath.Join(rootDir, courseDirName, urlNameToXMLFileName(c.URLName))); err == nil {
	//	fullCourseXML, err := ioutil.ReadFile(filepath.Join(rootDir, courseDirName, urlNameToXMLFileName(c.URLName)))
	//	if err != nil {
	//		return nil, err
	//	}
	//	fullCourseData := &Course{}
	//	err = xml.Unmarshal(fullCourseXML, fullCourseData)
	//	if err != nil {
	//		return nil, err
	//	}
	//	fullCourseData.URLName = c.URLName
	//	if c.Org != "" {
	//		fullCourseData.Org = c.Org
	//	}
	//	if c.CourseCode != "" {
	//		fullCourseData.CourseCode = c.CourseCode
	//	}
	//	c = fullCourseData
	//}
	//for i := range c.Chapters {
	//	err = c.Chapters[i].resolveRecursive(rootDir)
	//	if err != nil {
	//		return nil, err
	//	}
	//}
	return c, nil
}

func exportCourseRecursive(course ir.Course, rootDir string) (err error) {
	if _, err := os.Stat(rootDir); err == nil {
		return errors.New("eocs: specified root course export directory must not exist, in order to ensure that no contents are incidentally overwritten")
	}
	err = os.MkdirAll(rootDir, 0775)
	if err != nil {
		return err
	}
	courseEOCS := &Course{
		URLName:     course.GetURLName(),
		DisplayName: course.GetDisplayName(),
		Org:         course.GetOrgName(),
		CourseCode:  course.GetCourseCode(),
		CourseImage: course.GetCourseImage(),
		Language:    course.GetLanguage(),
	}
	err = writeIndexYAML(rootDir, courseEOCS)
	if err != nil {
		return err
	}

	err = appendIRChaptersToCourse(courseEOCS, course.GetChapters())
	if err != nil {
		return err
	}
	wg := sync.WaitGroup{}
	for chapIdx, chap := range courseEOCS.Chapters {
		wg.Add(1)
		go func(rd string, cIdx int, c *Chapter) {
			Log.Info("Starting to export chapter: ", c.DisplayName)
			err = exportChapterRecursive(rd, cIdx, c)
			Log.Info("Returned from chapter export: ", c.DisplayName)
			if err != nil {
				Log.Fatalf("eocs: chapter export routine encountered fatal error: %s", err.Error())
			}
			wg.Done()
		}(rootDir, chapIdx, chap)
	}
	wg.Wait()
	return nil
}

func exportChapterRecursive(rootDir string, index int, chap *Chapter) (err error) {
	dirName := filepath.Join(rootDir, concatDirName(index, chap.DisplayName))
	err = os.MkdirAll(dirName, 0775)
	if err != nil {
		return err
	}
	err = writeIndexYAML(dirName, chap)
	if err != nil {
		return
	}
	for seqIdx, seq := range chap.Sequentials {
		err = exportSequentialRecursive(dirName, seqIdx, seq)
		if err != nil {
			return
		}
	}
	return
}

func exportSequentialRecursive(rootDir string, index int, seq *Sequential) (err error) {
	dirName := filepath.Join(rootDir, concatDirName(index, seq.DisplayName))
	err = os.MkdirAll(dirName, 0775)
	if err != nil {
		return err
	}
	err = writeIndexYAML(dirName, seq)
	if err != nil {
		return
	}
	for vertIdx, vert := range seq.Verticals {
		err = exportVerticalRecursive(dirName, vertIdx, vert)
		if err != nil {
			return
		}
	}
	return
}

func exportVerticalRecursive(rootDir string, index int, vert *Vertical) (err error) {
	dirName := filepath.Join(rootDir, concatDirName(index, vert.DisplayName))
	err = os.MkdirAll(dirName, 0775)
	if err != nil {
		return err
	}
	err = writeIndexYAML(dirName, vert)
	if err != nil {
		return
	}

	for blkIdx, blk := range vert.Blocks {
		err = exportBlock(dirName, blkIdx, blk)
		if err != nil {
			return
		}
	}
	return
}

func exportBlock(rootDir string, index int, blk *Block) (err error) {
	fileName := filepath.Join(rootDir, concatDirName(index, blk.DisplayName))
	switch blk.GetBlockType() {
	case "exleditor":
		blk.MarshalREPL(rootDir, concatDirName(index, blk.DisplayName))
		return
	case "problem":
		fileName += ".prob.md"
	case "md", "html":
		fileName += ".md"
	default:
		return errors.New(fmt.Sprintf("eocs: unsupported block type: %s", blk.GetBlockType()))
	}
	contents, err := blk.GetContentMD()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fileName, []byte(contents), 0755)
	if err != nil {
		return err
	}
	return nil
}

type Course struct {
	URLName     string     `yaml:"url_name"`
	DisplayName string     `yaml:"display_name"`
	Org         string     `yaml:"org"`
	CourseCode  string     `yaml:"course"`
	CourseImage string     `yaml:"course_image"`
	Language    string     `yaml:"language"`
	Chapters    []*Chapter `yaml:"-"`
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

func (course *Course) GetCourseCode() string {
	return course.CourseCode
}

func (course *Course) GetCourseImage() string {
	return course.CourseImage
}

func (course *Course) GetLanguage() string {
	return course.Language
}

func (course *Course) GetExtraAttributes() map[string]string {
	return map[string]string{}
}

func (course *Course) GetChapters() []ir.Chapter {
	return chaptersToIRChapters(course.Chapters)
}
