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
	// TODO: validate samples
	tokens, _ := tokenizePluralRule(pluralRule)

	relation := constructAndConditionChain(tokens)

	kind, _ := readNextToken(tokens, tokenOr)
	for kind != tokenNotFound {
		relation = func(o operands) bool {
			newRelation := constructAndConditionChain(tokens)
			return relation(o) || newRelation(o)
		}

		kind, _ = readNextToken(tokens, tokenOr)
	}

	if len(*tokens) > 0 {
		panic(fmt.Sprint("Unexpectedly have additional tokens: ", *tokens))
	}

	return func(d decimal.Decimal) bool {
		return relation(createOperands(d))
	}
}

func constructAndConditionChain(tokens *[]token) relation {
	relation := constructListRelation(tokens)

	kind, _ := readNextToken(tokens, tokenAnd)
	for kind != tokenNotFound {
		relation = func(o operands) bool {
			newRelation := constructListRelation(tokens)
			return relation(o) && newRelation(o)
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
		relation = func(o operands) bool {
			nextRelation := constructSingleRelation(tokens, accessor, isEqualityOperation)
			return relation(o) || nextRelation(o)
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
		accessor = func(o operands) decimal.Decimal {
			return accessor(o).Mod(modValue)
		}
	}

	return accessor
}

const (
	tokenNotFound = iota - 1
	tokenOperandName
	tokenAnd
	tokenOr
	tokenEquals
	tokenNotEquals
	tokenModulus
	tokenComma
	tokenRange
	tokenNumber
)

type token struct {
	Kind  int
	Value string
}

func readNextToken(tokens *[]token, expectedKinds ...int) (kind int, value string) {
	if len(*tokens) == 0 {
		panic("No tokens remaining.")
	}

	for expectedKind := range expectedKinds {
		if t := (*tokens)[0]; t.Kind == expectedKind {
			kind = expectedKind
			value = t.Value
			*tokens = (*tokens)[1:]
			return
		}
	}

	return tokenNotFound, ""
}

func mustReadNextToken(tokens *[]token, expectedKinds ...int) (int, string) {
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
