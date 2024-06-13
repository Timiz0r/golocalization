package gettext_test

import (
	"fmt"
	"slices"
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

func TestParsesPlurals(t *testing.T) {
	documentText := genericHeader + `
msgctxt "foo"
msgid "bar"
msgid_plural "bars"
msgstr[0] "bazs0"
msgstr[1] "bazs1"
msgstr[2] "bazs2"`

	doc, err := gettext.ParseDocumentString(documentText)
	if err != nil {
		t.Error("Error parsing document: ", err)
	}
	testEntryCount(t, &doc, 1)

	testContextualEntry(t, &doc, "foo", "bar")

	entry := doc.Entries[1]
	if entry.PluralId != "bars" {
		t.Errorf("Expected plural id %v, got %v.", "bars", entry.PluralId)
	}

	expectedPluralValues := [3]string{"bazs0", "bazs1", "bazs2"}
	if !slices.Equal(expectedPluralValues[:], entry.PluralValues) {
		t.Errorf("Expected plural values of %v, got %v.", expectedPluralValues, entry.PluralValues)
	}
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
