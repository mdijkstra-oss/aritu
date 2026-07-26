package scenario

import "testing"

func TestParseSemver(t *testing.T) {
	cases := []struct {
		name      string
		raw       string
		want      Version
		wantError bool
	}{
		{
			name: "reads major minor and patch from a dotted triple",
			raw:  "1.4.2",
			want: Version{Major: 1, Minor: 4, Patch: 2},
		},
		{
			name:      "rejects a version missing the patch component",
			raw:       "1.4",
			wantError: true,
		},
		{
			name:      "rejects a version whose minor component is not a number",
			raw:       "1.x.2",
			wantError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseSemver(tc.raw)

			if (err != nil) != tc.wantError {
				t.Fatalf("ParseSemver(%q) error = %v; want an error: %t", tc.raw, err, tc.wantError)
			}
			if got != tc.want {
				t.Fatalf("ParseSemver(%q) = %+v; want %+v", tc.raw, got, tc.want)
			}
		})
	}
}
