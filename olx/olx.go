package olx

import (
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/eocsuri"
	"github.com/exlskills/eocsutil/ir"
	"os"
)

var Log = config.Cfg().GetLogger()

func NewOLXExtFmt() *OLX {
	return &OLX{}
}

type OLX struct {
}

func (o *OLX) Import(fromUri string) (toIntermediateRepresentation ir.Course, err error) {
	rootDir, err := eocsuri.GetAbsolutePathFromFileURI(fromUri)
	if err != nil {
		return nil, err
	}
	return resolveCourseRecursive(rootDir)
}

func (o *OLX) Export(fromIntermediateRepresentation ir.Course, toUri string, forceExport bool) (err error) {
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
	err = os.MkdirAll(rootDir, 0775)
	if err != nil {
		return err
	}
	return exportCourseRecursive(fromIntermediateRepresentation, rootDir)
}
