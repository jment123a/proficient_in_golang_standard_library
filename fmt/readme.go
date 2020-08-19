/*
一些名词：

verb：fmt.Printf("%v", 123)中的%v，可以机翻为"动词"

操作数：fmt.Printf("%v", 123)中的123

进制前缀：
    0x或0X：16进制前缀
	   0o或0O：8进制前缀

数字字序：用于将数字转为不同进制的字符串
	   ldigits = "0123456789abcdefx"：16进制小写数字字序
	   例：想要将u=123（0111 1011）转为16进制字符串
	   for u >= 16 {
		   i--
		   buf[i] = digits[u&0xF]
		   u >>= 4
	   }
   digits[0111]（7）为8，digits[1011]（11）为b，得出7b
参数索引：用于将操作数作为宽度或精度，重新排序
    例：fmt.Printf("'%[3]*.[2]*[1]f'",1.0,2,3)
    [3]：为参数索引，为第3个操作数，后续verb会以第4个操作数为起点输出
Print.go

函数命名规则
   Print：默认输出为os.Stdout
   F开头：要求输入Writer作为参数
   S开头：返回要输出的字符串

   f结尾：要求输入格式化字符串
   ln结尾：程序结尾会输出换行符

Print调用链：
   Printf（实例化pp）
      |
   doPrintf（遍历format）
      |				   \
   printArg（输出参数）	printValue（输出反射参数）
      |				   /
   p.fmtXX（根据操作数类型检查verb）
      |
   p.fmt.fmtXX（将操作数格式化为[]byte）
      |
   p.buf.write（写入缓冲区）

基础结构：
   type pp struct {
      buf buffer //缓冲区，格式化操作都会将数据暂存在buf中，最后再写入io.Writer中
   	 arg interface{} //正在进行格式化处理的操作数
      value reflect.Value //正在进行格式化处理的需要"通过反射实现输出"的操作数
   	 fmt fmt //进行格式化操作的结构体
      ...
   }
   type fmt struct {
   	 buf *buffer //注：pp的缓冲区指针
   	 fmtFlags //注：verb设置的标志，包括宽度、精度等
      ...
   }
工作原理：创建一个pp，遍历format字符串，根据verb更新的fmtFlags格式化操作数arg写入buf中，最后写入io.Writer中

format.go

函数命名规则：
   S结尾：参数为字符串，表示格式化字符串
   Bs结尾：参数为字节切片，表示格式化字节切片
   x结尾：参数多了一个digits，只有ldigits（0123456789abcdefx）与udigits（"0123456789ABCDEFX"）两个方式，表示将参数转为16进制
   Sx就是将字符串转为16进制，Bx就是将字节切片转为16进制，Sbx就是将字符串或字节切片转为16进制
   Q结尾：表示将字符串添加单引号或双引号（使用Go语法安全地转义的单引号字符文字）
   C结尾：参数为Rune，表示格式化Unicode字符
   Qc就是将Rune格式化为带有单引号或双引号的Unicode字符

scan.go

函数命名规则：
   Scan：默认输入源为os.Stdin
   S开头：输入源为字符串
   F开头：输入源为io.Reader

   ln结尾：Scan需要通过换行符\n结尾
   f结尾：需要传入格式化字符串

调用链：
   Scanf（实例化ss）
      |
   doScanf（遍历format）
      |
   scanOne（赋值参数）
      |
   scanXX（检查verb与操作数）
      |
   Read（从数据源读取数据）

数据结构：
   type ss struct {
      rs    io.RuneScanner // 数据源
   	 buf   buffer         // 缓冲区，通常在一个函数内使用与清空
	     count int            // 从format中读取并处理了verb的数量
   }

打印演示：

   type a struct{
      a1 int
   }

常规:
   %v	fmt.Printf("%v\n", a{123})		{123}				默认格式
   %+v	fmt.Printf("%+v\n", a{123})		{a1:123}			显示字段名称
   %#v	fmt.Printf("%#v\n", a{123})		main.a{a1:123}		go语法表示形式
   %T	fmt.Printf("%T\n", a{123})		main.a				类型
   %%	fmt.Printf("%%\n")				%					%号
布尔类型：
   %t	fmt.Printf("%t",true);		true		"true"和"false"字符串
整数:
   %b	fmt.Printf("%b\n",123)		1111011		2进制
   %c	fmt.Printf("%c\n",123)		{			ASCII
   %d	fmt.Printf("%d\n",123)		123			10进制，默认格式
   %o	fmt.Printf("%o\n",123)		173			8进制
   %O	fmt.Printf("%O\n",123)		0o173		带0o前缀的8进制
   %q	fmt.Printf("%q\n",123)		'{'			go语法表示形式的ASCII
   %x	fmt.Printf("%x\n",123)		7b			小写16进制
   %X	fmt.Printf("%X\n",123)		7B			大写16进制
   %U	fmt.Printf("%U\n",123)		U+007B		Unicode格式

   %#b	fmt.Printf("%#b\n",123)		0b1111011	带0b前缀的2进制
   %#c	fmt.Printf("%#c\n",123)		{
   %#d	fmt.Printf("%#d\n",123)		123
   %#o	fmt.Printf("%#o\n",123)		0173		带0前缀的8进制
   %#O	fmt.Printf("%#O\n",123)		0o0173		带0o0前缀的8进制
   %#q	fmt.Printf("%#q\n",123)		'{'
   %#x	fmt.Printf("%#x\n",123)		0x7b		带0x前缀的小写16进制
   %#X	fmt.Printf("%#X\n",123)		0X7B		带0X前缀的大写16进制
   %#U	fmt.Printf("%#U\n",123)		U+007B '{'  带ASCII后缀的Unicode格式
浮点和复杂成分：
   %b	fmt.Printf("%b\n",123.0)					8655355533852672p-46	指数为2的无小数科学计数法，采用strconv.FormatFloat的形式为'b'格式，例如 -123456p-78
   %e	fmt.Printf("%e\n",123.0)					1.230000e+02			科学计数法，e为小写
   %E	fmt.Printf("%E\n",123.0)					1.230000E+02			科学计数法，E为大写
   %f	fmt.Printf("%f\n",123.0)					123.000000				小数点但无指数
   %F	fmt.Printf("%F\n",123.0)					123.000000				与%f相同
   %g	fmt.Printf("%g	%g\n",123.0,1234567.0)		123	1.234567e+06		数字较小时使用%f，数字较大时使用%e
   %G	fmt.Printf("%G	%G\n",123.0,1234567.0)		123	1.234567E+06		数字较小时使用%F，数字较大时使用%E
   %x	fmt.Printf("%x\n",123.0)					0x1.ecp+06				小写带0x前缀16进制
   %X	fmt.Printf("%X\n",123.0)					0X1.ECP+06				大写带0x前缀16进制
字符串与字节切片
   %s	fmt.Printf("%s\n","123")	123			字符串默认格式
   %q	fmt.Printf("%q\n","123")	"123"		go语法表示形式的字符串
   %x	fmt.Printf("%x\n","zzz")	7a7a7a		小写16进制
   %X	fmt.Printf("%X\n","zzz")	7A7A7A		大写16进制

   %#s	fmt.Printf("%#s\n","123")	123
   %#q	fmt.Printf("%#q\n","123")	`123`		反引号
   %#x	fmt.Printf("%#x\n","zzz")	0x7a7a7a	带0x前缀的小写16进制
   %#X	fmt.Printf("%#X\n","zzz")	0X7A7A7A	带0X前缀的大写16进制
切片:
   %p	fmt.Printf("%p",[]int{1,2,3,4,5})	0xc000080030	第0个元素的指针地址，前缀为0x

	  %#p	fmt.Printf("%#p",[]int{1,2,3,4,5})	c000078030		不显示0x前缀
指针:
   %p	fmt.Printf("%p\n",&a)		0xc000084020								带0x前缀的16进制指针
   %b	fmt.Printf("%b\n",&a)		1100000000000000000010000100000000100000	2进制指针
   %d	fmt.Printf("%d\n",&a)		824634261536								10进制指针
   %o	fmt.Printf("%o\n",&a)		14000002040040								8进制指针
   %x	fmt.Printf("%x\n",&a)		c000084020									小写16进制指针
   %X	fmt.Printf("%X\n",&a)		C000084020									大写16进制指针

   %#p	fmt.Printf("%#p\n",&a)		c00007c020									不带0x前缀
   %#b	fmt.Printf("%#b\n",&a)		0b1100000000000000000001111100000000100000	带0b前缀
   %#d	fmt.Printf("%#d\n",&a)		824634228768
   %#o	fmt.Printf("%#o\n",&a)		014000001740040								带0前缀
   %#x	fmt.Printf("%#x\n",&a)		0xc00007c020								带0x前缀
   %#X	fmt.Printf("%#X\n",&a)		0XC00007C020								带0X前缀
特殊情况：
   fmt.Printf("%3.2f",12345.6789)				12345.68		最少输出3个字符，精度为2
   fmt.Printf("'%3.2v'","123456789")				' 12'			最少输出3个字符，其余使用空格填充，默认左填充
   fmt.Printf("'%10.2f'",1.0)					'      1.00'	最少输出10个字符，其余使用空格填充，默认左填充
   fmt.Printf("%[3]v    %[2]v    %v",1,2,3)		3    2    3		参数索引，输出第3个操作数和第2个操作数，以第2个操作数为起点输出下一个操作数
   fmt.Printf("%[3]*.[2]*[1]f",1.0,2,3)			1.00			将第3个操作数作为宽度，将第2个操作数作为精度，输出第1个数
   fmt.Printf("%[2]v    %v",1,2,3)				2    3			输出第2个操作数，以第2个操作数为起点输出下一个操作数
其他字符：
   fmt.Printf("'%010.2f'",1.0)					'0000001.00'	最少输出10个字符，精度为2，其余使用0填充，默认左填充
   fmt.Printf("'%-10v'",1)						'1         '	最少输出10个字符，右侧填充，默认左填充


*/