package esmodels

import (
	"github.com/globalsign/mgo/bson"
	"time"
)

type VersionedContent struct {
	ID            string    `bson:"_id"`
	LatestVersion int       `bson:"latest_version"`
	Contents      []Content `bson:"contents"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`
}

type Content struct {
	ID      bson.ObjectId     `bson:"_id"`
	Version int               `bson:"version"`
	Content IntlStringWrapper `bson:"content"`
}
