package main

import (
	"fmt"
)

func f(a1 uint8, a2 [2]uint8, a3 uint8) (r1 struct {
	x uint8
	y [2]uint8
}, r2 string) {
	r1.x = a1
	r1.y = a2
	r2 = "r2"
	return
}

func main() {
	 var a1 uint8 = 1
	var a3 uint8 = 2
	r1, r2 := f(a1, [2]uint8{a1, a3}, a3)
	fmt.Println(r1, r2)
}
