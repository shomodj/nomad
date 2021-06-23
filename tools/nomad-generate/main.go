package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
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
	flag.Var(&excludedFieldFlags, "exclude", "list of Fields to exclude from Copy")
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
	file    *ast.File
	Targets []*TargetType
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
	// Build Targets
	for _, typeName := range typeNameFlags {
		t := &TargetType{Name: typeName}
		g.Targets = append(g.Targets, t)

		if g.file != nil {
			ast.Inspect(g.file, t.visitFields)
		}
	}

	var err error
	//if err = g.render("copy"); err != nil {
	//	fmt.Printf("could not render copy: %v\n", err)
	//}

	if err = g.render("equals"); err != nil {
		fmt.Printf("could not render equals: %v\n", err)
	}

	//if err = g.render("diff"); err != nil {
	//	fmt.Printf("could not render diff: %v\n", err)
	//}
	//
	//if err = g.render("merge"); err != nil {
	//	fmt.Printf("could not render merge: %v\n", err)
	//}
}

func (g * Generator) render(targetFunc string) error {
	var err error
	var file *os.File

	if file, err = os.OpenFile(fmt.Sprintf("../../nomad/structs/structs.%s.go", targetFunc), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666); err != nil {
		return err
	}

	w := bufio.NewWriter(file)

	if err = g.write(w, fmt.Sprintf("./structs.%s.tmpl", targetFunc), g); err != nil {
		return err
	}

	if err = w.Flush(); err != nil {
		return err
	}

	if err = file.Close(); err != nil {
		return err
	}

	return nil
}

func (g * Generator) write(w io.Writer, fileName string, data interface{}) error {
	tmpl, err := template.ParseFiles(fileName)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}

type TargetField struct {
	Name     string
	Field    *ast.Field
	TypeName string
}

func (f *TargetField) IsPrimitive() bool {
	return !(f.IsPointer() || f.IsStruct() || f.IsArray())
}

func (f *TargetField) IsArray() bool {
	return f.TypeName == "array"
}

func (f *TargetField) IsStruct() bool {
	return f.TypeName == "struct"
}

func (f *TargetField) IsPointer() bool {
	return f.TypeName == "pointer"
}

func (f *TargetField) resolveType(node ast.Node) bool {
	if len (f.TypeName) < 1 {
		switch node.(type) {
		case *ast.Field:
			if node.(*ast.Field).Names[0].Name == f.Name {
				switch node.(*ast.Field).Type.(type) {
				case *ast.Ident:
					f.TypeName = node.(*ast.Field).Type.(*ast.Ident).Name
					// For direct struct references (not pointers) the type Name will be returned
					// so we correct it here.
					if !f.IsPrimitive() {
						f.TypeName = "struct"
					}
				case *ast.ArrayType, *ast.MapType:
					f.TypeName = "array"
				case *ast.StructType:
					f.TypeName = "struct"
				case *ast.StarExpr:
					f.TypeName = "pointer"
				}
			}
		default:
			f.TypeName = fmt.Sprintf("%+v", node)
		}
	}
	return true
}

type TargetType struct {
	Name           string // Name of the type we're generating methods for
	methods        []string
	excludedFields []string
	Fields         []*TargetField
}

func (t *TargetType) Abbr() string {
	return strings.ToLower(string(t.Name[0]))
}

func (t *TargetType) Methods() [] string {
	if t.methods == nil {
		var m []string
		for _, method := range methodFlags {
			if strings.Contains(method, t.Name) {
				md := strings.TrimPrefix(method, fmt.Sprintf("%s.", t.Name))
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
			if strings.Index(excludedField, t.Name) > -1 {
				e = append(e, strings.TrimPrefix(excludedField, fmt.Sprintf("%s.", t.Name)))
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

	for _, field := range t.Fields {
		builder.WriteString(fmt.Sprintf(
			"\tif %s.%s != instance.%s return false\n",
			t.Abbr(),
			field.Name,
			field.Name))
	}

	return builder.String()
}

func (t *TargetType) visitFields(node ast.Node) bool {
	switch node.(type) {
	case *ast.TypeSpec:
		typeSpec := node.(*ast.TypeSpec)
		if typeSpec.Name.Name == t.Name {
			expr := typeSpec.Type.(*ast.StructType)
			for _, field := range expr.Fields.List {
				if t.fieldIsExcluded(field.Names[0].Name) {
					continue
				}

				targetField := &TargetField{Name: field.Names[0].Name, Field: field}
				t.Fields = append(t.Fields, targetField)
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