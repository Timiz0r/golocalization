package gettext

import (
	"fmt"
	"strings"
)

type Entry struct {
	Header     EntryHeader
	IsObsolete bool
	Context    string

	Id    string
	Value string

	PluralId     string
	PluralValues []string

	Lines []Line
}

type EntryLineParseError struct {
	Line   Line
	Reason string
}

type EntryParseError struct {
	Reason string
}

// the original c# implementation has a keyword->value map,
// which makes this method a bit more maintainable when supporting new keywords
// but in practice, this doesn't happen, and hard-coded keywords provide a simpler implementation
func ParseEntry(lines []Line) (Entry, error) {
	entry := Entry{
		Lines:  lines,
		Header: ExtractEntryHeader(lines),
	}

	//these will be plopped into the entry later
	var context, id, value, pluralId strings.Builder
	// in practice, there are 6 plural types and special zero, so usually 7 or less
	pluralValues := make(map[int]*strings.Builder, 7)

	// we currently use this for validation against len(pluralValues),
	// but we could hypothetically support missing values, as well. but we don't!
	var maxPluralIndex int
	var currentValue *strings.Builder
	var nonObsoleteLinesFound bool
	var shouldBePlural bool

	var unsupportedKeywords []string

	for _, line := range lines {
		if line.IsMarkedObsolete {
			line, error := ParseLine(line.Comment.Comment[2:])
			if error != nil {
				return Entry{}, EntryLineParseError{line, "Failed to parse obsolete line."}
			}
			entry.IsObsolete = true
		} else {
			nonObsoleteLinesFound = true
		}

		if !line.Keyword.IsEmpty {
			if line.Keyword.IsIndexed {
				idx := line.Keyword.Index
				maxPluralIndex = max(maxPluralIndex, idx)

				if _, ok := pluralValues[idx]; ok {
					return Entry{}, EntryLineParseError{line, fmt.Sprintf("Duplicate plural index found: %v.", idx)}
				} else {
					currentValue = &strings.Builder{}
					pluralValues[idx] = currentValue
				}
			} else {
				switch line.Keyword.Keyword {
				case "msgctxt":
					currentValue = &context
				case "msgid":
					currentValue = &id
				case "msgid_plural":
					currentValue = &pluralId
					shouldBePlural = true
				case "msgstr":
					currentValue = &value
				default:
					unsupportedKeywords = append(unsupportedKeywords, line.Keyword.Keyword)
				}

				if currentValue.Len() != 0 {
					return Entry{}, EntryLineParseError{line, "Keyword has already appeared in a prior line."}
				}
			}
		}

		if !line.Value.IsEmpty {
			// if the line is a keyword+value, then this cannot be hit
			if currentValue == nil {
				return Entry{}, EntryLineParseError{line, "Found a string-only line without a prior keyworded line."}
			}

			currentValue.WriteString(line.Value.Value)
		}
	}

	if entry.IsObsolete && nonObsoleteLinesFound {
		return Entry{}, EntryParseError{"Entry has lines marked obsolete but has non-obsolete lines as well."}
	}

	if len(pluralValues) != maxPluralIndex+1 {
		return Entry{}, EntryParseError{
			fmt.Sprintf("Expected a plural count of '%v' based on the highest found index, but only found '%v' plural entries.", maxPluralIndex+1, len(pluralValues)),
		}
	}

	if shouldBePlural && len(pluralValues) == 0 {
		return Entry{}, EntryParseError{"Plural id provided, but no plurals found."}
	}

	if len(pluralValues) > 0 && !shouldBePlural {
		return Entry{}, EntryParseError{"Plurals provided, but no plural id found."}
	}

	if len(unsupportedKeywords) > 0 {
		s := strings.Join(unsupportedKeywords, ", ")
		return Entry{}, EntryParseError{fmt.Sprint("Found unsupported keywords in entry: ", s)}
	}

	entry.Context = context.String()
	entry.Id = id.String()
	entry.Value = value.String()
	entry.PluralId = pluralId.String()

	entry.PluralValues = make([]string, len(pluralValues))
	for i := 0; i < len(pluralValues); i++ {
		entry.PluralValues[i] = pluralValues[i].String()
	}

	return entry, nil
}

func (e EntryLineParseError) Error() string {
	return fmt.Sprintf("Failed to parse line for entry. %v Line: %v", e.Reason, e.Line.RawLine)
}

func (e EntryParseError) Error() string {
	return fmt.Sprintf("Failed to parse entry. %v", e.Reason)
}
