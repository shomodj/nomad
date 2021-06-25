package main

import (
	"bytes"
	"embed"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"
)

//go:embed structs.copy.tmpl
var copyTmpl embed.FS

//go:embed structs.equals.tmpl
var equalsTmpl embed.FS

//go:embed structs.diff.tmpl
var diffTmpl embed.FS

//go:embed structs.merge.tmpl
var mergeTmpl embed.FS

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return fmt.Sprintf("%s", *s)
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// Generator holds the state of the analysis. Primarily used to buffer
// the output for format.Source.
type Generator struct {
	packageDir string
	files      []*ast.File
	Targets    map[string]*TargetType
	typeSpecs  map[string]*TypeSpecNode

	typeNames      []string
	methods        []string
	excludedFields []string
}

func main() {
	var excludedFieldFlags stringSliceFlag
	var typeNameFlags stringSliceFlag
	var methodFlags stringSliceFlag
	var packageDir string

	flag.Var(&excludedFieldFlags, "exclude", "list of Fields to exclude from Copy")
	flag.Var(&typeNameFlags, "type", "types for which to generate Copy methodFlags")
	flag.Var(&methodFlags, "method", "methodFlags to generate - defaults to all")
	flag.StringVar(&packageDir, "packageDir", "./", "The source dir to target")
	flag.Parse()

	if len(typeNameFlags) == 0 {
		fmt.Println("at least one -type flag needed to generate Copy")
		os.Exit(2)
	}

	g := &Generator{
		packageDir:     packageDir,
		typeNames:      typeNameFlags,
		methods:        methodFlags,
		excludedFields: excludedFieldFlags,
		typeSpecs:      map[string]*TypeSpecNode{},
	}

	err := run(g)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
}

func run(g *Generator) error {
	var err error
	var pkgs []*packages.Package

	if pkgs, err = g.loadPackages(); err != nil {
		return fmt.Errorf("error loading packages: %v", err)
	}

	if err = g.parsePackages(pkgs); err != nil {
		return fmt.Errorf("error parsing packages: %v", err)
	}

	if len(pkgs) == 0 {
		return fmt.Errorf("did not parse any packages")
	}

	if err = g.analyze(); err != nil {
		return fmt.Errorf("error analyzing: %v", err)
	}

	if len(g.typeSpecs) == 0 {
		return fmt.Errorf("did not analyze any types")
	}

	if err = g.generate(); err != nil {
		return fmt.Errorf("error generating: %v", err)
	}

	return nil
}

func (g *Generator) loadPackages() ([]*packages.Package, error) {
	// TODO: See which of these we really need
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

	cfg := &packages.Config{
		Dir:  g.packageDir, // this is the relative path to the source we want to parse
		Mode: loadMode,
	}

	pkgs, err := packages.Load(cfg, ".") // this pattern means load all go files
	return pkgs, err
}

func (g *Generator) parsePackages(pkgs []*packages.Package) error {
	for _, pkg := range pkgs {
		// TODO: Iterate and compose error
		if len(pkg.Errors) > 0 {
			return pkg.Errors[0]
		}

		for _, goFile := range pkg.GoFiles {
			if err := g.parseGoFile(goFile); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Generator) parseGoFile(goFile string) error {
	// Create the AST by parsing src.
	fileSet := token.NewFileSet() // positions are relative to fset
	file, err := parser.ParseFile(fileSet, goFile, nil, 0)
	if err != nil {
		fmt.Printf("could not parse file: %v\n", err)
		os.Exit(2)
	}

	g.files = append(g.files, file)

	// Gather targets
	for _, node := range file.Decls {
		switch t := node.(type) {
		case *ast.GenDecl:
			g.evaluateTarget(t)
		}
	}

	// Evaluate which methods targets already have
	for _, node := range file.Decls {
		switch t := node.(type) {
		case *ast.FuncDecl:
			g.evaluateTargetMethods(t)
		}
	}

	return nil
}

func (g *Generator) evaluateTarget(genDecl *ast.GenDecl) {
	for _, spec := range genDecl.Specs {
		switch spec.(type) {
		case *ast.TypeSpec:
			typeSpec := spec.(*ast.TypeSpec)

			switch typeSpec.Type.(type) {
			case *ast.StructType:
				if g.isTarget(typeSpec.Name.Name) {
					t := &TargetType{Name: typeSpec.Name.Name, g: g}
					g.Targets[t.Name] = t
					ast.Inspect(spec, t.visitFields)
				}
			}
		}
	}
}

var evaluateMethods = []string{"Copy", "Equals", "Diff", "Merge"}

func (g *Generator) evaluateTargetMethods(funcDecl *ast.FuncDecl) {
	if funcDecl.Recv != nil {
		for _, method := range evaluateMethods {
			if method == funcDecl.Name.Name {
				g.evaluateExistingMethod(funcDecl, funcDecl.Name.Name)
			}
		}
	}
}

func (g *Generator) evaluateExistingMethod(funcDecl *ast.FuncDecl, methodName string) {
	var methodRecv string
	if stex, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
		methodRecv = stex.X.(*ast.Ident).Name
	} else if id, ok := funcDecl.Recv.List[0].Type.(*ast.Ident); ok {
		methodRecv = id.Name
	}

	target, ok := g.Targets[methodRecv]
	if ok {
		target.ExistingMethods = append(target.ExistingMethods, funcDecl.Name.Name)
	}
}

func (g *Generator) isTarget(name string) bool {
	for _, typeName := range g.typeNames {
		if name == typeName {
			return true
		}
	}
	return false
}

func (g *Generator) generate() error {
	var err error
	if err = g.render("copy"); err != nil {
		return errors.New(fmt.Sprintf("generate.copy: %v", err))
	}

	if err = g.render("equals"); err != nil {
		return errors.New(fmt.Sprintf("generate.equals: %v", err))
	}

	//if err = g.render("diff"); err != nil {
	//	return errors.New(fmt.Sprintf("generate.diff: %v", err))
	//}

	//if err = g.render("merge"); err != nil {
	//	return errors.New(fmt.Sprintf("generate.merge: %v", err))
	//}

	return nil
}

func (g *Generator) render(targetFunc string) error {
	var err error
	targetFileName := fmt.Sprintf("./structs.%s.go", targetFunc)

	var templateFile embed.FS

	switch targetFunc {
	case "copy":
		templateFile = copyTmpl
	case "equals":
		templateFile = equalsTmpl
	case "diff":
		templateFile = diffTmpl
	case "merge":
		templateFile = mergeTmpl
	}

	var buf bytes.Buffer
	err = g.write(&buf, templateFile)
	if err != nil {
		return err
	}

	formatted := g.format(buf.Bytes())

	err = os.WriteFile(targetFileName, formatted, 0744)
	if err != nil {
		return err
	}

	return nil
}

func (g *Generator) write(w io.Writer, file embed.FS) error {
	if len(g.Targets) < 1 {
		return errors.New("generate.render.write: no targets found")
	}
	tmpl, err := template.ParseFS(file, "*")
	if err != nil {
		return errors.New(fmt.Sprintf("generate.render.write: %v", err))
	}
	return tmpl.Execute(w, g)
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
	g        *Generator
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

					ts := f.g.typeSpecs[ident]

					f.ValueType = &TargetField{
						TypeName: elemTypeName,
						isCopier: ts != nil && ts.isCopier(),
						g:        f.g,
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

					ts := f.g.typeSpecs[ident]
					f.ValueType = &TargetField{
						TypeName: valueTypeName,
						isCopier: ts != nil && ts.isCopier(),
						g:        f.g,
					}
					f.KeyType = &TargetField{
						TypeName: t.Key.(*ast.Ident).Name,
						g:        f.g,
					}

				case *ast.StructType:
					f.TypeName = "struct"
					// TODO: where can we get the Ident from?
					//ts := f.g.typeSpecs[ident]
					//f.isCopier = ts != nil && ts.isCopier()

				case *ast.StarExpr:
					f.TypeName = "pointer"
					ident := t.X.(*ast.Ident).Name
					ts := f.g.typeSpecs[ident]
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
	Name            string // Name of the type we're generating methods for
	methods         []string
	excludedFields  []string
	Fields          []*TargetField
	ExistingMethods []string
	g               *Generator
}

func (t *TargetType) Abbr() string {
	return strings.ToLower(string(t.Name[0]))
}

func (t *TargetType) targetsMethod(methodName string) bool {
	for _, method := range t.Methods() {
		if strings.ToLower(method) == "all" {
			return true
		}
		if strings.ToLower(methodName) == strings.ToLower(method) {
			return true
		}
	}
	return false
}

func (t *TargetType) needsMethod(methodName string) bool {
	for _, existing := range t.ExistingMethods {
		if existing == methodName {
			return false
		}
	}
	return true
}

func (t *TargetType) IsCopy() bool {
	return t.targetsMethod("copy") && t.needsMethod("Copy")
}

func (t *TargetType) IsEquals() bool {
	return t.targetsMethod("equals") && t.needsMethod("Equals")
}

func (t *TargetType) IsDiff() bool {
	return t.targetsMethod("diff") && t.needsMethod("Diff")
}

func (t *TargetType) IsMerge() bool {
	return t.targetsMethod("merge") && t.needsMethod("Merge")
}

func (t *TargetType) Methods() []string {
	if t.methods == nil {
		var m []string
		for _, method := range t.g.methods {
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
		for _, excludedField := range t.g.excludedFields {
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

				targetField := &TargetField{Name: field.Names[0].Name, Field: field, g: t.g}
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
