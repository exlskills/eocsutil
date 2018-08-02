package esmodels

import (
	"time"
)

type Section struct {
	ID        string            `bson:"_id"`
	Title     IntlStringWrapper `bson:"title"`
	Headline  IntlStringWrapper `bson:"headline"`
	Index     int               `bson:"index"`
	Cards     CardsWrapper      `bson:"cards"`
	CreatedAt time.Time         `bson:"created_at"`
	UpdatedAt time.Time         `bson:"updated_at"`
}
