package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerate_Equals(t *testing.T) {
	require := require.New(t)

	g := &Generator{
		typeNames:      []string{"Job"},
		packageDir:     "../../nomad/structs",
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
	err = g.write(&buf, "./structs.equals.tmpl", g)
	require.NoError(err)

	formatted := g.format(buf.Bytes())
	got := string(formatted)

	require.Contains(got, "if j.ID != other.ID {")
	require.Contains(got, "if j.Dispatched != other.Dispatched {")

	// TODO: is this "&" right?
	require.Contains(got, "if !j.Multiregion.Equals(&other.Multiregion) {")

	// TODO: this is a struct, is this right?
	require.Contains(got, "if j.Update != other.Update {")

	// TODO: missing this
	// require.Contains(got, `
	// for i, v := range j.Datacenters {
	// 	if v != other.Datacenters[i] {
	// 		return false
	// `)

	// TODO: missing this
	// require.Contains(got, `
	// for i, v := range j.TaskGroups {
	// 	if !v.Equals(other.TaskGroups[i]) {
	// 		return false
	// `)
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
