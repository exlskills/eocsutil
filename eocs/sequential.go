package eocs

import (
	"github.com/exlskills/eocsutil/ir"
	"time"
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
	URLName     string      `yaml:"url_name"`
	DisplayName string      `yaml:"display_name"`
	Graded      bool        `yaml:"graded"`
	Format      string      `yaml:"format"`
	Verticals   []*Vertical `yaml:"-"`
	UpdatedAt   time.Time   `yaml:"-"`
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
	return map[string]string{}
}

func (seq *Sequential) GetVerticals() []ir.Vertical {
	return verticalsToIRVerticals(seq.Verticals)
}

func (seq *Sequential) SetUpdatedAt(updatedAt time.Time) {
	seq.UpdatedAt = updatedAt
}
