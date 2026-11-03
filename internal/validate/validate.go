package validate

import (
	"regexp"
	"strconv"

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
	// max 는 글자 수를 센다. bcrypt 처럼 바이트 수에 한도가 있는 값은 이 태그로 막는다.
	_ = v.RegisterValidation("max_bytes", func(fl validator.FieldLevel) bool {
		limit, err := strconv.Atoi(fl.Param())
		if err != nil {
			panic("max_bytes: 정수 인자가 필요하다: " + fl.Param())
		}
		return len(fl.Field().String()) <= limit
	})
}

func Struct(s interface{}) error {
	return v.Struct(s)
}
