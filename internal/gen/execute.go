package tgen

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func Execute() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("GET WD\n\t", err)
	}
	genConf := new(genConf)
	var names string
	var dir string
	fileName := os.Getenv("GOFILE")

	flag.StringVar(&dir, "dir", "", "Дериктория сканирования")
	flag.StringVar(&genConf.dir, "outdir", "./mocks", "Папка куда будут генерироваться  файлы, если пусто то создает папку mocks в дериктории файла")
	flag.StringVar(&genConf.fileName, "outfile", fileName, "Имя выходного файла, если пустое то mock+имя файла")
	flag.StringVar(&genConf.Pkg, "outpkg", "", "Имя выходного пакета, если пусто то mock+имя пакета")
	flag.StringVar(&names, "name", "", "Через запятую перечисление имен интерфейсов, если пусто то генерирует все интерфейсы")
	flag.Parse()

	if !filepath.IsAbs(dir) {
		dir = filepath.Join(wd, dir)
	}
	if !filepath.IsAbs(genConf.dir) {
		genConf.dir = filepath.Join(dir, genConf.dir)
	}
	nameList := strings.Split(names, ",")
	if names == "" {
		nameList = nil
	}

	sc := &scanConf{wd: wd, dir: dir, names: nameList}
	sc.scanPkg()
	genConf.Src = sc.pkg
	if len(nameList) > 0 {
		for _, name := range nameList {
			if iface, ok := genConf.Src.Ifaces[name]; ok {
				iface.IsPrint = true
			}
		}
	} else {
		for _, iface := range genConf.Src.Ifaces {
			iface.IsPrint = true
		}
	}

	if genConf.Pkg == "" {
		genConf.Pkg = genConf.Src.pkgName + "mock"
	}
	if genConf.fileName == "" {
		genConf.fileName = genConf.Src.pkgName + "_mock.go"
	}

	generate(genConf)
}
