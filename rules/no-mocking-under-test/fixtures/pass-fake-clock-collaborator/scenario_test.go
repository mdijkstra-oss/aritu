package scenario

import (
	"testing"
	"time"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time { return c.now }

func TestClassifySessionGradesSessionsByRemainingLifetime(t *testing.T) {
	now := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		expiresAt time.Time
		want      SessionState
	}{
		{name: "expired an hour ago", expiresAt: now.Add(-time.Hour), want: SessionExpired},
		{name: "expiring exactly now", expiresAt: now, want: SessionExpired},
		{name: "two minutes of life left", expiresAt: now.Add(2 * time.Minute), want: SessionNeedsRenewal},
		{name: "an hour of life left", expiresAt: now.Add(time.Hour), want: SessionActive},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			session := Session{Token: "tok-1", ExpiresAt: tc.expiresAt}

			got := ClassifySession(fixedClock{now: now}, session)

			if got != tc.want {
				t.Errorf("ClassifySession() = %s, want %s", got, tc.want)
			}
		})
	}
}
