package eocs

import (
	"github.com/exlskills/eocsutil/ir"
)

func verticalsToIRVerticals(verts []*Vertical) []ir.Vertical {
	irVerts := make([]ir.Vertical, 0, len(verts))
	for _, c := range verts {
		irVerts = append(irVerts, c)
	}
	return irVerts
}

func appendIRVerticalsToSequential(seq *Sequential, verts []ir.Vertical) (err error) {
	seq.Verticals = make([]*Vertical, 0, len(verts))
	for _, v := range verts {
		newV := &Vertical{
			URLName:     v.GetURLName(),
			DisplayName: v.GetDisplayName(),
		}
		err = appendIRBlocksToVertical(newV, v.GetBlocks())
		if err != nil {
			return err
		}
		seq.Verticals = append(seq.Verticals, newV)
	}
	return nil
}

type Vertical struct {
	URLName     string   `yaml:"url_name"`
	DisplayName string   `yaml:"display_name"`
	Blocks      []*Block `yaml:"-"`
}

func (vert *Vertical) GetDisplayName() string {
	return vert.DisplayName
}

func (vert *Vertical) GetURLName() string {
	return vert.URLName
}

func (vert *Vertical) GetExtraAttributes() map[string]string {
	return map[string]string{}
}

func (vert *Vertical) GetBlocks() []ir.Block {
	return blocksToIRBlocks(vert.Blocks)
}
