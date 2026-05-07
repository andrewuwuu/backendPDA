package timeutil

import (
	"sync"
	"time"
)

var (
	jakartaLoc  *time.Location
	jakartaOnce sync.Once
)

// JakartaLocation returns the Asia/Jakarta time.Location, loaded once.
// Falls back to a fixed WIB zone (+07:00) if the timezone database is unavailable.
func JakartaLocation() *time.Location {
	jakartaOnce.Do(func() {
		loc, err := time.LoadLocation("Asia/Jakarta")
		if err != nil {
			loc = time.FixedZone("WIB", 7*60*60)
		}
		jakartaLoc = loc
	})
	return jakartaLoc
}

// NowJakarta returns the current time in the Jakarta timezone.
func NowJakarta() time.Time {
	return time.Now().In(JakartaLocation())
}

// TruncateToHour truncates a time value to the beginning of its hour.
func TruncateToHour(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}

// TruncateToJakartaHour converts a time to the Jakarta timezone and truncates
// it to the beginning of its hour.
func TruncateToJakartaHour(t time.Time) time.Time {
	return TruncateToHour(t.In(JakartaLocation()))
}
