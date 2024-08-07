package main

import (
	"flag"
	"os"
	"strings"

	"github.com/0LuigiCode0/tshort/internal"
)

func main() {
	var outDir string
	var outFileName string
	var outPkg string
	var intGen string

	inFileName := os.Getenv("GOFILE")
	inPkg := os.Getenv("GOPACKAGE")

	flag.StringVar(&outDir, "outdir", "./mocks", "Папка куда будут генерироваться  файлы, если пусто то создает папку mocks в дериктории файла")
	flag.StringVar(&outFileName, "outfilename", "mock"+inFileName, "Имя выходного файла, если пустое то mock+имя файла")
	flag.StringVar(&outPkg, "outpkg", "mock"+inPkg, "Имя выходного пакета, если пусто то mock+имя пакета")
	flag.StringVar(&intGen, "intgen", "", "Через запятую перечисление имен интерфейсов, если пусто то генерирует все интерфейсы")
	flag.Parse()

	intGenNames := map[string]struct{}{}
	for _, s := range strings.SplitN(intGen, ",", 0) {
		intGenNames[s] = struct{}{}
	}

	f := internal.Scan(inFileName, intGenNames)
	internal.Generate(f, outDir, outFileName, outPkg)
}
