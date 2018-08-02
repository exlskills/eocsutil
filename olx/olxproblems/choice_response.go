package olxproblems

type ChoiceResponse struct {
	Label         ProblemLabel   `xml:"label"`
	CheckboxGroup *CheckboxGroup `xml:"checkboxgroup"`
}
