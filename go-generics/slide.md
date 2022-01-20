---
marp: true
theme: default
paginate: true
style: |
    section h1 {
        font-size: 2em;
        text-align: center;
    }
    section h2 {
        position: absolute;
        top: 1.2em;
        left: 0;
        right: 0;
        font-size: 1.5em;
        text-align: center;
    }
    section h3 {
        font-size: 1.2em;
    }
    section h4 {
        font-size: 1em;
    }
    section p {
        font-size: 1em;
        text-align: center;
    }
    section li {
        font-size: 1em;
    }
    section.left p {
        text-align: left;
    }
---

# Go 语言泛型介绍

by [Jin](https://github.com/JinWuZhao)

----------

<!-- header: 'Go 语言泛型介绍 by [Jin](https://github.com/JinWuZhao)' -->

## 最受开发者期待的特性

![w:28em](go-need-features.png)

----------

## 什么是泛型

### Polymorphism

#### Ad-hoc

- Overloading
- Coercion

#### Universal

- Inclusion
- Parametric (Generics)

By Christopher Strachey, Luca Cardelli, Peter Wegner.

----------

## 准备工作

```sh
mkdir generics
cd generics
docker run --rm -it \
    -v $PWD:/go/src/generics \
    -w /go/src/generics golang:1.18beta1-buster bash
go mod init example/generics
touch main.go
```

----------

## 类型参数

```text
func Foo[T any, ...](parameters...) ReturnType
```

- **T**：类型名
- **any**：约束

----------

## 泛型函数

```go
func SumInteger[T constraints.Integer](a, b T) T {
    return a + b
}
```

```go
sum := SumInteger[int](1, 2)
```

----------

## 泛型结构体

```go
type Vector[T any] struct {
    inner []T
}

func (m *Vector[T]) Len() int {
    return len(m.inner)
}

func (m *Vector[T]) Get(index int) T {
    return m.inner[index]
}

```

----------

## 泛型内置容器

```go
type Array[T any] [8]T

type Slice[T any] []T

type Map[K comparable, V any] map[K]V

type Chan[T] chan T
```

----------

## 泛型接口

```go
type Iterator[T any] interface {
    Next() bool
    Value() T
}
```

----------

## 实现泛型接口

```go
type ListIter[T any] struct {
    index int
    inner []T
}

func (m *ListIter[T]) Next() (bool) {
    ...
}

func (m *ListIter[T]) Value() T {
    ...
}
```

----------

## 其它

```go
type Fn[A any, B any] func(A) B

type MyInt[T fmt.Stringer] int
...
```

----------

## 约束

- **any** 相当于 **interface{}**
- **comparable** 可以使用 **==** 和 **!=** 操作符

----------

## 定义约束：方法

```go
type Stringer interface {
    String() string
}
```

----------

## 定义约束：非接口类型

```go
type Integer interface {
    int64
}

type Integer interface {
    ~int64
}
```

----------

## 定义约束：联合类型

```go
type Integer interface {
    int | int8 | int16 | int32 | int64
}

type Integer interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64
}
```

----------

## 定义约束：复合元素

```go
type StringerInteger interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64
    String() string
}
```

----------

## 类型集合I

- 非接口类型 **T** ：

**{int}**, **{float32}**, **{string}** ...

- 空接口 **interface {}** 或 **any** ：

所有类型

- **interface { f1(); f2(); f3() ... }**：

**{t : t declared {f1, f2, f3 ...}}**（简写为 **{t: decl(t, Sfs)}** ）

- **interface { M; N }**（ **interface M {...}**, **interface N {...}** ）：

**SM** ∩ **SN**

----------

## 类型集合II

- **interface { ~T }**：

**{t : t ~ T}**（底层类型为 **T** 的类型）

- **interface { X | Y | Z }**：

**SX** ∪ **SY** ∪ **SZ**

- **interface { M; X | Y | Z; f1(); f2(); f3() ...}**：

**SM** ∩ (**SX** ∪ **SY** ∪ **SZ**) ∩ **{ t : decl(t, Sfs)}**

- **interface { int; string }**：

**{int}** ∩ **{string}** = **∅**

----------

## 实例化I

```go
// SOrdered: {t : t ~ int} ∪ {t : t ~ int8} ∪ {t : t ~ int16} ∪ ...
type Ordered interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 |
        ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
        ~float32 | ~float64 |
        ~string
}

func Max[T Ordered](a, b T) T {
    if a > b {
        return a
    }
    return b
}
```

----------

## 实例化II

```go
func main() {
    // {int} ⊆ SOrdered.
    Max[int](1, 2)

    // {string} ⊆ SOrdered. 
    Max[string]("a", "b")
    
    // INVALID: []byte ⊄ SOrdered.
    Max[[]byte]([]byte("hello"))
}
```

----------

## 类型集合的操作：接口方法

```go
// T 关联的类型集合为所有实现了 fmt.Stringer 接口的类型
func JoinSlice[T fmt.Stringer](elements []T, seperator string) string {
    var ret string
    for i, v := range elements {
        ret += v.String() // 类型T的对象可以使用其方法
        if i < len(elements) - 1 {
            ret += seperator
        }
    }
}
```

----------

## 类型集合的操作：比较

```go
// T 关联的类型集合为所有可比较的类型
func SearchIndex[T comparable](elements []T, target T) int {
    index := -1
    for i, e := range elements {
        if e == target { // 类型T的对象可以使用“==”比较
            index = i
            break
        }
    }
    return index
}
```

----------

## 类型集合的操作：基本数据类型

```go
// 有序类型，下面构成Ordered的的所有类型都是可以比较大小的
type Ordered interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 |
        ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
        ~float32 | ~float64 |
        ~string
}

// T 关联的类型集合为 Ordered 对应的类型集合
func Max[T Ordered](a, b T) T {
    if a >= b { // 可以比较大小
        return a
    }
    return b
}
```

----------

## 类型集合的操作：内置容器Slice

```go
// 自定义泛型Slice约束
type MySlice[T any] interface {
    []T
}

func ForEachMySlice[T any, S MySlice[T]](s S) {
    for _, e := range s {
        fmt.Println(e)
    }
}
```

----------

## 类型集合的操作：内置容器Map

```go
// 自定义泛型Map约束
type MyMap[K comparable, V any] interface {
    map[K]V
}

func ForEachMyMap[K comparable, V any, M MyMap[K, V]](m M) {
    for k, v := range m {
        fmt.Println(k, v)
    }
}
```

----------

## 类型集合的操作：内置容器Channel

```go
// 自定义泛型Channel约束
type MyChan[T any] interface {
    chan T
}

func ForEachChan[T any, C MyChan[T]](c C) {
    for v := range c {
        fmt.Println(v)
    }
}
```

----------

## 标准库提供的约束[constraints](https://github.com/golang/go/blob/go1.18beta1/src/constraints/constraints.go)

```go
type Signed interface {
        ~int | ~int8 | ~int16 | ~int32 | ~int64
}

type Unsigned interface {
        ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type Integer interface {
        Signed | Unsigned
}
...
```

----------

## 类型推断

- 指定类型参数

```go
sum := SumInteger[int](1, 2)
```

- 推断类型参数

```go
sumVal := SumInteger(1, 2, 3)
```

----------

## 类型合一化（Type unification）

<!-- _class: left -->
类型参数 **T1**、**T2**，**[]map[int]bool** 可与以下类型合一化：

- **[]map[int]bool**
- **T1** (**T1** -> **[]map[int]bool**)
- **[]T1** (**T1** -> **map[int]bool**)
- **[]map[T1]T2** (**T1** -> **int**, **T2** -> **bool**)

----------

## 函数参数类型推断I

```go
func Map[F, T any](s []F, f func(F) T) []T {
    r := make([]T, len(s))
    for i, v := range s {
        r[i] = f(v)
    }
    return r
}

// []int : []F, { F -> int }.
// func(i int) string : func(F) T, { F -> int，T -> string }.
// Map[int, string](...).
strs := Map([]int{1, 2, 3}, strconv.Itoa)
```

----------

## 函数参数类型推断II

```go
func NewPair[F any](f1, f2 F) *Pair[F] { ... }

// int : F, { F -> int }.
// NewPair[int](...).
NewPair(1, 2)

// untyped int : F, { F -> ? }.
// int64 : F, { F -> int64 }.
// NewPair[int64](...).
NewPair(1, int64(2))
```

----------

## 约束类型推断I

<!-- _class: left -->
**[T Constraint1[V], V Constraint2]**，对于 **T** 可进行类型推断仅当：

- **Constraint1** 对应的类型集合中只包含一个非接口类型；
- **Constraint1** 对应的类型集合中的所有类型的底层类型相同；

----------

## 约束类型推断II

```go
type SC[E any] interface {
    []E
}

func DoubleDefined[S SC[E], E constraints.Number](s S) S {
    r := make(S, len(s))
    for i, v := range s {
        r[i] = v + v
    }
    return r
}

func main() {
    v := DoubleDefined([]int{1})
    // ...
}
```

----------

## 推断步骤

<!-- _class: left -->
如果可以进行约束类型推断，那么函数的类型推断步骤如下：

1. 使用已知的类型参数构建映射关系；
2. 进行约束类型推断；
3. 使用有类型的实参进行函数类型推断；
4. 再次进行约束类型推断；
5. 针对剩余的无类型实参使用默认类型进行函数类型推断；
6. 再次进行约束类型推断；

----------

## 引用自身的约束I

```go
type Equaler interface {
    Equal(Equaler) bool
}

func Index[T Equaler](s []T, e T) int {
    for i, v := range s {
        if e.Equal(v) {
            return i
        }
    }
    return -1
}
```

----------

## 引用自身的约束II

```go
func Index[T interface { Equal(T) bool }](s []T, e T) int {
    for i, v := range s {
        if e.Equal(v) {
            return i
        }
    }
    return -1
}
```

----------

## 约束中同时包含方法和元素I

```go
type StringableSignedInteger interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64
    String() string
}

type MyInt int

func (mi MyInt) String() string {
    return fmt.Sprintf("MyInt(%d)", mi)
}
```

----------

## 约束中同时包含方法和元素II

```go
// type set: Ø
type StringableSignedInteger2 interface {
    int | int8 | int16 | int32 | int64
    String() string
}
```

----------

## 指针方法I

```go
type Setter interface {
    Set(string)
}

func FromStrings[T Setter](s []string) []T {
    result := make([]T, len(s))
    for i, v := range s {
        result[i].Set(v)
    }
    return result
}
```

----------

## 指针方法II

```go
type Settable int

func (p *Settable) Set(s string) {
    i, _ := strconv.Atoi(s)
    *p = Settable(i)
}

func F() {
    // INVALID
    nums := FromStrings[Settable]([]string{"1", "2"})
    ...
}
```

----------

## 指针方法III

```go
type Setter[B any] interface {
    Set(string)
    *B // 非接口类型元素
}

func FromStrings[T any, PT Setter[T]](s []string) []T {
    result := make([]T, len(s))
    for i, v := range s {
        p := PT(&result[i])
        p.Set(v)
    }
    return result
}
```

----------

## 类型转换

```go
type Integer interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 |
        ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type Number interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 |
        ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | float32 | float64
}

func Convert[To Integer, From Number](from From) To {
    to := To(from)
    if From(to) != from {
        panic("conversion out of range")
    }
    return to
}
```

----------

## 反射

```go
fmt.Println(reflect.TypeOf(List[int]{1}))
// 运行输出 List[int]
```

----------

## 存在的问题：返回零值

### 语法上不存在通用的零值表示

```go
func ZeroVal[T any]() T {
    var zero T
    return zero
}
```

----------

## 存在的问题：识别泛型类型I

```go
type Float interface {
    ~float32 | ~float64
}

func NewtonSqrt[T Float](v T) T {
    var iterations int
    switch (interface{})(v).(type) {
    case float32:
        iterations = 4
    case float64:
        iterations = 5
    default:
        panic(fmt.Sprintf("unexpected type %T", v))
    }
    // Code omitted.
}
```

----------

## 存在的问题：识别泛型类型II

```go
type MyFloat float32
NewtonSqrt(MyFloat(64)) // panic
```

----------

## 存在的问题：不支持参数化的方法

```go
type S struct{}

func (S) Identity[T any](v T) T { return v }
```

----------

## 不支持的特性

- 泛型类型特化
- 基于泛型的元编程
- 高级抽象
- 协变和逆变
- 柯里化
- 可变类型参数
- 参数化非类型值

----------

## 参考文献

- <https://go.dev/doc/tutorial/generics>
- <https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md>
