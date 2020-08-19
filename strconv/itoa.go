// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。
package strconv

import "math/bits"

const fastSmalls = true //为小整数启用快速路径

// FormatUint 返回给定进制i的字符串表示形式，为2 <= 进制 <= 36。结果使用小写字母'a'至'z'表示数字值 >= 10。
func FormatUint(i uint64, base int) string { //注：
	if fastSmalls && i < nSmalls && base == 10 { //注：10进制100以内的正数且启动快速路径，根据常量直接获取
		return small(int(i))
	}
	_, s := formatBits(nil, i, base, false, false)
	return s
}

// FormatInt 返回给定base中i的字符串表示形式，表示2 <= base <=36。结果使用小写字母'a'到'z'表示数字值> = 10。
func FormatInt(i int64, base int) string {
	if fastSmalls && 0 <= i && i < nSmalls && base == 10 { //注：10进制100以内的正数且启动快速路径，根据常量直接获取
		return small(int(i))
	}
	_, s := formatBits(nil, uint64(i), base, i < 0, false)
	return s
}

// Itoa 等效于FormatInt(int64(i), 10)。
func Itoa(i int) string { //注：将i作为10进制数字转为字符串
	return FormatInt(int64(i), 10)
}

// AppendInt 将由FormatInt生成的整数i的字符串形式附加到dst，然后返回扩展缓冲区。
func AppendInt(dst []byte, i int64, base int) []byte {
	if fastSmalls && 0 <= i && i < nSmalls && base == 10 { //注：10进制100以内的正数且启动快速路径，根据常量直接获取
		return append(dst, small(int(i))...)
	}
	dst, _ = formatBits(dst, uint64(i), base, i < 0, true)
	return dst
}

// AppendUint 将由FormatUint生成的无符号整数i的字符串形式附加到dst，然后返回扩展缓冲区。
func AppendUint(dst []byte, i uint64, base int) []byte {
	if fastSmalls && i < nSmalls && base == 10 {
		return append(dst, small(int(i))...)
	}
	dst, _ = formatBits(dst, i, base, false, true)
	return dst
}

// small 返回0 <= i < nSmalls的i的字符串。
func small(i int) string { //注：返回i的字符串形式
	if i < 10 { //注：如果i<10，返回1位数的i字符串
		return digits[i : i+1]
	}
	return smallsString[i*2 : i*2+2] //注：如果i>10，返回最大2位的i字符串
}

const nSmalls = 100

const smallsString = "00010203040506070809" + //注：00-99
	"10111213141516171819" +
	"20212223242526272829" +
	"30313233343536373839" +
	"40414243444546474849" +
	"50515253545556575859" +
	"60616263646566676869" +
	"70717273747576777879" +
	"80818283848586878889" +
	"90919293949596979899"

const host32bit = ^uint(0)>>32 == 0 //注：本机是否为32位

const digits = "0123456789abcdefghijklmnopqrstuvwxyz" //注：数字与字母

// formatBits 计算给定基数中u的字符串表示形式。
// 如果设置了neg，则u被视为int64负值。 如果设置了append_，
// 则将字符串追加到dst并将结果字节片作为第一个结果值返回； 否则，将字符串作为第二个结果值返回。
//
func formatBits(dst []byte, u uint64, base int, neg, append_ bool) (d []byte, s string) {
	if base < 2 || base > len(digits) {
		panic("strconv: illegal AppendInt/FormatInt base") //恐慌："非法的AppendInt/FormatInt进制"
	}
	// 2 <= base && base <= len(digits)

	var a [64 + 1]byte // +1用于以2为底的64位值的符号
	i := len(a)

	if neg {
		u = -u
	}

	// 转换位
	// 我们在可能的地方使用uint值，因为即使在32位计算机上，它们也可以放入单个寄存器中。
	if base == 10 { //注：10进制32位数字，且>1e9
		// 常见情况：对/使用常量，因为编译器可以将其优化为乘法+移位
		if host32bit {
			// 使用32位运算转换低位数字
			for u >= 1e9 {
				// 由于在32位计算机上运行时函数会计算64位除法和模运算，因此请避免在q = a/b之外使用r = a%b。
				q := u / 1e9          //注：1234 / 1000 = 1
				us := uint(u - q*1e9) // u%1e9适合一个单位，注：1234 - 1 * 1000 = 234
				for j := 4; j > 0; j-- {
					is := us % 100 * 2 //注：234 % 100 * 2 = 68
					us /= 100          //注： 234 / 100 = 2
					i -= 2
					a[i+1] = smallsString[is+1] //注：52
					a[i+0] = smallsString[is+0] //注：51
				}
				// us < 10，因为它包含前9位数字us的最后一位数字。
				i--
				a[i] = smallsString[us*2+1] //注：50

				u = q //注：u = 1
			}
			// u < 1e9
		}

		// u保证适合一个单位
		us := uint(u)
		for us >= 100 {
			is := us % 100 * 2
			us /= 100
			i -= 2
			a[i+1] = smallsString[is+1]
			a[i+0] = smallsString[is+0]
		}

		// us < 100
		is := us * 2 //注：1 * 2 = 2
		i--
		a[i] = smallsString[is+1] //注：49
		if us >= 10 {
			i--
			a[i] = smallsString[is]
		}

	} else if isPowerOfTwo(base) {
		// Use shifts and masks instead of / and %.
		// Base is a power of 2 and 2 <= base <= len(digits) where len(digits) is 36.
		// The largest power of 2 below or equal to 36 is 32, which is 1 << 5;
		// i.e., the largest possible shift count is 5. By &-ind that value with
		// the constant 7 we tell the compiler that the shift count is always
		// less than 8 which is smaller than any register width. This allows
		// the compiler to generate better code for the shift operation.
		shift := uint(bits.TrailingZeros(uint(base))) & 7
		b := uint64(base)
		m := uint(base) - 1 // == 1<<shift - 1
		for u >= b {
			i--
			a[i] = digits[uint(u)&m]
			u >>= shift
		}
		// u < base
		i--
		a[i] = digits[uint(u)]
	} else {
		// general case
		b := uint64(base)
		for u >= b {
			i--
			// Avoid using r = a%b in addition to q = a/b
			// since 64bit division and modulo operations
			// are calculated by runtime functions on 32bit machines.
			q := u / b
			a[i] = digits[uint(u-q*b)]
			u = q
		}
		// u < base
		i--
		a[i] = digits[uint(u)]
	}

	// add sign, if any
	if neg {
		i--
		a[i] = '-'
	}

	if append_ {
		d = append(dst, a[i:]...)
		return
	}
	s = string(a[i:])
	return
}

func isPowerOfTwo(x int) bool { //注：x是否是2的指数
	return x&(x-1) == 0 //注：除第1位之外要求其余都是0，例1：1000 0000，例2：1000
}
