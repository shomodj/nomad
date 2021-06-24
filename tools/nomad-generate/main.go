package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"
)

var g *Generator
var excludedFieldFlags stringSliceFlag
var typeNameFlags stringSliceFlag
var methodFlags stringSliceFlag
var packageName string

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return fmt.Sprintf("%s", *s)
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	run()
}

func run() {
	flag.Var(&excludedFieldFlags, "exclude", "list of Fields to exclude from Copy")
	flag.Var(&typeNameFlags, "type", "types for which to generate Copy methodFlags")
	flag.Var(&methodFlags, "method", "methodFlags to generate - defaults to all")
	flag.StringVar(&packageName, "packageName", "./","The source dir to target")
	flag.Parse()

	if len(typeNameFlags) == 0 {
		fmt.Println("at least one -type flag needed to generate Copy")
		os.Exit(2)
	}

	g = &Generator{
		typeSpecs: map[string]*TypeSpecNode{},
	}

	var err error
	var pkgs []*packages.Package
	if pkgs, err = loadPackages(); err != nil {
		fmt.Println(fmt.Sprintf("error loading packages: %v", err))
		os.Exit(2)
	}

	if err = parsePackages(pkgs);  err != nil {
		fmt.Println(fmt.Sprintf("error parsing packages: %v", err))
		os.Exit(2)
	}

	if err = g.analyze(); err != nil {
		fmt.Println(fmt.Sprintf("error analyzing: %v", err))
		os.Exit(2)
	}

	if err = g.generate(); err != nil {
		fmt.Println(fmt.Sprintf("error generating: %v", err))
		os.Exit(2)
	}
}

func loadPackages() ([]*packages.Package, error) {
	const loadMode = packages.NeedName |
		packages.NeedFiles |
		packages.NeedCompiledGoFiles |
		packages.NeedImports |
		packages.NeedDeps |
		packages.NeedExportsFile |
		packages.NeedTypes |
		packages.NeedSyntax |
		packages.NeedTypesInfo |
		packages.NeedTypesSizes |
		packages.NeedModule

	cfg := &packages.Config{Mode: loadMode}
	pkgs, err := packages.Load(cfg, packageName)
	return pkgs, err
}

func parsePackages(pkgs []*packages.Package) error {
	for _, pkg := range pkgs {

		if len(pkg.Errors) > 0 {
			return pkg.Errors[0]
		}

		for _, goFile := range pkg.GoFiles {
			// Create the AST by parsing src.
			fileSet := token.NewFileSet() // positions are relative to fset
			file, err := parser.ParseFile(fileSet, goFile, nil, 0)
			if err != nil {
				fmt.Printf("could not parse file: %v\n", err)
				os.Exit(2)
			}

			g.files = append(g.files, file)

			for _, node := range file.Decls {
				switch node.(type) {

				case *ast.GenDecl:
					genDecl := node.(*ast.GenDecl)
					for _, spec := range genDecl.Specs {
						switch spec.(type) {
						case *ast.TypeSpec:
							typeSpec := spec.(*ast.TypeSpec)

							switch typeSpec.Type.(type) {
							case *ast.StructType:
								if isTarget(typeSpec.Name.Name) {
									t := &TargetType{Name: typeSpec.Name.Name}
									g.Targets = append(g.Targets, t)
									ast.Inspect(file, t.visitFields)
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func isTarget(name string) bool {
	for _, typeName := range typeNameFlags {
		if name == typeName { return true}
	}
	return false
}

// Generator holds the state of the analysis. Primarily used to buffer
// the output for format.Source.
type Generator struct {
	files      []*ast.File
	Targets   []*TargetType
	typeSpecs map[string]*TypeSpecNode
}

func (g *Generator) generate() error {
	var err error
	if err = g.render("copy"); err != nil {
		fmt.Printf("could not render copy: %v\n", err)
	}

	if err = g.render("equals"); err != nil {
		return errors.New(fmt.Sprintf("generate.equals: %v", err))
	}

	//if err = g.render("diff"); err != nil {
	//	fmt.Printf("could not render diff: %v\n", err)
	//}
	//
	//if err = g.render("merge"); err != nil {
	//	fmt.Printf("could not render merge: %v\n", err)
	//}

	return nil
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

	// TODO: replace ioutil
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

func (t *TargetType) hasMethod(methodName string) bool {
	for _, method := range t.Methods() {
		if strings.ToLower(method) == "all" { return true }
		if strings.ToLower(methodName) == strings.ToLower(method) { return true }
	}
	return false
}

func (t *TargetType) IsCopy() bool {
	return t.hasMethod("copy")
}

func (t *TargetType) IsEquals() bool {
	return t.hasMethod("equals")
}

func (t *TargetType) IsDiff() bool {
	return t.hasMethod("diff")
}

func (t *TargetType) IsMerge() bool {
	return t.hasMethod("merge")
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
