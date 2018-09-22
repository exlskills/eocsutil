package olxproblems

import (
	"encoding/xml"
	"github.com/exlskills/eocsutil/mdutils"
	"github.com/dlclark/regexp2"
)

var (
	reAmpSingle = regexp2.MustCompile("/(?<=`.*)(&amp;)(?=.*`)/g", 0)
	reAmpMulti = regexp2.MustCompile("/(?<=```.*)(&amp;)(?=.*```)/gms", 0)
	reGtSingle = regexp2.MustCompile("/(?<=`.*)(&gt;)(?=.*`)/g", 0)
	reGtMulti = regexp2.MustCompile("/(?<=```.*)(&gt;)(?=.*```)/gms", 0)
	reLtSingle = regexp2.MustCompile("/(?<=`.*)(&lt;)(?=.*`)/g", 0)
	reLtMulti = regexp2.MustCompile("/(?<=```.*)(&lt;)(?=.*```)/gms", 0)
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
	inner, err := reAmpSingle.Replace(inner, "&", -1, -1)
	if err != nil {
		return "", err
	}
	inner, err = reAmpMulti.Replace(inner, "&", -1, -1)
	if err != nil {
		return "", err
	}
	inner, err = reGtSingle.Replace(inner, ">", -1, -1)
	if err != nil {
		return "", err
	}
	inner, err = reGtMulti.Replace(inner, ">", -1, -1)
	if err != nil {
		return "", err
	}
	inner, err = reLtSingle.Replace(inner, "<", -1, -1)
	if err != nil {
		return "", err
	}
	inner, err = reLtMulti.Replace(inner, "<", -1, -1)
	if err != nil {
		return "", err
	}
	return inner, err
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
