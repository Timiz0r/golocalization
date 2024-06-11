package gettext

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/text/language"
)

type DocumentHeader struct {
	Tag         language.Tag
	PluralRules PluralRules
}

type DocumentHeaderParseError struct {
	Entry  Entry
	Reason string
}

func CreateHeaderFromEntry(entry Entry) (DocumentHeader, error) {
	if entry.IsContextual {
		return DocumentHeader{}, DocumentHeaderParseError{entry, "Header must not be contextual."}
	}

	matches := languageExtractor.FindStringSubmatch(entry.Value)
	if matches == nil {
		return DocumentHeader{}, DocumentHeaderParseError{entry, "Language not found."}
	}
	rawLanguageValue := matches[1]
	var languageValue string

	matches = languageParser.FindStringSubmatch(rawLanguageValue)
	if matches == nil {
		languageValue = rawLanguageValue
	} else {
		languageValue = matches[1]

		if getTextVariant := matches[3]; len(getTextVariant) > 0 {
			if variant, ok := getTextVariantMap[getTextVariant]; ok {
				languageValue = fmt.Sprint(languageValue, "-", variant)
			} else {
				return DocumentHeader{}, DocumentHeaderParseError{entry, fmt.Sprint("Unable to parse variant of language: ", rawLanguageValue)}
			}
		}

		if country := matches[2]; len(country) > 0 {
			languageValue = fmt.Sprint(languageValue, "-", country)
		}
	}

	var d PluralRulesDefinition

	for _, rawPluralRules := range pluralRuleExtractor.FindAllStringSubmatch(entry.Value, -1) {
		rule := rawPluralRules[2]
		switch strings.ToLower(rawPluralRules[1]) {
		case "zero":
			d.Zero = rule
		case "one":
			d.One = rule
		case "two":
			d.Two = rule
		case "few":
			d.Few = rule
		case "many":
			d.Many = rule
		case "other":
			d.Other = rule
		}
	}

	return DocumentHeader{
		language.Make(languageValue),
		d.Parse(),
	}, nil
}

func (e DocumentHeaderParseError) Error() string {
	return fmt.Sprint("Failed to parse document header: ", e.Reason)
}

var (
	languageExtractor   = regexp.MustCompile(`(?im)^Language: (.+)$`)
	languageParser      = regexp.MustCompile(`(?i)([a-z]+)(?:_([a-z]+))?(?:@([a-z]+))?`)
	pluralRuleExtractor = regexp.MustCompile(`(?im)^X-PluralRules-([a-z]+): *(.*)$`)
	getTextVariantMap   = map[string]string{
		"latin":       "Latn",
		"cyrillic":    "Cyrl",
		"adlam":       "Adlm",
		"javanese":    "Java",
		"arabic":      "Arab",
		"devanagari":  "Deva",
		"mongolian":   "Mong",
		"bangla":      "Beng",
		"gurmukhi":    "Guru",
		"olchiki":     "Olck",
		"tifinagh":    "Tfng",
		"vai":         "Vaii",
		"simplified":  "Hans",
		"traditional": "Hant",
	}
)
