package cfg

type Record struct {
	// index 0
	IndexInt int "index"
	// index 1
	IndexStr string "index"
	_Number  int32
	Str      string
	Arr1     [2]int
	Arr2     [3][2]int
	Arr3     []int
	St       struct {
		Name string "name"
		Num  int    "num"
	}
	M map[string]int
}
