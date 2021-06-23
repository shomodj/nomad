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

var filePath string
var g *Generator

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
	cwd = strings.TrimSuffix(cwd, "/tools/nomad-generate") + "/nomad/structs"

	filePath = filepath.Join(cwd, fileName)

	g = &Generator{}
	err = g.parseFile(filePath)
	if err != nil {
		fmt.Printf("could not parse file: %v\n", err)
		os.Exit(2)
	}

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
	targets 	  []*TargetType
	//	pkg            *Package // Package we are scanning.
}

func (g *Generator) Printf(format string, args ...interface{}) {
	_, _  = fmt.Fprintf(&g.buf, format, args...)
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
	// Build targets
	for _, typeName := range typeNameFlags {
		t := &TargetType{name: typeName}
		g.targets = append(g.targets, t)

		if g.file != nil {
			ast.Inspect(g.file, t.visitFields)
		}
	}

	err := g.renderTemplates()
	if err != nil {
		fmt.Printf("could not render templates: %v\n", err)
	}
}

func (g *Generator) renderTemplates() error {
	var err error
	if err = g.renderEquals(); err != nil {
		return err
	}

	return nil
}

func (g * Generator) renderEquals() error {

	return nil
}

type TargetField struct {
	name string
	field *ast.Field
	typeName string
}

func (f *TargetField) IsPrimitive() bool {
	return !(f.IsPointer() || f.IsStruct() || !f.IsArray())
}

func (f *TargetField) IsArray() bool {
	return f.typeName == "array"
}

func (f *TargetField) IsStruct() bool {
	return f.typeName == "struct"
}

func (f *TargetField) IsPointer() bool {
	return f.typeName == "pointer"
}

func (f *TargetField) resolveType(node ast.Node) bool {
	//if f.name == "Update" {
	//	fmt.Println(fmt.Sprintf("%+v", node))
	//}
	if len (f.typeName) < 1 {
		switch node.(type) {
		case *ast.Field:
			if node.(*ast.Field).Names[0].Name == f.name {
				switch node.(*ast.Field).Type.(type) {
				case *ast.Ident:
					f.typeName = node.(*ast.Field).Type.(*ast.Ident).Name
					// For direct struct references (not pointers) the type name will be returned
					// so we correct it here.
					if !f.IsPrimitive() {
						f.typeName = "struct"
					}
				case *ast.ArrayType, *ast.MapType:
					f.typeName = "array"
				case *ast.StructType:
					f.typeName = "struct"
				case *ast.StarExpr:
					f.typeName = "pointer"
				}
			}
		default:
			f.typeName = fmt.Sprintf("%+v", node)
		}
	}
	return true
}

type TargetType struct {
	name           string // name of the type we're generating methods for
	methods        []string
	excludedFields []string
	fields         []*TargetField
}

func (t *TargetType) Abbr() string {
	return strings.ToLower(string(t.name[0]))
}

func (t *TargetType) Methods() [] string {
	if t.methods == nil {
		var m []string
		for _, method := range methodFlags {
			if strings.Contains(method, t.name) {
				md := strings.TrimPrefix(method, fmt.Sprintf("%s.", t.name))
				m = append(m, md)
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
			if strings.Index(excludedField, t.name) > -1 {
				e = append(e, strings.TrimPrefix(excludedField, fmt.Sprintf("%s.", t.name)))
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
			field.name,
			field.name))
	}

	return builder.String()
}

func (t *TargetType) visitFields(node ast.Node) bool {
	switch node.(type) {
	case *ast.TypeSpec:
		typeSpec := node.(*ast.TypeSpec)
		if typeSpec.Name.Name == t.name {
			expr := typeSpec.Type.(*ast.StructType)
			for _, field := range expr.Fields.List {
				if field.Names[0].Name == "Stop" {
					fmt.Println("found")
				}

				if t.fieldIsExcluded(field.Names[0].Name) {
					continue
				}

				targetField := &TargetField{name: field.Names[0].Name, field: field}
				t.fields = append(t.fields, targetField)
				ast.Inspect(field, targetField.resolveType)
			}
		}
	}
	return true
}

func (t *TargetType) fieldIsExcluded(name string) bool {

	for _, exclude := range t.ExcludedFields() {
		if exclude == name {
			return true
		}
	}

	return false
}

// Equals function template. Expects to be fed lower case first letter of type,
// and type name x2. TODO: avoid passing type name 2x
const equalsTmpl = `func (%s *%s) Equals(instance %s) bool {
%s
	return true
}
`