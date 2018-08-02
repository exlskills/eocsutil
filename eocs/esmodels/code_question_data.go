package esmodels

import (
	"github.com/exlskills/eocsutil/wsenv"
	"github.com/globalsign/mgo/bson"
)

type CodeQuestionData struct {
	ID             bson.ObjectId                   `json:"_id"`
	APIVersion     int                             `json:"api_version"`
	EnvironmentKey string                          `json:"environment"`
	SrcFiles       map[string]*wsenv.WorkspaceFile `json:"src"`
	TmplFiles      map[string]*wsenv.WorkspaceFile `json:"tmpl"`
	TestFiles      map[string]*wsenv.WorkspaceFile `json:"test"`
}
