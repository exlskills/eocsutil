package ghmodels

type Repository struct {
	ID int64 `json:"id"`
	Name string `json:"name"`
	FullName string `json:"full_name"`
	ArchiveURL string `json:"archive_url"`
}
