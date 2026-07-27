package scenario

import (
	"reflect"
	"testing"
)

func TestParseHeaderValueSplitsTheTokenFromItsParameters(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want HeaderValue
	}{
		{
			name: "a content type with a charset",
			raw:  "text/html; charset=utf-8",
			want: HeaderValue{
				Token:  "text/html",
				Params: []Param{{Name: "charset", Value: "utf-8"}},
			},
		},
		{
			name: "a multipart type carrying two parameters",
			raw:  "multipart/form-data; boundary=abc; charset=utf-8",
			want: HeaderValue{
				Token: "multipart/form-data",
				Params: []Param{
					{Name: "boundary", Value: "abc"},
					{Name: "charset", Value: "utf-8"},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ParseHeaderValue(tc.raw); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseHeaderValue(%q) = %+v, want %+v", tc.raw, got, tc.want)
			}
		})
	}
}
