package esmodels

import (
	"time"
)

type UnitsWrapper struct {
	ID        string    `bson:"_id"`
	Units     []Unit    `bson:"Units"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}
