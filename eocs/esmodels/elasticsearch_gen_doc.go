package esmodels

type ElasticsearchGenDoc struct {
	ID          string
	DocType     string `json:"doc_type"`
	Title       string `json:"title"`
	Headline    string `json:"headline"`
	TextContent string `json:"text_content"`
	CodeContent string `json:"code_content"`
	DocRef      string `json:"doc_ref"`
}
