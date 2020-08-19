// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package strconv

import "errors"

// lower(c) 当且仅当c是该小写字母或等效的大写字母时，它是小写字母。
// 而不是写c =='x' || c =='X'可以写lower(c) == 'x'。
// 请注意，较低的非字母会产生其他非字母。
func lower(c byte) byte { //注：将ASCII字母转为小写
	// 注：'x' - 'X' = 0010 0000
	// 例1：c = 'g'		103		0110 0111
	// 得：0110 0111，为g本身
	//
	// 例2：c = 'G'		71		0100 0111
	// 得：0110 0111，为小写g

	return c | ('x' - 'X') //注：将第2位变为1（32），大写字母 + 32 = 小写字母
}

// ErrRange 指示值超出目标类型的范围。
var ErrRange = errors.New("value out of range") //注：值超出范围

// ErrSyntax 指示一个值对于目标类型没有正确的语法。
var ErrSyntax = errors.New("invalid syntax") //注：无效的语法

// NumError 记录转换失败。
type NumError struct { //注：字符串转为数字的错误
	Func string // 失败的函数（ParseBool，ParseInt，ParseUint，ParseFloat），注：发生错误的函数
	Num  string // 输入，注：转换数字出错的字符串
	Err  error  // 转换失败的原因（例如ErrRange，ErrSyntax等），注：转换失败的原因
}

func (e *NumError) Error() string { //注：输出错误，函数解析错误
	return "strconv." + e.Func + ": " + "parsing " + Quote(e.Num) + ": " + e.Err.Error()
}

func (e *NumError) Unwrap() error { return e.Err } //注：返回error

func syntaxError(fn, str string) *NumError { //注：工厂函数，生成"无效的语法"错误
	return &NumError{fn, str, ErrSyntax}
}

func rangeError(fn, str string) *NumError { //注：工厂函数，生成"值超出范围"错误
	return &NumError{fn, str, ErrRange}
}

func baseError(fn, str string, base int) *NumError { //注：工厂函数，生成"无效的进制"错误
	return &NumError{fn, str, errors.New("invalid base " + Itoa(base))}
}

func bitSizeError(fn, str string, bitSize int) *NumError { //注：工厂函数，生成"无效的位数"错误
	return &NumError{fn, str, errors.New("invalid bit size " + Itoa(bitSize))}
}

const intSize = 32 << (^uint(0) >> 63) //注：int的位数，问：为什么不把这些封装到一个包里

// IntSize 是int或uint值的大小（以位为单位）。
const IntSize = intSize

const maxUint64 = 1<<64 - 1 //注：uint64的最大值

// ParseUint 类似于ParseInt，但用于无符号数字。
func ParseUint(s string, base int, bitSize int) (uint64, error) { //注：将s转为base进制bitSize正整数并返回
	const fnParseUint = "ParseUint"

	if s == "" {
		return 0, syntaxError(fnParseUint, s)
	}

	base0 := base == 0

	s0 := s
	switch {
	case 2 <= base && base <= 36:
		// 有效进制 没事做

	case base == 0:
		// 查找八进制十六进制前缀。
		base = 10
		if s[0] == '0' {
			switch {
			case len(s) >= 3 && lower(s[1]) == 'b': //注：0b是2进制
				base = 2
				s = s[2:]
			case len(s) >= 3 && lower(s[1]) == 'o': //注：0o是8进制
				base = 8
				s = s[2:]
			case len(s) >= 3 && lower(s[1]) == 'x': //注：0x是10进制
				base = 16
				s = s[2:]
			default: //注：0开头是8进制
				base = 8
				s = s[1:]
			}
		}

	default:
		return 0, baseError(fnParseUint, s0, base) //注：进制错误
	}

	if bitSize == 0 { //注：默认位数与int相同
		bitSize = int(IntSize)
	} else if bitSize < 0 || bitSize > 64 {
		return 0, bitSizeError(fnParseUint, s0, bitSize) //注：位数错误
	}

	// Cutoff 是最小的数字，以使cutoff * base > maxUint64。
	// 在常见情况下使用编译时常量。
	var cutoff uint64 //注：再乘以进制就会溢出
	switch base {
	case 10:
		cutoff = maxUint64/10 + 1
	case 16:
		cutoff = maxUint64/16 + 1
	default:
		cutoff = maxUint64/uint64(base) + 1
	}

	maxVal := uint64(1)<<uint(bitSize) - 1 //注：64位无符号整数最大值

	underscores := false
	var n uint64
	for _, c := range []byte(s) {
		var d byte
		switch {
		case c == '_' && base0: //注：遇到下划线，并且不知道进制
			underscores = true
			continue
		case '0' <= c && c <= '9': //注：0-9
			d = c - '0'
		case 'a' <= lower(c) && lower(c) <= 'z': //注：a-z
			d = lower(c) - 'a' + 10
		default:
			return 0, syntaxError(fnParseUint, s0) //注：语法错误
		}

		if d >= byte(base) { //注：1位数超过进制
			return 0, syntaxError(fnParseUint, s0) //注：语法错误
		}

		if n >= cutoff {
			// n*base 超界
			return maxVal, rangeError(fnParseUint, s0) //注：超界错误
		}
		n *= uint64(base) //注：进位

		n1 := n + uint64(d)
		if n1 < n || n1 > maxVal {
			// n+v 超界
			return maxVal, rangeError(fnParseUint, s0)
		}
		n = n1
	}

	if underscores && !underscoreOK(s0) {
		return 0, syntaxError(fnParseUint, s0)
	}

	return n, nil
}

// ParseInt 解析给定进制（0，2到36）和位大小（0到64）中的字符串s，并返回相应的值i。
//
// 如果base参数为0，则字符串的前缀隐含真实的base："0b"为2，"0”或"0o"为8，"0x"为16，否则为10。
// 同样，仅对于以0为底的参数，下划线字符是允许的，如Go语法对整数文字所定义的。
//
// bitSize参数指定结果必须适合的整数类型。位大小0、8、16、32和64分别对应于int，int8，int16，int32和int64。
// 如果bitSize小于0或大于64，则返回错误。
//
// ParseInt返回的错误的具体类型为*NumError并包含err.Num = s。如果s为空或包含无效数字，则err.Err = ErrSyntax，返回值为0；否则，返回0。
// 如果与s对应的值不能用给定大小的有符号整数表示，则err.Err = ErrRange，并且返回的值是适当的bitSize和sign的最大大小整数。
func ParseInt(s string, base int, bitSize int) (i int64, err error) { //注：将s转为base进制bitSize整数并返回
	const fnParseInt = "ParseInt"

	if s == "" {
		return 0, syntaxError(fnParseInt, s) //注：空字符串返回错误
	}

	// 挑选前缀标志。
	s0 := s
	neg := false
	if s[0] == '+' {
		s = s[1:]
	} else if s[0] == '-' {
		neg = true
		s = s[1:]
	}

	// 转换无符号并检查范围。
	var un uint64
	un, err = ParseUint(s, base, bitSize)              //注：转为uint
	if err != nil && err.(*NumError).Err != ErrRange { //注：如果转为uint失败，返回错误
		err.(*NumError).Func = fnParseInt
		err.(*NumError).Num = s0
		return 0, err
	}

	if bitSize == 0 { //注：没要求位数，则与int相同
		bitSize = int(IntSize)
	}

	cutoff := uint64(1 << uint(bitSize-1)) //注：1后面63个0
	if !neg && un >= cutoff {              //注：不是负数，但是符号位是1，返回错误
		return int64(cutoff - 1), rangeError(fnParseInt, s0)
	}
	if neg && un > cutoff { //注：是负数，但是 >0，返回错误
		return -int64(cutoff), rangeError(fnParseInt, s0)
	}
	n := int64(un)
	if neg {
		n = -n
	}
	return n, nil
}

// Atoi 等效于ParseInt(s, 10, 0)，转换为int类型。
func Atoi(s string) (int, error) { //注：将字符串s转为int返回
	const fnAtoi = "Atoi"

	sLen := len(s)
	if intSize == 32 && (0 < sLen && sLen < 10) ||
		intSize == 64 && (0 < sLen && sLen < 19) {
		// 适合int类型的小整数的快速路径。
		s0 := s
		if s[0] == '-' || s[0] == '+' { //注：检查符号
			s = s[1:]
			if len(s) < 1 {
				return 0, &NumError{fnAtoi, s0, ErrSyntax}
			}
		}

		n := 0
		for _, ch := range []byte(s) { //注：获取数字
			ch -= '0'
			if ch > 9 {
				return 0, &NumError{fnAtoi, s0, ErrSyntax}
			}
			n = n*10 + int(ch)
		}
		if s0[0] == '-' { //注：判断负号
			n = -n
		}
		return n, nil
	}

	// 无效，较大或带下划线整数的慢速路径。
	i64, err := ParseInt(s, 10, 0)
	if nerr, ok := err.(*NumError); ok {
		nerr.Func = fnAtoi
	}
	return int(i64), err
}

// underscoreOK 报告是否允许s中的下划线。
// 在此功能中检查它们可以使所有解析器简单地跳过它们。
// 下划线必须仅出现在数字之间或基本前缀与数字之间。
func underscoreOK(s string) bool {
	// a跟踪我们看到的最后一个字符（类）：
	//  ^ 为数字的开头，
	//  0 表示数字或基本前缀，
	//  _ 为下划线，
	//  ! 以上都不是。
	saw := '^'
	i := 0

	// 可选标志。
	if len(s) >= 1 && (s[0] == '-' || s[0] == '+') { //注：正负号
		s = s[1:]
	}

	// 可选的基本前缀。
	hex := false
	if len(s) >= 2 && s[0] == '0' && (lower(s[1]) == 'b' || lower(s[1]) == 'o' || lower(s[1]) == 'x') { //注：遇到0b、0o、0x
		i = 2
		saw = '0' // 基本前缀算作"下划线作为数字分隔符"的数字
		hex = lower(s[1]) == 'x'
	}

	// 数字正确。
	for ; i < len(s); i++ {
		// 数字总是可以的。
		if '0' <= s[i] && s[i] <= '9' || hex && 'a' <= lower(s[i]) && lower(s[i]) <= 'f' {
			saw = '0'
			continue
		}
		// 下划线必须跟在数字后面。
		if s[i] == '_' {
			if saw != '0' {
				return false
			}
			saw = '_'
			continue
		}
		// 下划线也必须跟数字。
		if saw == '_' {
			return false
		}
		// 看到非数字，非下划线。
		saw = '!'
	}
	return saw != '_'
}
