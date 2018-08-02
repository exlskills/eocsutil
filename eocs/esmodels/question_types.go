package esmodels

var olxToQuestionType = map[string]string{
	"stringresponse":         "WSCQ",
	"optionresponse":         "", // TODO
	"multiplechoiceresponse": "MCSA",
	"numericalresponse":      "", // TODO
	"choiceresponse":         "MCMA",
}

func ESTypeFromOLXType(olxType string) string {
	return olxToQuestionType[olxType]
}
