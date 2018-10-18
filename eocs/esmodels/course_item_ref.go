package esmodels

type CourseItemRef struct {
	CourseID  string `bson:"course_id,omitempty"`
	UnitID    string `bson:"unit_id,omitempty"`
	SectionID string `bson:"section_id,omitempty"`
	CardID    string `bson:"card_id,omitempty"`
}
