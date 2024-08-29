package tgen

import (
	"bufio"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	tutils "github.com/0LuigiCode0/tshort/internal/utils"
)

type scanConf struct {
	pkg         *_package
	indexMapper map[string]map[int]string
	wd          string
	dir         string
	names       []string
}

type _package struct {
	c          *scanConf
	Imports    *imports
	Ifaces     ifaces
	pkgName    string
	moduleName string
	include    map[string]*scanConf
}

// -------------------------------------------------------------------------- //
// MARK:imports{}
// -------------------------------------------------------------------------- //

type (
	imports struct {
		Imports           map[string]*_import
		aliasMapper       map[string]string
		pathImportToAlias map[string]string
		versions          map[string]string
	}
	_import struct {
		Path    string
		IsPrint bool
	}
)

func (imps imports) String() (out string) {
	ss := make([]string, 0, len(imps.Imports))
	for alias, imp := range imps.Imports {
		if imp.IsPrint {
			ss = append(ss, alias+" "+imp.Path+" "+strconv.FormatBool(imp.IsPrint))
		}
	}
	return tutils.Join("\n", ss...)
}

func (imps imports) _Import() (out string) {
	ss := make([]string, 0, len(imps.Imports))
	for alias, imp := range imps.Imports {
		ss = append(ss, "IMPORT:\n\t"+alias+" "+imp.Path+" "+strconv.FormatBool(imp.IsPrint))
	}
	return tutils.Join("\n", ss...)
}

// -------------------------------------------------------------------------- //
// MARK:includes{}
// -------------------------------------------------------------------------- //

type (
	includes map[string]*include
	include  struct {
		methods *methods
		imports *imports
	}
)

// -------------------------------------------------------------------------- //
// MARK:ifaces{}
// -------------------------------------------------------------------------- //

type (
	ifaces map[string]*iface
	iface  struct {
		scan       *scan
		raw        *ast.FieldList
		Generics   *generics
		nestedList nestedList
		Methods    methods
		IsPrint    bool
	}
)

// -------------------------------------------------------------------------- //
// MARK:generics{}
// -------------------------------------------------------------------------- //

type (
	generics struct {
		indexMapper map[int]string
		aliasMapper map[string]string
		params      params
	}
)

func (gens generics) String() (out string) {
	if len(gens.params) == 0 {
		return ""
	}

	ss := make([]string, 0, len(gens.params))
	for _, gen := range gens.params {
		ss = append(ss, gen.value+" "+gen.exp.String())
	}
	return "[" + tutils.Join(",", ss...) + "]"
}

func (gens generics) StringName() (out string) {
	if len(gens.params) == 0 {
		return ""
	}

	ss := make([]string, 0, len(gens.params))
	for _, gen := range gens.params {
		ss = append(ss, gen.value)
	}
	return "[" + tutils.Join(",", ss...) + "]"
}

// -------------------------------------------------------------------------- //
// MARK:methods{}
// -------------------------------------------------------------------------- //

type (
	methods map[string]*method
	method  struct {
		In  params
		Out params
	}
)

func (m methods) String() string {
	ss := make([]string, 0, len(m))
	for name, method := range m {
		ss = append(ss, "METHOD:\n\t"+name+"("+method.In.String()+")"+"("+method.Out.String()+")")
	}
	return tutils.Join("\n", ss...)
}

// -------------------------------------------------------------------------- //
// MARK:nestedList{}
// -------------------------------------------------------------------------- //

type (
	nestedList map[string][]*nested
	nested     struct {
		generics *generics
		name     string
	}
)

func (n nestedList) String() string {
	ss := make([]string, 0, len(n))
	for module, includes := range n {
		for _, include := range includes {
			ss = append(ss, "INCLUDE:\n\t"+module+"."+include.name+include.generics.String())
		}
	}
	return tutils.Join("\n", ss...)
}

// -------------------------------------------------------------------------- //
// MARK:params{}
// -------------------------------------------------------------------------- //

type (
	params []*_unitParam
)

func (p params) String() string {
	ss := make([]string, 0, len(p))
	for _, param := range p {
		ss = append(ss, param.value+" "+param.exp.String())
	}
	return tutils.Join(",", ss...)
}

func (p params) Names() string {
	ss := make([]string, 0, len(p))
	for _, param := range p {
		ss = append(ss, param.value)
	}
	return tutils.Join(",", ss...)
}

func (p params) Types() string {
	ss := make([]string, 0, len(p))
	for _, param := range p {
		ss = append(ss, param.exp.String())
	}
	return tutils.Join(",", ss...)
}

// -------------------------------------------------------------------------- //
// MARK:scanPkg()
// -------------------------------------------------------------------------- //

func (c *scanConf) scanPkg() {
	if c.dir == "" {
		return
	}
	if c.pkg == nil {
		c.pkg = &_package{
			c:       c,
			Imports: &imports{Imports: map[string]*_import{}, aliasMapper: map[string]string{}, pathImportToAlias: map[string]string{}, versions: map[string]string{}},
			Ifaces:  ifaces{},
			include: map[string]*scanConf{},
		}
	}

	scanDir, err := parser.ParseDir(token.NewFileSet(), c.dir, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go") && strings.HasSuffix(fi.Name(), ".go")
	}, parser.SkipObjectResolution)
	if err != nil {
		log.Fatal("PARSE PKG\n\t", err)
	}
	for name, dir := range scanDir {
		c.pkg.pkgName = name
		c.pkg.scanImports(dir.Files)
		c.pkg.scanInterfaces(dir.Files)
		c.pkg.scanMod()
	}

	// fmt.Println(c.pkg.Imports._Import())
	// for name, v := range c.pkg.Ifaces {
	// 	fmt.Println(name, v.Generics.String(), v.Generics.StringName())
	// 	fmt.Println(name, v.Methods.String())
	// 	fmt.Println(name, v.nestedList.String())
	// }

	include := map[string][]struct {
		name        string
		indexMapper map[int]string
	}{}
	for _, iface := range c.pkg.Ifaces {
		for module, includes := range iface.nestedList {
			for _, v := range includes {
				include[module] = append(include[module], struct {
					name        string
					indexMapper map[int]string
				}{name: v.name, indexMapper: v.generics.indexMapper})
			}
		}
	}
	for module, includes := range include {
		if module != "" {
			curModule = c.pkg.Imports.aliasMapper[module]
			wd, dir := c.pkg.findPath(module)
			names := make([]string, 0, len(includes))
			indexMapper := make(map[string]map[int]string, len(includes))
			for _, include := range includes {
				names = append(names, include.name)
				indexMapper[include.name] = include.indexMapper
			}

			sc := &scanConf{wd: wd, dir: dir, names: names, indexMapper: indexMapper}
			sc.pkg = &_package{
				c:          sc,
				Imports:    c.pkg.Imports,
				Ifaces:     ifaces{},
				moduleName: module,
				include:    map[string]*scanConf{},
			}
			sc.scanPkg()
			c.pkg.include[module] = sc
		}
	}
}

// -------------------------------------------------------------------------- //
// MARK:scanMod()
// -------------------------------------------------------------------------- //

var reg = regexp.MustCompile(`[.](.*?\/)`)

func (pkg *_package) scanMod() {
	fs, err := os.ReadDir(pkg.c.wd)
	if err != nil {
		log.Fatal("SCAN DIR\n\t", err)
	}
	for _, entry := range fs {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".mod") {
			// Open go.mod
			f, err := os.Open(entry.Name())
			if err != nil {
				log.Fatal("DIR\n\t", err)
			}
			defer f.Close()
			buf := bufio.NewScanner(f)

			// Scan module name
			buf.Scan()
			line := buf.Text()
			if err != nil {
				log.Fatal("DIR\n\t", err)
			}
			pkg.moduleName = string(line[7:])

			// Scan version by import
			versions := map[string]string{}
			var require bool

			for buf.Scan() {
				line = buf.Text()
				if line != "" {
					if !require {
						if strings.HasPrefix(line, "require (") {
							require = true
							continue
						} else if path, ok := strings.CutPrefix(line, "require "); ok {
							line = path
						} else {
							continue
						}
					} else if line == ")" {
						require = false
						continue
					}

					vers := strings.SplitN(line, " ", 3)
					if len(vers) > 1 && vers[len(vers)-1] == "// indirect" {
						continue
					}
					versions[vers[0]] = vers[1]
				}
			}

			for _, path := range pkg.Imports.Imports {
				if !strings.HasPrefix(path.Path, pkg.moduleName) && reg.MatchString(path.Path) {
					for absPath, v := range versions {
						if strings.HasPrefix(path.Path, absPath) {
							pkg.Imports.versions[absPath] = v
						}
					}
				}
			}

			f.Close()
		}
	}
	// dir = filepath.Dir(dir)
	// }
}

// -------------------------------------------------------------------------- //
// MARK:scanImports()
// -------------------------------------------------------------------------- //

func (pkg *_package) scanImports(files map[string]*ast.File) {
	for _, f := range files {
		for _, decl := range f.Decls {
			if decl, ok := decl.(*ast.GenDecl); ok && decl.Tok == token.IMPORT {
				for _, spec := range decl.Specs {
					if impType, ok := spec.(*ast.ImportSpec); ok {
						var alias string
						var baseAlias string
						pkgPath := impType.Path.Value[1 : len(impType.Path.Value)-1]
						if impType.Name == nil {
							alias = path.Base(pkgPath)
						} else {
							alias = impType.Name.Name
						}
						if baseAlias, ok = pkg.Imports.pathImportToAlias[pkgPath]; !ok {
							baseAlias = "_" + alias + "_"
							pkg.Imports.pathImportToAlias[pkgPath] = baseAlias
						}
						if _, ok := pkg.Imports.Imports[baseAlias]; !ok {
							pkg.Imports.Imports[baseAlias] = &_import{Path: pkgPath}
						}

						pkg.Imports.aliasMapper[alias] = baseAlias
					}
				}
			}
		}
	}
}

// -------------------------------------------------------------------------- //
// MARK:scanInterfaces()
// -------------------------------------------------------------------------- //

func (pkg *_package) scanInterfaces(files map[string]*ast.File) {
	ifaces := make(ifaces)
	for _, f := range files {
		for _, decl := range f.Decls {
			if decl, ok := decl.(*ast.GenDecl); ok && decl.Tok == token.TYPE {
				for _, spec := range decl.Specs {
					if types, ok := spec.(*ast.TypeSpec); ok {
						if _if, ok := types.Type.(*ast.InterfaceType); ok {
							scan := &scan{imports: pkg.Imports}
							if mapper, ok := pkg.c.indexMapper[types.Name.Name]; ok {
								scan.indexMapper = mapper
							}

							iface := &iface{
								scan:     scan,
								raw:      _if.Methods,
								Generics: scan.scanGenerics(types.TypeParams),
							}
							scan.genericAlias = iface.Generics.aliasMapper

							ifaces[types.Name.Name] = iface
						}
					}
				}
			}
		}
	}
	if pkg.c.names == nil {
		pkg.Ifaces = ifaces
		for _, iface := range ifaces {
			iface.scanFields(iface.raw)
		}
	} else {
		for _, name := range pkg.c.names {
			if iface, ok := ifaces[name]; ok {
				iface.scanFields(iface.raw)
				pkg.Ifaces[name] = iface
			}
		}
	}
}

// -------------------------------------------------------------------------- //
// MARK:scanGenerics()
// -------------------------------------------------------------------------- //

func (s *scan) scanGenerics(paramList *ast.FieldList) (out *generics) {
	out = new(generics)
	out.params = s.scanParams(paramList)
	out.aliasMapper = map[string]string{}
	for i, gen := range out.params {
		var newName string
		if name, ok := s.indexMapper[i]; ok {
			newName = name
		} else {
			newName = "_g" + strconv.Itoa(i)
		}
		out.aliasMapper[gen.value] = newName
		gen.value = newName
	}
	return
}

// -------------------------------------------------------------------------- //
// MARK:scanFields()
// -------------------------------------------------------------------------- //

func (_if *iface) scanFields(fieldList *ast.FieldList) {
	if fieldList == nil {
		return
	}
	_if.Methods = make(methods, len(fieldList.List))
	_if.nestedList = make(nestedList, len(fieldList.List))

	for _, f := range fieldList.List {
		switch t := f.Type.(type) {
		case *ast.FuncType:
			_if.Methods[tutils.JoinF("", func(i int, s *ast.Ident) (string, bool) { return s.Name, true }, f.Names...)] = _if.scanMethod(t)
		default:
			// разбираем вложения на класс, модуль и прочее
			module, inc := _if.scanNested(t)
			_if.nestedList[module] = append(_if.nestedList[module], inc)
		}
	}
}

// -------------------------------------------------------------------------- //
// MARK:scanMethod()
// -------------------------------------------------------------------------- //

func (_if *iface) scanMethod(f *ast.FuncType) *method {
	in := _if.scan.scanParams(f.Params)
	out := _if.scan.scanParams(f.Results)
	for i, param := range in {
		param.value = "_in" + strconv.Itoa(i)
	}
	for i, param := range out {
		param.value = "_out" + strconv.Itoa(i)
	}

	return &method{
		In:  in,
		Out: out,
	}
}

// -------------------------------------------------------------------------- //
// MARK:scanNested()
// -------------------------------------------------------------------------- //

func (_if *iface) scanNested(f ast.Expr) (string, *nested) {
	var module string
	inc := &nested{
		generics: &generics{
			indexMapper: map[int]string{},
			aliasMapper: map[string]string{},
			params:      []*_unitParam{},
		},
	}

	switch t := f.(type) {
	case *ast.IndexExpr:
		inc.generics.params = _if.scan.scanParams(&ast.FieldList{List: []*ast.Field{{Type: t.Index}}})
		f = t.X
	case *ast.IndexListExpr:
		inc.generics.params = _if.scan.scanParams(&ast.FieldList{List: tutils.Convert(t.Indices, func(i int, exp ast.Expr) (*ast.Field, bool) { return &ast.Field{Type: exp}, true })})
		f = t.X
	}

	for i, gen := range inc.generics.params {
		inc.generics.indexMapper[i] = gen.String()
	}

	switch f := f.(type) {
	case *ast.Ident:
		inc.name = f.Name
	case *ast.SelectorExpr:
		inc.name = f.Sel.Name
		module = f.X.(*ast.Ident).Name
	}

	return module, inc
}

// -------------------------------------------------------------------------- //
// MARK:scanType()
// -------------------------------------------------------------------------- //

type scan struct {
	indexMapper  map[int]string
	genericAlias map[string]string
	imports      *imports
}

var curModule string

var baseType = map[string]struct{}{
	"int":     {},
	"int8":    {},
	"int16":   {},
	"int32":   {},
	"int64":   {},
	"uint":    {},
	"uint8":   {},
	"uint16":  {},
	"uint32":  {},
	"uint64":  {},
	"string":  {},
	"byte":    {},
	"rune":    {},
	"float32": {},
	"float64": {},
	"bool":    {},
	"any":     {},
	"error":   {},
}

func (s *scan) scanType(exp ast.Expr) (out iunit) {
	if exp == nil {
		return unitEmpty
	}
	switch exp := exp.(type) {
	case *ast.BasicLit:
		out = unit(exp.Value)
	case *ast.Ident:
		// единичные значения
		if _, ok := baseType[exp.Name]; ok {
			out = unit(exp.Name)
		} else {
			if newOut, ok := s.genericAlias[exp.Name]; ok {
				out = unit(newOut)
			} else {
				out = unitComp(exp.Name, curModule)
				if imp, ok := s.imports.Imports[curModule]; ok {
					imp.IsPrint = true
				}
			}
		}
	case *ast.IndexExpr:
		// параметры дженериков
		out = unitGen(s.scanType(exp.X), []*_unitParam{unitParam("", s.scanType(exp.Index))})
	case *ast.IndexListExpr:
		// параметры дженериков
		args := make([]*_unitParam, 0, len(exp.Indices))
		for _, arg := range exp.Indices {
			args = append(args, unitParam("", s.scanType(arg)))
		}

		out = unitGen(s.scanType(exp.X), args)
	case *ast.Ellipsis:
		// перечисления
		out = unitEllipsis(s.scanType(exp.Elt))
	case *ast.SelectorExpr:
		// последовательный вызов
		sel := exp.X.(*ast.Ident).Name
		// if importAliasMapper
		if trueName, ok := s.imports.aliasMapper[sel]; ok {
			if imp, ok := s.imports.Imports[trueName]; ok || trueName == curModule {
				imp.IsPrint = true
			}
			sel = trueName
		}
		out = unitComp(exp.Sel.Name, sel)
	case *ast.StarExpr:
		// указатель
		out = unitPtr(s.scanType(exp.X))
	case *ast.ArrayType:
		// массивы и слайсы
		if exp.Len == nil {
			out = unitSlice(s.scanType(exp.Elt))
		} else {
			out = unitArray(s.scanType(exp.Len), s.scanType(exp.Elt))
		}
	case *ast.ChanType:
		// каналы с направлением
		chanDirect := chan_
		switch exp.Dir {
		case ast.RECV:
			chanDirect = chan_out
		case ast.SEND:
			chanDirect = chan_in
		}

		out = unitChan(chanDirect, s.scanType(exp.Value))
	case *ast.FuncType:
		// функции
		_in := s.scanParams(exp.Params)
		_out := s.scanParams(exp.Results)

		for _, param := range _in {
			param.value = ""
		}
		for _, param := range _out {
			param.value = ""
		}

		out = unitFunc(_in, _out)
	case *ast.MapType:
		out = unitMap(unitParam("", s.scanType(exp.Key)), unitParam("", s.scanType(exp.Value)))
	default:
		tutils.Print("unknown type", exp)
		out = unit("")
	}
	return
}

// -------------------------------------------------------------------------- //
// MARK:scanParam()
// -------------------------------------------------------------------------- //

func (s *scan) scanParam(param *ast.Field) params {
	out := make(params, 0, len(param.Names))
	_type := s.scanType(param.Type)
	if len(param.Names) > 0 {
		for _, name := range param.Names {
			out = append(out, unitParam(name.Name, _type))
		}
	} else {
		out = append(out, unitParam("", _type))
	}
	return out
}

func (s *scan) scanParams(paramList *ast.FieldList) params {
	if paramList == nil {
		return nil
	}
	out := make(params, 0, len(paramList.List))
	for _, param := range paramList.List {
		out = append(out, s.scanParam(param)...)
	}

	return out
}

// -------------------------------------------------------------------------- //
// MARK:findPath()
// -------------------------------------------------------------------------- //

func (pkg *_package) findPath(module string) (wd, dir string) {
	impPath, ok := pkg.Imports.Imports[pkg.Imports.aliasMapper[module]]
	if ok {
		if reg.MatchString(impPath.Path) {
			if pkgPath, ok := strings.CutPrefix(impPath.Path, pkg.moduleName); ok {
				return pkg.c.wd, filepath.Join(pkg.c.wd, pkgPath)
			}

			var ver string
			pkgPath := impPath.Path
			for ver, ok = pkg.Imports.versions[pkgPath]; !ok; ver, ok = pkg.Imports.versions[pkgPath] {
				pkgPath = path.Dir(pkgPath)
			}
			basePath, _ := strings.CutPrefix(impPath.Path, pkgPath)
			wd = filepath.Join(build.Default.GOPATH, "pkg/mod", pkgPath+"@"+ver)
			return wd, filepath.Join(wd, basePath)
		}
		wd = filepath.Join(build.Default.GOROOT, "src")
		return wd, filepath.Join(wd, impPath.Path)
	}
	return "", ""
}
