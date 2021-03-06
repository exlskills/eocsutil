package esmodels

import "time"

type Card struct {
	ID            string            `bson:"_id"`
	Title         IntlStringWrapper `bson:"title"`
	Headline      IntlStringWrapper `bson:"headline"`
	Index         int               `bson:"index"`
	ContentID     string            `bson:"content_id"`
	QuestionIDs   []string          `bson:"question_ids"`
	CourseItemRef CourseItemRef     `bson:"course_item_ref"`
	GithubEditURL string            `bson:"github_edit_url,omitempty"`
	Tags          []string          `bson:"tags"`
	CreatedAt     time.Time         `bson:"created_at"`
	UpdatedAt     time.Time         `bson:"updated_at"`
}
