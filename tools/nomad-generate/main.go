package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
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

	g = &Generator{
		typeSpecs: map[string]*TypeSpecNode{},
	}
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

	g.analyze()
	g.generate()
}

// Generator holds the state of the analysis. Primarily used to buffer
// the output for format.Source.
type Generator struct {
	file      *ast.File
	Targets   []*TargetType
	typeSpecs map[string]*TypeSpecNode
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
	if err = g.render("copy"); err != nil {
		fmt.Printf("could not render copy: %v\n", err)
	}

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

func (g *Generator) render(targetFunc string) error {
	var err error
	targetFileName := fmt.Sprintf("../../nomad/structs/structs.%s.go", targetFunc)

	var buf bytes.Buffer
	err = g.write(&buf, fmt.Sprintf("./structs.%s.tmpl", targetFunc), g)
	if err != nil {
		return err
	}

	formatted := g.format(buf.Bytes())

	err = ioutil.WriteFile(targetFileName, formatted, 0744)
	if err != nil {
		return err
	}

	return nil
}

func (g *Generator) write(w io.Writer, fileName string, data interface{}) error {
	tmpl, err := template.ParseFiles(fileName)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}

func (g *Generator) format(buf []byte) []byte {
	src, err := format.Source(buf)
	if err != nil {
		fmt.Printf("invalid Go generated: %s\n", err) // should never happen
		return buf
	}
	return src
}

type TargetField struct {
	Name     string
	Field    *ast.Field
	TypeName string

	KeyType   *TargetField // the type of a map key
	ValueType *TargetField // the type of a map or array value

	isCopier bool // does this type implement Copy
}

func (f *TargetField) IsPrimitive() bool {
	return !(f.IsPointer() || f.IsStruct() || f.IsArray() || f.IsMap())
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

func (f *TargetField) IsMap() bool {
	return f.TypeName == "map"
}

func (f *TargetField) IsCopier() bool {
	return f.isCopier
}

func (f *TargetField) resolveType(node ast.Node) bool {
	if len(f.TypeName) < 1 {
		switch node.(type) {
		case *ast.Field:
			if node.(*ast.Field).Names[0].Name == f.Name {
				switch t := node.(*ast.Field).Type.(type) {
				case *ast.Ident:
					f.TypeName = node.(*ast.Field).Type.(*ast.Ident).Name
					// For direct struct references (not pointers) the type
					// Name will be returned so we correct it here.
					if !f.IsPrimitive() {
						f.TypeName = "struct"
					}
				case *ast.ArrayType:
					f.TypeName = "array"

					var elemTypeName string
					var ident string

					expr, ok := t.Elt.(*ast.StarExpr)
					if ok {
						ident = expr.X.(*ast.Ident).Name
						elemTypeName = "*" + ident
					} else {
						ident = t.Elt.(*ast.Ident).Name
						elemTypeName = ident
					}

					ts := g.typeSpecs[ident]

					f.ValueType = &TargetField{
						TypeName: elemTypeName,
						isCopier: ts != nil && ts.isCopier(),
					}

				case *ast.MapType:
					f.TypeName = "map"

					var valueTypeName string
					var ident string

					expr, ok := t.Value.(*ast.StarExpr)
					if ok {
						ident = expr.X.(*ast.Ident).Name
						valueTypeName = "*" + ident
					} else {
						ident = t.Value.(*ast.Ident).Name
						valueTypeName = ident
					}

					ts := g.typeSpecs[ident]
					f.ValueType = &TargetField{
						TypeName: valueTypeName,
						isCopier: ts != nil && ts.isCopier(),
					}
					f.KeyType = &TargetField{
						TypeName: t.Key.(*ast.Ident).Name,
					}

				case *ast.StructType:
					f.TypeName = "struct"
					// TODO: where can we get the Ident from?
					//ts := g.typeSpecs[ident]
					//f.isCopier = ts != nil && ts.isCopier()

				case *ast.StarExpr:
					f.TypeName = "pointer"
					ident := t.X.(*ast.Ident).Name
					ts := g.typeSpecs[ident]
					f.isCopier = ts != nil && ts.isCopier()
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

func (t *TargetType) Methods() []string {
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

func (t *TargetType) ExcludedFields() []string {
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
