package pdf

import (
	"errors"
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/eocsuri"
	"github.com/exlskills/eocsutil/ir"
	"os"
)

var Log = config.Cfg().GetLogger()

func NewPDFExtFmt() *PDF {
	return &PDF{}
}

type PDF struct {
}

func (o *PDF) Import(fromUri string) (toIntermediateRepresentation ir.Course, err error) {
	return nil, errors.New("pdf extfmt does not support import")
}

func (o *PDF) Export(fromIntermediateRepresentation ir.Course, toUri string, forceExport bool) (err error) {
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
