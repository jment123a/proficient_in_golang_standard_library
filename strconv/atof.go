// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package strconv

// 十进制转换为二进制浮点数。
// 算法：
//  1）将输入存储在多精度十进制中。
//  2）将小数乘以/除以2直到在[0.5，1）范围内
//  3）乘以2^精度并四舍五入得到尾数。

import "math"

var optimize = true // 设置为false以强制进行慢路径转换以进行测试

func equalIgnoreCase(s1, s2 string) bool { //注：返回s1是否等于s2（不区分大小写）
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ { //注：遍历s1
		c1 := s1[i]
		if 'A' <= c1 && c1 <= 'Z' { //注：s1转为小写
			c1 += 'a' - 'A'
		}
		c2 := s2[i]
		if 'A' <= c2 && c2 <= 'Z' { //注：s2转为大写
			c2 += 'a' - 'A'
		}
		if c1 != c2 { //注：不同则返回false
			return false
		}
	}
	return true //注：全部相同则返回true
}

func special(s string) (f float64, ok bool) { //注：将s格式化为无穷大与非数字f与是否格式化成功ok
	if len(s) == 0 {
		return
	}
	switch s[0] {
	default:
		return
	case '+':
		if equalIgnoreCase(s, "+inf") || equalIgnoreCase(s, "+infinity") {
			return math.Inf(1), true
		}
	case '-':
		if equalIgnoreCase(s, "-inf") || equalIgnoreCase(s, "-infinity") {
			return math.Inf(-1), true
		}
	case 'n', 'N':
		if equalIgnoreCase(s, "nan") {
			return math.NaN(), true
		}
	case 'i', 'I':
		if equalIgnoreCase(s, "inf") || equalIgnoreCase(s, "infinity") {
			return math.Inf(1), true
		}
	}
	return
}

func (b *decimal) set(s string) (ok bool) {
	i := 0
	b.neg = false   //注：为负数
	b.trunc = false //注：截断

	// 可选标志
	if i >= len(s) {
		return
	}
	switch { //注：是否为负数
	case s[i] == '+':
		i++
	case s[i] == '-':
		b.neg = true
		i++
	}

	// 数字
	sawdot := false    //注：遇到.
	sawdigits := false //注：遇到数字
	for ; i < len(s); i++ {
		switch {
		case s[i] == '_':
			// readFloat已检查下划线
			continue
		case s[i] == '.':
			if sawdot { //注：遇到第2个.返回
				return
			}
			sawdot = true
			b.dp = b.nd
			continue

		// 例1：b.d = ""，b.nd = 0，b.dp = 0，s = "0123"
		// 执行以下代码1次：b.d = ""，b.nd = 0，b.dp = -1，s = "0123"
		// 执行以下代码2次：发生截取
		//
		// 例2：b.d = "12345"，b.nd = 5，b.dp = 0，s = "0123"
		// 执行以下代码1次：b.d = ""，b.nd = 0，b.dp = -1，s = "0123"
		// 执行以下代码2次：
		case '0' <= s[i] && s[i] <= '9':
			sawdigits = true
			if s[i] == '0' && b.nd == 0 { // 忽略前导零
				b.dp--
				continue
			}
			if b.nd < len(b.d) {
				b.d[b.nd] = s[i]
				b.nd++
			} else if s[i] != '0' {
				b.trunc = true
			}
			continue
		}
		break
	}
	if !sawdigits {
		return
	}
	if !sawdot {
		b.dp = b.nd
	}

	// optional exponent moves decimal point.
	// if we read a very large, very long number,
	// just be sure to move the decimal point by
	// a lot (say, 100000).  it doesn't matter if it's
	// not the exact number.
	if i < len(s) && lower(s[i]) == 'e' {
		i++
		if i >= len(s) {
			return
		}
		esign := 1
		if s[i] == '+' {
			i++
		} else if s[i] == '-' {
			i++
			esign = -1
		}
		if i >= len(s) || s[i] < '0' || s[i] > '9' {
			return
		}
		e := 0
		for ; i < len(s) && ('0' <= s[i] && s[i] <= '9' || s[i] == '_'); i++ {
			if s[i] == '_' {
				// readFloat already checked underscores
				continue
			}
			if e < 10000 {
				e = e*10 + int(s[i]) - '0'
			}
		}
		b.dp += e * esign
	}

	if i != len(s) {
		return
	}

	ok = true
	return
}

// readFloat 从浮点字符串表示形式读取十进制尾数和指数。 如果该数字无效，则返回ok == false。
func readFloat(s string) (mantissa uint64, exp int, neg, trunc, hex, ok bool) { //注：将字符串表示的s读取十进制的整数mantissa，返回整数mantissa，指数的位置xpe，是否为负数neg，是否截取字符串trunc，是否为16进制hex，是否获取成功ok
	// 形参：
	// mantissa：数据
	// exp：指数在数字中的位置
	// neg：是否为负数
	// trunc：是否截取
	// hex：是否为16进制
	// ok：是否成功转换为float
	//
	// underscores：是否出现下划线
	// base：进制
	// maxMantDigits：数据的最大位数
	// expChar：指数词（e或p）
	// sawdot：是否遇到小数点
	// sawdigits：是否遇到数字
	// nd：数据的位长度
	// ndMant：数据的位数
	// dp:数据小数点的位置
	i := 0
	underscores := false

	// 可选标志
	if i >= len(s) {
		return
	}
	switch { //注：遇到符号
	case s[i] == '+':
		i++
	case s[i] == '-':
		neg = true
		i++
	}

	// 数字
	base := uint64(10)
	maxMantDigits := 19 // 10^19 适合uint64
	expChar := byte('e')
	if i+2 < len(s) && s[i] == '0' && lower(s[i+1]) == 'x' { //注：s的连续两个字符是0x或0X
		base = 16          //注：16进制
		maxMantDigits = 16 // 16^16 适合uint64
		i += 2
		expChar = 'p'
		hex = true
	}
	sawdot := false
	sawdigits := false
	nd := 0
	ndMant := 0
	dp := 0
	for ; i < len(s); i++ {
		switch c := s[i]; true {
		case c == '_':
			underscores = true //注：出现下划线
			continue

		case c == '.':
			if sawdot {
				return
			}
			sawdot = true //注：出现小数点.
			dp = nd
			continue

		case '0' <= c && c <= '9': //注：遇到0-9
			sawdigits = true         //注：出现数字
			if c == '0' && nd == 0 { // 忽略前导零
				dp--
				continue
			}
			nd++
			if ndMant < maxMantDigits { //注：当前数字为超过最大数值位数
				mantissa *= base
				mantissa += uint64(c - '0')
				ndMant++
			} else if c != '0' { //注：超过了则标记截取
				trunc = true
			}
			continue
			// 例：如果s = "abc"
			// 执行以下代码第1次： 'a' - 'a' + 10 = 10（1010）
			// 执行以下代码第2次： 'b' - 'a' + 10 = 11，10 * 16 + 11 = 171（1010 1011）
			// 执行以下代码第3次： 'c' - 'a' + 10 = 12, 171 * 16 + 12 = 2748（1010 1011 1100）
		case base == 16 && 'a' <= lower(c) && lower(c) <= 'f': //注：遇到a-f
			sawdigits = true
			nd++
			if ndMant < maxMantDigits {
				mantissa *= 16
				mantissa += uint64(lower(c) - 'a' + 10)
				ndMant++
			} else {
				trunc = true
			}
			continue
		}
		break
	}
	if !sawdigits { //注：没有出现数字
		return
	}
	if !sawdot { //注：没有出现小数点
		dp = nd
	}

	if base == 16 { //注：#
		dp *= 4
		ndMant *= 4
	}

	// 可选指数移动小数点。
	// 如果我们读取的是非常大的非常长的数字，请确保将小数点移动很多（例如100000）。 这不是确切的数字也没关系。
	if i < len(s) && lower(s[i]) == expChar { //注：遇到指数（e或p）
		i++
		if i >= len(s) { //注：e或p必须要有数字
			return
		}
		esign := 1 //注：指数为正还是负
		if s[i] == '+' {
			i++
		} else if s[i] == '-' {
			i++
			esign = -1
		}
		if i >= len(s) || s[i] < '0' || s[i] > '9' { //注：指数必须为数字
			return
		}
		e := 0                                                                 //注：e后面的指数
		for ; i < len(s) && ('0' <= s[i] && s[i] <= '9' || s[i] == '_'); i++ { //注：计算指数
			if s[i] == '_' {
				underscores = true
				continue
			}
			if e < 10000 {
				e = e*10 + int(s[i]) - '0'
			}
		}
		dp += e * esign
	} else if base == 16 {
		// 必须具有指数。
		return
	}

	if i != len(s) {
		return
	}

	if mantissa != 0 {
		exp = dp - ndMant
	}

	if underscores && !underscoreOK(s) { //注：下划线格式错误
		return
	}

	ok = true
	return
}

// 十进制的十进制幂到二的二进制幂。
var powtab = []int{1, 3, 6, 9, 13, 16, 19, 23, 26}

func (d *decimal) floatBits(flt *floatInfo) (b uint64, overflow bool) {
	var exp int
	var mant uint64

	// Zero is always a special case.
	if d.nd == 0 {
		mant = 0
		exp = flt.bias
		goto out
	}

	// Obvious overflow/underflow.
	// These bounds are for 64-bit floats.
	// Will have to change if we want to support 80-bit floats in the future.
	if d.dp > 310 {
		goto overflow
	}
	if d.dp < -330 {
		// zero
		mant = 0
		exp = flt.bias
		goto out
	}

	// Scale by powers of two until in range [0.5, 1.0)
	exp = 0
	for d.dp > 0 {
		var n int
		if d.dp >= len(powtab) {
			n = 27
		} else {
			n = powtab[d.dp]
		}
		d.Shift(-n)
		exp += n
	}
	for d.dp < 0 || d.dp == 0 && d.d[0] < '5' {
		var n int
		if -d.dp >= len(powtab) {
			n = 27
		} else {
			n = powtab[-d.dp]
		}
		d.Shift(n)
		exp -= n
	}

	// Our range is [0.5,1) but floating point range is [1,2).
	exp--

	// Minimum representable exponent is flt.bias+1.
	// If the exponent is smaller, move it up and
	// adjust d accordingly.
	if exp < flt.bias+1 {
		n := flt.bias + 1 - exp
		d.Shift(-n)
		exp += n
	}

	if exp-flt.bias >= 1<<flt.expbits-1 {
		goto overflow
	}

	// Extract 1+flt.mantbits bits.
	d.Shift(int(1 + flt.mantbits))
	mant = d.RoundedInteger()

	// Rounding might have added a bit; shift down.
	if mant == 2<<flt.mantbits {
		mant >>= 1
		exp++
		if exp-flt.bias >= 1<<flt.expbits-1 {
			goto overflow
		}
	}

	// Denormalized?
	if mant&(1<<flt.mantbits) == 0 {
		exp = flt.bias
	}
	goto out

overflow:
	// ±Inf
	mant = 0
	exp = 1<<flt.expbits - 1 + flt.bias
	overflow = true

out:
	// Assemble bits.
	bits := mant & (uint64(1)<<flt.mantbits - 1)
	bits |= uint64((exp-flt.bias)&(1<<flt.expbits-1)) << flt.mantbits
	if d.neg {
		bits |= 1 << flt.mantbits << flt.expbits
	}
	return bits, overflow
}

// 精确的10的幂。
var float64pow10 = []float64{ //注：在调用时初始化
	1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
	1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
	1e20, 1e21, 1e22,
}
var float32pow10 = []float32{1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10} //注：在调用时初始化

// 如果可能的话，完全在浮点数学运算中将十进制表示形式精确地转换为64位浮点f，则可以这样做，避免了使用decimalToFloatBits。
// 三种常见情况：
// 值是正整数
// 值是正整数 * 10的幂
// 值是正整数 / 10的幂
// 这些都会产生可能不精确但正确的舍入答案。
func atof64exact(mantissa uint64, exp int, neg bool) (f float64, ok bool) {
	if mantissa>>float64info.mantbits != 0 {
		return
	}
	f = float64(mantissa)
	if neg {
		f = -f
	}
	switch {
	case exp == 0:
		// an integer.
		return f, true
	// Exact integers are <= 10^15.
	// Exact powers of ten are <= 10^22.
	case exp > 0 && exp <= 15+22: // int * 10^k
		// If exponent is big but number of digits is not,
		// can move a few zeros into the integer part.
		if exp > 22 {
			f *= float64pow10[exp-22]
			exp = 22
		}
		if f > 1e15 || f < -1e15 {
			// the exponent was really too large.
			return
		}
		return f * float64pow10[exp], true
	case exp < 0 && exp >= -22: // int / 10^k
		return f / float64pow10[-exp], true
	}
	return
}

// If possible to compute mantissa*10^exp to 32-bit float f exactly,
// entirely in floating-point math, do so, avoiding the machinery above.
func atof32exact(mantissa uint64, exp int, neg bool) (f float32, ok bool) {
	if mantissa>>float32info.mantbits != 0 {
		return
	}
	f = float32(mantissa)
	if neg {
		f = -f
	}
	switch {
	case exp == 0:
		return f, true
	// Exact integers are <= 10^7.
	// Exact powers of ten are <= 10^10.
	case exp > 0 && exp <= 7+10: // int * 10^k
		// If exponent is big but number of digits is not,
		// can move a few zeros into the integer part.
		if exp > 10 {
			f *= float32pow10[exp-10]
			exp = 10
		}
		if f > 1e7 || f < -1e7 {
			// the exponent was really too large.
			return
		}
		return f * float32pow10[exp], true
	case exp < 0 && exp >= -10: // int / 10^k
		return f / float32pow10[-exp], true
	}
	return
}

// atofHex converts the hex floating-point string s
// to a rounded float32 or float64 value (depending on flt==&float32info or flt==&float64info)
// and returns it as a float64.
// The string s has already been parsed into a mantissa, exponent, and sign (neg==true for negative).
// If trunc is true, trailing non-zero bits have been omitted from the mantissa.
func atofHex(s string, flt *floatInfo, mantissa uint64, exp int, neg, trunc bool) (float64, error) {
	maxExp := 1<<flt.expbits + flt.bias - 2
	minExp := flt.bias + 1
	exp += int(flt.mantbits) // mantissa now implicitly divided by 2^mantbits.

	// Shift mantissa and exponent to bring representation into float range.
	// Eventually we want a mantissa with a leading 1-bit followed by mantbits other bits.
	// For rounding, we need two more, where the bottom bit represents
	// whether that bit or any later bit was non-zero.
	// (If the mantissa has already lost non-zero bits, trunc is true,
	// and we OR in a 1 below after shifting left appropriately.)
	for mantissa != 0 && mantissa>>(flt.mantbits+2) == 0 {
		mantissa <<= 1
		exp--
	}
	if trunc {
		mantissa |= 1
	}
	for mantissa>>(1+flt.mantbits+2) != 0 {
		mantissa = mantissa>>1 | mantissa&1
		exp++
	}

	// If exponent is too negative,
	// denormalize in hopes of making it representable.
	// (The -2 is for the rounding bits.)
	for mantissa > 1 && exp < minExp-2 {
		mantissa = mantissa>>1 | mantissa&1
		exp++
	}

	// Round using two bottom bits.
	round := mantissa & 3
	mantissa >>= 2
	round |= mantissa & 1 // round to even (round up if mantissa is odd)
	exp += 2
	if round == 3 {
		mantissa++
		if mantissa == 1<<(1+flt.mantbits) {
			mantissa >>= 1
			exp++
		}
	}

	if mantissa>>flt.mantbits == 0 { // Denormal or zero.
		exp = flt.bias
	}
	var err error
	if exp > maxExp { // infinity and range error
		mantissa = 1 << flt.mantbits
		exp = maxExp + 1
		err = rangeError(fnParseFloat, s)
	}

	bits := mantissa & (1<<flt.mantbits - 1)
	bits |= uint64((exp-flt.bias)&(1<<flt.expbits-1)) << flt.mantbits
	if neg {
		bits |= 1 << flt.mantbits << flt.expbits
	}
	if flt == &float32info {
		return float64(math.Float32frombits(uint32(bits))), err
	}
	return math.Float64frombits(bits), err
}

const fnParseFloat = "ParseFloat"

func atof32(s string) (f float32, err error) {
	if val, ok := special(s); ok {
		return float32(val), nil
	}

	mantissa, exp, neg, trunc, hex, ok := readFloat(s)
	if !ok {
		return 0, syntaxError(fnParseFloat, s)
	}

	if hex {
		f, err := atofHex(s, &float32info, mantissa, exp, neg, trunc)
		return float32(f), err
	}

	if optimize {
		// Try pure floating-point arithmetic conversion.
		if !trunc {
			if f, ok := atof32exact(mantissa, exp, neg); ok {
				return f, nil
			}
		}
		// Try another fast path.
		ext := new(extFloat)
		if ok := ext.AssignDecimal(mantissa, exp, neg, trunc, &float32info); ok {
			b, ovf := ext.floatBits(&float32info)
			f = math.Float32frombits(uint32(b))
			if ovf {
				err = rangeError(fnParseFloat, s)
			}
			return f, err
		}
	}

	// Slow fallback.
	var d decimal
	if !d.set(s) {
		return 0, syntaxError(fnParseFloat, s)
	}
	b, ovf := d.floatBits(&float32info)
	f = math.Float32frombits(uint32(b))
	if ovf {
		err = rangeError(fnParseFloat, s)
	}
	return f, err
}

func atof64(s string) (f float64, err error) {
	if val, ok := special(s); ok {
		return val, nil
	}

	mantissa, exp, neg, trunc, hex, ok := readFloat(s)
	if !ok {
		return 0, syntaxError(fnParseFloat, s)
	}

	if hex {
		return atofHex(s, &float64info, mantissa, exp, neg, trunc)
	}

	if optimize {
		// Try pure floating-point arithmetic conversion.
		if !trunc {
			if f, ok := atof64exact(mantissa, exp, neg); ok {
				return f, nil
			}
		}
		// Try another fast path.
		ext := new(extFloat)
		if ok := ext.AssignDecimal(mantissa, exp, neg, trunc, &float64info); ok {
			b, ovf := ext.floatBits(&float64info)
			f = math.Float64frombits(b)
			if ovf {
				err = rangeError(fnParseFloat, s)
			}
			return f, err
		}
	}

	// Slow fallback.
	var d decimal
	if !d.set(s) {
		return 0, syntaxError(fnParseFloat, s)
	}
	b, ovf := d.floatBits(&float64info)
	f = math.Float64frombits(b)
	if ovf {
		err = rangeError(fnParseFloat, s)
	}
	return f, err
}

// ParseFloat 以bitSize指定的精度将字符串s转换为浮点数：float32为32或float64为64。
// 当bitSize = 32时，结果仍为float64类型，但可以将其转换为float32而无需更改其值。
// ParseFloat接受十进制和十六进制浮点数语法。
// 如果s格式正确且在有效的浮点数附近，则ParseFloat返回使用IEEE754无偏舍入舍入的最接近的浮点数。
// （仅当十六进制表示中的位数比尾数多时，才解析十六进制浮点值。）
// ParseFloat返回的错误的具体类型为*NumError并包含err.Num = s。
// 如果s的语法格式不正确，则ParseFloat返回err.Err = ErrSyntax。
// 如果s的语法格式正确，但与给定大小的最大浮点数相差超过1/2 ULP，则ParseFloat返回f = ±Inf，err.Err = ErrRange。
// ParseFloat将字符串"NaN", "+Inf"和"-Inf" 识别为它们各自的特殊浮点值。匹配时忽略大小写。
func ParseFloat(s string, bitSize int) (float64, error) { //注：将s转为bitSize（32、64）位浮点数并返回该浮点数与错误
	if bitSize == 32 { //注：如果要求浮点位数为32
		f, err := atof32(s) //注：转为float32
		return float64(f), err
	}
	return atof64(s) //注：否则转为float64
}
