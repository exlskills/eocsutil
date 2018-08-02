package esmodels

import "time"

type EmbeddedDocRef struct {
	DocID     string    `bson:"doc_id"`
	Level     string    `bson:"level"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}
