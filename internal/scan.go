package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path"
	"strconv"
)

type file struct {
	raw            []byte
	Pkg            string
	Imports        []*base
	ImportsInclude map[string]struct{}
	Interfaces     []*_interface
}

type base struct {
	Name []string
	Arg  string
}

type _interface struct {
	base
	Generics []*base
	Methods  []*_func
}

type _func struct {
	base
	Generics []*base
	In       []*base
	Out      []*base
}

func Scan(fileName string, intNames map[string]struct{}) *file {
	fs := token.NewFileSet()
	raw, err := os.ReadFile(fileName)
	if err != nil {
		log.Fatal(err)
	}
	f, err := parser.ParseFile(fs, fileName, nil, 0)
	if err != nil {
		log.Fatal(err)
	}

	file := new(file)
	file.raw = raw
	file.scanImports(f.Imports)
	file.scanInterfaces(f.Scope.Objects, intNames)

	return file
}

func (f *file) scanImports(imports []*ast.ImportSpec) {
	f.Imports = make([]*base, 0, len(imports))
	f.ImportsInclude = make(map[string]struct{}, len(imports))
	for _, _import := range imports {
		imp := new(base)
		imp.Arg = _import.Path.Value
		if _import.Name == nil {
			imp.Name = []string{path.Ext(imp.Arg)}
		} else {
			imp.Name = []string{_import.Name.Name}
		}
		f.Imports = append(f.Imports, imp)
	}
}

func (f *file) scanInterfaces(objs map[string]*ast.Object, intNames map[string]struct{}) {
	f.Interfaces = make([]*_interface, 0, len(objs))
	for name, obj := range objs {
		if _, ok := intNames[name]; len(intNames) > 0 && !ok {
			continue
		}
		if v, ok := obj.Decl.(*ast.TypeSpec); ok {
			generics := f._scanParams(v.TypeParams, "g")
			if v, ok := v.Type.(*ast.InterfaceType); ok {
				_interface := &_interface{}
				_interface.Name = []string{name}
				_interface.Generics = generics
				_interface.Methods = f._scanMethods(v.Methods)
				f.Interfaces = append(f.Interfaces, _interface)
			}
		}
	}
}

func (f *file) _scanParams(typeParams *ast.FieldList, prefix string) []*base {
	if typeParams != nil && typeParams.List != nil {
		params := make([]*base, 0, len(typeParams.List))
		for i, v := range typeParams.List {
			arg := string(f.raw[v.Type.Pos()-1 : v.Type.End()-1])
			if len(v.Names) > 0 {
				for _, name := range v.Names {
					params = append(params, &base{[]string{name.Name}, arg})
				}

			} else {
				params = append(params, &base{[]string{prefix + strconv.Itoa(i)}, arg})
			}
		}
		return params
	}
	return nil
}
func (f *file) _scanMethods(methods *ast.FieldList) []*_func {
	if methods != nil {
		funcs := make([]*_func, 0, len(methods.List))
		for _, method := range methods.List {
			if methodType, ok := method.Type.(*ast.FuncType); ok {
				for _, name := range method.Names {
					_func := &_func{}
					_func.Name = []string{name.Name}
					_func.Generics = f._scanParams(methodType.TypeParams, "g")
					_func.In = f._scanParams(methodType.Params, "i")
					_func.Out = f._scanParams(methodType.Results, "o")
					funcs = append(funcs, _func)
				}
			}
		}
		return funcs
	}
	return nil
}
