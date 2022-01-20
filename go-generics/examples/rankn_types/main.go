// Rank-N Types示例
// Go语言只实现了Rank-1

package main

func id[T any](v T) T {
	return v
}

// f 无法定义为多态函数
func Foo[T any](f func(T) T) T {
	id(1) // YES
	id("") // YES
	f(1) // NO
	f("") // NO
	//...
}

// 假想的定义 Rank-2 Type
func FooRank2[T any](f func[forall.T](T) T) T { 
	//... 
}

// 假想的定义 Rank-3 Type
func FooRank3[T any](f func[forall.T](func[forall.T](T) T) T) T { 
	//...
}

func main() {}