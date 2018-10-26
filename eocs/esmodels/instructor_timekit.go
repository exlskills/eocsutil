package esmodels

type InstructorTimekit struct {
	Intervals []TimekitInterval `bson:"intervals" json:"intervals" yaml:"intervals"`
}

type TimekitInterval struct {
	Credits         int64  `bson:"credits" json:"credits" yaml:"credits"`
	ProjectID       string `bson:"project_id" json:"project_id" yaml:"project_id"`
	DurationSeconds int64  `bson:"duration_seconds" json:"duration_seconds" yaml:"duration_seconds"`
}
