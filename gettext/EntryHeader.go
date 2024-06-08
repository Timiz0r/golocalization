package gettext

import (
	"slices"
	"strings"
)

type EntryHeader struct {
	References []string
	Flags      []string
}

func ExtractEntryHeader(lines []Line) EntryHeader {
	headerComments := make([]string, 0, len(lines))
	for _, line := range lines {
		if !line.IsCommentOrWhiteSpace || line.IsMarkedObsolete {
			break
		}

		if line.IsComment {
			headerComments = append(headerComments, line.Comment.Comment)
		}
	}

	references := make([]string, len(headerComments))
	for _, comment := range headerComments {
		if strings.HasPrefix(comment, ":") {
			references = append(references, strings.TrimSpace(comment[1:]))
		}
	}

	var flags []string
	for _, comment := range headerComments {
		if strings.HasPrefix(comment, ",") {
			flags = strings.Split(comment, ",")

			flags = slices.DeleteFunc(flags, func(flag string) bool {
				return len(flag) == 0
			})

			for i, flag := range flags {
				flags[i] = strings.TrimSpace(flag)
			}
		}
	}

	return EntryHeader{References: references, Flags: flags}
}
