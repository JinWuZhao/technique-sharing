# Go语言底层探究

## 前言

// TODO

## 准备工作

本篇内容包含大量实验，读者们可以事先准备下本地的实验环境，边看边试验效果更佳。

- Linux系统环境
真机、虚拟机、WSL2都可，且需要安装Docker。

- 代码示例
请将代码示例[examples](examples)下载到本地，后面的实验中，以此目录作为工作目录。

- Go编译器
这里使用的Docker镜像，golang:1.16-buster, golang:1.17-buster, golang:1.18-buster，可以提前pull下来。  

启动命令参考，后文启动Go开发环境皆以此方式执行：

```sh
docker run --rm -it -v $PWD:/go/src/examples -w /go/src/examples --security-opt seccomp=unconfined golang:1.16-buster bash
```

命令中的参数 *--security-opt seccomp=unconfined* 是提供权限给gdb可以关闭内存地址随机化，方便我们做研究。

- GDB
在容器中安装：

```sh
apt-get update
apt-get install gdb
```

- 其它工具
objdump、readelf（容器环境自带）

## 内存布局

话题开始之前我们先回顾一些基础知识。

### 虚拟内存

虚拟内存是计算机系统内存管理的一种技术。它使得应用程序认为自己拥有连续可用的内存（一个连续完整的地址空间），而实际上物理内存通常被分隔成多个内存碎片，还有部分暂时存储在外部磁盘上，在需要时进行数据交换。  
如下图中所示，左边是应用程序的角度看到的内存空间，右边是在物理内存以及磁盘上上可能的空间分布。本文中只关注应用进程的虚拟内存空间，所以后文提到的“内存”皆指虚拟内存。  
![虚拟内存](virtual-memory.png)

### Go程序的内存分布

操作系统中，一个应用程序启动了之后，其进程的内存空间中，数据是怎样分布的呢？我们通过一个实验来亲眼看看。  

- 启动Go1.16开发环境
- 编译代码示例

```sh
# 注意这里的 -gcflags="-l" 用于关闭内联
go build -v -gcflags="-l" -o app ./memory_layout
```

- 使用gdb调试程序

启动gdb

```sh
gdb app
```

以下为gdb命令：

```text
b 10 # 在第10行设置断点
r # 运行程序

# 触发断点...

i proc mapping # 打印进程的内存布局，相当于"cat /proc/<pid>/maps"
```

执行了以上命令后，可以得到类似下面的结果：

```text
Mapped address spaces:

          Start Addr           End Addr       Size     Offset objfile
            0x400000           0x498000    0x98000        0x0 /go/src/examples/app
            0x498000           0x535000    0x9d000    0x98000 /go/src/examples/app
            0x535000           0x54b000    0x16000   0x135000 /go/src/examples/app
            0x54b000           0x57e000    0x33000        0x0 [heap]
        0xc000000000       0xc004000000  0x4000000        0x0 
      0x7fffd12e9000     0x7fffd369a000  0x23b1000        0x0 
      0x7fffd369a000     0x7fffe381a000 0x10180000        0x0 
      0x7fffe381a000     0x7fffe381b000     0x1000        0x0 
      0x7fffe381b000     0x7ffff56ca000 0x11eaf000        0x0 
      0x7ffff56ca000     0x7ffff56cb000     0x1000        0x0 
      0x7ffff56cb000     0x7ffff7aa0000  0x23d5000        0x0 
      0x7ffff7aa0000     0x7ffff7aa1000     0x1000        0x0 
      0x7ffff7aa1000     0x7ffff7f1a000   0x479000        0x0 
      0x7ffff7f1a000     0x7ffff7f1b000     0x1000        0x0 
      0x7ffff7f1b000     0x7ffff7f9a000    0x7f000        0x0 
      0x7ffff7f9a000     0x7ffff7ffa000    0x60000        0x0 
      0x7ffff7ffa000     0x7ffff7ffd000     0x3000        0x0 [vvar]
      0x7ffff7ffd000     0x7ffff7fff000     0x2000        0x0 [vdso]
      0x7ffffffde000     0x7ffffffff000    0x21000        0x0 [stack]
```

上面展示的是进程的内存布局，**Start Addr** 和 **End Addr** 为在内存中的开始地址和结束地址，**Size** 为数据大小（Byte），**objfile** 为内存映射的文件（如果存在的话则展示文件路径，其它带方括号的名称含义暂且忽略），**Offset** 为对应映射的文件中的偏移位置（未映射到文件的Offset为0x0）。  

示例项目 **memory_layout** 的代码中定义了一个全局变量 **gInt**，我们可以看一下它在内存中的位置。  
执行以下gdb命令：

```text
p &main.gInt # 打印gInt的地址
# 输出 (int *) 0x578288 <main.gInt>
```

联系前文的内存布局，可以发现 **main.gInt** 的地址 **0x578288** 位于 **0x54b000-0x57e000** 地址段的空间中（后面对应的objfile名称为 **heap** ）。  
再执行以下gdb命令：

```text
i symbol 0x578288 # 查询该地址对应的符号以及映射文件中的段（ELF文件中的Section）
# 输出 main.gInt in section .noptrbss of /go/src/examples/app
```

可以看到此地址来源于可执行文件app中的 **.noptrbss** 段，这个是Go编译器进行链接的时候自定义的段，意为非指针BSS段，BSS段是用于存储未初始化的全局变量的，那么这个自定义段就表示非指针的全局变量。  

示例项目 **memory_layout** 的代码中还定义了一个函数 **sum(a, b int) int**，我们看看这个函数位于哪里。  
执行以下gdb命令：

```text
p main.sum # 打印sum的信息
# 输出 {void (int, int, int)} 0x497700 <main.sum>
```

我们得到了sum函数的地址 **0x497700** ，继续查询该地址的符号：  

``` text
i symbol 0x497700
# 输出 main.sum in section .text of /go/src/examples/app
```

可以看到这个地址来源于可执行文件app的 **.text** 段，这个用于存储程序指令的段（也就是代码部分）。联系前文的内存布局，也可以看到这个函数在内存空间中的位置 **0x400000-0x498000** 地址段。  

再来看一下示例项目 **memory_layout** 中函数 **sum** 的参数 **a** 和 **b**。  
执行以下gdb命令：

```text
p &a # 打印a的地址
# 输出 (int *) 0xc000092f38
p &b # 打印b的地址
# 输出 (int *) 0xc000092f40
```

联系前文的内存布局，可以看到这两个参数的地址位于 **0xc000000000-0xc004000000** 地址段中，大家应该知道，**Go 1.17** 以前的版本中，函数的参数数据是分配在栈空间上的，这个地址段便包含了Go程序进程中的栈空间。  

gdb中还可以看到可执行文件中的各个段在内存中的分布情况：  

```text
i files
```

会得到以下信息：

```text
Symbols from "/go/src/examples/app".
Native process:
        Using the running image of child LWP 1137.
        While running this, GDB does not access memory from...
Local exec file:
        `/go/src/examples/app', file type elf64-x86-64.
        Entry point: 0x465760
        0x0000000000401000 - 0x00000000004977ca is .text
        0x0000000000498000 - 0x00000000004dbb64 is .rodata
        0x00000000004dbd00 - 0x00000000004dc42c is .typelink
        0x00000000004dc440 - 0x00000000004dc490 is .itablink
        0x00000000004dc490 - 0x00000000004dc490 is .gosymtab
        0x00000000004dc4a0 - 0x00000000005343a0 is .gopclntab
        0x0000000000535000 - 0x0000000000535020 is .go.buildinfo
        0x0000000000535020 - 0x00000000005432e4 is .noptrdata
        0x0000000000543300 - 0x000000000054aa90 is .data
        0x000000000054aaa0 - 0x00000000005781f0 is .bss
        0x0000000000578200 - 0x000000000057d510 is .noptrbss
        0x0000000000400f9c - 0x0000000000401000 is .note.go.buildid
        0x00007ffff7ffd120 - 0x00007ffff7ffd15c is .hash in system-supplied DSO at 0x7ffff7ffd000
        0x00007ffff7ffd160 - 0x00007ffff7ffd1a8 is .gnu.hash in system-supplied DSO at 0x7ffff7ffd000
        0x00007ffff7ffd1a8 - 0x00007ffff7ffd298 is .dynsym in system-supplied DSO at 0x7ffff7ffd000
        0x00007ffff7ffd298 - 0x00007ffff7ffd2f6 is .dynstr in system-supplied DSO at 0x7ffff7ffd000
        0x00007ffff7ffd2f6 - 0x00007ffff7ffd30a is .gnu.version in system-supplied DSO at 0x7ffff7ffd000
        0x00007ffff7ffd310 - 0x00007ffff7ffd348 is .gnu.version_d in system-supplied DSO at 0x7ffff7ffd000
        0x00007ffff7ffd348 - 0x00007ffff7ffd458 is .dynamic in system-supplied DSO at 0x7ffff7ffd000
        0x00007ffff7ffd458 - 0x00007ffff7ffd798 is .rodata in system-supplied DSO at 0x7ffff7ffd000
        0x00007ffff7ffd798 - 0x00007ffff7ffd7ec is .note in system-supplied DSO at 0x7ffff7ffd000
        0x00007ffff7ffd7ec - 0x00007ffff7ffd820 is .eh_frame_hdr in system-supplied DSO at 0x7ffff7ffd000
        0x00007ffff7ffd820 - 0x00007ffff7ffd910 is .eh_frame in system-supplied DSO at 0x7ffff7ffd000
        0x00007ffff7ffd910 - 0x00007ffff7ffddaa is .text in system-supplied DSO at 0x7ffff7ffd000
        0x00007ffff7ffddaa - 0x00007ffff7ffde1f is .altinstructions in system-supplied DSO at 0x7ffff7ffd000
        0x00007ffff7ffde1f - 0x00007ffff7ffde3b is .altinstr_replacement in system-supplied DSO at 0x7ffff7ffd000
```

从上面信息中可以看到 **Entry point: 0x465760** 这一行，地址 **0x465760** 为程序的入口，可以看以下这个地址的详细信息：  

```text
i symbol 0x465760
# 输出 _rt0_amd64_linux in section .text of /go/src/examples/app

list *_rt0_amd64_linux # 查看该函数的定义
# 输出结果
0x465760 is in _rt0_amd64_linux (/usr/local/go/src/runtime/rt0_linux_amd64.s:8).
3       // license that can be found in the LICENSE file.
4
5       #include "textflag.h"
6
7       TEXT _rt0_amd64_linux(SB),NOSPLIT,$-8
8               JMP     _rt0_amd64(SB)
9
10      TEXT _rt0_amd64_linux_lib(SB),NOSPLIT,$0
11              JMP     _rt0_amd64_lib(SB)

list *_rt0_amd64
# 输出结果
0x4621c0 is in _rt0_amd64 (/usr/local/go/src/runtime/asm_amd64.s:15).
10      // _rt0_amd64 is common startup code for most amd64 systems when using
11      // internal linking. This is the entry point for the program from the
12      // kernel for an ordinary -buildmode=exe program. The stack holds the
13      // number of arguments and the C-style argv.
14      TEXT _rt0_amd64(SB),NOSPLIT,$-8
15              MOVQ    0(SP), DI       // argc
16              LEAQ    8(SP), SI       // argv
17              JMP     runtime·rt0_go(SB)
18
19      // main is common startup code for most amd64 systems when using

i address runtime.rt0_go # 打印runtime.rt0_go函数的地址
# 输出 Symbol "runtime.rt0_go" is a function at address 0x4621e0.

list *0x4621e0
# 输出结果
0x4621e0 is in runtime.rt0_go (/usr/local/go/src/runtime/asm_amd64.s:91).
86
87      // Defined as ABIInternal since it does not use the stack-based Go ABI (and
88      // in addition there are no calls to this entry point from Go code).
89      TEXT runtime·rt0_go<ABIInternal>(SB),NOSPLIT,$0
90              // copy arguments forward on an even stack
91              MOVQ    DI, AX          // argc
92              MOVQ    SI, BX          // argv
93              SUBQ    $(4*8+7), SP            // 2args 2auto
94              ANDQ    $~15, SP
95              MOVQ    AX, 16(SP)

# 默认只输出10行，想查看更多，可以用以下命令设置最大显示行数为100行
set listsize 100
```

以上例子给出了一个从入口探寻go程序的执行过程的一个方法，感兴趣的朋友可自行尝试。

### Go语言函数调用过程

各位可能有了解过一些函数调用中的知识，比如函数的栈帧、栈指针、压栈和弹栈等等，这些底层的知识在很多语言中都存在，现在我们在Go程序中回顾一下。  

#### 栈

栈（Stack）是进程内存中的一片连续的区域，发生函数的调用的时候，程序会在这片区域上申请一段空间用以存放参数、局部变量等信息，函数返回时，会将这段空间销毁。  
程序对这片内存区域的操作是按照栈结构的“先进后出”来进行的，通常栈由内存中的高地址向低地址增长，即栈底位于高地址处，栈顶位于低地址处，栈增长的行为叫做“压栈/入栈”（push），栈减小的行为叫做“弹栈/出栈"（pop）。  
以下为栈空间的示意图：  
![栈](stack.png)

#### 寄存器

寄存器（Register）是CPU用来暂存指令、数据和地址的存储器，容量非常有限，但是读写速度很快。  
CPU中的寄存器有很多种，不同的CPU架构的寄存器种类和数量也各有不同，这里以x86_64为例介绍一些常用的寄存器。  

- 数据寄存器
用来存放操作数、结果和信息的多个寄存器，如AX、BX、CX、DX等。

- 指令指针寄存器IP
又称作程序计数器（PC），用来储存要执行的下一条指令的地址，每执行一条指令后IP寄存器的值都会变化。

- 栈指针寄存器SP
用来存储栈内存区域的地址，总是指向栈顶，可以通过对SP的偏移操作来表示压栈和弹栈。

- 基址寄存器BP
用于备份SP的值，在栈中辅助寻址用。

64位CPU的单个寄存器最多可以存储8字节的数据，对寄存器的访问支持指定数据宽度：8位、16位、32位、64位。后文中你可能会在汇编指令中看到如al、ah、ax、eax、rax这样的寄存器名称，这些都是AX寄存器的不同访问方式，注意并不是所有寄存器都支持这样的访问方式。

#### 栈帧

栈帧也叫过程活动记录，是编译器用来实现过程/函数调用的一种数据结构。  
每一次的函数调用，都会在调用栈上维护一个独立的栈帧（Stack Frame），每个独立的栈帧通常包括：  

- 传递给函数的参数
- 函数的返回到调用者的地址(return address)
- 临时变量空间，包含函数中的局部变量和编译器自动生成的临时数据。
- 函数调用者的BP寄存器值(caller BP)

我们来做一个实验，使用Go 1.16编译示例 **function_call**：

```sh
# 为了能让结果更清晰，我们使用 -gcflags="-N -l" 来关闭优化和内联
go build -v -gcflags="-N -l" -o app ./function_call
```

使用gdb调试该程序：

```sh
gdb app
```

在第6行设置一个断点并运行触发该断点，该断点位于sumSquare函数中：

```text
b 6
r
```

查看当前断点处的调用栈数据：

```text
i frame 0
# 输出以下内容
Stack frame at 0xc000112ec8:
 rip = 0x497709 in main.sum (/go/src/examples/function_call/main.go:6); saved rip = 0x49777f
 called by frame at 0xc000112f08
 source language unknown.
 Arglist at 0xc000112eb8, args: a=1, b=4, ~r2=0
 Locals at 0xc000112eb8, Previous frame's sp is 0xc000112ec8
 Saved registers:
  rip at 0xc000112ec0

i frame 1
# 输出以下内容
Stack frame at 0xc000112f08:
 rip = 0x49777f in main.sumSquare (/go/src/examples/function_call/main.go:12); saved rip = 0x4977d7
 called by frame at 0xc000112f88, caller of frame at 0xc000112ec8
 source language unknown.
 Arglist at 0xc000112ec0, args: a=1, b=2, ~r2=0
 Locals at 0xc000112ec0, Previous frame's sp is 0xc000112f08
 Saved registers:
  rip at 0xc000112f00

i frame 2
# 输出以下内容
Stack frame at 0xc000112f88:
 rip = 0x4977d7 in main.main (/go/src/examples/function_call/main.go:16); saved rip = 0x434f56
 caller of frame at 0xc000112f08
 source language unknown.
 Arglist at 0xc000112f00, args: 
 Locals at 0xc000112f00, Previous frame's sp is 0xc000112f88
 Saved registers:
  rip at 0xc000112f80
```

先解释下上面的内容：

- 命令 i frame n
输出相对于断点处的栈帧信息，n=0 代表断点处所在的函数，n=1 则是更外一层的函数（即caller），以此类推。
- Stack frame at 0xnnnnnn
栈帧的起始地址，栈帧从此地址开始往低地址处延申。不过习惯上通常不把这个地址作为栈帧的起始，这里我们先入乡随俗，大家阅读其它资料时留意下防止产生困惑。
- rip = 0xnnnnn in main.foo (...); saved rip = 0xnnnnn
rip就是IP寄存器（64位值)，这里指在此函数中要执行的下一条指令（后面跟着源代码中的位置），可能是断点处接下来的指令（frame 0 的函数），也可能是调用其它函数后的下一条指令。 saved rip 是指Caller调用该函数后的下一条指令，可以观察下上面的三个栈帧的rip和saved rip的值，会发现saved rip其实就是caller的rip。
- called by frame at 0xnnnnnnnnn, caller of frame at 0xnnnnnnnnn
called by frame at 0xnnnnnnnnn是指该函数的caller的栈帧位置，caller of frame at 0xnnnnnnnnn指的是该函数中发生函数调用时callee栈帧位置。
- Arglist at 0xnnnnnnnnn, args: ...
看描述是参数列表的起始地址，但其实并不是，要往高地址偏移一段跳过saved rip部分才是该函数的参数列表。
- Locals at 0xnnnnnnnnn, Previous frame's sp is 0xnnnnnnnnn
Locals at 0xnnnnnnnnn是局部变量的起始地址，这里的值跟Arglist一样了，我猜可能是gdb分辨不出来实际局部变量的位置。Previous frame's sp指前一个栈帧的栈顶也就是当前栈帧的起始地址。
- Saved registers: rip at 0xnnnnnnnnn
在栈帧中保存的寄存器值所在的地址，这里的rip指的就是上面的saved rip，这个saved rip我们一般称之为return address。

再介绍一些gdb的小技巧，以方便各位查看调用栈和寄存器的数据。

- 打印内存中的值

> x/nfu addr
含义为以f格式打印从addr开始的n个长度单元为u的内存值。参数具体含义如下:  
n：输出单元的个数，可以省略，默认为1。  
f：是输出格式。比如x是以16进制形式输出，o是以8进制形式输出等等。  
u：标明一个单元的长度。b是一个byte，h是两个byte（halfword），w是四个byte（word），g是八个byte（giant word）。  
addr: 内存地址
比如打印 **stack frame 0: 0xc000112f08** 的值：

```text
x/xg 0xc000112ec8
# 输出 0xc000112ec8:   0x0000000000000001
```

addr还可以是表达式：

```text
x/gx 0xc000112ec8+8 # 打印内存中0xc000112ec8向高地址偏移8个字节处的值，注意后面的8是十进制的
# 输出 0xc000112ed0:   0x0000000000000004
```

- 打印寄存器的值
  
```text
i register rsp # 输出单个寄存器的值
i registers # 输出当前用到的所有寄存器的值
```

使用gdb探索上面栈帧的输出信息，我们能描绘出当前调用栈的结构如下图：  
![调用栈](call-stack.png)

上面图中的caller BP指的是caller中BP寄存器的值，大家可以留意下图中的caller BP指向的位置。

## 语法、寄存器

与AT&T ASM对比，伪寄存器

## 手写汇编

实现函数

## 参数传递（栈和寄存器，参数类型）

压栈过程，参数传递方法，栈传递参数和寄存器传递参数，各参数类型的传递方法。

## 控制结构

## 泛型实现

Go 1.18Beta2

## 其他知识

sadfasdf
