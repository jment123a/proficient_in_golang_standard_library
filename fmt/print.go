// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fmt

import (
	"internal/fmtsort"
	"io"
	"os"
	"reflect"
	"sync"
	"unicode/utf8"
)

// 用于buffer.WriteString的字符串。
// 这比使用buffer.Write与字节数组的开销要少。
// 注：拼接输出时常用的字符串常量
const (
	commaSpaceString  = ", "                      //注：元素之间的间隔符，多用于%#v
	nilAngleString    = "<nil>"                   //注：未知格式的nil
	nilParenString    = "(nil)"                   //注：已知格式的nil
	nilString         = "nil"                     //注：空指针
	mapString         = "map["                    //注：map的字符串
	percentBangString = "%!"                      //注：错误开头字符串
	missingString     = "(MISSING)"               //注：verb多于操作数数量，缺少操作数
	badIndexString    = "(BADINDEX)"              //注：没有参数索引对应的操作数
	panicString       = "(PANIC="                 //注：自定义格式化接口异常
	extraString       = "%!(EXTRA "               //注：存在未输出的操作数
	badWidthString    = "%!(BADWIDTH)"            //注：错误的宽度设置
	badPrecString     = "%!(BADPREC)"             //注：错误的精度设置
	noVerbString      = "%!(NOVERB)"              //注：错误的verb
	invReflectString  = "<invalid reflect.Value>" //注：非法的反射类型
)

// State 代表传递给自定义格式程序的打印机状态。
// 它提供对io.Writer接口的访问以及有关操作数的格式说明符的标志和选项的信息。
type State interface {
	// Write 是调用以发出要打印的格式化输出的函数。
	Write(b []byte) (n int, err error) //注：向缓冲区写入b，返回写入长度n与错误err
	// Width 返回width选项的值以及是否已设置。
	Width() (wid int, ok bool) //注：获取设置的宽度与是否设置了宽度
	// Precision 返回precision选项的值以及是否已设置。
	Precision() (prec int, ok bool) //注：获取设置的精度与是否设置了精度

	// Flag 报告是否已设置标志c（一个字符）。
	Flag(c int) bool //注：查询是否设置了标志c
}

// Formatter 是由具有自定义格式器的值实现的接口。
// Format的实现可以调用Sprint(f)或Fprint(f)等。
// 生成其输出。
type Formatter interface {
	Format(f State, c rune)
}

// Stringer 由具有String方法的任何值实现，该方法定义该值的``本机''格式。
// String方法用于将作为操作数传递的值打印为接受字符串的任何格式，或打印到未格式化的打印机（如Print）。
type Stringer interface {
	String() string
}

// GoStringer 由具有GoString方法的任何值实现，该方法定义该值的Go语法。
// GoString方法用于将作为操作数传递的值打印为%#v格式。
type GoStringer interface {
	GoString() string
}

// 使用简单的[]byte而不是bytes.Buffer可以避免较大的依赖性。
type buffer []byte //注：fmt包的缓冲区

func (b *buffer) write(p []byte) { //注：向缓冲区b写入字节数组p
	*b = append(*b, p...)
}

func (b *buffer) writeString(s string) { //注：向缓冲区b写入字符串s
	*b = append(*b, s...)
}

func (b *buffer) writeByte(c byte) { //注：向缓冲区b写入单个字符c
	*b = append(*b, c)
}

func (bp *buffer) writeRune(r rune) { //注：向缓冲区bp写入一个rune
	if r < utf8.RuneSelf { //注：如果r是单个字节
		*bp = append(*bp, byte(r)) //注：直接写入
		return
	}

	//注：如果b的len：5，cap：6
	// 以下循环条件为5 + 4 > 6，循环4次，向b追加4个0
	// b的len：9，cap：12，此时9 + 4 == 12
	b := *bp
	n := len(b)
	for n+utf8.UTFMax > cap(b) { //注：使bp可以容纳最大4个字节的Rune
		b = append(b, 0)
	}
	w := utf8.EncodeRune(b[n:n+utf8.UTFMax], r) //注：将r转为[]byte存放至b的最后4个位置中
	*bp = b[:n+w]
}

// pp 用于存储打印机的状态，并与sync.Pool一起重用以避免分配。
type pp struct {
	buf buffer //注：缓冲区

	// arg 将当前项保存为interface{}。
	arg interface{} //注：正在进行格式化的操作数，在printArg中使用

	// value 用于代替arg的反射值。
	value reflect.Value //注：需要通过反射实现打印的操作数，代替arg，在printValue中使用

	// fmt 用于格式化基本项目，例如整数或字符串。
	fmt fmt //注：用于格式化

	// reordered 记录格式字符串是否使用了参数重新排序。
	reordered bool //注：如果fmt.Printf("%[2]v%v%v",1,2,3,4,5)，输出：234，参数索引是否重新计数
	// goodArgNum 记录最近的重新排序指令是否有效。
	goodArgNum bool //注：这次获取的参数索引是否有效
	// panicking 由catchPanic设置，以避免无限恐慌，恢复，恐慌，...递归。
	panicking bool //注：执行用户自定义格式化时是否捕获到异常，用于避免无限恐慌
	// erroring 在打印错误字符串以防止调用handleMethods时设置。
	erroring bool //注：是否在进行错误verb处理，防止调用handleMethods形成无限递归
	// wrapErrs 在格式字符串可能包含%w动词时设置。
	wrapErrs bool //注：是否输出错误
	// wrappedErr 记录%w的目标。
	wrappedErr error //注：输出的错误
}

var ppFree = sync.Pool{ //注：pp的对象缓冲池
	New: func() interface{} { return new(pp) },
}

// newPrinter 分配一个新的pp结构或获取一个缓存的pp结构。
func newPrinter() *pp { //工厂函数
	p := ppFree.Get().(*pp) //注：获取一个pp
	p.panicking = false     //注：重置恐慌
	p.erroring = false      //注：重置错误
	p.wrapErrs = false      //注：#重置%w标记
	p.fmt.init(&p.buf)      //注：初始化p.fmt，将p的buf指针赋值给p.fmt.buf
	return p
}

// free 将使用的pp结构保存在ppFree中； 避免每次调用分配。
func (p *pp) free() { //注：将p放回对象缓冲池中
	// 正确使用sync.Pool要求每个条目具有大约相同的内存成本。
	// 为了在存储的类型包含可变大小的缓冲区时获得此属性，我们对最大缓冲区添加了硬限制以放回池中。
	//
	//参见https://golang.org/issue/23199
	if cap(p.buf) > 64<<10 { //注：如果p的缓冲区buf分配空间超过65536则不放回池中
		return
	}

	p.buf = p.buf[:0]         //注：清空缓冲区
	p.arg = nil               //注：清空参数
	p.value = reflect.Value{} //注：清空操作数
	p.wrappedErr = nil        //注：#重置%w标记
	ppFree.Put(p)             //注：放入池中
}

func (p *pp) Width() (wid int, ok bool) { return p.fmt.wid, p.fmt.widPresent } //注：返回p是否要求了宽度，宽度是多少

func (p *pp) Precision() (prec int, ok bool) { return p.fmt.prec, p.fmt.precPresent } //注：返回p是否要了精度，精度是多少

func (p *pp) Flag(b int) bool { //注：根据b查询p的标志
	switch b {
	case '-':
		return p.fmt.minus
	case '+':
		return p.fmt.plus || p.fmt.plusV
	case '#':
		return p.fmt.sharp || p.fmt.sharpV
	case ' ':
		return p.fmt.space
	case '0':
		return p.fmt.zero
	}
	return false
}

// 实现Write，以便我们可以在pp上（通过State）调用Fprintf，以便在自定义verb中递归使用。
func (p *pp) Write(b []byte) (ret int, err error) { //注：向p.buf追加b，返回写入的长度ret和错误err
	p.buf.write(b) //注：向缓冲区追加b
	return len(b), nil
}

//实现WriteString，以便我们可以在pp上（通过State）调用io.WriteString，以提高效率。
func (p *pp) WriteString(s string) (ret int, err error) { //注：向p.buf追加s，返回写入的长度ret和错误err
	p.buf.writeString(s) //注：向缓冲区追加s
	return len(s), nil
}

// 这些例程以"f"结尾并采用格式字符串。
// 注：
// F开头：要求输入Writer作为参数
// S开头：返回要输出的字符串

// f结尾：要求输入格式化字符串
// ln结尾：程序结尾会输出换行符

// 注：
// 调用链：
// Printf（实例化pp）——doPrintf（遍历format）——printArg（输出参数）——p.fmtXX（根据操作数类型检查verb）——p.fmt.fmtXX（将操作数格式化为[]byte）——p.buf.write（写入缓冲区）
// Print（实例化pp）——doPrint（遍历操作数）——printArg（输出参数）

// Fprintf 根据格式说明符格式化并写入w。
// 返回写入的字节数以及遇到的任何写入错误。
func Fprintf(w io.Writer, format string, a ...interface{}) (n int, err error) { //注：根据format格式化参数a写入w中，返回写入的长度n与错误err
	p := newPrinter()
	p.doPrintf(format, a)
	n, err = w.Write(p.buf) //注：将p的缓冲区写入w
	p.free()                //注：释放p
	return
}

// Printf 根据格式说明符格式化并写入标准输出。
// 返回写入的字节数以及遇到的任何写入错误。
func Printf(format string, a ...interface{}) (n int, err error) { //注：根据format格式化参数a写入os.Stdout中，返回写入的长度n与错误err
	return Fprintf(os.Stdout, format, a...)
}

// Sprintf 根据格式说明符设置格式，并返回结果字符串。
func Sprintf(format string, a ...interface{}) string { //注：根据format格式化参数a，返回格式化后的文本
	p := newPrinter()
	p.doPrintf(format, a)
	s := string(p.buf) //注：将缓冲区的内容转为字符串返回
	p.free()
	return s
}

// 这些例程不采用格式字符串

// Fprint 格式使用其操作数的默认格式并写入w。
//如果都不是字符串，则在操作数之间添加空格。
//返回写入的字节数以及遇到的任何写入错误。
func Fprint(w io.Writer, a ...interface{}) (n int, err error) { //注：将a转为字符串，写入w中
	p := newPrinter()
	p.doPrint(a)
	n, err = w.Write(p.buf)
	p.free()
	return
}

// Print 格式使用其操作数的默认格式并写入标准输出。
// 如果都不是字符串，则在操作数之间添加空格。
// 返回写入的字节数以及遇到的任何写入错误。
func Print(a ...interface{}) (n int, err error) { //注：将a转为字符串，写入os.Stdout中
	return Fprint(os.Stdout, a...)
}

// Sprint 使用其操作数的默认格式设置格式，并返回结果字符串。
//如果都不是字符串，则在操作数之间添加空格。
func Sprint(a ...interface{}) string { //注：将a转为字符串返回
	p := newPrinter()
	p.doPrint(a)
	s := string(p.buf)
	p.free()
	return s
}

//这些例程以'ln'结尾，不使用格式字符串，始终在操作数之间添加空格，并在最后一个操作数之后添加换行符。

// Fprintln 格式使用其操作数的默认格式并写入w。
// 始终在操作数之间添加空格，并添加换行符。
// 返回写入的字节数以及遇到的任何写入错误。
func Fprintln(w io.Writer, a ...interface{}) (n int, err error) { //注：将a转为字符串，结尾添加换行符，写入w中
	p := newPrinter()
	p.doPrintln(a)
	n, err = w.Write(p.buf)
	p.free()
	return
}

// Println 格式使用其操作数的默认格式并写入标准输出。
// 始终在操作数之间添加空格，并添加换行符。
// 返回写入的字节数以及遇到的任何写入错误。
func Println(a ...interface{}) (n int, err error) { //注：将a转为字符串，结尾添加换行符，写入os.Stdout
	return Fprintln(os.Stdout, a...)
}

// Sprintln 使用其操作数的默认格式设置格式，并返回结果字符串。
// 始终在操作数之间添加空格，并添加换行符。
func Sprintln(a ...interface{}) string { //注：将a转为字符串返回，结尾添加换行符
	p := newPrinter()
	p.doPrintln(a)
	s := string(p.buf)
	p.free()
	return s
}

// getField 获取结构值的第i个字段。
// 如果字段本身是接口，则返回接口内部事物的值，而不是接口本身。
func getField(v reflect.Value, i int) reflect.Value { //注：获取结构体v的字段，如果该字段是已分配空间的接口，返回指向的数据
	val := v.Field(i)                                    //注：获取结构体v的第i个字段
	if val.Kind() == reflect.Interface && !val.IsNil() { //注：如果v是接口，并且v已分配空间
		val = val.Elem() //注：获取接口或指针的值
	}
	return val
}

// tooLarge 报告整数的大小是否太大而不能用作格式化宽度或精度。
func tooLarge(x int) bool { //注：报告x是否太大或太小（界限为+-100000）
	const max int = 1e6 //注：100000
	return x > max || x < -max
}

// parsenum 将ASCII转换为整数。 如果没有数字，则num为0（且isnum为false）。
func parsenum(s string, start, end int) (num int, isnum bool, newi int) { //注：将s[start: end]转为数字，遇到第一个非数字就返回，返回转换的数字num，是否是数字isnum，字符串的已索引位置newi
	// 返回第一个最长的s[start: end+1]转换的数字
	// 如果s[0, 5] = "12a34"，返回12
	if start >= end { //注：start不能超过end
		return 0, false, end
	}
	for newi = start; newi < end && '0' <= s[newi] && s[newi] <= '9'; newi++ { //注：遍历s的start位置至end位置，且s的元素只能为0-9
		if tooLarge(num) { //注：num是否超过+-100000
			return 0, false, end //溢出; 疯狂的长号很有可能。
		}
		num = num*10 + int(s[newi]-'0') //注：'9' - '0' = 9
		isnum = true
	}
	return
}

func (p *pp) unknownType(v reflect.Value) { //注：输出未知的类型信息
	if !v.IsValid() { //注：v的类型不合法
		p.buf.writeString(nilAngleString) //注：输出<nil>
		return
	}
	p.buf.writeByte('?')
	p.buf.writeString(v.Type().String())
	p.buf.writeByte('?') //注：输出?v的类型?
}

func (p *pp) badVerb(verb rune) { //注：非法的verb，输出错误信息
	p.erroring = true                    //注：出现错误
	p.buf.writeString(percentBangString) //注：输出%!
	p.buf.writeRune(verb)                //注：输出verb
	p.buf.writeByte('(')                 //注：输出(
	switch {
	case p.arg != nil: //注：如果操作数不为空
		p.buf.writeString(reflect.TypeOf(p.arg).String()) //注：输出操作数的类型
		p.buf.writeByte('=')                              //注：输出=
		p.printArg(p.arg, 'v')                            //注：输出操作数的默认类型的值，%!z(int=1)
	case p.value.IsValid(): //注：验证反射类型操作数是否合法
		p.buf.writeString(p.value.Type().String()) //注：输出类型
		p.buf.writeByte('=')                       //注：输出=
		p.printValue(p.value, 'v', 0)              //注：输出默认类型的值，%!z(<nil>)
	default:
		p.buf.writeString(nilAngleString) //注：输出<nil>
	}
	p.buf.writeByte(')') //注：输出)
	p.erroring = false   //注：结束错误
}

func (p *pp) fmtBool(v bool, verb rune) { //注：检查verb，将布尔类型转为字符串写入p.fmt
	switch verb {
	case 't', 'v': //注：检查格式化布尔类型的verb
		p.fmt.fmtBoolean(v)
	default:
		p.badVerb(verb)
	}
}

// fmt0x64 通过临时设置Sharp标志，按要求将uint64格式化为十六进制格式，并根据要求为其添加0x或不添加0x前缀。
func (p *pp) fmt0x64(v uint64, leading0x bool) { //注：将v输出为16进制，无符号，小写，根据leading0x设置是否显示前缀
	sharp := p.fmt.sharp                            //注：备份sharp设置
	p.fmt.sharp = leading0x                         //注：是否添加前缀0x或0X
	p.fmt.fmtInteger(v, 16, unsigned, 'v', ldigits) //注：输出16进制的无符号数v，打印默认格式，小写
	p.fmt.sharp = sharp                             //注：还原sharp设置
}

// fmtInteger 格式化有符号或无符号整数。
func (p *pp) fmtInteger(v uint64, isSigned bool, verb rune) { //注：根据verb输出数字v，根据isSigned设置是否显示符号
	switch verb {
	case 'v': //注：默认类型%d
		if p.fmt.sharpV && !isSigned { //注：如果为%#v且不显示符号，输出无符号16进制，显示前缀0x
			p.fmt0x64(v, true)
		} else {
			p.fmt.fmtInteger(v, 10, isSigned, verb, ldigits) //注：否则输出
		}
	case 'd': //注：10进制数字
		p.fmt.fmtInteger(v, 10, isSigned, verb, ldigits) //注：10进制，小写
	case 'b': //注：2进制数字
		p.fmt.fmtInteger(v, 2, isSigned, verb, ldigits) //注：2进制，小写
	case 'o', 'O': //注：8进制与8进制加0o前缀数字
		p.fmt.fmtInteger(v, 8, isSigned, verb, ldigits) //注：8进制小写
	case 'x': //注：16进制小写数字
		p.fmt.fmtInteger(v, 16, isSigned, verb, ldigits) //注：16进制，小写
	case 'X': //注：16进制大写数字
		p.fmt.fmtInteger(v, 16, isSigned, verb, udigits) //注：进制，大写
	case 'c': //注：Rune
		p.fmt.fmtC(v) //注：输出Rune
	case 'q': //注：输出转义的Rune
		if v <= utf8.MaxRune {
			p.fmt.fmtQc(v)
		} else {
			p.badVerb(verb)
		}
	case 'U': //注：Unicode
		p.fmt.fmtUnicode(v) //注：输出Unicode，例：v=123 输出：U+007B '{'
	default:
		p.badVerb(verb)
	}
}

// fmtFloat 格式化浮点数。 将每个verb的默认精度指定为fmt_float调用中的最后一个参数。
func (p *pp) fmtFloat(v float64, size int, verb rune) { //注：#根据verb格式化v
	switch verb {
	case 'v': //注：默认格式%g
		p.fmt.fmtFloat(v, size, 'g', -1)
	case 'b', 'g', 'G', 'x', 'X':
		p.fmt.fmtFloat(v, size, verb, -1)
	case 'f', 'e', 'E':
		p.fmt.fmtFloat(v, size, verb, 6)
	case 'F':
		p.fmt.fmtFloat(v, size, 'f', 6)
	default:
		p.badVerb(verb)
	}
}

// fmtComplex 使用fmtFloat将r = real(v)和j = imag(v)的复数v格式化为（r + ji）进行r和j格式化。
func (p *pp) fmtComplex(v complex128, size int, verb rune) { //注：根据verb输出复杂类型v
	// 确保在调用fmtFloat之前找到所有不受支持的verb，以免产生错误的错误字符串。
	switch verb {
	case 'v', 'b', 'g', 'G', 'x', 'X', 'f', 'F', 'e', 'E': //注：复杂类型只支持这些verb
		oldPlus := p.fmt.plus
		p.buf.writeByte('(')              //注：输出(
		p.fmtFloat(real(v), size/2, verb) //注：输出v的实部
		// 虚部总是有一个标志。
		p.fmt.plus = true
		p.fmtFloat(imag(v), size/2, verb) //注：输出v的虚部
		p.buf.writeString("i)")           //注：输出i)
		p.fmt.plus = oldPlus
	default:
		p.badVerb(verb)
	}
}

func (p *pp) fmtString(v string, verb rune) { //注：将v写入缓冲区，根据verb格式化
	switch verb {
	case 'v':
		if p.fmt.sharpV { //注：%#v打印字符串
			p.fmt.fmtQ(v)
		} else {
			p.fmt.fmtS(v)
		}
	case 's':
		p.fmt.fmtS(v)
	case 'x':
		p.fmt.fmtSx(v, ldigits)
	case 'X':
		p.fmt.fmtSx(v, udigits)
	case 'q':
		p.fmt.fmtQ(v)
	default:
		p.badVerb(verb)
	}
}

func (p *pp) fmtBytes(v []byte, verb rune, typeString string) { //注：将v根据verb类型写入缓冲区，typeString为v的数据类型
	switch verb {
	case 'v', 'd': //注：%v或%d，转为10进制数字
		if p.fmt.sharpV { //注：%#v或%#d
			p.buf.writeString(typeString) //注：输出类型
			if v == nil {
				p.buf.writeString(nilParenString) //注：如果为空，输出(nil)
				return
			}
			p.buf.writeByte('{') //注：输出{
			for i, c := range v {
				if i > 0 {
					p.buf.writeString(commaSpaceString) //注：每个元素之间输出", "
				}
				p.fmt0x64(uint64(c), true) //注：将v的每个元素输出为包含0x或0X前缀的的16进制数
			}

			p.buf.writeByte('}') //注：输出}，如果fmt.Printf("%v", []byte{1, 2, 3})，输出：[]byte{0x1, 0x2, 0x3}
		} else {
			p.buf.writeByte('[') //注：输出[
			for i, c := range v {
				if i > 0 { //注：每个元素之间输出空格
					p.buf.writeByte(' ')
				}
				p.fmt.fmtInteger(uint64(c), 10, unsigned, verb, ldigits) //注：输出整数
			}
			p.buf.writeByte(']') //注：输出]，如果fmt.Printf("%v", []byte{1, 2, 3})，输出：[1 2 3]

		}
	case 's':
		p.fmt.fmtBs(v)
	case 'x':
		p.fmt.fmtBx(v, ldigits)
	case 'X':
		p.fmt.fmtBx(v, udigits)
	case 'q':
		p.fmt.fmtQ(string(v))
	default:
		p.printValue(reflect.ValueOf(v), verb, 0)
	}
}

func (p *pp) fmtPointer(value reflect.Value, verb rune) { //注：将指针v指向数据的指针地址根据verb写入缓冲区
	var u uintptr
	switch value.Kind() { //注：获取引用类型的指针
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		u = value.Pointer()
	default:
		p.badVerb(verb)
		return
	}

	switch verb {
	case 'v': //注：打印默认格式
		if p.fmt.sharpV { //注：如果verb是%#v，输出(value的类型)(nil或0x指针)
			p.buf.writeByte('(')                     //注：输出(
			p.buf.writeString(value.Type().String()) //注：输出操作数的类型的字符串形式
			p.buf.writeString(")(")                  //注：输出)(
			if u == 0 {                              //注：如果操作数的指针为0
				p.buf.writeString(nilString) //注：输出nil
			} else {
				p.fmt0x64(uint64(u), true) //注：输出16进制小写的，显示前缀0x的u
			}
			p.buf.writeByte(')') //注：输出)，如果fmt.Printf("%#v",&a),输出：(*int)(0xc00007c020)
		} else {
			if u == 0 { //注：输出<nil>
				p.fmt.padString(nilAngleString)
			} else { //注：输出根据!sharp显示前缀0x的16进制小写的u（%p默认是加0x的，%#p变为不加0x，与%v与%#v相反）
				p.fmt0x64(uint64(u), !p.fmt.sharp) //注：如果fmt.Printf("%v",&a)，输出：0xc000016060
			}
		}
	case 'p': //注：打印指针
		p.fmt0x64(uint64(u), !p.fmt.sharp) //注：输出根据!sharp显示前缀0x的16进制小写的u（默认输出0x，sharp设置为true之后取反，为不显示0x）
	case 'b', 'o', 'd', 'x', 'X': //注：数值类型
		p.fmtInteger(uint64(u), unsigned, verb)
	default:
		p.badVerb(verb)
	}
}

func (p *pp) catchPanic(arg interface{}, verb rune, method string) {
	// 注：
	// type A struct{
	// 	a1 int
	// }
	// func (a A) GoString() string{
	// 	panic("panic")
	// 	return "panic"
	// }
	// func main() {
	// 	var gs fmt.GoStringer
	// 	gs = A{123}
	// 	fmt.Printf("%#v",gs)
	// }
	// 输出：%!v(PANIC=GoString method: panic)
	// 注：为用户定义的自定义处理方法捕获恐慌
	if err := recover(); err != nil { //注：捕获到了恐慌
		// 如果是nil指针，则只需说"<nil>"。
		// 最可能的原因是Stringer无法防止nil或值接收器的nil指针，无论哪种情况，"<nil>"都是一个不错的结果。
		if v := reflect.ValueOf(arg); v.Kind() == reflect.Ptr && v.IsNil() { //注：操作数为空指针
			p.buf.writeString(nilAngleString) //注：写入<nil>
			return
		}

		//否则，打印简短的紧急消息。 大多数时候，恐慌值会很好地打印出来。
		if p.panicking { //注：是否恐慌
			//嵌套恐慌； printArg中的递归不能成功。
			panic(err)
		}

		oldFlags := p.fmt.fmtFlags
		//对于此输出，我们需要默认行为。
		p.fmt.clearflags()

		p.buf.writeString(percentBangString) //注：输出%!
		p.buf.writeRune(verb)                //注：输出verb
		p.buf.writeString(panicString)       //注：输出(PANIC="
		p.buf.writeString(method)            //注：输出方法名称
		p.buf.writeString(" method: ")       //注： method:
		p.panicking = true
		p.printArg(err, 'v') //注： 输出错误
		p.panicking = false
		p.buf.writeByte(')') //注：输出)

		p.fmt.fmtFlags = oldFlags
	}
}

func (p *pp) handleMethods(verb rune) (handled bool) { //注：执行自定义处理方法，参数为verb，返回操作数是否支持处理方法handled
	if p.erroring { //注：是否在进行错误verb处理时被调用，是否递归调用了
		return
	}
	if verb == 'w' { //注：%w
		// 除了与Errorf一起使用%w，多次或与非错误arg一起使用之外，%w是无效的。
		err, ok := p.arg.(error)
		if !ok || !p.wrapErrs || p.wrappedErr != nil { //注：如果操作数不能实现error，或出现错误
			p.wrappedErr = nil
			p.wrapErrs = false
			p.badVerb(verb) //注：输出错误的verb
			return true
		}
		p.wrappedErr = err //注：记录错误方法
		// 如果arg是Formatter，则将'v'作为verb传递给它。
		verb = 'v'
	}

	// 是格式化程序吗？
	if formatter, ok := p.arg.(Formatter); ok { //注：如果实现了自定义格式化接口Formatter，则执行Format方法
		handled = true
		defer p.catchPanic(p.arg, verb, "Format")
		formatter.Format(p, verb)
		return
	}

	// 如果我们正在执行Go语法，并且该参数知道如何提供它，请立即处理。
	if p.fmt.sharpV {
		if stringer, ok := p.arg.(GoStringer); ok { //注：如果verb为%#v，且操作数实现了GoStringer接口，执行GoString方法
			handled = true
			defer p.catchPanic(p.arg, verb, "GoString")
			// 不加修饰地打印GoString的结果。
			p.fmt.fmtS(stringer.GoString())
			return
		}
	} else {
		// 如果根据格式可接受字符串，请查看该值是否满足字符串值接口之一。
		// Println等。将动词设置为％v，这是“可字符串化的”。
		switch verb {
		case 'v', 's', 'x', 'X', 'q':
			// 是错误还是Stringer？
			// 复制正文是必要的：
			// 必须在调用方法之前进行已设置的处理并延迟catchPanic。
			switch v := p.arg.(type) {
			case error: //注：是否为error格式，输出Error()
				handled = true
				defer p.catchPanic(p.arg, verb, "Error")
				p.fmtString(v.Error(), verb)
				return

			case Stringer: //注：是否为Stringer格式，输出String()
				handled = true
				defer p.catchPanic(p.arg, verb, "String")
				p.fmtString(v.String(), verb)
				return
			}
		}
	}
	return false
}

func (p *pp) printArg(arg interface{}, verb rune) { //注：根据verb格式化arg写入值缓冲区
	p.arg = arg //注：arg为当前要输出的1个操作数
	p.value = reflect.Value{}

	if arg == nil { //注：如果操作数为nil
		switch verb {
		case 'T', 'v': //注：如果verb为默认格式
			p.fmt.padString(nilAngleString) //注：填充<nil>，满足宽度要求
		default:
			p.badVerb(verb) //注：写入错误的verb
		}
		return
	}

	// 特殊处理注意事项。
	// %T（值的类型）和%p（其地址）是特殊的； 我们总是先做他们。
	switch verb {
	case 'T': //注：go语法表示
		p.fmt.fmtS(reflect.TypeOf(arg).String()) //注：arg的类型字符串形式，根据精度要求截取
		return
	case 'p': //注：打印指针
		p.fmtPointer(reflect.ValueOf(arg), 'p')
		return
	}

	// 某些类型可以无需反射即可完成。
	switch f := arg.(type) {
	case bool:
		p.fmtBool(f, verb)
	case float32:
		p.fmtFloat(float64(f), 32, verb)
	case float64:
		p.fmtFloat(f, 64, verb)
	case complex64:
		p.fmtComplex(complex128(f), 64, verb)
	case complex128:
		p.fmtComplex(f, 128, verb)
	case int:
		p.fmtInteger(uint64(f), signed, verb)
	case int8:
		p.fmtInteger(uint64(f), signed, verb)
	case int16:
		p.fmtInteger(uint64(f), signed, verb)
	case int32:
		p.fmtInteger(uint64(f), signed, verb)
	case int64:
		p.fmtInteger(uint64(f), signed, verb)
	case uint:
		p.fmtInteger(uint64(f), unsigned, verb)
	case uint8:
		p.fmtInteger(uint64(f), unsigned, verb)
	case uint16:
		p.fmtInteger(uint64(f), unsigned, verb)
	case uint32:
		p.fmtInteger(uint64(f), unsigned, verb)
	case uint64:
		p.fmtInteger(f, unsigned, verb)
	case uintptr:
		p.fmtInteger(uint64(f), unsigned, verb)
	case string:
		p.fmtString(f, verb)
	case []byte:
		p.fmtBytes(f, verb, "[]byte")
	case reflect.Value:
		//使用特殊方法处理可提取值，因为printValue不会在depth为0时处理它们。
		if f.IsValid() && f.CanInterface() { //注：f合法且为已导出的字段
			p.arg = f.Interface() //注：转为空接口
			if p.handleMethods(verb) {
				return
			}
		}
		p.printValue(f, verb, 0)
	default:
		// 如果类型不简单，则可能有方法。
		if !p.handleMethods(verb) {
			//需要使用反射，因为该类型没有可用于格式化的接口方法。
			p.printValue(reflect.ValueOf(f), verb, 0)
		}
	}
}

// printValue 与printArg相似，但以反射值而不是interface{}值开头。
// 它不处理'p'和'T'动词，因为它们应该已经由printArg处理。
func (p *pp) printValue(value reflect.Value, verb rune, depth int) {
	// 如果尚未由printArg处理（深度 == 0），则使用特殊方法处理值。
	if depth > 0 && value.IsValid() && value.CanInterface() { //注：v是否合法，是否为已导出的字段，且深度大于0
		p.arg = value.Interface() //注：将value转为空接口
		if p.handleMethods(verb) {
			return
		}
	}
	p.arg = nil
	p.value = value

	switch f := value; value.Kind() {
	case reflect.Invalid: //注：非法类型
		if depth == 0 { //注：深度为0
			p.buf.writeString(invReflectString) //注：输出<invalid reflect.Value>
		} else {
			switch verb {
			case 'v':
				p.buf.writeString(nilAngleString) //注：输出<nil>
			default:
				p.badVerb(verb)
			}
		}
	case reflect.Bool:
		p.fmtBool(f.Bool(), verb)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p.fmtInteger(uint64(f.Int()), signed, verb)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p.fmtInteger(f.Uint(), unsigned, verb)
	case reflect.Float32:
		p.fmtFloat(f.Float(), 32, verb)
	case reflect.Float64:
		p.fmtFloat(f.Float(), 64, verb)
	case reflect.Complex64:
		p.fmtComplex(f.Complex(), 64, verb)
	case reflect.Complex128:
		p.fmtComplex(f.Complex(), 128, verb)
	case reflect.String:
		p.fmtString(f.String(), verb)
		// a := make(map[int]string, 2)
		// a[0] = "a"
		// a[1] = "b"
		// fmt.Printf("%#v",a)
		// 输出：map[int]string{0:"a", 1:"b"}
		// fmt.Printf("%v",a)
		// 输出：map[0:a 1:b]
	case reflect.Map:
		if p.fmt.sharpV { //注：%#v输出map
			p.buf.writeString(f.Type().String()) //注：输出map的类型
			if f.IsNil() {
				p.buf.writeString(nilParenString) //注：输出(nil)，例：map[int]string(nil)
				return
			}
			p.buf.writeByte('{') //注：输出{
		} else {
			p.buf.writeString(mapString) //注：输出map[
		}
		sorted := fmtsort.Sort(f) //注：#
		for i, key := range sorted.Key {
			if i > 0 { //注：每个元素之间输出,或空格
				if p.fmt.sharpV {
					p.buf.writeString(commaSpaceString)
				} else {
					p.buf.writeByte(' ')
				}
			}
			p.printValue(key, verb, depth+1) //注：#
			p.buf.writeByte(':')
			p.printValue(sorted.Value[i], verb, depth+1)
		}
		if p.fmt.sharpV {
			p.buf.writeByte('}') //注：输出}
		} else {
			p.buf.writeByte(']') //注：输出]
		}
		// type a struct {
		//     a1 int
		// 	   b
		// }
		// type b struct {
		// 	    b1 int
		// }
		// func main() {
		//     Printf("%#v", a{1, b{2}})
		// }
		// 输出：main.a{a1:1, b:main.b{b1:2}}
	case reflect.Struct:
		if p.fmt.sharpV {
			p.buf.writeString(f.Type().String()) //注：输出结构体类型
		}
		p.buf.writeByte('{')                //注：输出{
		for i := 0; i < f.NumField(); i++ { //注：遍历字段数量
			if i > 0 { //注：每个元素之间，如果%#v输出', '否则输出' '
				if p.fmt.sharpV {
					p.buf.writeString(commaSpaceString)
				} else {
					p.buf.writeByte(' ')
				}
			}
			if p.fmt.plusV || p.fmt.sharpV {
				if name := f.Type().Field(i).Name; name != "" { //注：获取v的第i个字段
					p.buf.writeString(name) //注：输出字段名称
					p.buf.writeByte(':')    //注：输出:
				}
			}
			p.printValue(getField(f, i), verb, depth+1) //注：嵌套输出字段，深度+1
		}
		p.buf.writeByte('}') //注：输出}
	case reflect.Interface:
		value := f.Elem() //注： 获取value指向数据的指针值
		if !value.IsValid() {
			if p.fmt.sharpV {
				p.buf.writeString(f.Type().String()) //注：输出类型
				p.buf.writeString(nilParenString)    //注：输出(nil)
			} else {
				p.buf.writeString(nilAngleString) //注：输出<nil>
			}
		} else {
			p.printValue(value, verb, depth+1) //注：嵌套是输出值，深度+1
		}

	// fmt.Printf("%v",[]int{1,2,3})
	// [1 2 3]
	// fmt.Printf("%#v",[]int{1,2,3})
	// []int{1, 2, 3}
	case reflect.Array, reflect.Slice:
		switch verb {
		case 's', 'q', 'x', 'X':
			// 处理上述verb专用的byte和uint8切片和数组。
			t := f.Type()
			if t.Elem().Kind() == reflect.Uint8 { //注：如果是Uint8类型
				var bytes []byte
				if f.Kind() == reflect.Slice { //注：如果是切片
					bytes = f.Bytes()
				} else if f.CanAddr() { //注：是可寻址的，通过反射实现，用于处理数组、切片与字符串
					bytes = f.Slice(0, f.Len()).Bytes() //注：value[0, f.len()]的字节数组
				} else { //注：不可寻址的
					// 我们有一个数组，但是不能对无法寻址的数组Slice()进行切片，因此我们需要手动构建切片。
					// 这是一种罕见的情况，但是如果反射可以提供更多帮助，那就太好了。
					bytes = make([]byte, f.Len())
					for i := range bytes {
						bytes[i] = byte(f.Index(i).Uint()) //注：将第i个元素转为数字
					}
				}
				p.fmtBytes(bytes, verb, t.String())
				return
			}
		}
		if p.fmt.sharpV {
			p.buf.writeString(f.Type().String())        //注：输出类型
			if f.Kind() == reflect.Slice && f.IsNil() { //注：如果切片为空
				p.buf.writeString(nilParenString) //注：输出(nil)
				return
			}
			p.buf.writeByte('{') //注：输出{
			for i := 0; i < f.Len(); i++ {
				if i > 0 {
					p.buf.writeString(commaSpaceString) //注：输出,
				}
				p.printValue(f.Index(i), verb, depth+1) //注：递归遍历第v的第i个元素，深度+1
			}
			p.buf.writeByte('}') //注：输出}
		} else {
			p.buf.writeByte('[') //注：输出[
			for i := 0; i < f.Len(); i++ {
				if i > 0 {
					p.buf.writeByte(' ') //注：输出空格
				}
				p.printValue(f.Index(i), verb, depth+1) //注：递归遍历第v的第i个元素，深度+1
			}
			p.buf.writeByte(']') //注：输出]
		}
	// fmt.Printf("%#v",&[]int{1,2,3})
	// &[]int{1, 2, 3}
	case reflect.Ptr:
		//指向数组，切片或结构的指针？可以，但不能嵌入（避免循环）
		if depth == 0 && f.Pointer() != 0 { //注：深度为0，且指针不为空
			switch a := f.Elem(); a.Kind() {
			case reflect.Array, reflect.Slice, reflect.Struct, reflect.Map:
				p.buf.writeByte('&')
				p.printValue(a, verb, depth+1) //注：前缀加&，原样输出
				return
			}
		}
		fallthrough
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		p.fmtPointer(f, verb)
	default:
		p.unknownType(f)
	}
}

// intFromArg 获取a的argNumth元素。 返回时，isInt报告参数是否具有整数类型。
func intFromArg(a []interface{}, argNum int) (num int, isInt bool, newArgNum int) { //注：从操作数组中取出第argNum个int，返回取出的值num，是否为数字IsInt，下一个参数索引newArgNum
	newArgNum = argNum
	if argNum < len(a) { //注：操作数索引小与操作数组的长度
		num, isInt = a[argNum].(int) // 几乎总是可以的。 注：取出第对应索引的操作数
		if !isInt {                  //注：如果不是int
			// 努力工作。
			switch v := reflect.ValueOf(a[argNum]); v.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64: //注：int类型统一转为int64
				n := v.Int()
				if int64(int(n)) == n {
					num = int(n)
					isInt = true
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr: //注：Uint统一转为int64
				n := v.Uint()
				if int64(n) >= 0 && uint64(int(n)) == n {
					num = int(n)
					isInt = true
				}
			default:
				//已经为0，为false。
			}
		}
		newArgNum = argNum + 1
		if tooLarge(num) { //注：操作数大于超过+-100000
			num = 0
			isInt = false
		}
	}
	return
}

// parseArgNumber 返回带括号的数字的值减1（显式参数数字为1索引，但我们希望为0索引）。
// 众所周知，左括号位于format[0]。
// 返回的值是索引，直到结束括号所消耗的字节数（如果存在）以及该数目是否可以解析。 如果没有关闭括号，则要消耗的字节将为1。
func parseArgNumber(format string) (index int, wid int, ok bool) { //注：获取format的下一个参数索引（格式：[index]），返回format字符串解析出的索引index，字符串索引到的位置wid，是否解析出数字ok
	//必须至少有3个字节：[n]。
	if len(format) < 3 { //注：因为要获取[i]，所以至少为3
		return 0, 1, false
	}

	//找到右括号。
	for i := 1; i < len(format); i++ { //注：遍历format，寻找]
		if format[i] == ']' {
			// width：	转换的数字
			// ok：		是否是数字
			// newi：	format索引到的位置
			width, ok, newi := parsenum(format, 1, i) //注：在format[1: i+1]处获取第1个最长的数字
			if !ok || newi != i {                     //注：]之前的字符串不能转为数字 或 字符串的索引的字符串转为数字的索引不同
				return 0, i + 1, false
			}
			return width - 1, i + 1, true // arg数字为1索引，并跳过括号。注：[index]从[1]开始，但是返回值以0开始，操作数也是从0开始
		}
	}
	return 0, 1, false
}

// 调用链：doPrint——argNumber（获取下一个参数索引[index]）——parseArgNumber（获取第一个参数索引[index]）——parsenum（将参数索引字符串转为数字）
// argNumber 返回要求值的下一个参数，该参数可以是传入的argNum的值，也可以是以format[i:]开头的括号中的整数的值。
// 它还返回i的新值，即要处理的格式的下一个字节的索引。
func (p *pp) argNumber(argNum int, format string, i int, numArgs int) (newArgNum, newi int, found bool) { //注：获取format[i:]的下一个参数索引
	//注：如果没有找到，默认参数索引为argNum，一共有numArgs个操作数
	// 如果找到了新的参数索引，返回参数索引newArgNum，format遍历到的位置newi与是否找到新的参数索引found，
	//
	// argNum：		默认操作数索引
	// format：		格式化字符串
	// i：			格式化字符串索引
	// numArgs：	操作数数量
	// newArgNum：	获取参数后的操作数的索引（参数索引）
	// newi：		获取参数后的格式化字符串索引
	// found：		是否获取到参数索引
	if len(format) <= i || format[i] != '[' { //注：索引为超过格式化字符串或!='['
		return argNum, i, false
	}
	p.reordered = true                           //注：字符串使用了参数重新排序
	index, wid, ok := parseArgNumber(format[i:]) //注：获取下一个参数索引
	if ok && 0 <= index && index < numArgs {     //注：如果获取到了参数索引，且参数索引<=0，且索引可以与操作数对应
		return index, i + wid, true
	}
	p.goodArgNum = false //注：最近的重新排序指令无效
	return argNum, i + wid, ok
}

func (p *pp) badArgNum(verb rune) { //注：输出%!verb(BADINDEX)
	p.buf.writeString(percentBangString)
	p.buf.writeRune(verb)
	p.buf.writeString(badIndexString)
}

func (p *pp) missingArg(verb rune) { //注：输出%!verb(MISSING)
	p.buf.writeString(percentBangString)
	p.buf.writeRune(verb)
	p.buf.writeString(missingString)
}

func (p *pp) doPrintf(format string, a []interface{}) { //注：遍历格式化字符串format，按制定格式打印操作数组a
	end := len(format)
	argNum := 0         // 我们按照非平凡的格式处理一个参数
	afterIndex := false // 格式上的前一项是类似[3]的索引。
	p.reordered = false // 注：p未使用参数重新排序

formatLoop: //注：遍历格式化字符串
	for i := 0; i < end; {
		p.goodArgNum = true               //注：p最近的重新排序指令有效
		lasti := i                        //注：上一个i的位置
		for i < end && format[i] != '%' { //注：遍历format到第一个%处
			i++
		}
		if i > lasti { //注：如果遍历时跳过了字符串常量
			p.buf.writeString(format[lasti:i]) //注：写入字符串常量
		}
		if i >= end { //注：format遍历结束
			//完成处理格式字符串
			break
		}

		//处理一个verb
		i++

		//我们有标志吗？
		p.fmt.clearflags() //注：清空格式化标记
	simpleFormat: //注：格式化简单格式
		for ; i < end; i++ {
			c := format[i] //注：%的下一个字，verb
			switch c {
			case '#': //注："%#"
				p.fmt.sharp = true
			case '0': //注："%0"
				p.fmt.zero = !p.fmt.minus //只允许零填充到左边。注：如果%-0，则zero为false
			case '+': //注："%+"
				p.fmt.plus = true
			case '-': //注："%-"
				p.fmt.minus = true
				p.fmt.zero = false // 不要在右边用零填充。
			case ' ': //注："% "
				p.fmt.space = true
			default:
				//不带精度，宽度或参数索引的ascii小写简单verb常见情况的快速路径。
				if 'a' <= c && c <= 'z' && argNum < len(a) { //注：如果verb在a与z之间，遍历操作数
					if c == 'v' {
						// Go语法
						p.fmt.sharpV = p.fmt.sharp //注：如果是%#v，设置sharp为false，sharpV为true
						p.fmt.sharp = false
						// 结构域语法
						p.fmt.plusV = p.fmt.plus //注：如果是%+v，设置plus为false，plusV为true
						p.fmt.plus = false
					}
					p.printArg(a[argNum], rune(c)) //注：格式化第argNum个操作数，格式为c
					argNum++
					i++
					continue formatLoop
				}
				//格式比简单的标记和verb更复杂，或者格式不正确。
				break simpleFormat //注：可能遇到了'['或'.'
			}
		}

		// 我们有一个明确的参数索引吗？
		// 注：当fmt.Sprintf("%[3]*.[2]*[1]f", 12.0, 2, 6)
		// %[3]*f：代表从操作数中第index个操作数作为宽度或精度
		// %[3]3f：错误
		// %[3].f:错误
		argNum, i, afterIndex = p.argNumber(argNum, format, i, len(a)) //注：获取format从i开始的下一个参数索引，argNum=3，i=4，afterIndex=true

		//我们有宽度吗？

		if i < end && format[i] == '*' { //注：format没有遍历结束并且[index]的下一个字符是*（这个参数索引是宽度），i=4
			i++                                                         //注：i=5
			p.fmt.wid, p.fmt.widPresent, argNum = intFromArg(a, argNum) //注：取出宽度、是否有宽度、下一个参数的索引值，wid=6，widPresent=true，argNum=4

			if !p.fmt.widPresent { //注：如果没设置宽度
				p.buf.writeString(badWidthString) //注：写入%!(BADWIDTH)
			}

			//我们有一个负宽度，所以取其值并确保设置minus
			if p.fmt.wid < 0 { //注：负宽度，设置minus为true
				p.fmt.wid = -p.fmt.wid
				p.fmt.minus = true
				p.fmt.zero = false //不要在右边用零填充。
			}
			afterIndex = false //注：没找到参数索引
		} else {
			p.fmt.wid, p.fmt.widPresent, i = parsenum(format, i, end) //注：在format[i: end+1]中取出数字，wid=0，widPresent=false，i=4
			if afterIndex && p.fmt.widPresent {                       // "%[3]2d"
				p.goodArgNum = false //注：#p最近的重新排序指令无效
			}
		}

		//我们有精度吗？
		if i+1 < end && format[i] == '.' { //注：同上（这个参数索引是精度），i=5
			i++
			if afterIndex { // "%[3].2d"，注：没有遇到*
				p.goodArgNum = false
			}
			argNum, i, afterIndex = p.argNumber(argNum, format, i, len(a)) //注：下一个参数索引，argNum=2，i=9，afterIndex=true
			if i < end && format[i] == '*' {
				i++                                                           //注：i=10
				p.fmt.prec, p.fmt.precPresent, argNum = intFromArg(a, argNum) //注：取出精度、是否有宽度、下一个参数的索引值
				//负精度参数没有意义
				if p.fmt.prec < 0 {
					p.fmt.prec = 0
					p.fmt.precPresent = false
				}
				if !p.fmt.precPresent {
					p.buf.writeString(badPrecString) //注：输出%!(BADPREC)
				}
				afterIndex = false
			} else { // "%[3].2d"
				p.fmt.prec, p.fmt.precPresent, i = parsenum(format, i, end) //注：取出精度
				if !p.fmt.precPresent {
					p.fmt.prec = 0
					p.fmt.precPresent = true
				}
			}
		}

		if !afterIndex { //注：如果之前获取的是宽度和精度，再次获取参数索引
			argNum, i, afterIndex = p.argNumber(argNum, format, i, len(a))
		}

		if i >= end { //注：如果遍历结束了
			p.buf.writeString(noVerbString) //注：%!(NOVERB)
			break
		}

		verb, size := rune(format[i]), 1
		if verb >= utf8.RuneSelf { //注：如果verb是多字节Rune，转为字符串
			verb, size = utf8.DecodeRuneInString(format[i:])
		}
		i += size

		switch {
		case verb == '%': // %不吸收操作数，并且忽略f.wid和f.prec。
			p.buf.writeByte('%') //注：写入%
		case !p.goodArgNum:
			p.badArgNum(verb)
		case argNum >= len(a): //没有剩余参数可为当前verb打印。
			p.missingArg(verb)
		case verb == 'v':
			//Go语法
			p.fmt.sharpV = p.fmt.sharp
			p.fmt.sharp = false
			//结构域语法
			p.fmt.plusV = p.fmt.plus
			p.fmt.plus = false
			fallthrough
		default:
			p.printArg(a[argNum], verb)
			argNum++
		}
	}

	// 检查是否有额外的参数，除非调用乱序访问了这些参数，在这种情况下，检测它们是否已全部使用成本太高，如果没有使用，则可以确定。
	if !p.reordered && argNum < len(a) { //注：当fmt.Printf("%f", 12.0, 1)，输出：%!(EXTRA int=1)
		p.fmt.clearflags()
		p.buf.writeString(extraString) //注：输出%!(EXTRA
		for i, arg := range a[argNum:] {
			if i > 0 {
				p.buf.writeString(commaSpaceString) //注：每个未打印的操作数之间添加一个逗号
			}
			if arg == nil {
				p.buf.writeString(nilAngleString) //注：<nil>
			} else {
				p.buf.writeString(reflect.TypeOf(arg).String()) //注：输出操作数的类型
				p.buf.writeByte('=')                            //注：输出=
				p.printArg(arg, 'v')                            //注：输出默认类型的操作数
			}
		}
		p.buf.writeByte(')') //注：输出)
	}
}

func (p *pp) doPrint(a []interface{}) { //注：遍历操作数a，使用默认格式（%v）输出
	prevString := false
	for argNum, arg := range a { //注：遍历操作数
		isString := arg != nil && reflect.TypeOf(arg).Kind() == reflect.String //注：操作数是否为字符串
		//在两个非字符串参数之间添加一个空格。
		// 注：fmt.Print("a", "b", 1, 2)，输出结果为"ab1 2"
		if argNum > 0 && !isString && !prevString { //注：不是第1个参数，并且不是字符串，并且上一个操作数不是字符串
			p.buf.writeByte(' ') //注：输出一个空格
		}
		p.printArg(arg, 'v') //注：输出一个默认格式的操作数
		prevString = isString
	}
}

// doPrintln 就像doPrint一样，但是总是在参数之间添加一个空格
// 和最后一个参数之后的换行符。
func (p *pp) doPrintln(a []interface{}) { //注：遍历操作数a，使用默认格式（%v）输出，最后输出一个换行符
	for argNum, arg := range a { //注：遍历操作数
		if argNum > 0 {
			p.buf.writeByte(' ') //注：操作数之间添加空格
		}
		p.printArg(arg, 'v') //注：输出一个默认格式的操作数
	}
	p.buf.writeByte('\n') //注：最后输出一个换行符
}
