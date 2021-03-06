# Go语言底层探究-基础篇

## 前言

本文带大家深入Go语言的底层，探究Go程序的运行原理。

## 准备工作

本篇内容包含大量实验，大家可以事先准备下本地的实验环境，边看边试验效果更佳。

- Linux系统环境
真机、虚拟机、WSL2都可，且需要安装Docker。

- 代码示例
请将代码示例[examples](examples)下载到本地，后面的实验中，以此目录作为工作目录。

- Go编译器
这里使用的Docker镜像，**golang:1.16-buster**, **golang:1.17-buster**，可以提前pull下来。  

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
objdump（容器环境自带）

## 内存布局

话题开始之前我们先回顾一些基础知识。

### 虚拟内存

虚拟内存是计算机系统内存管理的一种技术。它使得应用程序认为自己拥有连续可用的内存（一个连续完整的地址空间），而实际上物理内存通常被分隔成多个内存碎片，还有部分暂时存储在外部磁盘上，在需要时进行数据交换。  
如下图中所示，左边是应用程序的角度看到的内存空间，右边是在物理内存以及磁盘上上可能的空间分布。本文中只关注进程的虚拟内存空间，所以后文提到的“**内存**”皆指虚拟内存。  
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

上面展示的是进程的内存布局，**Start Addr** 和 **End Addr** 为在内存中的开始地址和结束地址，**Size** 为数据大小（Byte），**objfile** 为内存映射的文件（如果存在的话则展示文件路径，其它带方括号的名称含义暂且忽略），**Offset** 为对应映射的文件中的偏移位置（未映射到文件的**Offset**为**0x0**）。  

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

可以看到这个地址来源于可执行文件app的 **.text** 段，这个用于存储程序指令的段（也就是Go代码编译成的机器指令）。联系前文的内存布局，也可以看到这个函数在内存空间中的位置 **0x400000-0x498000** 地址段。  

再来看一下示例项目 **memory_layout** 中函数 **sum** 的参数 **a** 和 **b**。  
执行以下gdb命令：

```text
p &a # 打印a的地址
# 输出 (int *) 0xc000092f38
p &b # 打印b的地址
# 输出 (int *) 0xc000092f40
```

联系前文的内存布局，可以看到这两个参数的地址位于 **0xc000000000-0xc004000000** 地址段中，这个地址段包含了Go程序进程中的调用栈。  

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

#### 调用栈

调用栈（Call stack）是进程内存中的一片连续的区域，发生函数的调用的时候，程序会在这片区域上申请一段空间用以存放参数、局部变量等信息，函数返回时，会将这段空间销毁。  
程序对这片内存区域的操作是按照栈结构的“先进后出”来进行的，通常栈由内存中的高地址向低地址增长，即栈底位于高地址处，栈顶位于低地址处，栈增长的行为叫做“压栈/入栈”（push），栈减小的行为叫做“弹栈/出栈"（pop）。  
以下为栈空间的示意图：  
![调用栈](stack.png)

#### 寄存器

寄存器（Register）是CPU用来暂存指令、数据和地址的存储器，容量非常有限，但是读写速度很快。  
CPU中的寄存器有很多种，不同的CPU架构的寄存器种类和数量也各有不同，这里以x86_64为例介绍一些常用的寄存器。  

- 数据寄存器
用来存放操作数、结果和信息的多个寄存器，如AX、BX、CX、DX等。

- 指令指针寄存器IP
又称作程序计数器（PC），用来储存要执行的下一条指令的地址，每执行一条指令后IP寄存器的值都会变化。

- 栈指针寄存器SP
用来存储调用栈内存区域的地址，总是指向栈顶，可以通过对SP的偏移操作来实现压栈和弹栈。

- 基指针寄存器BP
又称作帧指针（Frame Pointer），在栈中辅助寻址用，通常保存的是函数临时变量区域的起始地址。

64位CPU的单个寄存器最多可以存储8字节的数据，对寄存器的访问支持指定位宽：8位、16位、32位、64位。后文中你可能会在汇编指令中看到如al、ah、ax、eax、rax这样的寄存器名称，这些都是对AX寄存器的不同位宽和区域的访问，注意并不是所有寄存器都支持这样的访问。

#### 栈帧

栈帧也叫过程活动记录，是编译器用来实现过程/函数调用的一种数据结构。  
每一次的函数调用，都会在调用栈上维护一个栈帧（Stack Frame），每个栈帧通常包括：  

- 传递给函数的参数和返回值。
- 函数的返回到调用者的指令地址（return address）。
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
输出相对于断点处的栈帧信息，**n=0** 代表断点处所在的函数，**n=1** 则是更外一层的函数（即caller），以此类推。
- Stack frame at 0xnnnnnn
栈帧的起始地址，栈帧从此地址开始往低地址处延申。
- rip = 0xnnnnn in main.foo (...); saved rip = 0xnnnnn
**rip** 就是IP寄存器（64位值)，这里指在此函数中要执行的下一条指令（后面跟着源代码中的位置），可能是断点处接下来的指令（**frame 0** 的函数），也可能是调用其它函数后的下一条指令。 **saved rip** 是指Caller调用该函数后的下一条指令，可以观察下上面的三个栈帧的 **rip** 和 **saved rip** 的值，会发现 **saved rip** 其实就是caller的 **rip** 。
- called by frame at 0xnnnnnnnnn, caller of frame at 0xnnnnnnnnn
**called by frame at 0xnnnnnnnnn**是指该函数的caller的栈帧位置，**caller of frame at 0xnnnnnnnnn**指的是该函数中发生函数调用时callee的栈帧位置。
- Arglist at 0xnnnnnnnnn, args: ...
看描述是参数列表的起始地址，但其实并不是，要往高地址偏移一段跳过 **saved rip** 部分才是该函数的参数列表。
- Locals at 0xnnnnnnnnn, Previous frame's sp is 0xnnnnnnnnn
**Locals at 0xnnnnnnnnn**是局部变量的起始地址，这里的值跟**Arglist**一样了，我猜可能是gdb分辨不出来实际局部变量的位置。**Previous frame's sp**指前一个栈帧的栈顶也就是当前栈帧的起始地址。
- Saved registers: rip at 0xnnnnnnnnn
在栈帧中保存的寄存器值所在的地址，这里的rip指的就是上面的 **saved rip**，这个 **saved rip** 我们一般称之为 **return address**。

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
![调用栈结构](call-stack.png)

上面图中的caller BP指的是caller中BP寄存器的值，大家可以留意下图中的caller BP指向的位置。图中需要留意的是，函数的参数和局部变量在调用栈上的排列顺序，这属于Go语言中的一种调用约定。

## 反汇编

我们编译Go代码得到的可执行文件中的 **.text** 段中的内容是给CPU执行的二进制机器指令，我们可将其反汇编为汇编指令，来探究下Go程序离机器最近时的样子。  

使用Go 1.16编译示例 **function_call**， 使用容器环境中自带的工具 **objdump** 对生成的可执行文件反汇编：

```sh
go build -v -gcflags="-N -l" -o app ./function_call
objdump -S -d ./app > app.dump
```

用文本编辑器打开生成的文件 **app.dump**，搜索字符串 **<main.main>:**，定位到出现的位置，下面隔两行找到 **func main() {** 下面的代码，直到文件末尾，下面节选出来（省略了一些辅助信息和不重要的部分）：

```text
  ...........................
  4977b3: 48 83 ec 78           sub    $0x78,%rsp
  4977b7: 48 89 6c 24 70        mov    %rbp,0x70(%rsp)
  4977bc: 48 8d 6c 24 70        lea    0x70(%rsp),%rbp

  4977c1: 48 c7 04 24 01 00 00  movq   $0x1,(%rsp)
  4977c8: 00 
  4977c9: 48 c7 44 24 08 02 00  movq   $0x2,0x8(%rsp)
  4977d0: 00 00 
  4977d2: e8 49 ff ff ff        callq  497720 <main.sumSquare>
  4977d7: 48 8b 44 24 10        mov    0x10(%rsp),%rax
  4977dc: 48 89 44 24 30        mov    %rax,0x30(%rsp)

  ...........................
  497870: 48 8b 6c 24 70        mov    0x70(%rsp),%rbp
  497875: 48 83 c4 78           add    $0x78,%rsp
  497879: c3                    retq   
```

上面的展示的是示例程序 **function_call** 中main函数的一部分反汇编内容。

### 汇编语言

汇编语言是一种低级语言，它使用助记符来代替和表示特定机器语言的操作指令。基本上汇编语言是与特定的机器语言指令集是一一对应的，本文中所采用的是**Intel x86_64**（又称**amd64**）架构的指令集。汇编语言有两大风格分别是**Intel**汇编和**AT&T**汇编。上文中使用的objdump工具得到的反汇编结果中的汇编语言是**AT&T**风格的（objdump工具默认采用AT&T汇编，也可以通过 **-M intel** 参数指定为Intel汇编）。  

回到上节最后的main函数的反汇编结果，可以看出存在一个统一的格式，以第一行为例：

- 4977b3
可执行文件 **.text** 段中指令的地址，也对应进程内存中的地址（回顾前文的内存布局一节），注意这里是16进制的。
- 48 83 ec 78
对应的机器指令16进制码，空格分隔的每一个数字是一字节，这条指令占四字节。
- sub $0x78,%rsp
这个才是真正的汇编指令了，语法大致是：操作指令 [操作数,...]，**sub**就是操作指令，**$0x78,%rsp**则是两个操作数。  
操作指令有很多种，取决于架构支持的指令集数量，常用的也就几十种。
每种操作指令各自都有操作数的要求，大多是0~2个。  
操作数可能是（包括但不限于）：  
  - 立即数
  **$**前缀的字面量的常数如：$0x1。
  - 寄存器
  **%**前缀的寄存器名称如：%rsp。
  - 直接寻址
  **FOO** 或者 **\$FOO** ，FOO指的是一个全局变量，带$的表示直接引用地址，不带的则是取内存的值。
  - 间接寻址
  格式为：immed32(basepointer, indexpointer, indexscale)，表示的地址为：immed32 + basepointer + indexpointer × indexscale。这里immed32是32位的立即数，basepointer是基础指针，indexpointer是索引指针，indexscale是索引的缩放值即倍数，basepointer和indexpointer一般是寄存器。

汇编语言暂时就讲这些，由于本文目的不是汇编教学，因此不会进行全面地介绍，有兴趣的朋友可以自行找相关资料学习。

### 汇编分析

我们大致分析一下上面dump出的汇编代码。  
下面是示例程序 **function_call** 种main函数里调用sumSquare函数的相关指令，为方便理解加上了一些注释：

```asm
  ; ...........................
  sub    $0x78,%rsp                  ; 用rsp的值减去0x78并存放到rsp里。
                                     ; 相当于将SP向低地址移动了0x78字节，
                                     ; 也就是调用栈的栈顶增长了0x78字节。

  mov    %rbp,0x70(%rsp)             ; 将rbp的值存放到内存中rsp+0x70地址处（相对寻址）。
                                     ; 相当于在距离栈顶0x70字节处存储了当前BP的值（即caller BP），
                                     ; 这个BP值位于内存中 (rsp+0x78, rsp+0x70] 区间内。

  lea    0x70(%rsp),%rbp             ; 将rsp+0x70地址处的内存值存放到rbp中。即修改了当前的BP。

  movq   $0x1,(%rsp)                 ; 将立即数0x1存放到内存中rsp地址处，
                                     ; 位于内存中 (rsp+0x8, rsp] 区间内。
                                     ; 该值是main.sumSquare的第一个参数。

  movq   $0x2,0x8(%rsp)              ; 将立即数0x2存放到内存中rsp+0x8地址处，
                                     ; 位于内存中 (rsp+0x10, rsp+0x8] 区间内。
                                     ; 该值是main.sumSquare的第二个参数。

  callq  497720 <main.sumSquare>     ; 调用内存中0x497720地址处的函数。同时将rsp增加0x8，
                                     ; 将rip的值存放到内存中rsp地址处，
                                     ; 相当于把当前IP（下一条指令的地址）压入栈顶（即return address）,
                                     ; 再把IP改为497720。
                                     ; 后面跟着的是该处的符号描述即main.sumSquare。

  mov    0x10(%rsp),%rax             ; 将内存中rsp+0x10地址处的值存放到rax中，
                                     ; rsp+0x10地址处存储着main.sumSquare的返回值。

  mov    %rax,0x30(%rsp)             ; 将rax的值存放到内存中rsp+0x30地址处。
                                     ; 这里是给后面省略的调用fmt.Println部分准备的。

  ; ...........................

  mov    0x70(%rsp),%rbp             ; 将内存中rsp+0x70地址处的值存放到rbp中，
                                     ; 这里呼应开头第二行的指令mov %rbp,0x70(%rsp)，
                                     ; 也就是恢复BP到原值（caller BP）。

  add    $0x78,%rsp                  ; 用rsp的值加上0x78并存放到rsp里。
                                     ; 相当于将SP向高地址移动了0x78字节，
                                     ; 即调用栈的栈顶减小了0x78字节（呼应第一行指令sub $0x78,%rsp）。

  retq                               ; 表示main.main函数返回，
                                     ; 这个行为会将IP的值改为main.main函数caller（即rumtime.main）
                                     ; 中的return address，并且将该return address从栈顶弹出。
```

以上汇编代码中，有个隐含的行为是：每一行指令执行后，IP寄存器都会自动增加（指向下一行指令的地址）。  

接下来我们再看看main.sumSquare函数的汇编代码，按照前文的方法，在 **app.dump** 中搜索字符串 **<main.sumSquare>:**，就能找到对应的dump信息。这里我们直接节选汇编代码部分（省略一些不重要的内容，上面汇编代码中注释过的相似内容，下面就不再赘述了）：

```asm
  sub    $0x38,%rsp         ; 栈顶增长0x38字节。
  mov    %rbp,0x30(%rsp)    ; 保存caller BP。
  lea    0x30(%rsp),%rbp    ; 修改当前BP。

  movq   $0x0,0x50(%rsp)    ; 将(rsp+0x58, rsp+0x50]区间置空。
                            ; 这里的rsp+0x50等价于main.main函数中的rsp+0x10，
                            ; 也就是main.sumSquare的返回值。
  
  mov    0x40(%rsp),%rax    ; 这里的rsp+0x40等价于main.main函数中的rsp，
                            ; 也就是main.sumSquare的第一个参数。
  
  mov    0x40(%rsp),%rcx    ; 这里引入一个新的寄存器rcx用于后面乘法运算和存放结果。
  
  imul   %rax,%rcx          ; 将rax与rcx的值相乘，结果存放于rcx中。即 sqa := a * a 。
  
  mov    %rcx,0x20(%rsp)    ; 这里rsp+0x20处的值作为局部变量sqa。
  
  mov    0x48(%rsp),%rax    ; 这里rsp+0x48等价于main.main函数中的rsp+0x8，
                            ; 也就是main.sumSquare的第二个参数。
  
  mov    0x48(%rsp),%rcx    ; 这里复用寄存器rcx用于后面乘法运算和存放结果。
  
  imul   %rax,%rcx          ; 将rax与rcx的值相乘，结果存放于rcx中。即 sqb := b * b 。
  
  mov    %rcx,0x18(%rsp)    ; 这里rsp+0x18处的值作为局部变量sqb。
  
  mov    0x20(%rsp),%rax    ; 将rsp+0x20地址处的内存值存放到rax中。
  
  mov    %rax,(%rsp)        ; 这里rsp处的值作为main.sum的第一个参数。
  
  mov    %rcx,0x8(%rsp)     ; 这里rsp+0x8处的值作为main.sum的第二个参数。
  
  callq  497700 <main.sum>  ; 调用0x497700地址处的函数 main.sum。
  
  mov    0x10(%rsp),%rax    ; 这里rsp+0x10处的值为main.sum的返回值。
  
  mov    %rax,0x28(%rsp)    ; 此处同样为main.sum的返回值，是冗余的临时变量。
  
  mov    %rax,0x50(%rsp)    ; 这里rsp+0x50为main.sumSquare的返回值。

  mov    0x30(%rsp),%rbp    ; 恢复BP为原值caller BP。
  add    $0x38,%rsp         ; 栈顶减小0x38字节。
  retq                      ; main.sumSquare函数返回。
```

上面的汇编代码中，可以看出main.sumSquare函数与main.main函数的汇编代码的开头和结尾非常的相似，都包含栈增长和减小、保存caller BP和恢复BP等操作。  
留意下同一个内存地址在main.sumSquare(callee)中与main.main(caller)中相对SP的偏移量，可以用此公式换算：  
> offset(caller) = offset(callee) - stacksize(callee) - size(return address)  
比如main.sumSquare中的rsp+0x40地址与main.main函数中的rsp地址中相对rsp的偏移量关系：
> 0 = 0x40 - $0x38 - 0x8

大家如果结合前面通过gdb信息描绘出的调用栈结构图来分析这节的汇编代码，会有更深刻的体会。  

最后我们再看看main.sum函数的汇编代码：

```asm
  movq   $0x0,0x18(%rsp)    ; 将内存中(0x10, 0x18]区间置空。
                            ; 这里rsp+0x18等价于main.sumSquare函数中的rsp+0x10地址，
                            ; 作为main.sum函数的返回值。

  mov    0x8(%rsp),%rax     ; 这里的rsp+0x8等价main.sumSquare函数中的rsp地址，
                            ; 作为main.sum的第一个参数。

  add    0x10(%rsp),%rax    ; 这里的rsp+0x10等价于main.sumSquare函数中的rsp+0x8地址，
                            ; 作为main.sum的第二个参数，与rax中的第一个参数相加，即：a + b。

  mov    %rax,0x18(%rsp)    ; 将上面加法运算得到的结果存放到返回值的地址处：rsp+0x18。
  retq                      ; main.sum函数返回。
```

应该会发现main.sum函数的汇编代码相比main.main和main.sumSquare函数简单了很多。main.sum里没有了对调用栈的操作，因为main.sum中不存在临时变量，使用的参数和返回值又都在main.sumSquare的栈帧中，而且这里也没有了对BP的保存和恢复操作（后面再说原因），因此main.sum的栈帧中就仅剩下return address了，return address的压栈是call指令自动完成的。

### Go语言的函数调用约定

从前文的一系列示例分析中，我们可以总结出一个通用的调用栈结构：

```text

                                       caller                                                                                 
       +---------------------->  +------------------+                                                                         
       |                         |  return address  |                                                                         
       |                         --------------------                                                                         
       |                         |                  |                                                                         
       |                         | caller parent BP |                                                                         
       |                         --------------------                                                                         
       |                         |                  |                                                                         
       |                         |   Local Var0     |                                                                         
       |                         --------------------                                                                         
       |                         |                  |                                                                         
       |                         |   .......        |                                                                         
       |                         --------------------                                                                         
       |                         |                  |                                                                         
       |                         |   Local VarN     |                                                                         
                                 --------------------                                                                         
 caller stack frame              |                  |                                                                         
                                 |   callee arg2    |                                                                         
       |                         |------------------|                                                                         
       |                         |                  |                                                                         
       |                         |   callee arg1    |                                                                         
       |                         |------------------|                                                                         
       |                         |                  |                                                                         
       |                         |   callee arg0    |                                                                         
       +---------------------->  ----------------------------------------------+    <-------------------------------+         
                                 |                  |                          |                                    |         
                                 |  return address  |  parent return address   |                                    |         
                                 +------------------+---------------------------                                    |         
                                                    |                          |                                    |         
                                                    |       caller BP          |                                    |         
                                                BP  ----------------------------                                    |         
                                                    |                          |                                    |         
                                                    |     Local Var0           |                                    |         
                                                    ----------------------------                                    |         
                                                    |                          |                                              
                                                    |     Local Var1           |                                              
                                                    ----------------------------                            callee stack frame
                                                    |                          |                                              
                                                    |       .....              |                                              
                                                    ----------------------------                                    |         
                                                    |                          |                                    |         
                                                    |     Local VarN           |                                    |         
                                                SP  ----------------------------                                    |         
                                                    |                          |                                    |         
                                                    |                          |                                    |         
                                                    |                          |                                    |         
                                                    |                          |                                    |         
                                                    |                          |                                    |         
                                                    +--------------------------+    <-------------------------------+         
                                                                                                                              
                                                              callee
```

Go语言的函数调用包含以下过程：

1. 如果callee存在参数，那么将callee的参数值按一定次序放入caller的栈帧中，顺序为Arg(n)、Arg(n-1)、Arg(n-2)...Arg(0)
2. 执行call指令，将return address压入栈顶，IP改为callee的地址
3. 如果callee中存在临时变量或者其它函数调用的参数，那么将SP向低地址偏移，使调用栈增长足够的空间作为callee的栈帧
4. 如果步骤3中调用栈有增长，那么保存当前BP（caller BP）到callee的栈帧中，修改当前BP为SP的值
5. 如果callee中存在临时变量，那么将变量值按一定次序放入caller的栈帧中，顺序为Var(0)、Var(1)、Var(2)...Var(n)
6. 如果callee中有调用其它函数，那么回到步骤1，注意回到1后的callee指代的是新的函数，caller指代的是前面的callee
7. 如果callee中没有其它调用了，那么执行完之后，将返回值放入caller的栈帧（如果有的话）
8. 恢复当前BP为caller BP（如果栈帧中保存了caller BP的话）
9. 将SP向高地址偏移，使步骤3中增长的调用栈恢复到原本的大小
10. 执行ret指令，使return address从栈顶弹出，并将IP改为return address，后续将回到caller继续执行

调用约定是一种定义函数从调用处接受参数以及返回结果的方法的约定。比如C++语言，就有__cdecl、__stdcall、__fastcall、naked、__pascal、__thiscall等多种调用约定。  
从Go1.12开始，Go编译器中定义了两种调用约定 **ABI0** 和 **ABIInternal** 。**ABI0**为旧有的调用约定，**ABIInternal**是不稳定的，会随着版本迭代发生变化的调用约定。  
**ABIInternal**用于Go语言定义的函数，而**ABI0**用于在汇编中定义的函数，对于跨ABI调用函数的情况，编译器会自动分析并提供相应的ABI包装器来实现透明的调用。
Go语言的调用约定，在Go1.16（包含）之前是基于栈的调用，也就是上文中描述的调用方式，是将函数参数和返回值都放在栈上来进行传递。  
Go1.17开始启用基于寄存器的调用，此版本中 **ABI0** 指原来的基于栈的调用约定，**ABIInternal** 就是基于寄存器的调用约定。

### 基于寄存器的调用

Go1.17版本开始引入了基于寄存器的调用。相比原来的单纯基于栈的调用，性能有了一定提升。  
该调用方式是使用栈和寄存器组合传递函数参数和返回值。每个参数或返回值要么完全在寄存器中传递，要么完全在栈中传递。因为访问寄存器通常比访问栈要快，所以参数和返回值会优先在寄存器中传递。但是任何包含非平凡数组或不完全适合剩余可用寄存器的参数或结果都会在栈上传递。  
每个体系结构都定义了一个整数寄存器序列和一个浮点寄存器序列。在高层次上，参数和结果被递归地分解为基本类型的值，并分配给对应序列中的寄存器。  
参数和返回值可以共享相同的寄存器，但不共享相同的栈空间。除了在栈上传递的参数和返回值之外，调用者还在栈上为所有基于寄存器的参数保留溢出空间。  
编译器通过一套算法来决定将参数和返回值分配在寄存器上还是栈上、决定如何分配寄存器。详情请阅读官方文档：[https://tip.golang.org/src/cmd/compile/abi-internal](https://tip.golang.org/src/cmd/compile/abi-internal)。  

我们可以尝试下使用Go1.17来生成示例程序 **function_call** 的汇编结果：

```sh
go build -v -gcflags="-N -l" -o app ./function_call
objdump -S -d ./app > app.dump
```

下面给出和前文相同形式的输出节选：

main函数：

```asm
  sub    $0x60,%rsp                  ; 扩展调用栈
  mov    %rbp,0x58(%rsp)             ; 保存caller BP
  lea    0x58(%rsp),%rbp             ; 更新BP

  mov    $0x1,%eax                   ; 第一个参数放在AX中
  mov    $0x2,%ebx                   ; 第二个参数放在BX中
  callq  47e320 <main.sumSquare>     ; 调用sumSquare
  mov    %rax,0x18(%rsp)             ; 从AX中取出sumSquare的返回值放入栈中，用于后续fmt.Println的相关调用
  ; ...........................

  mov    0x58(%rsp),%rbp             ; 恢复BP
  add    $0x60,%rsp                  ; 回收调用栈
  retq                               ; 函数返回
```

sumSquare函数：

```asm
  sub    $0x38,%rsp               
  mov    %rbp,0x30(%rsp)
  lea    0x30(%rsp),%rbp

  mov    %rax,0x40(%rsp)        ; 将AX的值放入栈中（参数1的溢出空间）
  mov    %rbx,0x48(%rsp)        ; 将BX的值放入栈中（参数2的溢出空间）
  movq   $0x0,0x10(%rsp)        ; 初始化返回值空间（溢出空间）
  mov    0x40(%rsp),%rcx        ; 将栈上的值放入CX中（来自参数1）
  mov    0x40(%rsp),%rdx        ; 将栈上的值放入DX中（来自参数1）
  imul   %rcx,%rdx              ; CX与DX相乘，结果放入DX中
  mov    %rdx,0x20(%rsp)        ; 将DX的值放入栈中（局部变量1）
  mov    0x48(%rsp),%rbx        ; 将栈上的值放入BX中（来自参数2）
  mov    0x48(%rsp),%rcx        ; 将栈上的值放入CX中（来自参数2）
  imul   %rcx,%rbx              ; CX与BX相乘，结果放入BX中（作为sum的参数2）
  mov    %rbx,0x18(%rsp)        ; 将BX的值放入栈中（局部变量2）
  mov    0x20(%rsp),%rax        ; 将栈上的值放入AX中（局部变量1，作为sum的参数1）
  callq  47e2e0 <main.sum>      ; 调用sum函数
  mov    %rax,0x28(%rsp)        ; 将AX的值放入栈中（sum的返回值）
  mov    %rax,0x10(%rsp)        ; 将AX的值放入栈中（返回值的溢出空间）

  mov    0x30(%rsp),%rbp
  add    $0x38,%rsp
  retq
```

sum函数：

```asm
  sub    $0x10,%rsp
  mov    %rbp,0x8(%rsp)
  lea    0x8(%rsp),%rbp

  mov    %rax,0x18(%rsp)         ; 将AX的值放入栈中（参数1的溢出空间）
  mov    %rbx,0x20(%rsp)         ; 将BX的值放入栈中（参数2的溢出空间）
  movq   $0x0,(%rsp)             ; 初始化返回值空间（溢出空间）
  mov    0x18(%rsp),%rax         ; 将栈上的值放入AX中（来自参数1）
  add    0x20(%rsp),%rax         ; 将栈上的值（来自参数2）与AX（来自参数1）相加放入AX中（返回值）
  mov    %rax,(%rsp)             ; 将AX放入栈中（返回值的溢出空间）

  mov    0x8(%rsp),%rbp
  add    $0x10,%rsp
  retq
```

分析下上面的汇编代码可以看到，虽然用了寄存器传递参数和返回值，但是也在栈上分配了参数的溢出空间，而且内存布局较之前基于栈的调用也发生了细微的变化。  
本文的实验中我们都是关闭了优化和内联（使用了 **-N -l** 选项），有兴趣的朋友可以试试启用优化，仅关闭内联（使用 **-l** 选项），来观察下汇编结果有哪些不同。

## Go汇编语言

下面使Go程序的主要编译过程：  
> 解析代码成AST -> 生成SSA IR -> 汇编 -> 链接 -> 生成可执行文件  

上面的 **汇编** 阶段生成汇编代码，得到的汇编结果中的指令地址是从0开始的偏移量的形式。  
**链接** 阶段会对汇编结果做调整，将地址信息从偏移量转换为可执行文件中的逻辑地址。  

Go语言编译器工具链中有实现自己的汇编器，所使用的汇编语言来自于[Plan9](https://en.wikipedia.org/wiki/Plan_9_from_Bell_Labs)汇编器。Go语言的汇编语法风格与常见的Intel汇编和AT&T汇编都有一定区别，我们可以使用以下两种方式得到Go汇编代码（以示例代码 **function_call** 为例）：

```sh
go tool compile -S -N -l function_call/main.go > app.go.dump
```

```sh
go tool objdump -S app > app.go.dump
```

这两种方式输出的文件与前文中直接使用objdump输出的格式是相似的。只不过 **go tool compile -S** 得到的只有main.go中代码对应的汇编结果，而 **go tool objdump -S** 得到的是可执行文件app里 **.text** 段的反汇编信息。而前者的dump文件由于其汇编结果尚未经过链接，因此其中的地址列是从 **0x0000 00000** 开始的偏移量，并且包含一些函数栈相关的信息，以及一些伪指令；后者的dump文件中的汇编代码已经经过了链接阶段，其中的地址列与直接objdump得到的是相同的，该dump文件中的内容精简了很多，基本和objdump出的信息差不多。  

这里我们采用 **go tool compile -S** 的汇编结果来进行后续的讲解。下面贴出 **function_call** 示例的Go汇编结果，同样是省略一些不重要的信息，大家可以与前文[汇编分析](#汇编分析)部分的AT&T汇编代码对照着看看：  

main函数：

```asm
; ........................
(function_call/main.go:15)  SUBQ  $120, SP
(function_call/main.go:15)  MOVQ  BP, 112(SP)
(function_call/main.go:15)  LEAQ  112(SP), BP

(function_call/main.go:16)  MOVQ  $1, (SP)
(function_call/main.go:16)  MOVQ  $2, 8(SP)
(function_call/main.go:16)  CALL  "".sumSquare(SB)
(function_call/main.go:16)  MOVQ  16(SP), AX
(function_call/main.go:16)  MOVQ  AX, "".ret+48(SP)
; ........................

(function_call/main.go:18)  MOVQ  112(SP), BP
(function_call/main.go:18)  ADDQ  $120, SP
(function_call/main.go:18)  RET
```

sumSquare函数：

```asm
(function_call/main.go:9)   SUBQ  $56, SP
(function_call/main.go:9)   MOVQ  BP, 48(SP)
(function_call/main.go:9)   LEAQ  48(SP), BP

(function_call/main.go:9)   MOVQ  $0, "".~r2+80(SP)
(function_call/main.go:10)  MOVQ  "".a+64(SP), AX
(function_call/main.go:10)  MOVQ  "".a+64(SP), CX
(function_call/main.go:10)  IMULQ  AX, CX
(function_call/main.go:10)  MOVQ  CX, "".sqa+32(SP)
(function_call/main.go:11)  MOVQ  "".b+72(SP), AX
(function_call/main.go:11)  MOVQ  "".b+72(SP), CX
(function_call/main.go:11)  IMULQ  AX, CX
(function_call/main.go:11)  MOVQ  CX, "".sqb+24(SP)
(function_call/main.go:12)  MOVQ  "".sqa+32(SP), AX
(function_call/main.go:12)  MOVQ  AX, (SP)
(function_call/main.go:12)  MOVQ  CX, 8(SP)
(function_call/main.go:12)  CALL  "".sum(SB)
(function_call/main.go:12)  MOVQ  16(SP), AX
(function_call/main.go:12)  MOVQ  AX, ""..autotmp_5+40(SP)
(function_call/main.go:12)  MOVQ  AX, "".~r2+80(SP)

(function_call/main.go:12)  MOVQ  48(SP), BP
(function_call/main.go:12)  ADDQ  $56, SP
(function_call/main.go:12)  RET
```

sum函数：

```asm
(function_call/main.go:5)  MOVQ  $0, "".~r2+24(SP)
(function_call/main.go:6)  MOVQ  "".a+8(SP), AX
(function_call/main.go:6)  ADDQ  "".b+16(SP), AX
(function_call/main.go:6)  MOVQ  AX, "".~r2+24(SP)
(function_call/main.go:6)  RET
```

汇编结果中贴心的给出了指令对应的源代码位置，这里也一并贴出来了，方便各位查看。

### 寄存器名称

Go汇编指令中的寄存器名称不用带前缀来表示宽度，比如AT&T汇编中的 **rax**、**eax** 等等，在Go汇编中统一写作 **AX**。  

### 指令与操作数

操作指令很多需要后缀表明操作的宽度，比如 **MOVEQ**、**SUBQ**等等。  
操作数的顺序，大部分都是用第二个操作数来存放结果，这一点跟AT&T汇编类似。了解过Intel汇编的朋友会发现刚好是相反的顺序。  

比如：

```asm
MOVQ $1,(SP)
```

对应AT&T汇编：

```asm
movq $0x1,(%rsp)
```

注意这里Go汇编的的立即数 **$1** 是十进制的。

### 伪寄存器

Go汇编中提供了几个伪寄存器：

- PC: 程序计数器。其实就是指IP寄存器，比较少用。
- SB: 静态基指针。用于引用全局的符号，使用**package.symbol(SB)**的方式来引用符号的地址，**"".symbol(SB)**表示当前包的符号。比如**CALL "".sum(SB)**，对应的AT&T汇编是**callq 497700**，汇编器会将指令中的 **"".sum(SB)** 转换为sum函数的地址偏移量，编译器在链接阶段会将其转换成逻辑地址**0x497700**。

### 通过SP寻址

通过**package.symbol+offset(SP)**的方式来进行寻址，比如 **"".sqa+32(SP)** 表示局部变量sqa，这里的**package.symbol**并没有什么实际用途，只是方便阅读。还有一种通过**offset(SP)**（不带symbol）的方式来寻址。

### 间接寻址

Go汇编中还会出现这种间接寻址方式：

```asm
offset(basepointer)(index*scale) ; address = basepointer + index * scale + offset (scale取值1、2、 4、8)
```

例如：

```asm
64(DI)(BX*2) ; address = DI + BX * 2 + 64
```

这个与前文提到过的AT&T汇编的间接寻址方式是等价的。

### 手写汇编

前文的对Go汇编的介绍和代码示例，都是基于**go tool compile -S**和**go tool objdump**得到的汇编结果来讲的，如果是手写汇编，会有一些区别，本文不是汇编教程，就不详细介绍了，有兴趣的可以看官方文档了解一下：[https://go.dev/doc/asm](https://go.dev/doc/asm)。

## 参考资料 & 扩展阅读

- [https://go.dev/doc/gdb](https://go.dev/doc/gdb)
  Go语言对于GDB的扩展。
- [https://tip.golang.org/src/cmd/compile/abi-internal](https://tip.golang.org/src/cmd/compile/abi-internal)
  Go语言内部ABI规范。
- [https://go.googlesource.com/proposal/+/master/design/27539-internal-abi.md](https://go.googlesource.com/proposal/+/master/design/27539-internal-abi.md)
  关于Go语言两种调用约定的提案。
- [https://go.googlesource.com/proposal/+/master/design/40724-register-calling.md](https://go.googlesource.com/proposal/+/master/design/40724-register-calling.md)
  关于Go语言基于寄存器的调用提案。
- [https://go.dev/doc/asm](https://go.dev/doc/asm)
  Go汇编官方文档。
- [https://github.com/cch123/asmshare/blob/master/layout.md](https://github.com/cch123/asmshare/blob/master/layout.md)
  Go汇编教程。
- [https://github.com/cch123/golang-notes/blob/master/assembly.md](https://github.com/cch123/golang-notes/blob/master/assembly.md)
  Go汇编完全解析。

## 相关讨论

[点击进入讨论区](https://github.com/JinWuZhao/technique-sharing/issues/2)
