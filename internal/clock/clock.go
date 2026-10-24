// Package clock: semua logika "hari ini" memakai WIB (UTC+7),
// terlepas dari zona waktu server.
package clock

import "time"

var WIB = time.FixedZone("WIB", 7*60*60)

func Now() time.Time { return time.Now().In(WIB) }

// DayBounds mengembalikan [awal hari, awal hari berikutnya) dalam WIB.
func DayBounds(t time.Time) (time.Time, time.Time) {
	t = t.In(WIB)
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, WIB)
	return start, start.Add(24 * time.Hour)
}
