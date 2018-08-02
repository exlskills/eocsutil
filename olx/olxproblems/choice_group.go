package olxproblems

type ChoiceGroup struct {
	Type    string   `xml:"type,attr"`
	Choices []Choice `xml:"choice"`
}
