package ir

type Block interface {
	GetDisplayName() string
	GetURLName() string
	GetContentType() string
	GetContent() string
	GetExtraAttributes() map[string]string
}
