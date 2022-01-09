package godb

import (
	"fmt"
	"strings"
)

// location represents a position in the SQL query
type location struct {
	line   uint
	column uint
}

// keyword Represents a keyword in an SQL query
type keyword string

// Supported keywords
const (
	SELECT keyword = "select"
	WHERE  keyword = "where"
	FROM   keyword = "from"
	AS     keyword = "as"
	TABLE  keyword = "table"
	CREATE keyword = "create"
	INSERT keyword = "insert"
	INTO   keyword = "into"
	VALUES keyword = "values"
	INT    keyword = "int"
	TEXT   keyword = "text"
)

// symbol represents special
type symbol string

const (
	SEMICOLON  symbol = ";"
	ASTERISK   symbol = "*"
	COMMA      symbol = ","
	LEFTPAREN  symbol = "("
	RIGHTPAREN symbol = ")"
)

type tokenKind uint

const (
	KEYWORD tokenKind = iota
	SYMBOL
	IDENTIFIER
	STRING
	NUMERIC
)

type token struct {
	value string
	kind  tokenKind
	loc   location
}

type cursor struct {
	pointer uint
	loc     location
}

func (t *token) equals(other *token) bool {
	return t.value == other.value && t.kind == other.kind
}

type lexer func(string, cursor) (*token, cursor, bool)

func lex(source string) ([]*token, error) {
	var tokens []*token
	var cur = cursor{}

lex:
	for cur.pointer < uint(len(source)) {
		lexers := []lexer{lexKeyword, lexSymbol, lexString, lexNumeric, lexIdentifier}
		for _, lexer := range lexers {
			if token, newCursor, ok := lexer(source, cur); ok {
				cur = newCursor
				if token != nil {
					tokens = append(tokens, token)
				}
				continue lex
			}
		}
		errorHint := ""
		if len(tokens) > 0 {
			errorHint = "after " + tokens[len(tokens)-1].value
		}
		return nil, fmt.Errorf("unidentified token %s, at %d:%d", errorHint, cur.loc.line, cur.loc.column)
	}
	return tokens, nil
}

func lexIdentifier(source string, ic cursor) (*token, cursor, bool) {
	if token, newCursor, ok := lexCharacterDelimited(source, ic, '"'); ok {
		return token, newCursor, true
	}

	cur := ic

	c := source[cur.pointer]

	// check the first character is an alphabet
	isAlphabet := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
	if !isAlphabet {
		return nil, ic, false
	}
	cur.pointer++
	cur.loc.column++

	value := []byte{c}
	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c = source[cur.pointer]

		isAlphabet := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		isNumber := c >= '0' && c <= '9'
		if isAlphabet || isNumber || c == '$' || c == '_' {
			value = append(value, c)
			cur.loc.column++
			continue
		}
		break
	}
	if len(value) == 0 {
		return nil, ic, false
	}

	return &token{
		value: strings.ToLower(string(value)),
		loc:   ic.loc,
		kind:  IDENTIFIER,
	}, cur, true
}

func lexCharacterDelimited(source string, ic cursor, delimiter byte) (*token, cursor, bool) {
	cur := ic

	if len(source[cur.pointer:]) == 0 {
		return nil, ic, false
	}

	if source[cur.pointer] != delimiter {
		return nil, ic, false
	}

	cur.loc.column++
	cur.pointer++

	var value []byte
	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c := source[cur.pointer]
		if c == delimiter {
			// Escape seq contains 2 same characters
			if cur.pointer+1 >= uint(len(source)) || source[cur.pointer+1] != delimiter {
				cur.pointer++
				cur.loc.column++
				return &token{
					value: string(value),
					loc:   ic.loc,
					kind:  STRING,
				}, cur, true
			} else {
				value = append(value, delimiter)
				cur.pointer++
				cur.loc.column++
			}
		}
		value = append(value, c)
		cur.loc.column++
	}
	return nil, ic, false
}

func lexString(source string, ic cursor) (*token, cursor, bool) {
	return lexCharacterDelimited(source, ic, '\'')
}

func lexSymbol(source string, ic cursor) (*token, cursor, bool) {
	cur := ic
	c := source[cur.pointer]

	cur.pointer++
	cur.loc.column++

	switch c {
	case '\n':
		cur.loc.line++
		cur.loc.column = 0
		fallthrough
	case '\t':
		fallthrough
	case ' ':
		return nil, cur, true
	}

	symbols := []symbol{
		COMMA,
		LEFTPAREN,
		RIGHTPAREN,
		SEMICOLON,
		ASTERISK,
	}

	var options []string
	for _, s := range symbols {
		options = append(options, string(s))
	}
	match := longestMatch(source, ic, options)
	if match == "" {
		return nil, ic, false
	}

	cur.pointer = ic.pointer + uint(len(match))
	cur.loc.column = ic.loc.column + uint(len(match))

	return &token{
		value: match,
		loc:   ic.loc,
		kind:  SYMBOL,
	}, cur, true
}

func lexNumeric(source string, ic cursor) (*token, cursor, bool) {
	cur := ic

	decimalFound := false
	exponentFound := false

	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c := source[cur.pointer]
		cur.loc.column++

		isDigit := c >= '0' && c <= '9'
		isDecimal := c == '.'
		isExponent := c == 'e' || c == 'E'

		// first character should be a number or a decimal
		if cur.pointer == ic.pointer {
			if !isDigit && !isDecimal {
				return nil, ic, false
			}
			decimalFound = isDecimal
			continue
		}
		if isDecimal {
			if decimalFound {
				return nil, ic, false
			}
			decimalFound = true
			continue
		}
		if isExponent {
			if exponentFound {
				return nil, ic, false
			}
			decimalFound = true
			exponentFound = true
			//Exponent should be followed by a number or sign
			if cur.pointer == uint(len(source)-1) {
				return nil, ic, false
			}
			cNext := source[cur.pointer+1]
			if cNext == '-' || cNext == '+' {
				cur.pointer++
				cur.loc.column++
			}
			continue
		}
		if !isDigit {
			break
		}
	}
	if cur.pointer == ic.pointer {
		return nil, ic, false
	}

	return &token{
		value: source[ic.pointer:cur.pointer],
		loc:   ic.loc,
		kind:  NUMERIC,
	}, cur, true
}

func lexKeyword(source string, ic cursor) (*token, cursor, bool) {
	cur := ic
	keywords := []keyword{
		SELECT,
		INSERT,
		VALUES,
		TABLE,
		CREATE,
		WHERE,
		FROM,
		INTO,
		INT,
		TEXT,
		AS,
	}

	var options []string
	for _, k := range keywords {
		options = append(options, string(k))
	}

	match := longestMatch(source, ic, options)
	if match == "" {
		return nil, ic, false
	}

	cur.pointer = ic.pointer + uint(len(match))
	cur.loc.column = ic.loc.column + uint(len(match))

	return &token{
		value: match,
		loc:   ic.loc,
		kind:  KEYWORD,
	}, cur, true
}

func longestMatch(source string, ic cursor, options []string) string {
	var value []byte
	var skiplist []int
	var match string

	cur := ic

	for cur.pointer < uint(len(source)) {
		value = append(value, strings.ToLower(string(source[cur.pointer]))...)
		cur.pointer++

	match:
		for i, option := range options {
			for _, skip := range skiplist {
				if i == skip {
					continue match
				}
			}

			if option == string(value) {
				skiplist = append(skiplist, i)
				if len(option) > len(match) {
					match = option
				}
				continue
			}

			sharesPrefix := string(value) == option[:cur.pointer-ic.pointer]
			tooLong := len(value) > len(option)
			if tooLong || !sharesPrefix {
				skiplist = append(skiplist, i)
			}
		}
		if len(skiplist) == len(options) {
			break
		}
	}
	return match
}
