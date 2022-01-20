package main

// func(T) T -> []T -> []T
func Map[T any](f func(T) T, list []T) []T {
	//...
	return nil
}

// func(T) T -> func([]T) []T
func Map1[T any](f func(T) T) func([]T) []T {
	//...
	return nil
}

// func(A) B -> func([]A) []B
func Map2[A, B any](f func(A) B) func([]A) []B {
	//...
	return nil
}

type Functor[A, B any] interface {
	Map(func(A) B) func([]A) []B
}

type Container[T any] interface {
	ID(T) T
}

// func(A) B -> func(C[A]) C[B]
// INVALID
type Functor2[A, B any, C Container] interface {
	Map(func(A) B) func(C[A]) C[B]
}
