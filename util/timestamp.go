package util

import (
	"strconv"
	"time"
)

func GetCurrentTimeMillis() string {
	timestamp_millis := time.Now().UnixNano() / 1e6
	timestamp_str := strconv.FormatInt(timestamp_millis, 10)

	return timestamp_str
}
