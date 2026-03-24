package main

import (
	"fmt"
	"log"

	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
)

func main() {
	program, err := parser.ParseFile(nil, "file.js", nil, 0)
	if err != nil {
		log.Fatal(err)
	}
	all_delcarations := program.DeclarationList
	fmt.Printf("%+v\n", all_delcarations)

	x := ast.Program{}
	_ = x

	fmt.Printf("body: %+v\n", program.Body)
	// So the nody identifies each node in the js
	// file, It could be the number of independent whole statements
	// a 'Function' is considered as a Node
	// 'import' statement is also a node
	// 'if' statement has the > Consequent, Alternative fields
	for _, st := range program.Body {
		fmt.Printf("%+v\n", st)
		fmt.Printf("%+v  || %+v\n\n", st.Idx0(), st.Idx1())
	}
}
