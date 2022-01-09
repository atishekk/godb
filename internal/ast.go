package godb

type Ast struct {
	Statements []*Statement
}

type StatementKind uint

const (
	SelectStmtKind StatementKind = iota
	CreateStmtKind
	InsertStmtKind
)

type expressionKind uint

const (
	literalKind expressionKind = iota
)

type expression struct {
	literal *token
	kind    expressionKind
}

// Types of statements

type SelectStatement struct {
	item []*expression
	from token
}

type InsertStatement struct {
	table  token
	values *[]*expression
}

type columnDefinition struct {
	name     token
	datatype token
}

type CreateStatement struct {
	name token
	cols *[]*columnDefinition
}

// Statement TODO: Think of a better way to build this union
type Statement struct {
	SelectStatement *SelectStatement
	CreateStatement *CreateStatement
	InsertStatement *InsertStatement
	Kind            StatementKind
}
