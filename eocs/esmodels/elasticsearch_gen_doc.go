package esmodels

type ElasticsearchGenDoc struct {
	ID          string `json:"-"`
	DocType     string `json:"doc_type"`
	Title       string `json:"title,omitempty"`
	Headline    string `json:"headline,omitempty"`
	TextContent string `json:"text_content,omitempty"`
	CodeContent string `json:"code_content,omitempty"`
	DocRef      string `json:"doc_ref,omitempty"`
}
