package esmodels

import "time"

type Exam struct {
	ID             string    `bson:"_id"`
	QuestionCount  int       `bson:"question_count"`
	CreatorID      string    `bson:"creator_id"`
	QuestionIDs    []string  `bson:"question_ids"`
	UseIDETestMode bool      `bson:"use_ide_test_mode"`
	Tags           []string  `bson:"tags"`
	TimeLimit      int       `bson:"time_limit"`
	EstTime        int       `bson:"est_time"`
	PassMarkPct    float64   `bson:"pass_mark_pct"`
	RandomOrder    bool      `bson:"random_order"`
	CreatedAt      time.Time `bson:"created_at"`
	UpdatedAt      time.Time `bson:"updated_at"`
}
