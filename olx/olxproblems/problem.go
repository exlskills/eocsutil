package olxproblems

import (
	"encoding/xml"
	"github.com/exlskills/eocsutil/mdutils"
)

type Problem struct {
	XMLName                xml.Name                `xml:"problem"`
	StringResponse         *StringResponse         `xml:"stringresponse,omitempty"`
	ChoiceResponse         *ChoiceResponse         `xml:"choiceresponse,omitempty"`
	MultipleChoiceResponse *MultipleChoiceResponse `xml:"multiplechoiceresponse,omitempty"`
	DemandHint             *DemandHint             `xml:"demandhint,omitempty"`
}

func NewProblemFromMD(md string) (prob *Problem, err error) {
	x, err := mdutils.MakeOLX(md)
	if err != nil {
		return nil, err
	}
	err = xml.Unmarshal([]byte(x), prob)
	return
}
