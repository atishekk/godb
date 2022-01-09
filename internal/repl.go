package godb

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func REPL() {
	mb := NewMemoryBackend()
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("GoDB")
	for {
		fmt.Print("> ")
		text, err := reader.ReadString('\n')
		text = strings.Replace(text, "\n", "", -1)

		ast, err := parse(text)
		if err != nil {
			panic(err)
		}

		for _, stmt := range ast.Statements {
			switch stmt.Kind {
			case CreateStmtKind:
				err := mb.CreateTable(stmt.CreateStatement)
				if err != nil {
					panic(err)
				}
				fmt.Println("ok")
			case InsertStmtKind:
				err := mb.Insert(stmt.InsertStatement)
				if err != nil {
					panic(err)
				}
				fmt.Println("ok")
			case SelectStmtKind:
				results, err := mb.Select(stmt.SelectStatement)
				if err != nil {
					panic(err)
				}
				for _, col := range results.Columns {
					fmt.Printf("| %s ", col.Name)
				}
				fmt.Println("|")

				for i := 0; i < 20; i++ {
					fmt.Printf("=")
				}
				fmt.Println()

				for _, result := range results.Row {
					fmt.Printf("|")

					for i, cell := range result {
						typ := results.Columns[i].Type
						s := ""
						switch typ {
						case IntType:
							s = fmt.Sprintf("%d", cell.AsInt())
						case TextType:
							s = cell.AsText()
						}
						fmt.Printf(" %s | ", s)
					}
					fmt.Println()
				}
				fmt.Println("ok")
			}
		}
	}
}
