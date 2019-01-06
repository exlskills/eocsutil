package ghmodels

type Repository struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	FullName   string    `json:"full_name"`
	ArchiveURL string    `json:"archive_url"`
	CloneURL   string    `json:"clone_url"`
	RepoOwner RepoOwner  `json:"owner"`
}

type Commit struct   {
	ID      string       `json:"id"`
	Message string       `json:"message"`
	Author  CommitAuthor `json:"author"`
}

type RepoOwner struct   {
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type CommitAuthor struct   {
	Name  string    `json:"name"`
	Email string    `json:"email"`
}