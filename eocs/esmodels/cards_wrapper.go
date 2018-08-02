package esmodels

import (
	"time"
)

type CardsWrapper struct {
	Cards     []Card    `bson:"Cards"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}
