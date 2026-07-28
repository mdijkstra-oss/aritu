package main

import (
	"io"
	"os"
)

// Escape sequences belong on a terminal and nowhere else, and NO_COLOR is the
// cross-tool convention for suppressing them (no-color.org).
func wantsColour(stream io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	file, isFile := stream.(*os.File)
	if !isFile {
		return false
	}
	return isCharacterDevice(file)
}

func isCharacterDevice(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
