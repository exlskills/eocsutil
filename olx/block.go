package olx

import (
	"encoding/xml"
	"fmt"
	"github.com/exlskills/eocsutil/ir"
	"github.com/exlskills/eocsutil/mdutils"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
			XMLName:     xml.Name{Local: b.GetBlockType()},
			URLName:     b.GetURLName(),
			DisplayName: b.GetDisplayName(),
			ExtraAttrs:  mapToXMLAttrs(b.GetExtraAttributes()),
		}
		var nodes []*BlockNode
		olxStr, err := b.GetContentOLX()
		Log.Info("olxStr: ", olxStr)
		if err != nil {
			if b.GetBlockType() == "html" {
				md, err := b.GetContentMD()
				if err != nil {
					// There's nothing else that we can do to recover at this point
					return err
				}
				olxStr, err = mdutils.MakeHTML(md, "github")
				Log.Info("md: ", olxStr)
				if err != nil {
					return errors.New(fmt.Sprintf("olx: error converting md to OLX: %s", err.Error()))
				}
			} else {
				return err
			}
		}
		err = xml.Unmarshal([]byte(olxStr), &nodes)
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
	URLName          string       `xml:"url_name,attr,omitempty"`
	DisplayName      string       `xml:"display_name,attr,omitempty"`
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
			block.ContentTreeBytes = htmlFileHTML
			// We have to do this because of a bug/feature in golang: https://github.com/golang/go/issues/20754
			if !strings.HasPrefix(string(htmlFileHTML), "<html>") {
				htmlFileHTML = append([]byte("<html>"), htmlFileHTML...)
				htmlFileHTML = append(htmlFileHTML, []byte("</html>")...)
			}
			err = xml.Unmarshal(htmlFileHTML, &block.ContentTree)
			if err != nil {
				Log.Errorf("Encountered invalid HTML in %s, error: %s", filepath.Join(rootDir, block.XMLName.Local, urlNameToHTMLFileName(block.Filename)), err.Error())
				return err
			}
			// Note: this is again due to the golang bug/feature described above. We will also *always* have the first element even if it's empty since we are adding the <html></html> above
			block.ContentTree = block.ContentTree[0].Nodes
			if block.ContentTree == nil {
				block.ContentTree = []*BlockNode{}
			}
		}
	}
	if block.ContentTree != nil && block.ContentTreeBytes == nil {
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

func (block *Block) GetBlockType() string {
	return block.ContentType
}

func (block *Block) GetContentMD() (string, error) {
	// In OLX, there is only one type that has markdown directly -- problem
	if block.ContentType == "problem" {
		return block.Markdown, nil
	}
	// And we can use mdutils to try to get MD from HTML
	if block.ContentType == "html" {
		return mdutils.MakeMD(string(block.ContentTreeBytes), "github")
	}
	return "", errors.New("olx: unable to return MD for non-problem block")
}

func (block *Block) GetContentOLX() (string, error) {
	// We always have OLX here :)
	return string(block.ContentTreeBytes), nil
}

func (block *Block) GetExtraAttributes() map[string]string {
	return xmlAttrsToMap(block.ExtraAttrs)
}
