package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return fmt.Sprintf("%s", *s)
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

var excludedFields stringSliceFlag
var typeNames stringSliceFlag

func main() {
	flag.Var(&excludedFields, "exclude", "list of fields to exclude from Copy")
	flag.Var(&typeNames, "type", "types for which to generate Copy methods")
	flag.Parse()

	if len(typeNames) == 0 {
		fmt.Println("at least one -type flag needed to generate Copy")
		os.Exit(2)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("could not get package working directory: %v\n", err)
		os.Exit(2)
	}
	fileName := os.Getenv("GOFILE")
	if fileName == "" {
		fmt.Println("GOFILE is unset")
		os.Exit(2)
	}

	exclusions := map[string]struct{}{}
	for _, field := range excludedFields {
		exclusions[field] = struct{}{}
	}

	g := &Generator{
		excludedFields: exclusions,
		typeSpecs:      map[string]*TypeSpecNode{},
	}
	g.analyzeFile(filepath.Join(cwd, fileName))

	for _, typeName := range typeNames {
		// we already know these types need Copy to be copied, because the
		// user asked us to generate their Copy methods!
		g.typeSpecs[typeName].setIsCopier()
	}

	for _, typeName := range typeNames {
		g.generate(typeName)
	}

	// TODO: write buffer to file instead
	// TODO: need to run goimports on results too
	out := g.format()
	fmt.Println("package", os.Getenv("GOPACKAGE"))
	fmt.Println(string(out))
}

// Generator holds the state of the analysis. Primarily used to buffer
// the output for format.Source.
type Generator struct {
	buf            bytes.Buffer // Accumulated output.
	file           *ast.File
	excludedFields map[string]struct{}
	typeSpecs      map[string]*TypeSpecNode
}

func (g *Generator) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

func (g *Generator) analyzeFile(filepath string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	g.file = f
	if g.file != nil {
		ast.Inspect(g.file, g.makeGraph)
		ast.Inspect(g.file, g.analyzeDecl)
	}
	return nil
}

func (g *Generator) generate(typeName string) {
	if g.file != nil {
		ct := &CopyType{
			g:        g,
			name:     typeName,
			recv:     strings.ToLower(typeName[:1]),
			excluded: g.excludedFields,
		}
		ast.Inspect(g.file, ct.genCopy)
	}
}

// TypeSpecNode is used to create a tree of typespecs and track if they
// implement (or need to implement) the Copy method.
type TypeSpecNode struct {
	name           string
	fields         map[string]*TypeSpecNode
	parents        map[string]*TypeSpecNode
	implementsCopy bool
}

// setIsCopier sets this type as Copy and all of its parents as well
func (t *TypeSpecNode) setIsCopier() {
	t.implementsCopy = true
	for _, p := range t.parents {
		p.implementsCopy = true
	}
}

func (t *TypeSpecNode) isCopier() bool {
	if t == nil {
		return false
	}
	return t.implementsCopy
}

func (g *Generator) makeGraph(node ast.Node) bool {
	switch t := node.(type) {
	case *ast.TypeSpec:
		expr, ok := t.Type.(*ast.StructType)
		if !ok {
			return true
		}
		var ts *TypeSpecNode
		typeName := t.Name.Name
		ts, ok = g.typeSpecs[typeName]
		if !ok {
			ts = &TypeSpecNode{
				name:    typeName,
				fields:  map[string]*TypeSpecNode{},
				parents: map[string]*TypeSpecNode{},
			}
			g.typeSpecs[typeName] = ts
		}
		for _, field := range expr.Fields.List {
			switch expr := field.Type.(type) {
			case *ast.StarExpr:
				ident, ok := expr.X.(*ast.Ident)
				if ok {
					fieldTs, ok := g.typeSpecs[ident.Name]
					if !ok {
						fieldTs = &TypeSpecNode{
							name:    ident.Name,
							fields:  map[string]*TypeSpecNode{},
							parents: map[string]*TypeSpecNode{},
						}
					}
					ts.fields[ident.Name] = fieldTs
					fieldTs.parents[typeName] = ts
					g.typeSpecs[ident.Name] = fieldTs
				}
			}
		}

	}
	return true
}

func (g *Generator) analyzeDecl(node ast.Node) bool {
	switch t := node.(type) {
	case *ast.TypeSpec:
		g.needsCopyMethod(t)
	case *ast.FuncDecl:
		// if we find a Copy method, cache it as one we've seen
		if t.Recv != nil && t.Name.Name == "Copy" {
			var methodRecv string
			if stex, ok := t.Recv.List[0].Type.(*ast.StarExpr); ok {
				methodRecv = stex.X.(*ast.Ident).Name
			} else if id, ok := t.Recv.List[0].Type.(*ast.Ident); ok {
				methodRecv = id.Name
			}
			ts, ok := g.typeSpecs[methodRecv]
			if ok {
				ts.setIsCopier()
			}
		}
	}
	return true
}

func (g *Generator) needsCopyMethod(t *ast.TypeSpec) bool {
	name := t.Name.Name

	ts, ok := g.typeSpecs[name]
	if !ok {
		return false // ignore interfaces TODO?
	}

	// check if this has been set by one of its children previously
	if ts.isCopier() {
		return true
	}
	for _, field := range ts.fields {
		if field.isCopier() {
			ts.setIsCopier()
			return true
		}
	}

	expr, ok := t.Type.(*ast.StructType)
	if !ok {
		return false
	}
	for _, field := range expr.Fields.List {
		switch expr := field.Type.(type) {
		case *ast.StarExpr:
			i, ok := expr.X.(*ast.Ident)
			if ok {
				child, ok := g.typeSpecs[i.Name]
				if ok {
					if child.isCopier() {
						ts.setIsCopier()
						return true
					}
				}
			}
		case *ast.StructType:
			return false // TODO: how do we get the type name here?
		case *ast.MapType, *ast.ArrayType:
			ts.setIsCopier()
			return true
		}
	}
	return false
}

func (g *Generator) format() []byte {
	src, err := format.Source(g.buf.Bytes())
	if err != nil {
		fmt.Printf("invalid Go generated: %s\n", err) // should never happen
		return g.buf.Bytes()
	}
	return src
}

type CopyType struct {
	g        *Generator
	name     string              // name of the type we're generating Copy for
	recv     string              // identifier of the receiver we're generating Copy for
	excluded map[string]struct{} // fields we should ignore

	// accumulated objects, each of which has its own copying behavior.
	// primitive fields are excluded because we do that at the top level
	blocks []string
}

func (ct *CopyType) genCopy(node ast.Node) bool {
	switch t := node.(type) {
	case *ast.TypeSpec:
		if t.Name.Name == ct.name {
			expr := t.Type.(*ast.StructType)
			for _, field := range expr.Fields.List {
				ct.genField(field) // generate block for each field
			}
			ct.genCopyMethod() // render the template to buffer
			return false       // we're done!
		}
	}
	return true
}

func (ct *CopyType) genField(field *ast.Field) {
	name := field.Names[0].Name
	if _, exclude := ct.excluded[ct.name+"."+name]; exclude {
		return
	}

	switch f := field.Type.(type) {
	case *ast.InterfaceType, *ast.FuncType:
		// TODO: return an error? just ignore?
		fmt.Printf("  cannot copy interface, channel, or function type: %s\n", name)
	case *ast.StarExpr:
		ct.genPointerField(f, name)
	case *ast.StructType:
		ct.genStructField(f, name)
	case *ast.MapType:
		ct.genMapField(f, name)
	case *ast.ArrayType:
		ct.genArrayField(f, name)
	default:
		// primitive types have already been copied
	}
}

const newRecv = "xx"

func (ct *CopyType) genCopyMethod() {

	data := struct {
		Old      string
		New      string
		TypeName string
		Blocks   []string
	}{
		Old:      ct.recv,
		New:      newRecv,
		TypeName: ct.name,
		Blocks:   ct.blocks,
	}

	decl := `
func ({{.Old}} *{{.TypeName}}) Copy() *{{.TypeName}} {
	if {{ .Old }} == nil {
		return nil
	}
	{{.New}} := new({{.TypeName}})
	*{{.New}} = *{{.Old}}
{{ range .Blocks }}{{.}}{{end}}

	return {{.New}}
}

`
	t := template.Must(template.New(ct.name).Parse(decl))
	t.Execute(&ct.g.buf, data) // TODO: send to the buffer
}

func (ct *CopyType) genPointerField(field *ast.StarExpr, name string) {

	typeName := field.X.(*ast.Ident).Name
	var buf bytes.Buffer
	var block string
	data := struct {
		Old      string
		New      string
		Name     string
		TypeName string
	}{
		Old:      fmt.Sprintf("%s.%s", ct.recv, name),
		New:      fmt.Sprintf("%s.%s", newRecv, name),
		Name:     name,
		TypeName: typeName,
	}

	ts := ct.g.typeSpecs[typeName]
	if ts.isCopier() {
		block = `
{{.New}} = {{.Old}}.Copy()
`
	} else {
		block = `
if {{.Old}} == nil {
	{{.New}} = nil
} else {
	{{.New}} = new({{.TypeName}})
	*{{.New}} = *{{.Old}}
}
`
	}
	t := template.Must(template.New(ct.name + "+" + typeName + "+" + name).Parse(block))
	err := t.Execute(&buf, data)
	if err != nil {
		panic(err)
	}
	ct.blocks = append(ct.blocks, buf.String())
}

func (ct *CopyType) genStructField(field *ast.StructType, name string) {

	typeName := name // TODO: this is wrong! how do we get the type name!?
	var buf bytes.Buffer
	var block string
	data := struct {
		Old      string
		New      string
		Name     string
		TypeName string
	}{
		Old:      fmt.Sprintf("%s.%s", ct.recv, name),
		New:      fmt.Sprintf("%s.%s", newRecv, name),
		Name:     name,
		TypeName: typeName,
	}
	var needsCopy bool
	if t, ok := ct.g.typeSpecs[typeName]; ok {
		needsCopy = t.isCopier()
	}
	if needsCopy {
		block = `
{{.New}} := {{.Old}}.Copy()
`
	} else {
		block = `
if {{.Old}} == nil {
	{{.New}} = nil
} else {
	{{.New}} := new({{.TypeName}})
	*{{.New}} = *{{.Old}}
}
`
	}
	t := template.Must(template.New("").Parse(block))
	t.Execute(&buf, data)
	ct.blocks = append(ct.blocks, buf.String())
}

func (ct *CopyType) genMapField(field *ast.MapType, name string) {
	keyTypeName := field.Key.(*ast.Ident).Name
	var typeName string
	var ts *TypeSpecNode
	expr, ok := field.Value.(*ast.StarExpr)
	if ok {
		ident, ok := expr.X.(*ast.Ident)
		if !ok {
			fmt.Println("no Ident")
			return // TODO: what to do here?
		}
		ts = ct.g.typeSpecs[ident.Name]
		typeName = "*" + ident.Name
	} else {
		ident, ok := field.Value.(*ast.Ident)
		if !ok {
			return // TODO: what to do here?
		}
		ts = ct.g.typeSpecs[ident.Name]
		typeName = ident.Name
	}

	var buf bytes.Buffer
	var block string
	data := struct {
		Old       string
		New       string
		Name      string
		KeyType   string
		ValueType string
	}{
		Old:       fmt.Sprintf("%s.%s", ct.recv, name),
		New:       fmt.Sprintf("%s.%s", newRecv, name),
		Name:      name,
		KeyType:   keyTypeName,
		ValueType: typeName,
	}
	if ts.isCopier() {
		block = `
{{.New}} = map[{{.KeyType}}]{{.ValueType}}{}
for k, v := range {{.Old}} {
	{{.New}}[k] = v.Copy()
}
`
	} else if keyTypeName == "string" && typeName == "string" {
		block = `
{{.New}} = helper.CopyMapStringString({{.Old}})
`
	} else if keyTypeName == "string" && typeName == "int" {
		block = `
{{.New}} = helper.CopyMapStringInt({{.Old}})
`
	} else if keyTypeName == "string" && typeName == "float64" {
		block = `
{{.New}} = helper.CopyMapStringFloat64({{.Old}})
`
	} else {
		block = `
{{.New}} = map[{{.KeyType}}]{{.ValueType}}{}
for k, v := range {{.Old}} {
	{{.New}}[k] = v
}
`
	}

	t := template.Must(template.New(name + "->" + typeName).Parse(block))
	t.Execute(&buf, data)
	ct.blocks = append(ct.blocks, buf.String())
}

func (ct *CopyType) genArrayField(field *ast.ArrayType, name string) {

	var elemTypeName string
	var ts *TypeSpecNode

	expr, ok := field.Elt.(*ast.StarExpr)
	if ok {
		ident, ok := expr.X.(*ast.Ident)
		if !ok {
			fmt.Println("no Ident")
			return // TODO: what to do here?
		}
		ts = ct.g.typeSpecs[ident.Name]
		elemTypeName = ident.Name
	} else {
		ident, ok := field.Elt.(*ast.Ident)
		if !ok {
			return // TODO: what to do here?
		}
		ts = ct.g.typeSpecs[ident.Name]
		elemTypeName = ident.Name
	}

	var buf bytes.Buffer
	var block string
	data := struct {
		Old             string
		New             string
		Name            string
		ElementTypeName string
	}{
		Old:             fmt.Sprintf("%s.%s", ct.recv, name),
		New:             fmt.Sprintf("%s.%s", newRecv, name),
		Name:            name,
		ElementTypeName: elemTypeName,
	}

	if ts.isCopier() {
		block = `
{{.New}} = make([]{{.ElementTypeName}}, len({{.Old}}))
for _, v := range {{.Old}} {
	{{.New}} = append({{.New}}, v.Copy())
}
`
	} else if elemTypeName == "string" {
		block = `
{{.New}} = helper.CopySliceString({{.Old}})
`
	} else if elemTypeName == "int" {
		block = `
{{.New}} = helper.CopySliceInt({{.Old}})
`
	} else {
		block = `
{{.New}} = make([]{{.ElementTypeName}}, len({{.Old}}))
for _, v := range {{.Old}} {
	{{.New}} = append({{.New}}, v)
}
`
	}

	t := template.Must(template.New(elemTypeName).Parse(block))
	t.Execute(&buf, data)
	ct.blocks = append(ct.blocks, buf.String())

}
