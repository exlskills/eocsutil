package esmodels

import "time"

type IntlStringWrapper struct {
	Strings   IntlStrings `bson:"intlString"`
	CreatedAt time.Time   `bson:"created_at"`
	UpdatedAt time.Time   `bson:"updated_at"`
}

type IntlStrings []IntlString

type IntlString struct {
	Content   string    `bson:"content"`
	IsDefault bool      `bson:"is_default"`
	Locale    string    `bson:"locale"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

func NewIntlStringWrapper(str, locale string) IntlStringWrapper {
	return IntlStringWrapper{
		Strings: []IntlString{
			{
				Content:   str,
				IsDefault: true,
				Locale:    locale,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
}
