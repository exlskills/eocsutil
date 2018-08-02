package eocs

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/exlskills/eocsutil/ir"
	"github.com/exlskills/eocsutil/mdutils"
	"github.com/exlskills/eocsutil/wsenv"
	"gopkg.in/yaml.v2"
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
		Log.Info("Importing block ...")
		defer Log.Info("Completed block ...")
		newB := &Block{
			BlockType:   b.GetBlockType(),
			URLName:     b.GetURLName(),
			DisplayName: b.GetDisplayName(),
		}
		if newB.BlockType == "problem" {
			md, err := b.GetContentMD()
			if err != nil {
				return err
			}
			newB.Markdown = md
		} else if newB.BlockType == "exleditor" {
			err = newB.UnmarshalREPL(b.GetExtraAttributes()["editor_config"])
			if err != nil {
				return err
			}
		} else if newB.BlockType == "html" {
			html, err := b.GetContentOLX()
			if err != nil {
				return err
			}
			md, err := mdutils.MakeMD(html, "github")
			if err != nil {
				return err
			}
			newB.Markdown = md
		} else {
			return errors.New(fmt.Sprintf("eocs: unsupported block type: %s", newB.BlockType))
		}
		vert.Blocks = append(vert.Blocks, newB)
	}
	return nil
}

type Block struct {
	BlockType   string     `yaml:"-"`
	URLName     string     `yaml:"url_name"`
	DisplayName string     `yaml:"display_name"`
	Markdown    string     `yaml:"-"`
	REPL        *BlockREPL `yaml:"-"`
}

func (block *Block) GetDisplayName() string {
	return block.DisplayName
}

func (block *Block) GetURLName() string {
	return block.URLName
}

func (block *Block) GetBlockType() string {
	return block.BlockType
}

func (block *Block) GetContentMD() (string, error) {
	// We always have MD
	return block.Markdown, nil
}

func (block *Block) GetContentOLX() (string, error) {
	// TODO if content type == HTML then we can (and should) easily convert out...
	return "", errors.New("eocs: does not yet support OLX content conversion, please use markdown")
}

func (block *Block) GetExtraAttributes() map[string]string {
	return map[string]string{}
}

func (block *Block) UnmarshalREPL(cfgJSON string) (err error) {
	wspc := wsenv.Workspace{}
	err = json.Unmarshal([]byte(cfgJSON), &wspc)
	if err != nil {
		return err
	}
	block.REPL = &BlockREPL{
		APIVersion:     1,
		EnvironmentKey: wspc.EnvironmentKey,
		Display: &BlockREPLDisplay{
			Height: "500px",
		},
		SrcFiles: wspc.Files["src"].Children["main"].Children["java"].Children["exlhub"].Children,
	}
	return nil
}

func (block *Block) MarshalREPL(rootDir, baseName string) (err error) {
	block.REPL.SourcePath = filepath.Join(".", baseName+".repl", "src")
	block.REPL.TestPath = filepath.Join(".", baseName+".repl", "test")
	outYAML, err := yaml.Marshal(block.REPL)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(rootDir, baseName+".repl.yaml"), outYAML, 0755)
	if err != nil {
		return err
	}

	filesDir := filepath.Join(rootDir, baseName+".repl")
	err = os.MkdirAll(filesDir, 0755)
	if err != nil {
		return err
	}

	srcDir := filepath.Join(filesDir, "src")
	err = os.MkdirAll(srcDir, 0755)
	if err != nil {
		return err
	}
	for name, file := range block.REPL.SrcFiles {
		err = ioutil.WriteFile(filepath.Join(srcDir, name), []byte(file.Contents), 0755)
		if err != nil {
			return err
		}
	}

	testDir := filepath.Join(filesDir, "test")
	err = os.MkdirAll(testDir, 0755)
	if err != nil {
		return err
	}
	for name, file := range block.REPL.TestFiles {
		err = ioutil.WriteFile(filepath.Join(testDir, name), []byte(file.Contents), 0755)
		if err != nil {
			return err
		}
	}
	return nil
}
