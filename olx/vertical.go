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
			ExtraAttrs:  mapToXMLAttrs(v.GetExtraAttributes()),
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
	XMLName     xml.Name   `xml:"vertical"`
	URLName     string     `xml:"url_name,attr"`
	DisplayName string     `xml:"display_name,attr"`
	ExtraAttrs  []xml.Attr `xml:",any,attr"`
	Blocks      []*Block   `xml:",any"`
}

func (vert *Vertical) resolveRecursive(rootDir string) (err error) {
	if _, err := os.Stat(filepath.Join(rootDir, verticalsDirName, urlNameToXMLFileName(vert.URLName))); err == nil {
		fullVertXML, err := ioutil.ReadFile(filepath.Join(rootDir, verticalsDirName, urlNameToXMLFileName(vert.URLName)))
		if err != nil {
			return err
		}
		fullVert := &Vertical{}
		err = xml.Unmarshal(fullVertXML, fullVert)
		if err != nil {
			return err
		}
		vert.DisplayName = fullVert.DisplayName
		vert.Blocks = fullVert.Blocks
	}
	if vert.DisplayName == "" {
		return errors.New(fmt.Sprintf("invalid vertical: %s", vert.URLName))
	}
	for i := range vert.Blocks {
		err = vert.Blocks[i].resolveRecursive(rootDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func (vert *Vertical) GetDisplayName() string {
	return vert.DisplayName
}

func (vert *Vertical) GetURLName() string {
	return vert.URLName
}

func (vert *Vertical) GetExtraAttributes() map[string]string {
	return xmlAttrsToMap(vert.ExtraAttrs)
}

func (vert *Vertical) GetBlocks() []ir.Block {
	return blocksToIRBlocks(vert.Blocks)
}
