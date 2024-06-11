package gettext

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Document struct {
	Header  DocumentHeader
	Entries []Entry
}

func CreateDocument(entries []Entry) (Document, error) {
	if len(entries) == 0 {
		return Document{}, DocumentMissingHeaderError{}
	}
	if len(entries[0].Id) > 0 {
		return Document{}, DocumentMissingHeaderError{}
	}

	header, err := CreateHeaderFromEntry(entries[0])
	if err != nil {
		return Document{}, err
	}
	return Document{header, entries}, nil
}

func ParseDocumentString(d string) (Document, error) {
	return ParseDocument(strings.NewReader(d))
}

func ParseDocument(r io.Reader) (Document, error) {
	scanner := bufio.NewScanner(r)

	lineCounter := 1
	var lines []Line
	for ; scanner.Scan(); lineCounter += 1 {
		line, err := ParseLine(scanner.Text())
		if err != nil {
			return Document{}, DocumentLineParseError{lineCounter, err}
		}

		lines = append(lines, line)
	}

	var ctx documentParsingContext
	for _, actualLine := range lines {
		// lineToInspect should be used when reading a line's properties.
		// actualLine, being the actual line, is what should be passed over to entries
		lineToInspect := actualLine
		if lineToInspect.IsMarkedObsolete {
			var err error
			lineToInspect, err = ParseLine(lineToInspect.Comment.Comment[2:])
			if err != nil {
				return Document{}, DocumentParseError{"Failed to parse obsolete line.", err}
			}
		}

		if lineToInspect.IsCommentOrWhiteSpace {
			ctx.AddComment(actualLine)
			continue
		}
		// is a string or a keyworded line after this point

		//since the header cannot be obsolete, we inspect the actual line here
		if !ctx.HaveHeader && actualLine.Keyword.Keyword == "msgctxt" {
			return Document{}, DocumentParseError{Reason: "The first entry should be the header, and the header should not have a msgctxt."}
		}
		if !ctx.HaveHeader && actualLine.Keyword.Keyword == "msgid" {
			if len(actualLine.Value.Raw) > 0 {
				return Document{}, DocumentParseError{Reason: fmt.Sprint("First entry must be a header with a blank 'msgid'. Found: ", actualLine.RawLine)}
			}
			ctx.HaveHeader = true

			ctx.PushComments()
			ctx.PushLine(actualLine)
			continue
		}
		//cannot be the header after this point; is non-keyworded string or keyworded line

		switch lineToInspect.Keyword.Keyword {
		case "msgctxt":
			if ctx.ProcessingContextualEntry {
				return Document{}, DocumentParseError{Reason: "Found two consecutive 'msgctxt' keyworded entries in a row without a 'msgid' between them."}
			}

			if err := ctx.StartNextEntry(); err != nil {
				return Document{}, err
			}
			ctx.ProcessingContextualEntry = true
			ctx.PushLine(actualLine)
		case "msgid":
			// if msgctxt not in current entry, then msgid indicates the start of a new entry
			// if it is in the current entry, then msgid is part of the current entry
			if !ctx.ProcessingContextualEntry {
				if err := ctx.StartNextEntry(); err != nil {
					return Document{}, err
				}
			}

			ctx.ProcessingContextualEntry = false
			ctx.PushLine(actualLine)
		default:
			//here, remaining are string lines and other keyworded lines, like lone strs and plural lines
			ctx.PushComments()
			ctx.PushLine(actualLine)
		}
	}

	ctx.PushComments()
	if err := ctx.FinishCurrentEntry(); err != nil {
		return Document{}, err
	}

	return CreateDocument(ctx.Entries)
}

type DocumentMissingHeaderError struct{}

func (e DocumentMissingHeaderError) Error() string {
	return "The first entry of a document must be a header, having an empty id."
}

type DocumentParseError struct {
	Reason          string
	UnderlyingError error
}

func (e DocumentParseError) Error() string {
	if e.UnderlyingError != nil {
		return fmt.Sprintf("%v. Underlying error: %v", e.Reason, e.UnderlyingError)
	}
	return e.Reason
}

type DocumentLineParseError struct {
	Line            int
	UnderlyingError error
}

func (e DocumentLineParseError) Error() string {
	return fmt.Sprintf("Failed to parse line %v: %v", e.Line, e.UnderlyingError)
}

type documentParsingContext struct {
	Entries      []Entry
	FoundEntries map[EntryKey]struct{}

	HaveHeader                bool
	ProcessingContextualEntry bool

	currentEntryLines   []Line
	currentCommentBlock []Line
}

func (ctx *documentParsingContext) AddComment(l Line) {
	ctx.currentCommentBlock = append(ctx.currentCommentBlock, l)
}

func (ctx *documentParsingContext) PushComments() {
	ctx.currentEntryLines = append(ctx.currentEntryLines, ctx.currentCommentBlock...)
	ctx.currentCommentBlock = nil
}

func (ctx *documentParsingContext) PushLine(l Line) {
	ctx.currentEntryLines = append(ctx.currentEntryLines, l)
}

func (ctx *documentParsingContext) StartNextEntry() error {
	foundWhitespace := false
	for _, l := range ctx.currentCommentBlock {
		if !foundWhitespace && l.IsWhiteSpace {
			foundWhitespace = true
			if err := ctx.FinishCurrentEntry(); err != nil {
				return err
			}
		}

		// in the event we found whitespace, that piece of whitespace starts the now current entry
		ctx.currentEntryLines = append(ctx.currentEntryLines, l)
	}
	ctx.currentCommentBlock = nil

	// or, in other words, if we didnt finish the entry earlier, finish it now
	if !foundWhitespace {
		if err := ctx.FinishCurrentEntry(); err != nil {
			return err
		}
	}

	return nil
}

func (ctx *documentParsingContext) FinishCurrentEntry() error {
	entry, err := ParseEntry(ctx.currentEntryLines)
	if err != nil {
		return err
	}

	if _, ok := ctx.FoundEntries[entry.EntryKey]; ok {
		return DocumentParseError{Reason: fmt.Sprintf("Duplicate entry found: %+v", entry.EntryKey)}
	}

	if ctx.FoundEntries == nil {
		ctx.FoundEntries = make(map[EntryKey]struct{})
	}
	ctx.FoundEntries[entry.EntryKey] = struct{}{}

	ctx.currentEntryLines = nil
	ctx.Entries = append(ctx.Entries, entry)

	return nil
}
