// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package fmt

import (
	"strconv"
	"unicode/utf8"
)

const (
	ldigits = "0123456789abcdefx" //小写字序，16进制转换用
	udigits = "0123456789ABCDEFX" //大写字序，16进制转换用
)

const (
	signed   = true  //注：有符号
	unsigned = false //注：无符号
)

// flags 放在单独的结构中以便于清除。
type fmtFlags struct {
	widPresent  bool //注：是否设置宽度
	precPresent bool //注：是否设置精度
	minus       bool //注：是否设置-，是否右侧填充
	plus        bool //注：是否设置+
	sharp       bool //注：是否设置#
	space       bool //注：是否设置' '
	zero        bool //注：是否设置0，是否填充0

	//对于%+v %#v格式，我们设置plusV/sharpV标志并清除plus/sharp标志，因为%+v和%#v实际上是在顶层设置的不同的无标志格式。
	plusV  bool //注：verb是否为%+v
	sharpV bool //注：verb是否为%#v
}

// fmt是Printf等使用的原始格式化程序。
// 它打印到必须单独设置的缓冲区中。
type fmt struct {
	buf *buffer //注：缓冲区

	fmtFlags //注：verb标志

	wid  int // 宽度
	prec int // 精度

	// intbuf足够大，可以存储带符号的int64的%b，并避免在32位体系结构的结构末尾填充。
	intbuf [68]byte //注：临时使用，数字或Rune进行转换的缓冲
}

func (f *fmt) clearflags() { //注：清空格式化标志
	f.fmtFlags = fmtFlags{}
}

func (f *fmt) init(buf *buffer) { //注：初始化f.buf为buf，清空标记
	f.buf = buf    //注：更新缓冲区对象
	f.clearflags() //注：清空格式化标志
}

// writePadding 生成n个字节的填充。
func (f *fmt) writePadding(n int) { //注：在f的缓冲区追加n个0或空格（根据f.zero是否为true）
	if n <= 0 { // 无需填充字节。
		return
	}
	buf := *f.buf
	oldLen := len(buf)
	newLen := oldLen + n
	// 为填充留出足够的空间。
	if newLen > cap(buf) { //注：为f的缓冲区扩容
		buf = make(buffer, cap(buf)*2+n)
		copy(buf, *f.buf)
	}
	//确定填充填充的字节。
	padByte := byte(' ')
	if f.zero {
		padByte = byte('0')
	}
	//用padByte填充填充。
	padding := buf[oldLen:newLen]
	for i := range padding {
		padding[i] = padByte
	}
	*f.buf = buf[:newLen]
}

// pad 将b附加到f.buf，在左侧(!f.minus)或右侧(f.minus)填充。
func (f *fmt) pad(b []byte) { //注：f.buf写入b，根据宽度于minus进行左填充或右填充
	if !f.widPresent || f.wid == 0 { //注：如果没有要求宽度，直接写入b
		f.buf.write(b)
		return
	}
	width := f.wid - utf8.RuneCount(b) //注：b转为Rune的长度
	if !f.minus {
		// 左填充
		f.writePadding(width)
		f.buf.write(b)
	} else {
		// 右填充
		f.buf.write(b)
		f.writePadding(width)
	}
}

// padString 将s附加到f.buf，在左侧（!f.minus）或右侧（f.minus）填充。
func (f *fmt) padString(s string) { //注：为f的缓冲区追加s，满足f.wid宽度要求，根据f.minus判断是否为右侧填充，填充物为空格或0（根据f.zero是否为true）
	if !f.widPresent || f.wid == 0 { //注：如果没有宽度要求
		f.buf.writeString(s) //注：f的缓冲区直接写入s
		return
	}
	width := f.wid - utf8.RuneCountInString(s) //注：f的宽度-s的字数
	if !f.minus {                              //注：设置了-
		// 左侧填充
		f.writePadding(width) //注：f的缓冲区填充长度为width的空格或0
		f.buf.writeString(s)  //注：f的缓冲区写入字符串s
	} else {
		// 右侧填充
		f.buf.writeString(s)  //注：f的缓冲区写入字符串s
		f.writePadding(width) //注：f的缓冲区填充长度为width的空格或0
	}
}

// fmtBoolean 格式化一个布尔值。
func (f *fmt) fmtBoolean(v bool) { //注：将布尔类型转为字符串写入f
	if v {
		f.padString("true")
	} else {
		f.padString("false")
	}
}

// fmtUnicode 将uint64格式化为"U+0078"或将f.sharp设置为“U+0078'x'”。
func (f *fmt) fmtUnicode(u uint64) { //注：打印Unicode
	buf := f.intbuf[0:]

	//使用默认精度设置，使用%#U（"U+FF FFFF FFFF FFFF FFFF"）格式化-1所需的最大buf长度为18，该长度适合已分配的intbuf，容量为68个字节。
	prec := 4
	if f.precPresent && f.prec > 4 { //注：如果精度要求>4
		prec = f.prec
		//计算"U+"，数字，" '"，字符，"'"所需的空间。
		width := 2 + prec + 2 + utf8.UTFMax + 1
		if width > len(buf) {
			buf = make([]byte, width)
		}
	}

	//格式化为buf，以buf[i]结尾。 从右到左更容易格式化数字。
	i := len(buf)

	//对于%#U，我们要在缓冲区的末尾添加一个空格和一个带引号的字符。
	//注：如果u = 123
	if f.sharp && u <= utf8.MaxRune && strconv.IsPrint(rune(u)) { //注：是否要额外输出转义后的字符，u是否为Rune，是否可以打印处转义后的字符

		i--
		buf[i] = '\'' //注： "'"
		i -= utf8.RuneLen(rune(u))
		utf8.EncodeRune(buf[i:], rune(u)) //注：buf = "{'"
		i--
		buf[i] = '\'' //注：buf = "'{'"
		i--
		buf[i] = ' ' //注：buf = " '{'"
	}
	// 将Unicode代码点u格式化为十六进制数。
	for u >= 16 { //注：buf = "B '{'"
		i--
		buf[i] = udigits[u&0xF]
		prec--
		u >>= 4
	}
	i--
	buf[i] = udigits[u] //注：buf = "7B '{'"
	prec--
	//在数字前加上零，直到达到要求的精度为止。
	for prec > 0 { //注：buf = "007B '{'"
		i--
		buf[i] = '0'
		prec--
	}
	// 添加前缀 "U+".
	i--
	buf[i] = '+'
	i--
	buf[i] = 'U' //注：buf = "U+007B '{'"

	oldZero := f.zero
	f.zero = false
	f.pad(buf[i:])
	f.zero = oldZero
}

// fmtInteger 格式化有符号和无符号整数。
func (f *fmt) fmtInteger(u uint64, base int, isSigned bool, verb rune, digits string) { //注：将u转为进制为base，根据isSigned是否显示符号，对应verb，根据digits设置大小写，写入f.buf种
	negative := isSigned && int64(u) < 0
	if negative { //注：如果有符号，而且u小于0
		u = -u //注：将u变为正数
	}

	buf := f.intbuf[0:]
	// 如果未设置精度或宽度，则已经分配的容量为68字节的f.intbuf足以进行整数格式化。
	if f.widPresent || f.precPresent { //注：设置了精度或宽度
		// 记入3个额外的字节，以便可能添加符号和"0x"。
		width := 3 + f.wid + f.prec // wid和prec总是正数。注：宽度为3位符号+宽度+精度
		if width > len(buf) {
			// 我们将需要更大的船。
			buf = make([]byte, width) //注：扩容buf
		}
	}

	// 要求额外的前导零位的两种方式：%.3d或%03d。
	// 如果都指定了f.zero标志，则将其忽略，并使用空格填充。
	prec := 0
	if f.precPresent { //注：设置了精度
		prec = f.prec
		// 精度为0且值为0表示除填充外，"不打印任何内容"。
		if prec == 0 && u == 0 {
			oldZero := f.zero
			f.zero = false
			f.writePadding(f.wid) //注：填充空格或0至足够的宽度
			f.zero = oldZero
			return
		}
	} else if f.zero && f.widPresent { //注：设置了0和宽度
		prec = f.wid
		if negative || f.plus || f.space { //注：u有符号且小于0或设置了+或设置了空格
			prec-- // 为标志留出空间
		}
	}

	// 因为从右到左打印更容易：将u格式化为buf，以buf[i]结尾。
	// 我们可以通过将32位的大小写拆分为一个单独的块来稍微加快速度，但这不值得重复，因此u有64位。
	i := len(buf)
	// 使用常数进行除法，并使用模数以获得更有效的代码。
	// 切换按流行程度排序的案例。
	switch base { //注：进制
	case 10:
		// 注：如果u = 123
		// next = 12	buf[i] = '03'
		// next = 1		buf[i-1] = '02'
		// next = 0		buf[i-2] = '01'
		for u >= 10 {
			i--
			next := u / 10
			buf[i] = byte('0' + u - next*10)
			u = next
		}
	case 16:
		// 注：
		// ldigits = "0123456789abcdefx"
		// 如果u = 123（0111 1011）
		// buf[i] = 'b'	取第1011（11）位	u = 0111
		// buf[i-1] = '7'	取第0111（7）位
		for u >= 16 {
			i--
			buf[i] = digits[u&0xF] //注：总是取u的后4位
			u >>= 4
		}
	case 8:
		// 注：
		// 如果u = 123（0111 1011）
		// buf[i] = '3'（011）		u = 0000 1111
		// buf[i-1] = '7'（111）	u = 0000 0001
		// buf[i-2] = '1'（001）
		for u >= 8 {
			i--
			buf[i] = byte('0' + u&7) //注：总是取u的后3位
			u >>= 3
		}
	case 2:
		// 注：
		// 如果u = 123（0111 1011），输出0111 1011
		for u >= 2 {
			i--
			buf[i] = byte('0' + u&1)
			u >>= 1
		}
	default:
		panic("fmt: unknown base; can't happen") //恐慌："未知进制；不可能发生"
	}
	i--
	buf[i] = digits[u]               //注：最后一位
	for i > 0 && prec > len(buf)-i { //注：设置了精度，填充0
		i--
		buf[i] = '0'
	}

	//各种前缀：0x - 等
	if f.sharp {
		switch base {
		case 2:
			// 添加前缀0b.
			i--
			buf[i] = 'b'
			i--
			buf[i] = '0'
		case 8:
			if buf[i] != '0' {
				i--
				buf[i] = '0'
			}
		case 16:
			// 添加前缀 0x 或 0X.
			i--
			buf[i] = digits[16]
			i--
			buf[i] = '0'
		}
	}
	if verb == 'O' { //注：%O需要添加0o前缀
		i--
		buf[i] = 'o'
		i--
		buf[i] = '0'
	}

	if negative { //注：如果有符号，而且u小于0
		i--
		buf[i] = '-'
	} else if f.plus { //注：如果需要显示+
		i--
		buf[i] = '+'
	} else if f.space { //注：如果需要显示空格
		i--
		buf[i] = ' '
	}

	//像以前的精度一样，已经使用零填充了左填充，或者由于显式设置的精度而忽略了f.zero标志。
	oldZero := f.zero
	f.zero = false
	f.pad(buf[i:])
	f.zero = oldZero
}

// truncate 将字符串s截断为指定的精度（如果存在）。
func (f *fmt) truncateString(s string) string { //注：返回符合精度要求的截取后的字符串
	if f.precPresent { //注：如果设置了精度
		n := f.prec
		for i := range s {
			n--
			if n < 0 {
				return s[:i] //注：返回符合精度要求的s
			}
		}
	}
	return s
}

// truncate 如果存在，则将字节片b截断为指定精度的字符串。
func (f *fmt) truncate(b []byte) []byte {
	if f.precPresent { //注：是否设置了精度
		n := f.prec
		for i := 0; i < len(b); { //注：遍历b的长度
			n--
			if n < 0 { //注：达到精度则返回截取后的b
				return b[:i]
			}
			wid := 1
			if b[i] >= utf8.RuneSelf { //注：计算下一个Rune占用多少字节
				_, wid = utf8.DecodeRune(b[i:])
			}
			i += wid //注：跳过一个Rune占用的字节
		}
	}
	return b
}

// 注：fmt函数命名方式
// S结尾：参数为字符串，表示格式化字符串
// Bs结尾：参数为字节切片，表示格式化字节切片
// x结尾：参数多了一个digits，只有ldigits（0123456789abcdefx）与udigits（"0123456789ABCDEFX"）两个方式，表示将参数转为16进制
// Sx就是将字符串转为16进制，Bx就是将字节切片转为16进制，Sbx就是将字符串或字节切片转为16进制
// Q结尾：表示将字符串添加单引号或双引号（使用Go语法安全地转义的单引号字符文字）
// C结尾：参数为Rune，表示格式化Unicode字符
// Qc就是将Rune格式化为带有单引号或双引号的Unicode字符

// fmtS 格式化字符串。
func (f *fmt) fmtS(s string) {
	s = f.truncateString(s) //注：根据精度截取字符串
	f.padString(s)          //注：f填充字符串s，复合宽度和精度要求
}

// fmtBs 格式化字节片b就像使用fmtS将其格式化为字符串一样。
func (f *fmt) fmtBs(b []byte) {
	b = f.truncate(b)
	f.pad(b)
}

// fmtSbx 将字符串或字节片格式化为其字节的十六进制编码。
func (f *fmt) fmtSbx(s string, b []byte, digits string) { //注：将s或b根据digits转义为16进制编码写入f，优先转换b
	length := len(b)
	if b == nil {
		// 没有字节切片。 假设字符串s应该被编码。
		length = len(s)
	}
	// 将长度设置为不处理超出精度要求的字节。
	if f.precPresent && f.prec < length { //注：根据精度限制s输出长度
		length = f.prec
	}
	// 考虑到f.sharp和f.space标志，计算编码宽度。
	width := 2 * length
	if width > 0 {
		if f.space {
			// 由两个十六进制编码的每个元素将获得前导0x或0X。
			if f.sharp { // 注：%#时，要增加0x或0X的宽度
				width *= 2
			}
			// 元素将以空格分隔。
			width += length - 1 //注：需要s或b的每个元素之间加一个空格的宽度
		} else if f.sharp {
			// 对于整个字符串，只会添加前导0x或0X。
			width += 2
		}
	} else { // 应该编码的字节片或字符串为空。
		if f.widPresent {
			f.writePadding(f.wid) //注：s与b均为nil，打印要求宽度的空格
		}
		return
	}
	// 处理左侧的填充。
	if f.widPresent && f.wid > width && !f.minus { //注：如果要求宽度大于要打印的字符串宽度，并且minus（反向填充）为false
		f.writePadding(f.wid - width) //注：左侧填充
	}
	// 将编码直接写入输出缓冲区。
	buf := *f.buf
	if f.sharp {
		// 添加前导0x或0X。
		buf = append(buf, '0', digits[16]) //注：buf填充0x或0X
	}
	var c byte
	for i := 0; i < length; i++ {
		if f.space && i > 0 { //注：如果设置了'% '，如果fmt.Printf("% x","123")，输出：31 32 33
			// 用空格分隔元素。
			buf = append(buf, ' ') //注：每个元素之间添加空格
			if f.sharp {           //注：如果设置了'% #'，如果fmt.Printf("% #x","123")，输出：0x31 0x32 0x33
				// 为每个元素添加前导0x或0X。
				buf = append(buf, '0', digits[16])
			}
		}
		if b != nil {
			c = b[i] // 从输入字节切片中获取一个字节。
		} else {
			c = s[i] // 从输入字符串中获取一个字节。
		}
		// 将每个字节编码为两个十六进制数字。
		// 注：fmt.Printf("%x","1")
		// 1的ASCII为49（0011 0001），49>>4为3（0011）digits得3，c&oxF（取后4位）为1（0001）digits得1
		// 最终"1"输出"31"
		buf = append(buf, digits[c>>4], digits[c&0xF])
	}
	*f.buf = buf
	// 处理右边的填充。
	if f.widPresent && f.wid > width && f.minus { //注：如果为反向填充(%-)，右填充
		f.writePadding(f.wid - width)
	}
}

// fmtSx 将字符串格式化为其字节的十六进制编码。
func (f *fmt) fmtSx(s, digits string) { //注：将字符串s转为16进制编码，大小写根据digits实现
	f.fmtSbx(s, nil, digits)
}

// fmtBx 将字节片格式化为其字节的十六进制编码。
func (f *fmt) fmtBx(b []byte, digits string) { //注：将字节数组b转为16进制编码，大小写根据digits实现
	f.fmtSbx("", b, digits)
}

// fmtQ 将字符串格式化为双引号，转义的Go字符串常量。
// 如果设置了f.sharp，则该字符串不包含制表符以外的任何控制字符，都可以返回原始（带反引号）的字符串。
func (f *fmt) fmtQ(s string) {
	s = f.truncateString(s) //注：截取字符串至要求的长度
	if f.sharp && strconv.CanBackquote(s) {
		f.padString("`" + s + "`") //注：填充`s`
		return
	}
	buf := f.intbuf[:0]
	if f.plus {
		f.pad(strconv.AppendQuoteToASCII(buf, s))
	} else {
		f.pad(strconv.AppendQuote(buf, s))
	}
}

// fmtC 将整数格式化为Unicode字符。
// 如果该字符不是有效的Unicode，它将显示'\ufffd'。
func (f *fmt) fmtC(c uint64) { //注：将c转为Rune写入f
	r := rune(c)
	if c > utf8.MaxRune { //注：如果c超过最大的Rune值
		r = utf8.RuneError //注：c的字符替换为错误字符
	}
	buf := f.intbuf[:0]
	w := utf8.EncodeRune(buf[:utf8.UTFMax], r) //注：将r转为Rune写入f
	f.pad(buf[:w])
}

// fmtQc 将整数格式化为单引号，转义的Go字符常量。
// 如果该字符不是有效的Unicode，它将显示'\ufffd'。
func (f *fmt) fmtQc(c uint64) {
	r := rune(c)
	if c > utf8.MaxRune {
		r = utf8.RuneError
	}
	buf := f.intbuf[:0]
	if f.plus {
		f.pad(strconv.AppendQuoteRuneToASCII(buf, r))
	} else {
		f.pad(strconv.AppendQuoteRune(buf, r))
	}
}

// fmtFloat 格式化float64。 它假定动词是strconv.AppendFloat的有效格式说明符，因此适合一个字节。
func (f *fmt) fmtFloat(v float64, size int, verb rune, prec int) { //注：#将v转为verb格式，默认精度为prec，写入f
	// 格式说明符中的显式精度优先于默认精度。
	if f.precPresent { //注：获取精度
		prec = f.prec
	}
	// 格式编号，必要时保留前导+符号的空间。
	num := strconv.AppendFloat(f.intbuf[:1], v, byte(verb), prec, size)
	if num[1] == '-' || num[1] == '+' {
		num = num[1:]
	} else {
		num[0] = '+'
	}
	// f.space表示添加前导空格而不是“ +”符号，除非f.plus明确要求使用该符号。
	if f.space && num[0] == '+' && !f.plus {
		num[0] = ' '
	}
	// 对Infinities和NaN的特殊处理，它们看起来不像数字，因此不应用零填充。
	if num[1] == 'I' || num[1] == 'N' {
		oldZero := f.zero
		f.zero = false
		// Remove sign before NaN if not asked for.
		if num[1] == 'N' && !f.space && !f.plus {
			num = num[1:]
		}
		f.pad(num)
		f.zero = oldZero
		return
	}
	// 尖锐的标志会强制为非二进制格式打印小数点，并保留尾随零，我们可能需要将其恢复。
	if f.sharp && verb != 'b' {
		digits := 0
		switch verb {
		case 'v', 'g', 'G', 'x':
			digits = prec
			//如果未明确设置精度，则使用精度6。
			if digits == -1 {
				digits = 6
			}
		}

		// Buffer pre-allocated with enough room for
		// exponent notations of the form "e+123" or "p-1023".
		var tailBuf [6]byte
		tail := tailBuf[:0]

		hasDecimalPoint := false
		// Starting from i = 1 to skip sign at num[0].
		for i := 1; i < len(num); i++ {
			switch num[i] {
			case '.':
				hasDecimalPoint = true
			case 'p', 'P':
				tail = append(tail, num[i:]...)
				num = num[:i]
			case 'e', 'E':
				if verb != 'x' && verb != 'X' {
					tail = append(tail, num[i:]...)
					num = num[:i]
					break
				}
				fallthrough
			default:
				digits--
			}
		}
		if !hasDecimalPoint {
			num = append(num, '.')
		}
		for digits > 0 {
			num = append(num, '0')
			digits--
		}
		num = append(num, tail...)
	}
	// We want a sign if asked for and if the sign is not positive.
	if f.plus || num[0] != '+' {
		// If we're zero padding to the left we want the sign before the leading zeros.
		// Achieve this by writing the sign out and then padding the unsigned number.
		if f.zero && f.widPresent && f.wid > len(num) {
			f.buf.writeByte(num[0])
			f.writePadding(f.wid - len(num))
			f.buf.write(num[1:])
			return
		}
		f.pad(num)
		return
	}
	// No sign to show and the number is positive; just print the unsigned number.
	f.pad(num[1:])
}
