package eocs

import (
	"errors"
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/eocsuri"
	"github.com/exlskills/eocsutil/ir"
	"os"
)

var Log = config.Cfg().GetLogger()

func NewEOCSFormat() *EOCS {
	return &EOCS{}
}

type EOCS struct {
}

func (e *EOCS) Import(fromUri string) (toIntermediateRepresentation ir.Course, err error) {
	rootDir, err := eocsuri.GetAbsolutePathFromFileURI(fromUri)
	if err != nil {
		return nil, err
	}
	return resolveCourseRecursive(rootDir)
}

func (e *EOCS) Export(fromIntermediateRepresentation ir.Course, toUri string, forceExport bool) (err error) {
	rootDir, err := eocsuri.GetAbsolutePathFromFileURI(toUri)
	if err != nil {
		return err
	}
	if forceExport {
		err = os.RemoveAll(rootDir)
		if err != nil {
			return err
		}
	}
	return exportCourseRecursive(fromIntermediateRepresentation, rootDir)
}

func (e *EOCS) Push(fromUri, toUri string) error {
	rootDir, err := eocsuri.GetAbsolutePathFromFileURI(fromUri)
	if err != nil {
		return err
	}
	course, err := resolveCourseRecursive(rootDir)
	if err != nil {
		return err
	}
	Log.Info("Course import complete!")
	dbName := os.Getenv("MGO_DB_NAME")
	if dbName == "" {
		return errors.New("for EOCS course conversion the MGO_DB_NAME environment variable must be set to the name of the MongoDB database to write to")
	}
	return upsertCourseRecursive(course, toUri, dbName)
}
