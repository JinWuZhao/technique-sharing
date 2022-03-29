# 函数调用约定的实验

实验环境：amd64架构，Docker镜像golang:1.17-buster
示例代码：[main.go](main.go)

## 未开启优化（-gcflags='-N -l'）

栈内存布局：

```text
49(SP): a3 spill space
48(SP): a1 spill space

<pointer-sized alignment>

42(SP): r1.y[1]
41(SP): r1.y[0]
40(SP): r1.x

<pointer-sized alignment>

33(SP): a2[1]
32(SP): a2[0]

<pointer-sized alignment>

16(SP): BP

8(SP): len(r2) spill space
(SP): r2 spill space
```

寄存器：

```text
AX: a1 -> r2
BX: a3 -> len(r2)
```

## 开启优化后(-gcflags='-l')

栈内存布局：

```text
18(SP): r1.y[1]
17(SP): r1.y[0]
16(SP): r1.x

<pointer-sized alignment>

9(SP): a2[1]
8(SP): a2[0]

<pointer-sized alignment>
```

寄存器：

```text
AX: a1 -> r2
BX: a3 -> len(r2)
```
