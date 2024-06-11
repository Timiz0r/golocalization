package gettext_test

import (
	"fmt"
	"testing"

	"github.com/Timiz0r/golocalization/gettext"
	"github.com/shopspring/decimal"
	"golang.org/x/text/language"
)

const genericHeader = `
msgid ""
msgstr "Language: ja\n"
`

func TestWhenFirstEntryNotHeader(t *testing.T) {
	documentText := `
msgid "foo"
msgstr "bar"`

	_, err := gettext.ParseDocumentString(documentText)

	if _, ok := err.(gettext.DocumentParseError); !ok {
		t.Errorf("Expected %T but got %T: %+v", gettext.DocumentParseError{}, err, err)
	}
}

func TestReturnsRightHeader_WhenJustLanguage(t *testing.T) {
	documentText := `
msgid ""
msgstr "Language: az_AZ@latin\n"`

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}
	header := doc.Header

	expectedTag := language.Make("az-Latn-AZ")
	if header.Tag != expectedTag {
		t.Errorf("Expected tag %v, got %v.", expectedTag, header.Tag)
	}

	if p := header.PluralRules.Evaluate(decimal.Zero); p != gettext.PluralTypeOther {
		t.Errorf("Expected %v because no plural rules results in %v, got %v.",
			gettext.PluralTypeOther, gettext.PluralTypeOther, p)
	}
}

func TestReturnsRightHeader_WhenPluralRulesPresent(t *testing.T) {
	expectedPluralRules := gettext.PluralRulesDefinition{
		Zero:  "n = 0 @integer 0 @decimal 0.0, 0.00, 0.000, 0.0000",
		One:   "n = 1 @integer 1 @decimal 1.0, 1.00, 1.000, 1.0000",
		Two:   "n % 100 = 2,22,42,62,82 or n % 1000 = 0 and n % 100000 = 1000..20000,40000,60000,80000 or n != 0 and n % 1000000 = 100000 @integer 2, 22, 42, 62, 82, 102, 122, 142, 1000, 10000, 100000, … @decimal 2.0, 22.0, 42.0, 62.0, 82.0, 102.0, 122.0, 142.0, 1000.0, 10000.0, 100000.0, …",
		Few:   "n % 100 = 3,23,43,63,83 @integer 3, 23, 43, 63, 83, 103, 123, 143, 1003, … @decimal 3.0, 23.0, 43.0, 63.0, 83.0, 103.0, 123.0, 143.0, 1003.0, …",
		Many:  "n != 1 and n % 100 = 1,21,41,61,81 @integer 21, 41, 61, 81, 101, 121, 141, 161, 1001, … @decimal 21.0, 41.0, 61.0, 81.0, 101.0, 121.0, 141.0, 161.0, 1001.0, …",
		Other: " @integer 4~19, 100, 1004, 1000000, … @decimal 0.1~0.9, 1.1~1.7, 10.0, 100.0, 1000.1, 1000000.0, …",
	}
	documentText := fmt.Sprintf(`
msgid ""
msgstr "Language: az_AZ@latin\n"
"X-PluralRules-Zero: %v\n"
"X-PluralRules-One: %v\n"
"X-PluralRules-Two: %v\n"
"X-PluralRules-Few: %v\n"
"X-PluralRules-Many: %v\n"
"X-PluralRules-Other: %v\n"`,
		expectedPluralRules.Zero,
		expectedPluralRules.One,
		expectedPluralRules.Two,
		expectedPluralRules.Few,
		expectedPluralRules.Many,
		expectedPluralRules.Other)

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}
	header := doc.Header

	expectedTag := language.Make("az-Latn-AZ")
	if header.Tag != expectedTag {
		t.Errorf("Expected tag %v, got %v.", expectedTag, header.Tag)
	}

	verifyPlural := func(d decimal.Decimal, pt gettext.PluralType) {
		if p := header.PluralRules.Evaluate(d); p != pt {
			t.Errorf("Expected %v for %v, got %v.", pt, d, p)
		}
	}

	verifyPlural(decimal.RequireFromString("0"), gettext.PluralTypeZero)
	verifyPlural(decimal.RequireFromString("1"), gettext.PluralTypeOne)
	verifyPlural(decimal.RequireFromString("2"), gettext.PluralTypeTwo)
	verifyPlural(decimal.RequireFromString("3"), gettext.PluralTypeFew)
	verifyPlural(decimal.RequireFromString("21"), gettext.PluralTypeMany)
	verifyPlural(decimal.RequireFromString("4"), gettext.PluralTypeOther)

	verifyPlural(decimal.RequireFromString("0.0"), gettext.PluralTypeZero)
	verifyPlural(decimal.RequireFromString("1.0"), gettext.PluralTypeOne)
	verifyPlural(decimal.RequireFromString("2.0"), gettext.PluralTypeTwo)
	verifyPlural(decimal.RequireFromString("3.0"), gettext.PluralTypeFew)
	verifyPlural(decimal.RequireFromString("21.0"), gettext.PluralTypeMany)
	verifyPlural(decimal.RequireFromString("0.1"), gettext.PluralTypeOther)

	// TODO: add a test for auto sample validation
}

func TestThrows_WhenHeaderEntryIncludesContext(t *testing.T) {
	documentText := `
msgctxt ""
msgid ""
msgstr "bar"`

	_, err := gettext.ParseDocumentString(documentText)

	if _, ok := err.(gettext.DocumentParseError); !ok {
		t.Errorf("Expected %T but got %T: %+v", gettext.DocumentParseError{}, err, err)
	}
}

func TestThrows_IfDuplicateEntryFound(t *testing.T) {
	documentText := genericHeader + `
msgid "foo"
msgstr "bar"

msgid "foo"
msgstr "baz"`

	_, err := gettext.ParseDocumentString(documentText)

	if _, ok := err.(gettext.DocumentParseError); !ok {
		t.Errorf("Expected %T but got %T: %+v", gettext.DocumentParseError{}, err, err)
	}

	documentText = genericHeader + `
msgctxt "apple"
msgid "foo"
msgstr "bar"

msgctxt "apple"
msgid "foo"
msgstr "baz"`

	if _, ok := err.(gettext.DocumentParseError); !ok {
		t.Errorf("Expected %T but got %T: %+v", gettext.DocumentParseError{}, err, err)
	}
}

func TestThrows_IfContextFollowedByContext(t *testing.T) {
	documentText := genericHeader + `
msgctxt "apple"

msgctxt "orange"
msgid "foo"
msgstr "bar"`

	_, err := gettext.ParseDocumentString(documentText)

	if _, ok := err.(gettext.DocumentParseError); !ok {
		t.Errorf("Expected %T but got %T: %+v", gettext.DocumentParseError{}, err, err)
	}
}

func TestParsesTwoEntries_WhenSameIdButDifferentContext(t *testing.T) {
	documentText := genericHeader + `
msgctxt "apple"
msgid "foo"
msgstr "bar"

msgctxt "orange"
msgid "foo"
msgstr "baz"`

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}

	testEntryCount(t, &doc, 2)
	testContextualEntry(t, &doc, "apple", "foo")
	testContextualEntry(t, &doc, "orange", "foo")
}

func TestParsesTwoEntries_WhenSameContextButDifferentId(t *testing.T) {
	documentText := genericHeader + `
msgctxt "apple"
msgid "foo"
msgstr "bar"

msgctxt "apple"
msgid "bar"
msgstr "baz"`

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}

	testEntryCount(t, &doc, 2)
	testContextualEntry(t, &doc, "apple", "foo")
	testContextualEntry(t, &doc, "apple", "bar")
}

func TestParsesTwoEntries_WhenOnlyFirstHasContext(t *testing.T) {
	documentText := genericHeader + `
msgctxt "apple"
msgid "foo"
msgstr "bar"

msgid "foo"
msgstr "baz"`

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}

	testEntryCount(t, &doc, 2)
	testContextualEntry(t, &doc, "apple", "foo")
	testEntry(t, &doc, "foo")
}

func TestParsesTwoEntries_WhenOnlySecondHasContext(t *testing.T) {
	documentText := genericHeader + `
msgid "foo"
msgstr "bar"

msgctxt "apple"
msgid "foo"
msgstr "baz"`

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}

	testEntryCount(t, &doc, 2)
	testEntry(t, &doc, "foo")
	testContextualEntry(t, &doc, "apple", "foo")
}

func TestParsesComments_WhenPlacedInAllSortsOfPlaces(t *testing.T) {
	documentText := genericHeader + `
# comment at start
msgid "foo" #comment inline to msgid
#comment between keywords
msgstr "bar"
#comment at end`

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}
	testEntryCount(t, &doc, 1)

	lines := doc.Entries[1].Lines
	testLine(t, lines, 0, "whitespace line", func(l gettext.Line) bool { return l.IsWhiteSpace }) //just becaus it's not 100% obvious reading code
	testLine(t, lines, 1, "comment", func(l gettext.Line) bool { return l.Comment.Comment == " comment at start" })
	testLine(t, lines, 2, "inline comment", func(l gettext.Line) bool {
		return l.Keyword.Keyword == "msgid" && l.Comment.Comment == "comment inline to msgid"
	})
	testLine(t, lines, 3, "comment", func(l gettext.Line) bool { return l.Comment.Comment == "comment between keywords" })
	testLine(t, lines, 5, "comment", func(l gettext.Line) bool { return l.Comment.Comment == "comment at end" })
}

func TestPostKeywordCommentsPartOfFirstEntry_WhenNoSpaceBetweenFirstAndSecondEntry(t *testing.T) {
	documentText := genericHeader + `
# comment at start
msgid "foo" #comment inline to msgid
#comment between keywords
msgstr "bar"
#comment at end
#another comment at end
msgid "bar"
msgstr "baz"`

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}
	testEntryCount(t, &doc, 2)

	lines := doc.Entries[1].Lines
	testLine(t, lines, len(lines)-1, "entry 1's last line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "another comment at end" })

	lines = doc.Entries[2].Lines
	testLine(t, lines, 0, "entry 2's first line to be msgid", func(l gettext.Line) bool { return l.Keyword.Keyword == "msgid" })
}

func TestPostKeywordCommentsSplitByLine(t *testing.T) {
	documentText := genericHeader + `
# comment at start
msgid "foo" #comment inline to msgid
#comment between keywords
msgstr "bar"
#comment at end
#another comment at end

#comment at start
msgid "bar"
msgstr "baz"`

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}
	testEntryCount(t, &doc, 2)

	lines := doc.Entries[1].Lines
	testLine(t, lines, len(lines)-1, "entry 1's last line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "another comment at end" })

	lines = doc.Entries[2].Lines
	testLine(t, lines, 0, "entry 2's 0th line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
	testLine(t, lines, 1, "entry 2's first line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "comment at start" })
}

func TestSecondEntryIncludesWhitespace_AfterFirstEntrySplitOffByWhitespace(t *testing.T) {
	documentText := genericHeader + `
msgid "foo"
msgstr "bar"

#comment 1

#comment 2
#comment 3
msgid "bar"


msgstr "baz"`

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}
	testEntryCount(t, &doc, 2)

	lines := doc.Entries[2].Lines
	testLine(t, lines, 0, "entry 2's 0th line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
	testLine(t, lines, 1, "entry 2's 1st line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "comment 1" })
	testLine(t, lines, 2, "entry 2's 2nd line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
	testLine(t, lines, 3, "entry 2's 3rd line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "comment 2" })
	testLine(t, lines, 4, "entry 2's 4th line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "comment 3" })
	testLine(t, lines, 6, "entry 2's 6th line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
	testLine(t, lines, 7, "entry 2's 7th line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
}

func TestLastEntryIncludesDoesNotSplitByWhitespace(t *testing.T) {
	documentText := genericHeader + `
msgid "foo"
msgstr "bar"

#comment 1


#comment 2

`

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}
	testEntryCount(t, &doc, 1)

	lines := doc.Entries[1].Lines
	testLine(t, lines, 0, "entry 1's 0th line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
	testLine(t, lines, 3, "entry 1's 3rd line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
	testLine(t, lines, 4, "entry 1's 4th line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "comment 1" })
	testLine(t, lines, 5, "entry 1's 5th line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
	testLine(t, lines, 6, "entry 1's 6th line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
	testLine(t, lines, 7, "entry 1's 7th line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "comment 2" })
	testLine(t, lines, 8, "entry 1's 8th line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })

	if len(lines) != 9 {
		t.Errorf("Expected %v lines for entry 1, got %v", 9, len(lines))
	}
}

func TestParsesMiddleObsoleteEntry(t *testing.T) {
	documentText := genericHeader + `
msgid "foo"
msgstr "bar"

#comment
#~ msgid "bar"
#comment
#~ msgstr "baz"
#~ "something" #inline comment
#comment

msgid "baz"
msgstr "wat"`

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}
	testEntryCount(t, &doc, 3)

	entry := doc.Entries[2]
	if entry.Id != "bar" {
		t.Errorf("Expected id %v, got %v", "bar", entry.Id)
	}
	if !entry.IsObsolete {
		t.Error("Expected obsolete")
	}
	if entry.Value != "bazsomething" {
		t.Errorf("Expected value %v, got %v", "bazsomething", entry.Value)
	}

	lines := entry.Lines
	testLine(t, lines, 0, "entry 2's 0th line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
	testLine(t, lines, 1, "entry 2's 1st line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "comment" })
	testLine(t, lines, 3, "entry 2's 3rd line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "comment" })
	testLine(t, lines, 5, "entry 2's 5th line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "~ \"something\" #inline comment" })
	testLine(t, lines, 6, "entry 2's 6th line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "comment" })
}

func TestParsesConsecutiveObsoleteEntries(t *testing.T) {
	documentText := genericHeader + `
#comment
#~ msgid "a"
#comment
#~ msgstr "baz"
#~ "something" #inline comment
#comment

#comment
#~ msgid "b"
#comment
#~ msgstr "baz"
#~ "something" #inline comment
#comment

#comment
#~ msgid "c"
#comment
#~ msgstr "baz"
#~ "something" #inline comment
#comment`

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}
	testEntryCount(t, &doc, 3)

	if e := doc.Entries[1]; e.Id != "a" {
		t.Errorf("Expected id %v, got %v", "a", e.Id)
	}
	if e := doc.Entries[2]; e.Id != "b" {
		t.Errorf("Expected id %v, got %v", "b", e.Id)
	}
	if e := doc.Entries[3]; e.Id != "c" {
		t.Errorf("Expected id %v, got %v", "c", e.Id)
	}

	for i := 1; i < 4; i += 1 {
		entry := doc.Entries[i]
		if !entry.IsObsolete {
			t.Errorf("Expected entry %v to be obsolete", i)
		}
		if entry.Value != "bazsomething" {
			t.Errorf("Expected entry %v to have value %v, got %v", i, "bazsomething", entry.Value)
		}

		lines := entry.Lines
		testLine(t, lines, 0, fmt.Sprintf("entry %v's 0th line to be whitespace", i),
			func(l gettext.Line) bool { return l.IsWhiteSpace })
		testLine(t, lines, 1, fmt.Sprintf("entry %v's 1st line to be comment", i),
			func(l gettext.Line) bool { return l.Comment.Comment == "comment" })
		testLine(t, lines, 3, fmt.Sprintf("entry %v's 3rd line to be comment", i),
			func(l gettext.Line) bool { return l.Comment.Comment == "comment" })
		testLine(t, lines, 5, fmt.Sprintf("entry %v's 5th line to be comment", i),
			func(l gettext.Line) bool { return l.Comment.Comment == "~ \"something\" #inline comment" })
		testLine(t, lines, 6, fmt.Sprintf("entry %v's 6th line to be comment", i),
			func(l gettext.Line) bool { return l.Comment.Comment == "comment" })
	}
}

func TestParsesObsoleteEntry_WithMidEntryWhitespace(t *testing.T) {
	documentText := genericHeader + `
msgid "foo"
msgstr "bar"

#comment

#~ msgid "bar"

#comment

#~ msgstr "baz"

#~ "something" #inline comment
#comment

msgid "baz"
msgstr "wat"`

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}
	testEntryCount(t, &doc, 3)

	entry := doc.Entries[2]
	if entry.Id != "bar" {
		t.Errorf("Expected id %v, got %v", "bar", entry.Id)
	}
	if !entry.IsObsolete {
		t.Error("Expected obsolete")
	}
	if entry.Value != "bazsomething" {
		t.Errorf("Expected value %v, got %v", "bazsomething", entry.Value)
	}

	lines := entry.Lines
	testLine(t, lines, 0, "entry 2's 0th line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
	testLine(t, lines, 1, "entry 2's 1st line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "comment" })
	testLine(t, lines, 2, "entry 2's 2nd line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
	testLine(t, lines, 4, "entry 2's 4th line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
	testLine(t, lines, 5, "entry 2's 5th line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "comment" })
	testLine(t, lines, 6, "entry 2's 6th line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
	testLine(t, lines, 8, "entry 2's 8th line to be whitespace", func(l gettext.Line) bool { return l.IsWhiteSpace })
	testLine(t, lines, 9, "entry 2's 9th line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "~ \"something\" #inline comment" })
	testLine(t, lines, 10, "entry 2's 10th line to be comment", func(l gettext.Line) bool { return l.Comment.Comment == "comment" })
}

func testEntry(t *testing.T, doc *gettext.Document, id string) {
	for _, e := range doc.Entries[1:] {
		if !e.IsContextual && e.Id == id {
			return
		}
	}

	t.Errorf("Unable to find entry with id of %v", id)
}

func testContextualEntry(t *testing.T, doc *gettext.Document, context string, id string) {
	for _, e := range doc.Entries[1:] {
		if e.IsContextual && e.Context == context && e.Id == id {
			return
		}
	}

	t.Errorf("Unable to find entry with context of %v and id of %v", context, id)
}

func testEntryCount(t *testing.T, doc *gettext.Document, expectedCount int) {
	if entryCount := len(doc.Entries); entryCount != expectedCount+1 {
		t.Errorf("Expected %v non-header entries, got %v", expectedCount, entryCount)
	}
}

func testLine(t *testing.T, lines []gettext.Line, i int, expectation string, predicate func(gettext.Line) bool) {
	if l := lines[i]; !predicate(l) {
		t.Errorf("Expected %v. Line: %v", expectation, l)
	}
}
