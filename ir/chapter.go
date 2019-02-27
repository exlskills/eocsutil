package ir

import "time"

type Chapter interface {
	GetDisplayName() string
	GetURLName() string
	GetExtraAttributes() map[string]string
	GetSequentials() []Sequential
	SetUpdatedAt(updatedAt time.Time)
}
