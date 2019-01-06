package eocs

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
)

type CPEditorEmbeddedBlock struct {
	XMLName xml.Name `xml:"div"`
	Class string `xml:"class,attr"`
	InnerXML string `xml:",innerxml"`
}

type cpEditorMeta struct {
	JS cpEditorPan `json:"js"`
	CSS cpEditorPan `json:"css"`
	HTML cpEditorPan `json:"html"`
	ShowPans []string `json:"showPans"`
	ActivePan string `json:"activePan"`
}

type cpEditorPan struct {
	Code string `json:"code"`
	Transformer string `json:"transformer"`
}

func NewCPEditorEmbeddedBlock(repl *BlockREPL) (*CPEditorEmbeddedBlock, error) {
	jsSrc := bytes.NewBufferString("")
	for key, val := range repl.SrcFiles {
		if val.IsDir || !strings.HasSuffix(key, ".js") {
			continue
		}
		jsSrc.WriteString(fmt.Sprintf("// %s\n", key))
		jsSrc.WriteString(val.Contents)
		jsSrc.WriteString("\n")
	}
	cpMeta := cpEditorMeta{
		JS: cpEditorPan{
			Code: jsSrc.String(),
			Transformer: "js",
		},
		CSS: cpEditorPan{
			Code: "",
			Transformer: "css",
		},
		HTML: cpEditorPan{
			Code: "",
			Transformer: "html",
		},
		ShowPans: []string{"js", "console"},
		ActivePan: "js",
	}
	cpMetaJson, err := json.Marshal(&cpMeta)
	if err != nil {
		return nil, err
	}
	return &CPEditorEmbeddedBlock{
		Class: "js-cp-boilerplate",
		InnerXML: string(cpMetaJson),
	}, nil
}

func (repl *CPEditorEmbeddedBlock) HTML() (b []byte, err error) {
	return xml.Marshal(repl)
}
