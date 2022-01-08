package godb

import (
	"errors"
	"fmt"
)

func tokenFromKeyword(k keyword) token {
	return token{
		kind:  KEYWORD,
		value: string(k),
	}
}

func tokenFromSymbol(s symbol) token {
	return token{
		value: string(s),
		kind:  SYMBOL,
	}
}

func helpMessage(tokens []*token, cursor uint, msg string) {
	var c *token
	if cursor < uint(len(tokens)) {
		c = tokens[cursor]
	} else {
		c = tokens[cursor-1]
	}
	fmt.Printf("[%d,%d]: %s got: %s\n", c.loc.line, c.loc.column, msg, c.value)
}

func expectToken(tokens []*token, cursor uint, t token) bool {
	if cursor >= uint(len(tokens)) {
		return false
	}
	return t.equals(tokens[cursor])
}

func parse(source string) (*Ast, error) {
	// lexical analysis get the tokens
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}

	a := Ast{}
	cursor := uint(0)

	for cursor < uint(len(tokens)) {
		stmt, newCursor, ok := parseStatement(tokens, cursor, tokenFromSymbol(SEMICOLON))
		if !ok {
			helpMessage(tokens, cursor, "Expected statement")
			return nil, errors.New("failed to parse, expected statement")
		}
		cursor = newCursor
		a.Statements = append(a.Statements, stmt)

		atLeastOneSemicolon := false
		for expectToken(tokens, cursor, tokenFromSymbol(SEMICOLON)) {
			cursor++
			atLeastOneSemicolon = true
		}
		if !atLeastOneSemicolon {
			helpMessage(tokens, cursor, "Expected semi-colon delimiter between statements")
			return nil, errors.New("missing semi-colon between statements")
		}
	}
	return &a, nil
}

func parseStatement(tokens []*token, initialCursor uint, delimiter token) (*Statement, uint, bool) {
	cursor := initialCursor

	if slct, newCursor, ok := parseSelectStatement(tokens, cursor, tokenFromSymbol(SEMICOLON)); ok {
		return &Statement{
			Kind:            SelectStmtKind,
			SelectStatement: slct,
		}, newCursor, true
	}

	if inst, newCursor, ok := parseInsertStatement(tokens, cursor, tokenFromSymbol(SEMICOLON)); ok {
		return &Statement{
			Kind:            InsertStmtKind,
			InsertStatement: inst,
		}, newCursor, true
	}

	if crt, newCursor, ok := parseCreateTableStatement(tokens, cursor, tokenFromSymbol(SEMICOLON)); ok {
		return &Statement{
			Kind:            CreateStmtKind,
			CreateStatement: crt,
		}, newCursor, true
	}
	return nil, initialCursor, false
}

func parseCreateTableStatement(tokens []*token, initialCursor uint, delimiter token) (*CreateStatement, uint, bool) {
	cursor := initialCursor

	if !expectToken(tokens, cursor, tokenFromKeyword(CREATE)) {
		return nil, initialCursor, false
	}
	cursor++

	if !expectToken(tokens, cursor, tokenFromKeyword(TABLE)) {
		return nil, initialCursor, false
	}
	cursor++

	name, newCursor, ok := parseToken(tokens, cursor, IDENTIFIER)
	if !ok {
		helpMessage(tokens, cursor, "Expected Table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	if !expectToken(tokens, cursor, tokenFromSymbol(LEFTPAREN)) {
		helpMessage(tokens, cursor, "Expected left parenthesis")
		return nil, initialCursor, false
	}
	cursor++

	cols, newCursor, ok := parseColumnDefinitions(tokens, cursor, tokenFromSymbol(RIGHTPAREN))
	if !ok {
		helpMessage(tokens, cursor, "Invalid column definition")
		return nil, initialCursor, false
	}
	cursor = newCursor

	if !expectToken(tokens, cursor, tokenFromSymbol(RIGHTPAREN)) {
		helpMessage(tokens, cursor, "Expected right parenthesis")
		return nil, initialCursor, false
	}
	cursor++

	return &CreateStatement{
		name: *name,
		cols: cols,
	}, cursor, true
}

func parseColumnDefinitions(tokens []*token, initialCursor uint, delimiter token) (*[]*columnDefinition, uint, bool) {
	cursor := initialCursor

	var cds []*columnDefinition
	for {
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}

		current := tokens[cursor]
		if delimiter.equals(current) {
			break
		}

		if len(cds) > 0 {
			if !expectToken(tokens, cursor, tokenFromSymbol(COMMA)) {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
			cursor++
		}

		id, newCursor, ok := parseToken(tokens, cursor, IDENTIFIER)
		if !ok {
			helpMessage(tokens, cursor, "Expected column name")
			return nil, initialCursor, false
		}
		cursor = newCursor

		ty, newCursor, ok := parseToken(tokens, cursor, KEYWORD)
		if !ok {
			helpMessage(tokens, cursor, "Expected column type")
			return nil, initialCursor, false
		}
		cursor = newCursor

		cds = append(cds, &columnDefinition{
			name:     *id,
			datatype: *ty,
		})
	}
	return &cds, cursor, true
}

func parseSelectStatement(tokens []*token, initialCursor uint, delimiter token) (*SelectStatement, uint, bool) {
	cursor := initialCursor

	if !expectToken(tokens, cursor, tokenFromKeyword(SELECT)) {
		return nil, initialCursor, false
	}
	cursor++
	slct := SelectStatement{}

	exps, newCursor, ok := parseExpressions(tokens, cursor, []token{tokenFromKeyword(FROM), delimiter})
	if !ok {
		return nil, initialCursor, false
	}

	slct.item = *exps
	cursor = newCursor

	if expectToken(tokens, cursor, tokenFromKeyword(FROM)) {
		cursor++
		from, newCursor, ok := parseToken(tokens, cursor, IDENTIFIER)
		if !ok {
			helpMessage(tokens, cursor, "Expected FROM token")
			return nil, initialCursor, false
		}
		slct.from = *from
		cursor = newCursor
	}
	return &slct, cursor, true
}

func parseToken(tokens []*token, initialCursor uint, kind tokenKind) (*token, uint, bool) {
	cursor := initialCursor

	if cursor >= uint(len(tokens)) {
		return nil, initialCursor, false
	}
	current := tokens[cursor]
	if current.kind == kind {
		return current, cursor + 1, true
	}
	return nil, initialCursor, false
}

func parseExpressions(tokens []*token, initialCursor uint, delimiters []token) (*[]*expression, uint, bool) {
	cursor := initialCursor
	var exps []*expression

outer:
	for {
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}

		current := tokens[cursor]
		for _, delimiter := range delimiters {
			if delimiter.equals(current) {
				break outer
			}
		}
		if len(exps) > 0 {
			if !expectToken(tokens, cursor, tokenFromSymbol(COMMA)) {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
			cursor++
		}

		exp, newCursor, ok := parseExpression(tokens, cursor, tokenFromSymbol(COMMA))
		if !ok {
			helpMessage(tokens, cursor, "Expected expression")
			return nil, initialCursor, false
		}
		cursor = newCursor

		exps = append(exps, exp)
	}
	return &exps, cursor, true
}

func parseExpression(tokens []*token, initialCursor uint, _ token) (*expression, uint, bool) {
	cursor := initialCursor

	kinds := []tokenKind{IDENTIFIER, NUMERIC, STRING}
	for _, kind := range kinds {
		t, newCursor, ok := parseToken(tokens, cursor, kind)
		if ok {
			return &expression{
				literal: t,
				kind:    literalKind,
			}, newCursor, true
		}
	}

	return nil, initialCursor, false
}

func parseInsertStatement(tokens []*token, initialCursor uint, delimiter token) (*InsertStatement, uint, bool) {
	cursor := initialCursor
	if !expectToken(tokens, cursor, tokenFromKeyword(INSERT)) {
		return nil, initialCursor, false
	}
	cursor++
	if !expectToken(tokens, cursor, tokenFromKeyword(INTO)) {
		return nil, initialCursor, false
	}
	cursor++

	table, newCursor, ok := parseToken(tokens, cursor, IDENTIFIER)
	if !ok {
		helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	if !expectToken(tokens, cursor, tokenFromKeyword(VALUES)) {
		helpMessage(tokens, cursor, "Expected VALUES")
		return nil, initialCursor, false
	}
	cursor++

	if !expectToken(tokens, cursor, tokenFromSymbol(LEFTPAREN)) {
		helpMessage(tokens, cursor, "Expected left paren")
		return nil, initialCursor, false
	}
	cursor++

	values, newCursor, ok := parseExpressions(tokens, cursor, []token{tokenFromSymbol(RIGHTPAREN)})
	if !ok {
		return nil, initialCursor, false
	}
	cursor = newCursor

	if !expectToken(tokens, cursor, tokenFromSymbol(RIGHTPAREN)) {
		helpMessage(tokens, cursor, "Expected right paren")
		return nil, initialCursor, false
	}
	cursor++

	return &InsertStatement{
		table:  *table,
		values: values,
	}, cursor, true
}
