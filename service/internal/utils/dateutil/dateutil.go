package dateutil

import (
	"time"
)

func GetCurrentTimeRFC3339() string {
	return time.Now().Format(time.RFC3339)
}
