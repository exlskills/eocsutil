package esmodels

import (
	"time"
)

type SectionsWrapper struct {
	Sections  []Section `bson:"Sections"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}
