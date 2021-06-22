package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
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

	generateEquals()
}

func generateEquals() {
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

	cwd = strings.TrimSuffix(cwd, "/tools/nomad-generate-equals") + "/nomad/structs"

	g := &Generator{excludedFields: excludedFields}
	g.parseFile(filepath.Join(cwd, fileName))

	fmt.Printf("cwd: %s\n", cwd)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GO") {
			fmt.Printf(" %s\n", kv)
		}
	}

	for _, typeName := range typeNames {
		g.generate(typeName)
	}
}

// Generator holds the state of the analysis. Primarily used to buffer
// the output for format.Source.
type Generator struct {
	buf            bytes.Buffer // Accumulated output.
	file           *ast.File
	excludedFields []string
	values         []string // Accumulator for constant values of that type.
	//	pkg            *Package // Package we are scanning.

}

func (g *Generator) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

func (g *Generator) parseFile(filepath string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	g.file = f
	return nil
}

func (g *Generator) generate(typeName string) {
	fmt.Printf("generating Equals for %s\n", typeName)
	if g.file != nil {
		t := &TargetType{typeName: typeName, excluded: g.excludedFields}
		ast.Inspect(g.file, t.genDecl)
		// I don't think we want to do this.
		g.values = append(g.values, t.fields...)

		if len(t.fields) > 0 {
			txt := fmt.Sprintf(
				equalsTmpl,
				strings.ToLower(string(t.typeName[0])),
				t.typeName,
				t.typeName,
				t.GenEqualsForValues())
			fmt.Println(txt)
			fmt.Println(txt)
		}

	}
}


type TargetType struct {
	typeName string   // name of the type we're generating Copy for
	excluded []string // fields we should ignore
	fields   []string // accumulated objects (TODO: what are we trying to do here?)
}

func (t *TargetType) GenEqualsForValues() string {
	builder := strings.Builder{}

	for _, field := range t.fields {
		builder.WriteString(fmt.Sprintf(
			"\tif %s.%s != instance.%s return false\n",
			strings.ToLower(string(t.typeName[0])),
			field,
			field))
	}

	return builder.String()
}


func (ct *TargetType) genDecl(node ast.Node) bool {
	var s string
	switch node.(type) {
	case *ast.TypeSpec:
		t := node.(*ast.TypeSpec)
		if t.Name.Name == ct.typeName {
			s = t.Name.Name
			fmt.Printf("%#v\n", t.Type)
			expr := t.Type.(*ast.StructType)
			for _, field := range expr.Fields.List {
				for _, exclude := range ct.excluded {
					if exclude == field.Names[0].Name {
						break
					}
				}
				ct.fields = append(ct.fields, field.Names[0].Name)
				fmt.Printf("field: %#v\n", field.Names[0].Name)
			}
		}
	}
	if s != "" {
		fmt.Printf("%s\n", s)
	}
	return true
}

// Equals function template. Expects to be fed lower case first letter of type,
// and type name x2. TODO: avoid passing type name 2x
const equalsTmpl = `func (%s *%s) Equals(instance %s) bool {
%s
	return true
}
`