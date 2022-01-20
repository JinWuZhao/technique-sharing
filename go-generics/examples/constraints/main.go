package main

type Int interface {
	int
}

type String interface {
	string
}

type Func interface {
	func(int) int
}

type MyStruct struct {
	Field1 int
	Field2 int
}

type Struct interface {
	MyStruct
}

type Pointer interface {
	*MyStruct
}

type MyInt int

type MyInteger interface {
	MyInt
}

type ApproxInt interface {
	~int
}

type ApproxString interface {
	~string
}

type ApproxFunc interface {
	~func(int) int
}

type ApproxStruct interface {
	~struct {
		Field1 int
		Field2 int
	}
}

type ApproxPointer interface {
	~*MyStruct
}

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | float32 | float64
}

func main() {}
