package main

import (
	"fmt"
	"strconv"
)

type Container[T any] interface {
	ID(T) T // Placeholder method
}

type Functor[A any, B any] interface {
	Map(func(A) B) func(Container[A]) Container[B]
}

type List[T any] []T

func (m List[T]) ID(v T) T {
	return v
}

type ListFunctor[A any, B any] struct{}

func (m ListFunctor[A, B]) Map(f func(A) B) func(Container[A]) Container[B] {
	return func(ta Container[A]) Container[B] {
		la := ta.(List[A])
		var tb List[B]
		for _, a := range la {
			tb = append(tb, f(a))
		}
		return tb
	}
}

func Map[A any, B any, F Functor[A, B]](f func(A) B, ta Container[A]) Container[B] {
	var functor F
	return functor.Map(f)(ta)
}

func main() {
	out := Map[int, string, ListFunctor[int, string]](
		func(v int) string {
			return strconv.FormatInt(int64(v), 10)
		},
		List[int]{1, 2, 3},
	)
	fmt.Println(out)
}
