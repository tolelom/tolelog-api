package validate

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var (
	v                        = validator.New()
	alphanumUnderscoreRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

func init() {
	_ = v.RegisterValidation("alphanum_underscore", func(fl validator.FieldLevel) bool {
		return alphanumUnderscoreRegexp.MatchString(fl.Field().String())
	})
}

func Struct(s interface{}) error {
	return v.Struct(s)
}
