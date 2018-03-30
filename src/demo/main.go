package main

import (
	"fmt"
)

func main() {

	a := []int32{1}

	a = append(a[0:0], a[0+1:]...)
	fmt.Println(a)
}
