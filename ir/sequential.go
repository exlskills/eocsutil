package ir

type Sequential interface {
	GetDisplayName() string
	GetURLName() string
	GetIsGraded() bool
	GetAssignmentType() string
	GetExtraAttributes() map[string]string
	GetVerticals() []Vertical
}
