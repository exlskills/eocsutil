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
	err = xml.Unmarshal([]byte(x), &prob)
	// Unescape things that had previously been escaped in the XML (because XML is annoying)
	if prob.StringResponse != nil {
		prob.StringResponse.Answer = unescapeInnerMDP(prob.StringResponse.Answer)
		prob.StringResponse.Label.InnerXML = unescapeInnerMDP(prob.StringResponse.Answer)
	}
	if prob.ChoiceResponse != nil {
		unescapeChoicesP(prob.ChoiceResponse.CheckboxGroup.Choices)
		prob.ChoiceResponse.Label.InnerXML = unescapeInnerMDP(prob.ChoiceResponse.Label.InnerXML)
	}
	if prob.MultipleChoiceResponse != nil {
		unescapeChoicesP(prob.MultipleChoiceResponse.ChoiceGroup.Choices)
		prob.MultipleChoiceResponse.Label.InnerXML = unescapeInnerMDP(prob.MultipleChoiceResponse.Label.InnerXML)
	}
	if prob.DemandHint != nil {
		prob.DemandHint.Hint = unescapeInnerMDP(prob.DemandHint.Hint)
	}
	return
}

func unescapeInnerMDP(inner string) string {
	newS, err := unescapeInnerMD(inner)
	if err != nil {
		panic(err)
	}
	return newS
}

func unescapeInnerMD(inner string) (string, error) {
	return mdutils.UnescapeMD(inner)
}

func unescapeChoicesP(cg []Choice) {
	if cg == nil {
		return
	}
	for ind := range cg {
		for chInd := range cg[ind].ChoiceHint {
			cg[ind].ChoiceHint[chInd].InnerXML = unescapeInnerMDP(cg[ind].ChoiceHint[chInd].InnerXML)
		}
		cg[ind].InnerXML = unescapeInnerMDP(cg[ind].InnerXML)
	}
}
