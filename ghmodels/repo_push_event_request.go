package ghmodels

type RepoPushEventRequest struct {
	Ref        string     `json:"ref"`
	Repository Repository `json:"repository"`
	Commits    []Commit   `json:"commits"`
	HeadCommit Commit     `json:"head_commit"`
}

