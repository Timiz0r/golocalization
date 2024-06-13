package gettext

import (
	"fmt"
	"strings"
	"text/scanner"
	"unicode"

	"github.com/shopspring/decimal"
)

type PluralRuleValidationError struct {
	RuleString     string
	InvalidSamples []decimal.Decimal
}

func (e PluralRuleValidationError) Error() string {
	return fmt.Sprintf("Plural rule '%v' failed sample validation. Invalid samples: %v",
		e.RuleString, e.InvalidSamples)
}

func validatePluralRule(pluralRule PluralRuleOperation, sampleString string, ruleString string) error {
	samples := parsePluralRuleSample(sampleString)
	var invalidSamples []decimal.Decimal

	for _, sample := range samples {
		result := pluralRule(sample)
		if !result {
			invalidSamples = append(invalidSamples, sample)
		}
	}

	if len(invalidSamples) > 0 {
		return PluralRuleValidationError{
			RuleString:     ruleString,
			InvalidSamples: invalidSamples,
		}
	}

	return nil
}

func parsePluralRuleSample(sample string) []decimal.Decimal {
	tokens := tokenizePluralRuleSample(sample)
	var results []decimal.Decimal

	var lowValue decimal.Decimal
	isRange := false
	for _, t := range tokens {
		if t.Kind == tokenIntegerSample || t.Kind == tokenDecimalSample {

			//since the decimal type represents both just fine, we don't otherwise care about these
			continue
		}

		var one = decimal.New(1, 0)

		switch t.Kind {
		case tokenNumber:
			n := decimal.RequireFromString(t.Value)

			if isRange {
				// easiest way to get number of decimal digits
				o := createOperands(n)
				increment := one.Pow(o.V.Neg())
				for d := lowValue.Add(increment); d.LessThanOrEqual(n); d = d.Add(increment) {
					results = append(results, d)
				}

				isRange = false
			} else {
				results = append(results, n)
				lowValue = n
			}
		case tokenRange:
			isRange = true

		case tokenIntegerSample:
			continue
		case tokenDecimalSample:
			continue
		case tokenComma:
			continue
		case tokenTripleDot:
			continue

		default:
			panic(fmt.Sprintf("Unknown token found: %v", t.Kind))
		}
	}

	return results
}

func tokenizePluralRuleSample(sample string) []token {
	var s scanner.Scanner
	s.Init(strings.NewReader(sample))
	s.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.SkipComments
	s.IsIdentRune = func(ch rune, i int) bool {
		// since we treat ... as an ident, we also treat … as one for consistency
		// and for letters, we need to be careful about i because we need to filter out exponential notation later on
		return ch == '@' || unicode.IsLetter(ch) && i > 0 || ch == '…' || ch == '.' && (i == 0 || i == 1 || i == 2)
	}

	var tokens []token
	const noValue = ""

	var foundExponentialNotation bool
ScanLoop:
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		// we dont support exponential notation, so we need to strip the prior number
		// and ignore the next number if we find it

		switch tok {
		case scanner.Ident:
			break
		case scanner.Int:
			if foundExponentialNotation {
				foundExponentialNotation = false
				continue ScanLoop
			}
			tokens = append(tokens, token{tokenNumber, s.TokenText()})
			continue ScanLoop
		case scanner.Float:
			if foundExponentialNotation {
				foundExponentialNotation = false
				continue ScanLoop
			}
			tokens = append(tokens, token{tokenNumber, s.TokenText()})
			continue ScanLoop
		case '~':
			tokens = append(tokens, token{tokenRange, noValue})
			continue ScanLoop
		case ',':
			tokens = append(tokens, token{tokenComma, noValue})
			continue ScanLoop
		case 'c':
			tokens = tokens[:len(tokens)-1]
			foundExponentialNotation = true
			continue ScanLoop
		case 'e':
			tokens = tokens[:len(tokens)-1]
			foundExponentialNotation = true
			continue ScanLoop
		default:
			panic(fmt.Sprintf("Unknown token '%v'", tok))
		}

		switch value := s.TokenText(); value {
		case "...":
			tokens = append(tokens, token{tokenTripleDot, noValue})
		case "…":
			tokens = append(tokens, token{tokenTripleDot, noValue})

		case "@integer":
			tokens = append(tokens, token{tokenIntegerSample, noValue})
		case "@decimal":
			tokens = append(tokens, token{tokenDecimalSample, noValue})

		default:
			panic(fmt.Sprintf("Unknown ident '%v'", value))
		}
	}

	return tokens
}
