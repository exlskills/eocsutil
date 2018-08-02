package olxproblems

type Choice struct {
	Correct    bool         `xml:"correct,attr"`
	ChoiceHint []ChoiceHint `xml:"choicehint"`
	InnerXML   string       `xml:",innerxml"`
}
