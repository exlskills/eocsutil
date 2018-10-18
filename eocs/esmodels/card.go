package esmodels

type Card struct {
	ID            string            `bson:"_id"`
	Title         IntlStringWrapper `bson:"title"`
	Headline      IntlStringWrapper `bson:"headline"`
	Index         int               `bson:"index"`
	ContentID     string            `bson:"content_id"`
	QuestionIDs   []string          `bson:"question_ids"`
	CardRef       DocRef            `bson:"card_ref"`
	CourseItemRef CourseItemRef     `bson:"course_item_ref"`
	Tags          []string          `bson:"tags"`
}
