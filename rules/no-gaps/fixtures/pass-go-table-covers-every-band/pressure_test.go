package scenario

import "testing"

func TestClassifyPressureGradesReadingsAgainstTheSafeRange(t *testing.T) {
	cases := []struct {
		name string
		psi  int
		want Band
	}{
		{"a flat tyre reads nothing", 0, BandLow},
		{"one psi under the safe range", 29, BandLow},
		{"exactly the bottom of the safe range", 30, BandNormal},
		{"comfortably inside the safe range", 33, BandNormal},
		{"exactly the top of the safe range", 35, BandNormal},
		{"one psi over the safe range", 36, BandHigh},
		{"over-inflated after a long drive", 120, BandHigh},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyPressure(tc.psi); got != tc.want {
				t.Errorf("ClassifyPressure(%d) = %d, want %d", tc.psi, got, tc.want)
			}
		})
	}
}
