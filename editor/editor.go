package editor

import (
	"fmt"
	"strings"
)

func AddTab(line string) string {
	return fmt.Sprintf("\t%s", line)
}

func RemoveTab(line string) string {
	// When \t is added to a line, BubbleTea adds
	// 4 space characters instead of one tab character.
	if strings.HasPrefix(line, "    ") {
		return line[4:]
	}
	return line
}
