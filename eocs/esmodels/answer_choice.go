package esmodels

import (
	"github.com/globalsign/mgo/bson"
	"time"
)

type AnswerChoice struct {
	ID          bson.ObjectId     `bson:"_id"`
	Sequence    int               `bson:"seq"`
	Text        IntlStringWrapper `bson:"text"`
	IsAnswer    bool              `bson:"is_answer"`
	Explanation IntlStringWrapper `bson:"explanation"`
	CreatedAt   time.Time         `bson:"created_at"`
	UpdatedAt   time.Time         `bson:"updated_at"`
}
