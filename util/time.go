package util

import "time"

//DateTimeFormat unified in the program.
const DateTimeFormat = "2006-01-02 15:04:05"

//Now returns date time in the format of "2006-01-02 15:04:05"
func Now() string {
	return time.Now().Format(DateTimeFormat)
}
