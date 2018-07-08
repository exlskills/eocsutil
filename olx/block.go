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

type Block struct {
	XMLName          xml.Name
	ContentType      string       `xml:"-"`
	URLName          string       `xml:"url_name,attr"`
	DisplayName      string       `xml:"display_name,attr"`
	Filename         string       `xml:"filename"`
	ExtraAttrs       []xml.Attr   `xml:",any,attr"`
	ContentTree      []*BlockNode `xml:",any"`
	ContentTreeBytes []byte       `xml:"-"`
}

type BlockNode struct {
	XMLName xml.Name
	Nodes   []*BlockNode `xml:",any"`
	Attrs   []xml.Attr   `xml:",any,attr"`
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
		block.DisplayName = fullBlock.DisplayName
		block.Filename = fullBlock.Filename
	}
	if block.ContentType == "html" && block.Filename != "" {
		if _, err := os.Stat(filepath.Join(rootDir, block.XMLName.Local, urlNameToHTMLFileName(block.Filename))); err == nil {
			htmlFileHTML, err := ioutil.ReadFile(filepath.Join(rootDir, block.XMLName.Local, urlNameToHTMLFileName(block.Filename)))
			if err != nil {
				return err
			}
			err = xml.Unmarshal(htmlFileHTML, block.ContentTree)
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
