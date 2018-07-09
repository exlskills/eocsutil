package olx

import (
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/exlskills/eocsutil/ir"
	"io/ioutil"
	"os"
	"path/filepath"
)

func sequentialsToIRSequentials(seqs []*Sequential) []ir.Sequential {
	irSeqs := make([]ir.Sequential, 0, len(seqs))
	for _, s := range seqs {
		irSeqs = append(irSeqs, s)
	}
	return irSeqs
}

func appendIRSequentialsToChapter(chap *Chapter, seqs []ir.Sequential) (err error) {
	chap.Sequentials = make([]*Sequential, 0, len(seqs))
	for _, s := range seqs {
		newS := &Sequential{
			URLName:     s.GetURLName(),
			DisplayName: s.GetDisplayName(),
			Graded:      s.GetIsGraded(),
			Format:      s.GetAssignmentType(),
			ExtraAttrs:  mapToXMLAttrs(s.GetExtraAttributes()),
		}
		err = appendIRVerticalsToSequential(newS, s.GetVerticals())
		if err != nil {
			return err
		}
		chap.Sequentials = append(chap.Sequentials, newS)
	}
	return nil
}

type Sequential struct {
	XMLName     xml.Name    `xml:"sequential"`
	URLName     string      `xml:"url_name,attr"`
	DisplayName string      `xml:"display_name,attr"`
	Graded      bool        `xml:"graded,attr"`
	Format      string      `xml:"format,attr"`
	ExtraAttrs  []xml.Attr  `xml:",any,attr"`
	Verticals   []*Vertical `xml:"vertical"`
}

func (seq *Sequential) resolveRecursive(rootDir string) (err error) {
	if _, err := os.Stat(filepath.Join(rootDir, sequentialsDirName, urlNameToXMLFileName(seq.URLName))); err == nil {
		fullSeqXML, err := ioutil.ReadFile(filepath.Join(rootDir, sequentialsDirName, urlNameToXMLFileName(seq.URLName)))
		if err != nil {
			return err
		}
		fullSeq := &Sequential{}
		err = xml.Unmarshal(fullSeqXML, fullSeq)
		seq.DisplayName = fullSeq.DisplayName
		seq.Graded = fullSeq.Graded
		seq.Format = fullSeq.Format
		seq.Verticals = fullSeq.Verticals
	}
	if seq.DisplayName == "" {
		return errors.New(fmt.Sprintf("invalid sequential: %s", seq.URLName))
	}
	for i := range seq.Verticals {
		err = seq.Verticals[i].resolveRecursive(rootDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func (seq *Sequential) GetDisplayName() string {
	return seq.DisplayName
}

func (seq *Sequential) GetURLName() string {
	return seq.URLName
}

func (seq *Sequential) GetIsGraded() bool {
	return seq.Graded
}

func (seq *Sequential) GetAssignmentType() string {
	return seq.Format
}

func (seq *Sequential) GetExtraAttributes() map[string]string {
	return xmlAttrsToMap(seq.ExtraAttrs)
}

func (seq *Sequential) GetVerticals() []ir.Vertical {
	return verticalsToIRVerticals(seq.Verticals)
}
