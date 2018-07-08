package extfmt

import "github.com/exlskills/eocsutil/ir"

type ExtFmt interface {
	Import(fromUri string) (toIntermediateRepresentation ir.Course, err error)
	Export(fromIntermediateRepresentation ir.Course, toUri string, forceExport bool) (err error)
}
