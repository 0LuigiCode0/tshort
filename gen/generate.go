package tgen

import (
	"log"
	"os"
	"path"
	"text/template"
)

type genConf struct {
	Src      *_package
	Pkg      string
	dir      string
	fileName string
}

// Генерирует фаил мока изходя из результатов scan()
func generate(c *genConf) {
	err := os.Mkdir(c.dir, os.ModeDir)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	outFile, err := os.OpenFile(path.Join(c.dir, c.fileName), os.O_CREATE|os.O_TRUNC|os.O_RDONLY, 0o744)
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	t, err := template.New("").Funcs(template.FuncMap{}).Parse(tmp)
	if err != nil {
		log.Fatal(err)
	}

	for _, iface := range c.Src.Ifaces {
		if iface.IsPrint {
			c.Src.fill(iface.Methods, iface.nestedList)
		}
	}

	err = t.Execute(outFile, c)
	if err != nil {
		log.Fatal(err)
	}
}

func (pkg *_package) fill(methods methods, nestedList nestedList) {
	for module, includes := range nestedList {
		if module == "" {
			for _, nest := range includes {

				incFace := pkg.Ifaces[nest.name]
				for name, _func := range incFace.Methods {
					methods[name] = _func
				}
				pkg.fill(methods, incFace.nestedList)
			}
		} else {
			_pkg := pkg.include[module].pkg
			for _, nest := range includes {
				incFace := _pkg.Ifaces[nest.name]
				for name, _func := range incFace.Methods {
					methods[name] = _func
				}
				_pkg.fill(methods, incFace.nestedList)
			}
		}
	}
}
