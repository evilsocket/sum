package master

import (
	"errors"
	"fmt"
	"github.com/evilsocket/sum/node/wrapper"

	. "github.com/evilsocket/sum/proto"

	"github.com/evilsocket/islazy/log"

	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/parser"
	"strings"
)

// used to mark a record resolution as failed
var recordNotFound = &Record{}

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

// Create a new astRaccoon
func NewAstRaccoon(source string) (*astRaccoon, error) {
	function, mergerFunction, err := parseAst(source)
	if err != nil {
		return nil, err
	}

	a := &astRaccoon{
		src:                source[:],
		parameters:         function.ParameterList.List,
		callNodes:          make([]*ast.CallExpression, 0),
		parametersToLookup: make(map[int]bool),
		MergerFunction:     mergerFunction,
	}
	ast.Walk(a, function.Body)
	return a, nil
}

// Parse the given JS code to extract first and merger functions
func parseAst(code string) (oracleFunction, mergerFunction *ast.FunctionLiteral, err error) {
	functionList := make([]*ast.FunctionLiteral, 0)

	program, err := parser.ParseFile(nil, "", code, 0)
	if err != nil {
		return nil, nil, err
	}

	for _, d := range program.DeclarationList {
		if fd, ok := d.(*ast.FunctionDeclaration); ok {
			functionList = append(functionList, fd.Function)
		}
	}

	if len(functionList) == 0 {
		return nil, nil, errors.New("no function provided")
	}

	oracleFunction = functionList[0]

	// search for a merger function
	for _, decl := range functionList {
		if decl == oracleFunction {
			continue
		}
		if !strings.HasPrefix(decl.Name.Name, "merge") {
			continue
		}

		if len(decl.ParameterList.List) != 1 {
			log.Warning("Function %s is not a merger function as it does not take 1 argument", decl.Name.Name)
			continue
		}

		mergerFunction = decl
		break
	}
	return
}

// Patch the code managed by this astRaccoon with the given records.
// Records are positional, the index identify which parameter has been resolved to that record.
func (a *astRaccoon) PatchCode(records []*Record) (newCode string, err error) {
	var compressed = make([]string, len(records))
	shift := file.Idx(0)
	newCode = a.src

	// lazy compression
	getCompressed := func(i int) (string, error) {
		if compressed[i] == "" && records[i] != nil {
			var err error
			var r = records[i]

			if r == recordNotFound {
				r = nil
			}

			compressed[i], err = wrapper.RecordToCompressedText(r)
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

// Get the source code of a given node
func (a *astRaccoon) getSource(n ast.Node) string {
	idx0 := n.Idx0() - 1
	idx1 := n.Idx1() - 1
	return a.src[idx0:idx1]
}

// Analyse a node of the AST and determine if it must be patched or not
func (a *astRaccoon) Enter(n ast.Node) ast.Visitor {
	if len(a.parameters) == 0 {
		return nil
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

// Needed to implement the otto.ast.Visitor interface
func (a *astRaccoon) Exit(n ast.Node) {}

// Check if the given index corresponds to a parameter that is used to lookup a record.
// i.e. if the i-th parameter is used as argument for `records.Find`
func (a *astRaccoon) IsParameterPositionARecordLookup(i int) bool {
	return a.parametersToLookup[i]
}

// Return an Oracle representing this astRaccoon
func (a *astRaccoon) AsOracle() *Oracle {
	return &Oracle{Id: a.ID, Name: a.Name, Code: a.src}
}

// Check if the given oracle is equal to the one managed by this raccoon
func (a *astRaccoon) IsEqualTo(oracle *Oracle) bool {
	return oracle.Name == a.Name && oracle.Code == a.src
}
