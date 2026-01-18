package parser

import (
	"strings"
)

func ParseInput(line string) []string {
	var args []string
	var current strings.Builder

	inSingle := false
	inDouble := false
	escaped := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			if inSingle {
				current.WriteByte(ch)
			} else if inDouble {
				if i+1 < len(line) {
					next := line[i+1]
					if next == '"' || next == '\\' || next == '$' || next == '`' || next == '\n' {
						escaped = true
					} else {
						current.WriteByte(ch)
					}
				}
			} else {
				escaped = true
			}
			continue
		}

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}

		if (ch == ' ' || ch == '\t') && !inSingle && !inDouble {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}
