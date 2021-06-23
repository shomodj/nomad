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

var excludedFieldFlags stringSliceFlag
var typeNameFlags stringSliceFlag
var methodFlags stringSliceFlag

func main() {
	run(os.Args)
}

func run(args []string) {
	flag.Var(&excludedFieldFlags, "exclude", "list of fields to exclude from Copy")
	flag.Var(&typeNameFlags, "type", "types for which to generate Copy methodFlags")
	flag.Var(&methodFlags, "method", "methodFlags to generate - defaults to all")
	flag.Parse()

	if len(typeNameFlags) == 0 {
		fmt.Println("at least one -type flag needed to generate Copy")
		os.Exit(2)
	}

	// TODO: replace all this filepathery
	fileName := os.Getenv("GOFILE")
	if fileName == "" {
		fmt.Println("GOFILE is unset")
		os.Exit(2)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("could not get package working directory: %v\n", err)
		os.Exit(2)
	}
	cwd = strings.TrimSuffix(cwd, "/tools/nomad-generate-equals") + "/nomad/structs"

	filePath := filepath.Join(cwd, fileName)

	g := &Generator{}
	g.parseFile(filePath)

	fmt.Printf("cwd: %s\n", cwd)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GO") {
			fmt.Printf(" %s\n", kv)
		}
	}

	g.generate()
}

// Generator holds the state of the analysis. Primarily used to buffer
// the output for format.Source.
type Generator struct {
	buf            bytes.Buffer // Accumulated output.
	file           *ast.File
	targets 	  []TargetType
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

func (g *Generator) generate() {
	for _, typeName := range typeNameFlags {
		t := &TargetType{Name: typeName}
		if g.file != nil {
			ast.Inspect(g.file, t.gatherFields)

			for _, methodName := range t.Methods() {
				if methodName == "equals" {
					g.generateEquals(t)
				}
			}
		}
	}
}

func (g *Generator) generateEquals(t *TargetType) {
	fmt.Printf("generating Equals for %s\n", t.Name)

	if len(t.fields) > 0 {
		txt := fmt.Sprintf(
			equalsTmpl,
			t.Abbr(),
			t.Name,
			t.Name,
			t.GenEqualsForValues())
		fmt.Println(txt)
		fmt.Println(txt)
	}
}

type TargetType struct {
	Name string // Name of the type we're generating methods for
	methods []string
	excludedFields []string
	fields []string
}

func (t *TargetType) Abbr() string {
	return strings.ToLower(string(t.Name[0]))
}

func (t *TargetType) Methods() [] string {
	if t.methods == nil {
		var m []string
		for _, method := range methodFlags {
			if strings.Index(method, t.Name) == -1 {
				m = append(m, strings.TrimPrefix(fmt.Sprintf("%s.", t.Name), method))
			}
		}

		if len(m) > 0 {
			t.methods = m
		} else {
			t.methods = make([]string, 0)
		}

	}
	return t.methods
}

func (t *TargetType) ExcludedFields() [] string {
	if t.excludedFields == nil {
		var e []string
		for _, excludedField := range excludedFieldFlags {
			if strings.Index(excludedField, t.Name) == -1 {
				e = append(e, strings.TrimPrefix(fmt.Sprintf("%s.", t.Name), excludedField))
			}
		}

		if len(e) > 0 {
			t.excludedFields = e
		} else {
			t.excludedFields = make([]string, 0)
		}

	}
	return t.excludedFields
}

func (t *TargetType) GenEqualsForValues() string {
	builder := strings.Builder{}

	for _, field := range t.fields {
		builder.WriteString(fmt.Sprintf(
			"\tif %s.%s != instance.%s return false\n",
			t.Abbr(),
			field,
			field))
	}

	return builder.String()
}


func (t *TargetType) gatherFields(node ast.Node) bool {
	var s string
	switch node.(type) {
	case *ast.TypeSpec:
		typeSpec := node.(*ast.TypeSpec)
		if typeSpec.Name.Name == t.Name {
			s = typeSpec.Name.Name
			fmt.Printf("%#v\n", typeSpec.Type)
			expr := typeSpec.Type.(*ast.StructType)
			for _, field := range expr.Fields.List {
				for _, exclude := range t.ExcludedFields() {
					if exclude == field.Names[0].Name {
						break
					}
				}
				t.fields = append(t.fields, field.Names[0].Name)
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
// and type Name x2. TODO: avoid passing type Name 2x
const equalsTmpl = `func (%s *%s) Equals(instance %s) bool {
%s
	return true
}
`