package esmodels

type ElasticsearchGenDoc struct {
	ID          string `json:"-"`
	DocType     string `json:"doc_type"`
	Title       string `json:"title,omitempty"`
	Headline    string `json:"headline,omitempty"`
	TextContent string `json:"text_content,omitempty"`
	CodeContent string `json:"code_content,omitempty"`
	CourseId    string `json:"course_id,omitempty"`
	UnitId      string `json:"unit_id,omitempty"`
	SectionId   string `json:"section_id,omitempty"`
	CardId      string `json:"card_id,omitempty"`
}
