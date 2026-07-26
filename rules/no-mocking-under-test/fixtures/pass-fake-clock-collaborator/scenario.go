package scenario

import "time"

type Clock interface {
	Now() time.Time
}

type Session struct {
	Token     string
	ExpiresAt time.Time
}

type SessionState int

const (
	SessionExpired SessionState = iota
	SessionNeedsRenewal
	SessionActive
)

const renewalWindow = 5 * time.Minute

func ClassifySession(clock Clock, session Session) SessionState {
	remaining := session.ExpiresAt.Sub(clock.Now())
	switch {
	case remaining <= 0:
		return SessionExpired
	case remaining <= renewalWindow:
		return SessionNeedsRenewal
	default:
		return SessionActive
	}
}

func (s SessionState) String() string {
	switch s {
	case SessionExpired:
		return "expired"
	case SessionNeedsRenewal:
		return "needs-renewal"
	case SessionActive:
		return "active"
	default:
		panic("unknown session state")
	}
}
