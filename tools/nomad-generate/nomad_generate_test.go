package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerate_All(t *testing.T) {
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

func TestGenerate_Copy(t *testing.T) {
	require := require.New(t)

	g := &Generator{
		typeNames:      []string{"Job"},
		packageDir:     "github.com/hashicorp/nomad/nomad/structs",
		methods:        []string{"Job.All"},
		excludedFields: []string{"Job.Payload"},
		typeSpecs:      map[string]*TypeSpecNode{},
	}

	pkgs, err := g.loadPackages()
	require.NoError(err)

	err = g.parsePackages(pkgs)
	require.NoError(err)

	err = g.analyze()
	require.NoError(err)

	// this is g.render without writing to disk
	var buf bytes.Buffer
	err = g.write(&buf, "./structs.copy.tmpl", g)
	require.NoError(err)

	formatted := g.format(buf.Bytes())
	got := string(formatted)

	require.Contains(got, "xx.Multiregion = j.Multiregion.Copy()")
	require.Contains(got, "xx.Meta = helper.CopyMapStringString(j.Meta)")
	require.Contains(got, `
	xx.Affinities = make([]*Affinity, len(j.Affinities))
	for _, v := range j.Affinities {
		xx.Affinities = append(xx.Affinities, v.Copy())
	}
`)

}
