package esmodels

type Question struct {
	ID              string            `bson:"id"`
	Data            interface{}       `bson:"data"`
	Points          float64           `bson:"points"`
	ComplexityLevel int               `bson:"compl_level"`
	QuestionType    string            `bson:"question_type"`
	QuestionText    IntlStringWrapper `bson:"question_text"`
	EstTimeSec      int               `bson:"est_time_sec"`
	Tags            []string          `bson:"tags"`
	DocRef          DocRef            `bson:"doc_ref"`
	Hint            IntlStringWrapper `bson:"hint"`
}