package esmodels

import (
	"github.com/globalsign/mgo/bson"
)

type CodeQuestionData struct {
	ID              bson.ObjectId     `json:"_id" bson:""`
	APIVersion      int               `json:"api_version" bson:"api_version"`
	EnvironmentKey  string            `json:"environment_key" bson:"environment_key"`
	SrcFiles        IntlStringWrapper `json:"src_files" bson:"src_files"`
	TmplFiles       IntlStringWrapper `json:"test_files" bson:"test_files"`
	TestFiles       string            `json:"tmpl_files" bson:"tmpl_files"`
	TestCommands    []string          `json:"test_commands" bson:"test_commands"`
	GradingStrategy string            `json:"grading_strategy" bson:"grading_strategy"`
	GradingTests    string            `json:"grading_tests" bson:"grading_tests"`
}
