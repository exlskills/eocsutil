package eocs

import (
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/ir"
)

var Log = config.Cfg().GetLogger()

func NewEOCSFormat() *EOCS {
	return &EOCS{}
}

type EOCS struct {
}

func (e *EOCS) Import(fromUri string) (toIntermediateRepresentation ir.Course, err error) {
	// TODO implement OLX import
	return nil, nil
}
func (e *EOCS) Export(fromIntermediateRepresentation ir.Course, toUri string, forceExport bool) (err error) {
	// TODO implement OLX export
	return nil
}
