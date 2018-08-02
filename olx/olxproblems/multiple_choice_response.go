package olxproblems

type MultipleChoiceResponse struct {
	Label       ProblemLabel `xml:"label"`
	ChoiceGroup *ChoiceGroup `xml:"choicegroup"`
}
