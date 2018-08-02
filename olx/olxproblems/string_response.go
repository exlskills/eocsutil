package olxproblems

type StringResponse struct {
	Label  ProblemLabel `xml:"label"`
	Answer string       `xml:"answer,attr"`
	Type   string       `xml:"type,attr"`
}
