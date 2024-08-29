package tgen

import (
	"testing"
)

func TestUnit(t *testing.T) {
	// A int
	// fmt.Println(unitParam("A", unit("int")))

	// // B mockmain.INT
	// fmt.Println(unitParam("B", unitComp("mockmain", unit("INT"))))

	// // C *string
	// fmt.Println(unitParam("C", unitPtr(unit("string"))))

	// // D chan string, D1 <-chan *string, D2 chan<- mockmain.INT
	// fmt.Println(unitParam("D", unitChan(chan_, unit("string"))))
	// fmt.Println(unitParam("D1", unitChan(chan_out, unitPtr(unit("string")))))
	// fmt.Println(unitParam("D2", unitChan(chan_in, unitComp("mockmain", unit("INT")))))

	// // E []string, E1 *[]mockmain.INT, E2 *[]*string, E3 [3]int
	// fmt.Println(unitParam("E", unitSlice(unit("string"))))
	// fmt.Println(unitParam("E1", unitPtr(unitSlice(unitComp("mockmain", unit("INT"))))))
	// fmt.Println(unitParam("E2", unitPtr(unitSlice(unitPtr((unit("string")))))))
	// // fmt.Println(unitParam("E3", unitArray("3", unit("int"))))

	// // F func(int, string) error, F1 func(int, string) (*mockmain.INT, error), F2 func([]*mockmain.INT, ...int)
	// fmt.Println(unitParam("F", unitFunc([]iunit{unit("int"), unit("string")}, []iunit{unit("error")})))
	// fmt.Println(unitParam("F1", unitFunc([]iunit{unit("int"), unit("string")}, []iunit{unitPtr(unitComp("mockmain", unit("INT"))), unit("error")})))
	// fmt.Println(unitParam("F2", unitFunc([]iunit{unitSlice(unitPtr(unitComp("mockmain", unit("INT")))), unitEllipsis(unit("int"))}, nil)))

	// // G INT[int], G1 *INT[int, []string]
	// fmt.Println(unitParam("G", unitGen(unit("INT"), []iunit{unit("int")})))
	// fmt.Println(unitParam("G1", unitPtr(unitGen(unit("INT"), []iunit{unit("int"), unitSlice(unit("string"))}))))
}
