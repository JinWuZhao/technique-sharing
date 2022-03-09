package main

import (
	"fmt"
)

var gInt int

func sum(a, b int) int {
	return a + b
}

func main() {
	gInt = sum(1, 2)
	fmt.Println(gInt)
}
