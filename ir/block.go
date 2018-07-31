package ir

type Block interface {
	GetDisplayName() string
	GetURLName() string
	GetBlockType() string
	GetContentOLX() (string, error)
	GetContentMD() (string, error)
	GetExtraAttributes() map[string]string
}
