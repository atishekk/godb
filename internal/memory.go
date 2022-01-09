package godb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
)

type MemoryCell []byte

func (mc MemoryCell) AsInt() int64 {
	var i int64
	err := binary.Read(bytes.NewBuffer(mc), binary.BigEndian, &i)
	if err != nil {
		panic(err)
	}
	return i
}

func (mc MemoryCell) AsText() string {
	return string(mc)
}

type table struct {
	columns     []string
	columnTypes []ColumnType
	rows        [][]MemoryCell
}

type MemoryBackend struct {
	tables map[string]*table
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		tables: make(map[string]*table),
	}
}

// Implementing the Backend Interface

func (mb *MemoryBackend) CreateTable(crt *CreateStatement) error {
	t := table{}
	mb.tables[crt.name.value] = &t

	if crt.cols == nil {
		return nil
	}

	for _, col := range *crt.cols {
		t.columns = append(t.columns, col.name.value)
		var dt ColumnType
		switch col.datatype.value {
		case "int":
			dt = IntType
		case "text":
			dt = TextType
		default:
			return ErrInvalidDatatype
		}

		t.columnTypes = append(t.columnTypes, dt)
	}
	return nil
}

func (mb *MemoryBackend) Insert(inst *InsertStatement) error {
	table, ok := mb.tables[inst.table.value]
	if !ok {
		return ErrTableDoesNotExist
	}
	if inst.values == nil {
		return nil
	}

	if len(*inst.values) != len(table.columns) {
		return ErrMissingValues
	}

	var row []MemoryCell

	for _, value := range *inst.values {
		if value.kind != literalKind {
			fmt.Println("Skipping non-literal")
			continue
		}
		row = append(row, mb.tokenToCell(value.literal))
	}
	table.rows = append(table.rows, row)
	return nil
}

func (mb *MemoryBackend) tokenToCell(t *token) MemoryCell {
	if t.kind == NUMERIC {
		buf := new(bytes.Buffer)
		i, err := strconv.Atoi(t.value)
		if err != nil {
			panic(err)
		}

		err = binary.Write(buf, binary.BigEndian, int64(i))
		if err != nil {
			panic(err)
		}
		return MemoryCell(buf.Bytes())
	} else if t.kind == STRING {
		return MemoryCell(t.value)
	}
	return nil
}

func (mb *MemoryBackend) Select(slct *SelectStatement) (*Results, error) {
	table, ok := mb.tables[slct.from.value]
	if !ok {
		return nil, ErrTableDoesNotExist
	}

	var results [][]Cell
	var columns []struct {
		Type ColumnType
		Name string
	}

	for i, row := range table.rows {
		var result []Cell
		isFirstRow := i == 0

		for _, exp := range slct.item {
			if exp.kind != literalKind {
				fmt.Println("skipping non-literal kinds, for now")
				continue
			}
			lit := exp.literal
			if lit.kind == IDENTIFIER {
				found := false
				for i, tableCol := range table.columns {
					if tableCol == lit.value {
						if isFirstRow {
							columns = append(columns, struct {
								Type ColumnType
								Name string
							}{
								Type: table.columnTypes[i],
								Name: lit.value,
							})
						}
						result = append(result, row[i])
						found = true
						break
					}
				}
				if !found {
					return nil, ErrColumnDoesNotExist
				}
				continue
			}
			return nil, ErrColumnDoesNotExist
		}
		results = append(results, result)
	}
	return &Results{
		Columns: columns,
		Row:     results,
	}, nil
}
