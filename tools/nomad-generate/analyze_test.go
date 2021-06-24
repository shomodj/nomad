package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyze_Copy(t *testing.T) {
	require := require.New(t)

	g := &Generator{
		packageName:    "github.com/hashicorp/nomad/tools/nomad-generate/main",
		typeNames:      []string{"Basket"},
		methods:        []string{"Basket.Copy"},
		excludedFields: []string{"Basket.Exclude"},
		typeSpecs:      map[string]*TypeSpecNode{},
	}

	pkgs, err := g.loadPackages()
	require.NoError(err)

	err = g.parsePackages(pkgs)
	require.NoError(err)

	g.analyze()

	require.True(g.typeSpecs["Basket"].isCopier())
	// require.Len(g.typeSpecs["Basket"].fields, 14) // ?

	require.True(g.typeSpecs["AppleWithReferenceFields"].isCopier())
	require.False(g.typeSpecs["BananaWithOnlyValueFields"].isCopier())
}
