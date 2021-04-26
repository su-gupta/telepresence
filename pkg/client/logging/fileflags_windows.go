package logging

import (
	"golang.org/x/sys/windows"
)

func fileFlags(flags int) int {
	return windows.FILE_SHARE_DELETE | flags
}
