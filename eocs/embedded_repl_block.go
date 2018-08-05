package eocs

import (
	"encoding/xml"
	"bytes"
	"net/url"
	)

type EXLcodeEmbeddedREPLBlock struct {
	XMLName xml.Name `xml:"div"`
	Src     string   `xml:"data-repl-src,attr,omitempty"`
	Test    string   `xml:"data-repl-test,attr,omitempty"`
	Tmpl    string   `xml:"data-repl-tmpl,attr,omitempty"`
	Width   string   `xml:"width,attr,omitempty"`
	Height  string   `xml:"height,attr,omitempty"`
}

func (repl *EXLcodeEmbeddedREPLBlock) GetEmbedURL() string {
	buf := bytes.NewBufferString("https://exlcode.com/repl?embedded=true&workspace=")
	buf.WriteString(url.QueryEscape(repl.Src))
	return buf.String()
}

func (repl *EXLcodeEmbeddedREPLBlock) IFrame() (b []byte, err error) {
	return xml.Marshal(EXLcodeEmbeddedREPLIFrame{
		SrcURL: repl.GetEmbedURL(),
		Width: "100%",
		Height: "500px",
	})
}


type EXLcodeEmbeddedREPLIFrame struct {
	XMLName xml.Name `xml:"iframe"`
	SrcURL     string   `xml:"src,attr,omitempty"`
	Width   string   `xml:"width,attr,omitempty"`
	Height  string   `xml:"height,attr,omitempty"`
}
