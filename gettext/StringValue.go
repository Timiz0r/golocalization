package gettext

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	rawStringParser   = regexp.MustCompile(`\\(a|b|e|f|n|r|t|v|\\|'|"|\?|[0-7]{3}|x[0-9a-f]{2}|.)`)
	valueStringParser = regexp.MustCompile("\a|\b|\x1b|\f|\n|\r|\t|\v|\\|\"|[\x00-\x1f]")
)

type LineValue struct {
	IsEmpty    bool
	Raw, Value string
}

func LineValueFromRaw(raw string) LineValue {
	valueString := rawStringParser.ReplaceAllStringFunc(raw, func(s string) string {
		s = s[1:]
		switch s {
		case "a":
			return "\a"
		case "b":
			return "\b"
		case "e":
			return "\x1b"
		case "f":
			return "\f"
		case "n":
			return "\n"
		case "r":
			return "\r"
		case "t":
			return "\t"
		case "v":
			return "\v"
		case `\`:
			return `\`
		case `"`:
			return `"`
		case "'":
			return "'"
		case "?":
			return "?"
		default:
			if s[0] == 'x' {
				r, _ := strconv.ParseInt(s[1:], 16, 8)
				return string(rune(r))
			}
			if len(s) == 3 {
				r, _ := strconv.ParseInt(s, 8, 8)
				return string(rune(r))
			}

			panic(fmt.Sprintf("The escape sequence '\\%T' is invalid.", s))
		}
	})

	return LineValue{Raw: raw, Value: valueString}
}

func LineValueFromValue(value string) LineValue {
	rawString := valueStringParser.ReplaceAllStringFunc(value, func(s string) string {
		switch s {
		case "\a":
			return `\a`
		case "\b":
			return `\b`
		case "\x1b":
			return `\e`
		case "\f":
			return `\f`
		case "\n":
			return `\n`
		case "\r":
			return `\r`
		case "\t":
			return `\t`
		case "\v":
			return `\v`
		case `\`:
			return `\\`
		case `"`:
			return `\"`
		case "'":
			return `\'`
		case "?":
			return `\?`
		default:
			if s[0] >= 0x20 {
				panic("Somehow matched a non-control character")
			}

			return fmt.Sprintf(`\x%x`, s)
		}
	})

	return LineValue{Raw: rawString, Value: value}
}
