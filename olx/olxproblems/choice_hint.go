package olxproblems

type ChoiceHint struct {
	Selected *bool  `xml:"selected,attr"`
	InnerXML string `xml:",innerxml"`
}
