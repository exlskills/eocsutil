package pdf

import (
	"fmt"
	"github.com/exlskills/eocsutil/ir"
	"github.com/exlskills/eocsutil/mdutils"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func exportCourseRecursive(course ir.Course, rootDir string) (err error) {
	courseMD, err := concatMarkdownFile(course)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(rootDir, "book.md"), []byte(courseMD), 0755)
	if err != nil {
		return err
	}
	err = mdutils.RunMarkdownPDFConverter(filepath.Join(rootDir, "book.md"), filepath.Join(rootDir, "book.pdf"))
	if err != nil {
		return err
	}
	return nil
}

func concatMarkdownFile(course ir.Course) (string, error) {
	mdStr := strings.Builder{}
	for _, chap := range course.GetChapters() {
		// Write the chapter heading
		mdStr.WriteString(fmt.Sprintf("\n# %s\n\n", chap.GetDisplayName()))
		for _, seq := range chap.GetSequentials() {
			// Write the section heading
			mdStr.WriteString(fmt.Sprintf("\n# %s\n\n", seq.GetDisplayName()))
			for _, vert := range seq.GetVerticals() {
				mdStr.WriteString(fmt.Sprintf("\n## %s\n\n", vert.GetDisplayName()))
				for _, blk := range vert.GetBlocks() {
					if blk.GetBlockType() != "html" {
						continue
					}
					if m, err := blk.GetContentMD(); err != nil {
						Log.Errorf("Encountered error getting markdown on block: %s; error: %s", blk.GetURLName(), err.Error())
						return "", err
					} else {
						mdStr.WriteString(m)
					}

				}
			}
		}
	}
	return mdStr.String(), nil
}
