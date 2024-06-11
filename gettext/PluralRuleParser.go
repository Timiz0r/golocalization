package gettext

import (
	"fmt"
	"strings"
	"text/scanner"
	"unicode"

	"github.com/shopspring/decimal"
)

type relation func(o operands) bool

// when panicing, would be nice to output an informative string
// but since these will generally be loaded from the official xml, panicing should not happen
func parsePluralRule(pluralRule string) PluralRuleOperation {
	// NOTE: this means we dont support plural rules without the sample string
	// since we can't differentiate between a zero-length (valid) rule and an non-existent rule
	// but this should not be a problem in practice
	if len(pluralRule) == 0 {
		return nil
	}

	tokens, sample := tokenizePluralRule(pluralRule)
	if len(*tokens) == 0 {
		return func(_ decimal.Decimal) bool {
			return true
		}
	}

	relation := constructAndConditionChain(tokens)

	kind, _ := readNextToken(tokens, tokenOr)
	for kind != tokenNotFound {
		oldRelation := relation
		newRelation := constructAndConditionChain(tokens)
		relation = func(o operands) bool {
			return oldRelation(o) || newRelation(o)
		}

		kind, _ = readNextToken(tokens, tokenOr)
	}

	if len(*tokens) > 0 {
		panic(fmt.Sprint("Unexpectedly have additional tokens: ", *tokens))
	}

	result := func(d decimal.Decimal) bool {
		return relation(createOperands(d))
	}

	validationError := validatePluralRule(result, sample, pluralRule)
	if validationError != nil {
		panic(validationError)
	}

	return result
}

func constructAndConditionChain(tokens *[]token) relation {
	relation := constructListRelation(tokens)

	kind, _ := readNextToken(tokens, tokenAnd)
	for kind != tokenNotFound {
		oldRelation := relation
		newRelation := constructListRelation(tokens)
		relation = func(o operands) bool {
			return oldRelation(o) && newRelation(o)
		}

		kind, _ = readNextToken(tokens, tokenAnd)
	}

	return relation
}

func constructListRelation(tokens *[]token) relation {
	accessor := constructAccessor(tokens)

	var isEqualityOperation bool
	switch kind, _ := mustReadNextToken(tokens, tokenEquals, tokenNotEquals); kind {
	case tokenEquals:
		isEqualityOperation = true
	case tokenNotEquals:
		isEqualityOperation = false
	default:
		panic("Cannot be hit because we verify tokenEquals and tokenNotEquals")
	}

	relation := constructSingleRelation(tokens, accessor, isEqualityOperation)

	kind, _ := readNextToken(tokens, tokenComma)
	for kind != tokenNotFound {
		oldRelation := relation
		newRelation := constructSingleRelation(tokens, accessor, isEqualityOperation)
		relation = func(o operands) bool {
			return oldRelation(o) || newRelation(o)
		}

		kind, _ = readNextToken(tokens, tokenComma)
	}

	return relation
}

func constructSingleRelation(tokens *[]token, accessor accessor, isEqualityOperation bool) relation {
	number := readNumber(tokens)
	highNumber, isRange := readRange(tokens)

	var r relation
	if isRange {
		r = func(o operands) (result bool) {
			n := accessor(o)
			result = n.GreaterThanOrEqual(number) && n.LessThanOrEqual(highNumber)
			if !isEqualityOperation {
				result = !result
			}
			return
		}
	} else {
		r = func(o operands) (result bool) {
			n := accessor(o)
			result = n.Equal(number)
			if !isEqualityOperation {
				result = !result
			}
			return
		}
	}
	return r
}

type accessor func(o operands) decimal.Decimal

var (
	accessorN = func(o operands) decimal.Decimal { return o.N }
	accessorI = func(o operands) decimal.Decimal { return o.I }
	accessorV = func(o operands) decimal.Decimal { return o.V }
	accessorW = func(o operands) decimal.Decimal { return o.W }
	accessorF = func(o operands) decimal.Decimal { return o.F }
	accessorT = func(o operands) decimal.Decimal { return o.T }
	// not supported and can only ever be zero
	accessorC = func(o operands) decimal.Decimal { return decimal.Zero }
	accessorE = func(o operands) decimal.Decimal { return decimal.Zero }
)

func constructAccessor(tokens *[]token) accessor {
	_, operandName := mustReadNextToken(tokens, tokenOperandName)
	modValue, hasModValue := readModulus(tokens)

	var accessor accessor
	switch operandName {
	case "n":
		accessor = accessorN
	case "i":
		accessor = accessorI
	case "v":
		accessor = accessorV
	case "w":
		accessor = accessorW
	case "f":
		accessor = accessorF
	case "t":
		accessor = accessorT
	case "c":
		accessor = accessorC
	case "e":
		accessor = accessorE
	default:
		panic(fmt.Sprint("Unknown operand name: ", operandName))
	}

	if hasModValue {
		oldAccessor := accessor
		accessor = func(o operands) decimal.Decimal {
			return oldAccessor(o).Mod(modValue)
		}
	}

	return accessor
}

type tokenKind int

const (
	tokenNotFound = tokenKind(iota - 1)
	tokenOperandName
	tokenAnd
	tokenOr
	tokenEquals
	tokenNotEquals
	tokenModulus
	tokenComma
	tokenRange
	tokenNumber

	// used only for sample parsing
	tokenIntegerSample
	tokenDecimalSample
	tokenTripleDot
)

type token struct {
	Kind  tokenKind
	Value string
}

func readNextToken(tokens *[]token, expectedKinds ...tokenKind) (kind tokenKind, value string) {
	if len(*tokens) == 0 {
		return tokenNotFound, ""
	}

	token := (*tokens)[0]
	for _, expectedKind := range expectedKinds {
		if token.Kind == tokenKind(expectedKind) {
			kind = tokenKind(expectedKind)
			value = token.Value
			*tokens = (*tokens)[1:]
			return
		}
	}

	return tokenNotFound, ""
}

func mustReadNextToken(tokens *[]token, expectedKinds ...tokenKind) (tokenKind, string) {
	kind, value := readNextToken(tokens, expectedKinds...)

	if kind == tokenNotFound {
		panic(fmt.Sprintf("Expected %v. Got %v.", expectedKinds, (*tokens)[0].Kind))
	}

	return kind, value
}

func readModulus(tokens *[]token) (decimal.Decimal, bool) {
	kind, _ := readNextToken(tokens, tokenModulus)
	if kind == tokenNotFound {
		return decimal.Zero, false
	}

	return readNumber(tokens), true
}

func readRange(tokens *[]token) (decimal.Decimal, bool) {
	kind, _ := readNextToken(tokens, tokenRange)
	if kind == tokenNotFound {
		return decimal.Zero, false
	}

	return readNumber(tokens), true
}

func readNumber(tokens *[]token) decimal.Decimal {
	_, rawNumber := mustReadNextToken(tokens, tokenNumber)
	result, err := decimal.NewFromString(rawNumber)
	if err != nil {
		panic(err)
	}
	return result
}

func tokenizePluralRule(pluralRule string) (*[]token, string) {
	var s scanner.Scanner
	s.Init(strings.NewReader(pluralRule))
	s.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.SkipComments
	s.IsIdentRune = func(ch rune, i int) bool {
		//group ! and = if possible for an easier parsing time
		return ch == '!' || ch == '=' || ch == '.' || unicode.IsLetter(ch)
	}

	var tokens []token
	const noValue = ""

ScanLoop:
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		switch tok {
		case scanner.Ident:
			break
		case '@':
			break ScanLoop
		case scanner.Int:
			tokens = append(tokens, token{tokenNumber, s.TokenText()})
			continue ScanLoop
		case '%':
			tokens = append(tokens, token{tokenModulus, noValue})
			continue ScanLoop
		case ',':
			tokens = append(tokens, token{tokenComma, noValue})
			continue ScanLoop
		default:
			panic(fmt.Sprintf("Unknown token '%v'", tok))
		}

		switch value := s.TokenText(); value {
		case "..":
			tokens = append(tokens, token{tokenRange, noValue})

		case "or":
			tokens = append(tokens, token{tokenOr, noValue})
		case "and":
			tokens = append(tokens, token{tokenAnd, noValue})

		case "=":
			tokens = append(tokens, token{tokenEquals, noValue})
		case "!=":
			tokens = append(tokens, token{tokenNotEquals, noValue})

		case "n":
			tokens = append(tokens, token{tokenOperandName, value})
		case "i":
			tokens = append(tokens, token{tokenOperandName, value})
		case "v":
			tokens = append(tokens, token{tokenOperandName, value})
		case "w":
			tokens = append(tokens, token{tokenOperandName, value})
		case "f":
			tokens = append(tokens, token{tokenOperandName, value})
		case "t":
			tokens = append(tokens, token{tokenOperandName, value})
		case "c":
			tokens = append(tokens, token{tokenOperandName, value})
		case "e":
			tokens = append(tokens, token{tokenOperandName, value})
		default:
			panic(fmt.Sprintf("Unknown ident '%v'", value))
		}
	}

	return &tokens, pluralRule[s.Offset:]
}

func (t tokenKind) String() string {
	switch t {

	case tokenNotFound:
		return "tokenNotFound"
	case tokenOperandName:
		return "tokenOperandName"
	case tokenAnd:
		return "tokenAnd"
	case tokenOr:
		return "tokenOr"
	case tokenEquals:
		return "tokenEquals"
	case tokenNotEquals:
		return "tokenNotEquals"
	case tokenModulus:
		return "tokenModulus"
	case tokenComma:
		return "tokenComma"
	case tokenRange:
		return "tokenRange"
	case tokenNumber:
		return "tokenNumber"
	case tokenIntegerSample:
		return "tokenIntegerSample"
	case tokenDecimalSample:
		return "tokenDecimalSample"
	case tokenTripleDot:
		return "tokenTripleDot"
	default:
		panic(fmt.Sprintf("Found invalid %T: %v", tokenNotFound, int(t)))
	}
}
