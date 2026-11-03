package dto

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
	"tolelom_api/internal/model"
)

func extensionValue(t *testing.T, v any, field, key string) int {
	t.Helper()
	f, ok := reflect.TypeOf(v).FieldByName(field)
	if !ok {
		t.Fatalf("%T has no field %s", v, field)
	}
	for _, ext := range strings.Split(f.Tag.Get("extensions"), ",") {
		if k, val, found := strings.Cut(ext, "="); found && k == key {
			n, err := strconv.Atoi(val)
			if err != nil {
				t.Fatalf("%T.%s %s=%q is not a number", v, field, key, val)
			}
			return n
		}
	}
	t.Fatalf("%T.%s has no %s extension", v, field, key)
	return 0
}

func TestSwaggerTagLimitMatchesModel(t *testing.T) {
	for _, v := range []any{CreatePostRequest{}, UpdatePostRequest{}} {
		if got := extensionValue(t, v, "Tags", "x-tag-max-runes"); got != model.TagNameMaxRunes {
			t.Errorf("%T.Tags x-tag-max-runes = %d, want %d", v, got, model.TagNameMaxRunes)
		}
	}
}

func TestSwaggerPasswordByteLimitMatchesValidation(t *testing.T) {
	cases := []struct {
		v     any
		field string
	}{
		{RegisterRequest{}, "Password"},
		{ChangePasswordRequest{}, "NewPassword"},
	}
	for _, tc := range cases {
		f, _ := reflect.TypeOf(tc.v).FieldByName(tc.field)
		want := "max_bytes=" + strconv.Itoa(extensionValue(t, tc.v, tc.field, "x-max-bytes"))
		if !strings.Contains(f.Tag.Get("validate"), want) {
			t.Errorf("%T.%s validate %q lacks %s", tc.v, tc.field, f.Tag.Get("validate"), want)
		}
	}
}
