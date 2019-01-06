package ir

import "time"

type Vertical interface {
	GetDisplayName() string
	GetURLName() string
	GetExtraAttributes() map[string]string
	GetBlocks() []Block
	SetUpdatedAt(updatedAt time.Time)
}
