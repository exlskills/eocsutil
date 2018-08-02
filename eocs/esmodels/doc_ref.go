package esmodels

import "time"

type DocRef struct {
	EmbeddedDocRef EmbeddedDocRefWrapper `bson:"EmbeddedDocRef"`
	CreatedAt      time.Time             `bson:"created_at"`
	UpdatedAt      time.Time             `bson:"updated_at"`
}
