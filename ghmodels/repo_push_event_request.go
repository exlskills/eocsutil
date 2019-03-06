package ghmodels

type RepoPushEventRequest struct {
	// This is defined as a string and later converted to go-git/plumbing.Reference
	Ref        string     `json:"ref"`
	Repository Repository `json:"repository"`
	Commits    []Commit   `json:"commits"`
	HeadCommit Commit     `json:"head_commit"`
}

