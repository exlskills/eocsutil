package ir

type Vertical interface {
	GetDisplayName() string
	GetURLName() string
	GetExtraAttributes() map[string]string
	GetBlocks() []Block
}
