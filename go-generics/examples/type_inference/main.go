package main

import (
	"fmt"
	"strings"
)

type Container[S fmt.Stringer] interface {
	~[]S
}

func Show[A Container[B], B fmt.Stringer](container A) string {
	strs := make([]string, len(container))
	for i, element := range container {
		strs[i] = element.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(strs, ","))
}

type Option[T any] struct {
	i *T
}

func Some[T any](v T) Option[T] {
	return Option[T]{
		i: &v,
	}
}

func None[T any]() Option[T] {
	return Option[T]{}
}

func (m Option[T]) String() string {
	if m.i != nil {
		return fmt.Sprintf("Some(%+v)", *m.i)
	}
	return "None"
}

type List[T any] []T

func NewList[T any](values ...T) List[T] {
	return append(List[T]{}, values...)
}

func main() {
	ls := NewList(Some(1), Some(2), Some(3), None[int]())
	fmt.Println(Show(ls))
}
