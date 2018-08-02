package esmodels

import (
	"time"
)

type Unit struct {
	ID                    string            `bson:"_id"`
	Sections              SectionsWrapper   `bson:"sections"`
	Title                 IntlStringWrapper `bson:"title"`
	Headline              IntlStringWrapper `bson:"headline"`
	FinalExamIDs          []string          `bson:"final_exams"`
	Index                 int               `bson:"index"`
	FinalExamWeightPct    float64           `bson:"final_exam_weight_pct"`
	AttemptsAllowedPerDay int               `bson:"attempts_allowed_per_day"`
	CreatedAt             time.Time         `bson:"created_at"`
	UpdatedAt             time.Time         `bson:"updated_at"`
}
