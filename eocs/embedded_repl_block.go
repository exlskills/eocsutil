package eocs

import "encoding/xml"

type EXLcodeEmbeddedREPLBlock struct {
	XMLName xml.Name `xml:"div"`
	Src     string   `xml:"data-repl-src,attr,omitempty"`
	Test    string   `xml:"data-repl-test,attr,omitempty"`
	Tmpl    string   `xml:"data-repl-tmpl,attr,omitempty"`
	Width   string   `xml:"width,attr,omitempty"`
	Height  string   `xml:"height,attr,omitempty"`
}
