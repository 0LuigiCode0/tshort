package main

import (
	"log"
	"os"
	"path"
	"strings"
	"text/template"
)

func generate(f *file, outDir, outFileName, outPkg string) {
	err := os.Mkdir(outDir, os.ModeDir)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	outFile, err := os.OpenFile(path.Join(outDir, outFileName), os.O_CREATE|os.O_TRUNC|os.O_RDONLY, 0744)
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	t, err := template.New("").Funcs(template.FuncMap{
		"isInclude": f.isInclude,
		"generic":   generic,
		"params":    params,
	}).Parse(tmp)
	if err != nil {
		log.Fatal(err)
	}
	f.Pkg = outPkg
	err = t.Execute(outFile, f)
	if err != nil {
		log.Fatal(err)
	}
}

func (b *base) String(incArg bool) string {
	out := strings.Join(b.Name, ",")
	if incArg {
		out += " " + b.Arg
	}
	return out
}

func (f *file) isInclude(key string) bool {
	_, ok := f.ImportsInclude[key]
	return ok
}

func generic(gencs []*base, incType bool) (out string) {
	out = params(gencs, incType)
	if len(gencs) > 0 {
		out = "[" + out + "]"
	}
	return
}
func params(params []*base, incType bool) string {
	paramsList := make([]string, 0, len(params))
	for _, param := range params {
		paramsList = append(paramsList, param.String(incType))
	}
	return strings.Join(paramsList, ",")
}
