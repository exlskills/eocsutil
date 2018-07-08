package olx

import (
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/eocsuri"
	"github.com/exlskills/eocsutil/ir"
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
	// TODO implement OLX export
	return nil
}
