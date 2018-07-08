package ir

type Chapter interface {
	GetDisplayName() string
	GetURLName() string
	GetExtraAttributes() map[string]string
	GetSequentials() []Sequential
}
