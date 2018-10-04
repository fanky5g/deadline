package deadline

import (
	"time"
)

// IsUnixTimePast compares current time vs passed in reference time to see if time is past
func IsUnixTimePast(refTime int64) bool {
	return refTime-time.Now().Unix() <= 0
}
