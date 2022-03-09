package main

import "fmt"

func sum(a, b int) int {
	return a + b
}

func sumSquare(a, b int) int {
	sqa := a * a
	sqb := b * b
	return sum(sqa, sqb)
}

func main() {
	ret := sumSquare(1, 2)
	fmt.Println(ret)
}
