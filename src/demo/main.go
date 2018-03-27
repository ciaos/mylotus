package main

import (
	"fmt"
)

func (a *A) Say() {
	fmt.Println("Say %v", a)
}

type A struct {
	a int32
}

func main() {
	a := &A{}
	a = nil
	a.Say()
}
