package main

import (
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {

	g := &Generator{
		typeNames:      []string{"Job"},
		packageDir:     "../../nomad/structs",
		methods:        []string{"Job.All"},
		excludedFields: []string{"Job.Payload"},
		typeSpecs:      map[string]*TypeSpecNode{},
	}
	err := run(g)
	require.NoError(t, err)
}

func TestFieldResolveType(t *testing.T) {
	g := &Generator{
		typeNames:      []string{"Job"},
		packageDir:     "github.com/hashicorp/nomad/nomad/structs",
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
