package cfg

type NewPlayer struct {
	InitCash  [][2]int
	InitHeros []int
	InitItems [][]int
	InitMail  struct {
		Title    string "title"
		Content  string "content"
		Expirets int64  "expirets"
	}
}
