package main

func Equal[T comparable](lhs, rhs T) bool {
	return lhs == rhs
}

type MyInt int

func main() {
	Equal(1, 1)
	Equal(MyInt(1), 1)
	Equal(1.0, 1.1)
	Equal("a", "b")
	Equal(struct{}{}, struct{}{})
}
