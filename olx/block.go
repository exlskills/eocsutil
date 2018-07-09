package olx

import (
	"encoding/xml"
	"github.com/exlskills/eocsutil/ir"
	"io/ioutil"
	"os"
	"path/filepath"
)

func blocksToIRBlocks(blocks []*Block) []ir.Block {
	irBlocks := make([]ir.Block, 0, len(blocks))
	for _, c := range blocks {
		irBlocks = append(irBlocks, c)
	}
	return irBlocks
}

func appendIRBlocksToVertical(vert *Vertical, blocks []ir.Block) (err error) {
	vert.Blocks = make([]*Block, 0, len(blocks))
	for _, b := range blocks {
		newB := &Block{
			XMLName:     xml.Name{Local: b.GetContentType()},
			URLName:     b.GetURLName(),
			DisplayName: b.GetDisplayName(),
			ExtraAttrs:  mapToXMLAttrs(b.GetExtraAttributes()),
		}
		var nodes []*BlockNode
		err = xml.Unmarshal([]byte(b.GetContent()), &nodes)
		if err != nil {
			return err
		}
		newB.ContentTree = nodes
		vert.Blocks = append(vert.Blocks, newB)
	}
	return nil
}

type Block struct {
	XMLName          xml.Name
	ContentType      string       `xml:"-"`
	URLName          string       `xml:"url_name,attr"`
	DisplayName      string       `xml:"display_name,attr"`
	Filename         string       `xml:"filename,attr,omitempty"`
	Markdown         string       `xml:"markdown,attr,omitempty"`
	ExtraAttrs       []xml.Attr   `xml:",any,attr"`
	ContentTree      []*BlockNode `xml:",any"`
	ContentTreeBytes []byte       `xml:"-"`
}

type BlockNode struct {
	XMLName  xml.Name
	CharData string       `xml:",chardata"`
	Nodes    []*BlockNode `xml:",any"`
	Attrs    []xml.Attr   `xml:",any,attr"`
}

func (block *Block) resolveRecursive(rootDir string) (err error) {
	block.ContentType = block.XMLName.Local
	if _, err := os.Stat(filepath.Join(rootDir, block.XMLName.Local, urlNameToXMLFileName(block.URLName))); err == nil {
		fullBlockXML, err := ioutil.ReadFile(filepath.Join(rootDir, block.XMLName.Local, urlNameToXMLFileName(block.URLName)))
		if err != nil {
			return err
		}
		fullBlock := &Block{}
		err = xml.Unmarshal(fullBlockXML, fullBlock)
		if err != nil {
			return err
		}
		block.DisplayName = fullBlock.DisplayName
		block.Filename = fullBlock.Filename
		block.Markdown = fullBlock.Markdown
		block.ExtraAttrs = fullBlock.ExtraAttrs
		block.ContentTree = fullBlock.ContentTree
	}

	if block.ContentType == "html" && block.Filename != "" {
		if _, err := os.Stat(filepath.Join(rootDir, block.XMLName.Local, urlNameToHTMLFileName(block.Filename))); err == nil {
			htmlFileHTML, err := ioutil.ReadFile(filepath.Join(rootDir, block.XMLName.Local, urlNameToHTMLFileName(block.Filename)))
			if err != nil {
				return err
			}
			err = xml.Unmarshal(htmlFileHTML, &block.ContentTree)
			if err != nil {
				return err
			}
		}
	}
	if block.ContentTree != nil {
		contentBytes, err := xml.Marshal(block.ContentTree)
		if err != nil {
			return err
		}
		block.ContentTreeBytes = contentBytes
	}
	return nil
}

func (block *Block) GetDisplayName() string {
	return block.DisplayName
}

func (block *Block) GetURLName() string {
	return block.URLName
}

func (block *Block) GetContentType() string {
	return block.ContentType
}

func (block *Block) GetContent() string {
	return string(block.ContentTreeBytes)
}

func (block *Block) GetExtraAttributes() map[string]string {
	return xmlAttrsToMap(block.ExtraAttrs)
}
