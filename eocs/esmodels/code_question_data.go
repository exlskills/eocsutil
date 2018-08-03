package esmodels

import (
		"github.com/globalsign/mgo/bson"
)

type CodeQuestionData struct {
	ID             bson.ObjectId                   `json:"_id"`
	APIVersion     int                             `json:"api_version"`
	EnvironmentKey string                          `json:"environment"`
	SrcFiles       string `json:"src"`
	TmplFiles      string `json:"tmpl"`
	TestFiles      string `json:"test"`
}
