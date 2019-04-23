package main

import (
	"context"
	"github.com/evilsocket/sum/proto"
	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/parser"
	log "github.com/sirupsen/logrus"
	. "github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestCreateOracle(t *testing.T) {
	ms := NewMuxService([]*NodeInfo{})

	arg := &sum.Oracle{}
	arg.Name = "alakazam"
	arg.Code = `
function findSimilar(id, threshold) {
    var v = records.Find(id);
    if( v.IsNull() == true ) {
        return ctx.Error("Vector " + id + " not found.");
    }

    var results = {};
    records.AllBut(v).forEach(function(record){
        var similarity = v.Cosine(record);
        if( similarity >= threshold ) {
           results[record.ID] = similarity
        }
    });

    return results;
}`

	resp, err := ms.CreateOracle(context.Background(), arg)
	Nil(t, err)
	True(t, resp.Success)
}

func TestAstRaccoon(t *testing.T) {
	code := `
function findSimilar(id, threshold) {
    var v = records.Find(id);
    if( v.IsNull() == true ) {
        return ctx.Error("Vector " + id + " not found.");
    }

    var results = {};
    records.AllBut(v).forEach(function(record){
        var similarity = v.Cosine(record);
        if( similarity >= threshold ) {
           results[record.ID] = similarity
        }
    });

	var x = record.Find(id);

    return results;
}`

	program, err := parser.ParseFile(nil, "", code, 0)
	Nil(t, err)
	Equal(t, 1, len(program.DeclarationList))

	raccoon := NewAstRaccoon(code, program.DeclarationList[0].(*ast.FunctionDeclaration).Function, nil)

	r := &sum.Record{Id: 1, Meta: map[string]string{"key": "value"}, Data: []float64{0.1, 0.2, 0.3}}
	newCode, err := raccoon.PatchCode([]*sum.Record{r, nil})
	Nil(t, err)

	expected := strings.Replace(code, "records.Find(id)", "records.New('eJziYBSSmDUTBHbaQ+iT9sZgcNleioeLOTu1Uoi1LDGnNBUQAAD//17vD1Y=')", -1)
	Equal(t, expected, newCode)
}
