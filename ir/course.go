package ir

type Course interface {
	GetDisplayName() string
	GetURLName() string
	GetOrgName() string
	GetCourseImage() string
	GetLanguage() string
	GetExtraAttributes() map[string]string
	GetChapters() []Chapter
}
