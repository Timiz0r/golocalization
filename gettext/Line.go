package gettext

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Line struct {
	Keyword Keyword
	Value   LineValue
	Comment Comment
	RawLine string

	IsCommentOrWhiteSpace, IsWhiteSpace, IsComment, IsMarkedObsolete bool
}

type LineParseError struct {
	Line string
}

func CompleteLine(keyword Keyword, lineValue LineValue, comment Comment) Line {
	var rawValueBuilder strings.Builder

	if !keyword.IsEmpty {

		rawValueBuilder.WriteString(keyword.Keyword)

		if keyword.IsIndexed {
			indexString := strconv.Itoa(keyword.Index)
			rawValueBuilder.Grow(2 + len(indexString))
			rawValueBuilder.WriteRune('[')
			rawValueBuilder.WriteString(indexString)
			rawValueBuilder.WriteRune(']')
		}

		if !lineValue.IsEmpty || !comment.IsEmpty {
			rawValueBuilder.WriteRune(' ')
		}
	}

	if !lineValue.IsEmpty {
		rawValueBuilder.Grow(2 + len(lineValue.Value))
		rawValueBuilder.WriteRune('"')
		rawValueBuilder.WriteString(lineValue.Value)
		rawValueBuilder.WriteRune('"')

		if !comment.IsEmpty {
			rawValueBuilder.WriteRune(' ')
		}
	}

	if !comment.IsEmpty {
		rawValueBuilder.Grow(1 + len(comment.Comment))
		rawValueBuilder.WriteRune('#')
		rawValueBuilder.WriteString(comment.Comment)
	}

	return *(&Line{
		Keyword: keyword,
		Value:   lineValue,
		Comment: comment,
		RawLine: rawValueBuilder.String(),
	}).populateExtraBools()
}

func CommentLine(comment Comment) Line {
	return CompleteLine(Keyword{IsEmpty: true}, LineValue{IsEmpty: true}, comment)
}

func ValueLine(lineValue LineValue) Line {
	return CompleteLine(Keyword{IsEmpty: true}, lineValue, Comment{IsEmpty: true})
}

func KeywordedValueLine(keyword Keyword, lineValue LineValue) Line {
	return CompleteLine(keyword, lineValue, Comment{IsEmpty: true})
}

var parseRegex = regexp.MustCompile(`` +
	`(?:` + // keyword (msgid, etc.) and optional index
	`` + `\s*(?<keyword>[^\s\["#]+)` + // could try [a-zA-Z0-9] or something, as well
	`` + `\s*(?:\[(?<index>\d+)\])?` +
	`)?` +
	`\s*(?:"` + // value
	`` + `(?<value>(?:[^"]|\\")*)` +
	`")?` +
	`\s*(?:` + // comment
	`` + `#(?<comment>.*)` +
	`)?`)

func ParseLine(line string) (Line, error) {
	matches, success := getMatches(parseRegex, line)
	if !success {
		return Line{}, LineParseError{line}
	}

	keyword := Keyword{IsEmpty: true}
	lineValue := LineValue{IsEmpty: true}
	comment := Comment{IsEmpty: true}

	if keywordMatch := matches["keyword"]; keywordMatch.success {
		if indexMatch := matches["index"]; indexMatch.success {
			if parsedIndex, err := strconv.Atoi(indexMatch.value); err == nil {
				keyword = IndexedKeyword(keywordMatch.value, parsedIndex)
			} else {
				return Line{}, LineParseError{fmt.Sprintf("Unable to parse index '%v'.", indexMatch.value)}
			}
		} else {
			keyword = SimpleKeyword(keywordMatch.value)
		}
	}

	if lineValueMatch := matches["value"]; lineValueMatch.success {
		lineValue = LineValueFromRaw(lineValueMatch.value)
	}

	if commentMatch := matches["comment"]; commentMatch.success {
		comment = Comment{Comment: commentMatch.value}
	}

	return *(&Line{
		Keyword: keyword,
		Value:   lineValue,
		Comment: comment,
		RawLine: line,
	}).populateExtraBools(), nil
}

func (err LineParseError) Error() string {
	return fmt.Sprint("Failed to parse line: ", err.Line)
}

type match struct {
	value   string
	success bool
}

func getMatches(r *regexp.Regexp, s string) (map[string]match, bool) {
	matchIndices := r.FindStringSubmatchIndex(s)
	if matchIndices[0] == -1 {
		return nil, false
	}

	matches := r.FindStringSubmatch(s)
	names := r.SubexpNames()
	parsedData := make(map[string]match, len(names))

	for i, name := range names {
		parsedData[name] = match{matches[i], matchIndices[i*2] != -1}
	}

	return parsedData, true
}

func (line *Line) populateExtraBools() *Line {
	line.IsCommentOrWhiteSpace = line.Keyword.IsEmpty && line.Value.IsEmpty
	line.IsWhiteSpace = line.IsCommentOrWhiteSpace && line.Comment.IsEmpty
	line.IsComment = line.IsCommentOrWhiteSpace && !line.Comment.IsEmpty
	line.IsMarkedObsolete = line.IsComment && line.Comment.Comment[0] == '~'

	return line
}
