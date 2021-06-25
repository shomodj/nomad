package main

import (
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyze_Copy(t *testing.T) {
	require := require.New(t)

	g := &Generator{
		packageDir:     "../../nomad/structs",
		typeNames:      []string{"Job"},
		methods:        []string{"Job.Copy"},
		excludedFields: []string{"Payload"},
		typeSpecs:      map[string]*TypeSpecNode{},
	}

	pkgs, err := g.loadPackages()
	require.NoError(err)

	err = g.parsePackages(pkgs)
	require.NoError(err)

	g.analyze()

	require.True(g.typeSpecs["Job"].isCopier())
	require.Len(g.typeSpecs["Job"].fields, 3) // 3 pointer fields

	got := func(typespec string) bool {
		return g.typeSpecs[typespec].isCopier()
	}

	require.True(got("Multiregion"), "Multiregion has pointer and array fields")
	require.True(got("PeriodicConfig"), "PeriodicConfig has a pointer field")
	require.True(got("ParameterizedJobConfig"), "ParameterizedJobConfig has array fields")
	require.True(got("UpdateStrategy"), "UpdateStrategy has a Copy method")

	require.False(got("TaskGroupSummary"), "TaskGroupSummary has only primitive fields")
}

func TestParse_FieldResolveType(t *testing.T) {
	g := &Generator{
		packageDir:     "../../nomad/structs",
		typeNames:      []string{"Job"},
		methods:        []string{"Job.All"},
		excludedFields: []string{"Job.Payload"},
		typeSpecs:      map[string]*TypeSpecNode{},
	}

	var jobTarget *TargetType

	for _, target := range g.Targets {
		if target.Name == "Job" {
			jobTarget = target
			break
		}
	}

	if jobTarget == nil {
		log.Fatal("unable to locate job target")
	}

	for _, field := range jobTarget.Fields {
		t.Log(fmt.Sprintf("%s: %s", field.Name, field.TypeName))
	}

	fmt.Println("TestFieldResolveType complete")
}
