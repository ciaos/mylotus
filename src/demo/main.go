package main

import (
	"fmt"
)

type B struct {
	bb int32
}

type A struct {
	la []*B
}

func main() {
	a := &A{
		la: make([]*B, 0, 10),
	}
	fmt.Println(a.la, len(a.la), cap(a.la))
}
