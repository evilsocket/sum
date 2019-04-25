package main

import (
	"fmt"
	. "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/wrapper"
	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/file"
	"strings"
)

// JS is garbage, this cute animal is responsible of digging in it.
type astRaccoon struct {
	ID        uint64
	Name      string
	src       string
	callNodes []*ast.CallExpression
	// parameters for the main function
	parameters         []*ast.Identifier
	parametersToLookup map[int]bool
	MergerFunction     *ast.FunctionLiteral
}

func NewAstRaccoon(source string, function, mergerFunction *ast.FunctionLiteral) *astRaccoon {
	a := &astRaccoon{
		src:                source[:],
		parameters:         function.ParameterList.List,
		callNodes:          make([]*ast.CallExpression, 0),
		parametersToLookup: make(map[int]bool),
		MergerFunction:     mergerFunction,
	}
	ast.Walk(a, function.Body)
	return a
}

func (a *astRaccoon) PatchCode(records []*Record) (newCode string, err error) {
	var compressed = make([]string, len(records))
	shift := file.Idx(0)
	newCode = a.src

	// lazy compression
	getCompressed := func(i int) (string, error) {
		if compressed[i] == "" && records[i] != nil {
			var err error
			compressed[i], err = wrapper.RecordToCompressedText(records[i])
			return compressed[i], err
		}
		return compressed[i], nil
	}

	// foreach parameter
	for i, p := range a.parameters {
		// that has been resolved to a valid record
		if records[i] == nil {
			continue
		}
		// foreach node that shall be patched
		for _, n := range a.callNodes {
			// whose argument is the current parameter
			arg := n.ArgumentList[0].(*ast.Identifier).Name
			if arg != p.Name {
				continue
			}

			// generate a compressed representation of the record
			compressedRecord, err := getCompressed(i)
			if err != nil {
				return "", err
			}

			// replace the node with the compressed string of the record
			idx0 := n.Idx0() + shift - 1
			idx1 := n.Idx1() + shift - 1
			newRecord := fmt.Sprintf("records.New('%s')", compressedRecord)
			newCode = newCode[:idx0] + newRecord + newCode[idx1:]
			shift += file.Idx(len(newRecord)) - (idx1 - idx0)
		}
	}

	return
}

func (a *astRaccoon) getSource(n ast.Node) string {
	idx0 := n.Idx0() - 1
	idx1 := n.Idx1() - 1
	return a.src[idx0:idx1]
}

func (a *astRaccoon) Enter(n ast.Node) ast.Visitor {
	if len(a.parameters) == 0 {
		return a
	}

	// search for a call expression with 1 argument
	callExpr, ok := n.(*ast.CallExpression)

	if !ok || len(callExpr.ArgumentList) != 1 {
		return a
	}

	callee := a.getSource(callExpr.Callee)
	callee = strings.Join(strings.Fields(callee), "")

	// that calls "records.Find"

	if callee != "records.Find" {
		return a
	}

	// with an identifier ( variable ) parameter

	arg, ok := callExpr.ArgumentList[0].(*ast.Identifier)

	if !ok {
		return a
	}

	// which is a variable specified as parameter of the function

	ok = false
	for i, p := range a.parameters {
		if p.Name == arg.Name {
			a.parametersToLookup[i] = true
			ok = true
			break
		}
	}
	if !ok {
		return a
	}

	a.callNodes = append(a.callNodes, callExpr)
	return nil // do not descend further into the AST
}

func (a *astRaccoon) Exit(n ast.Node) {}

func (a *astRaccoon) IsParameterPositionARecordLookup(i int) bool {
	return a.parametersToLookup[i]
}

func (a *astRaccoon) AsOracle() *Oracle {
	return &Oracle{Id: a.ID, Name: a.Name, Code: a.src}
}
