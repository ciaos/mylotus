package cfg

type ShopItem struct {
	// index 0
	ItemID      int32 "index"
	Category    int32
	Name        string
	Icon        string
	Description string
	Award       [][3]int
	Price       [][2]int
	Discount    float32
	Order       int32
}
