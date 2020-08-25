// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

// 二进制到十进制浮点转换。
// 算法：
//  1）将尾数存储为多精度十进制
//  2）按指数移动小数
//  3）读取数字并格式化

package strconv

import "math"

// TODO：移到其他地方？注：确实
type floatInfo struct {
	mantbits uint //注：尾数的位置
	expbits  uint //注：指数的位置
	bias     int  //注：最小指数
}

var float32info = floatInfo{23, 8, -127}   // 注：32位float，1为符号位，23位尾数，8位指数，指数范围为-127 ~ 128
var float64info = floatInfo{52, 11, -1023} // 注：64位float，1为符号位，52位尾数，11位指数，指数范围为-1023 ~ 1024

// FormatFloat 根据fmt和精度 prec格式将浮点数f转换为字符串。
// 假设原始结果是从bitSize位的浮点值（float32为32，float64为64）获得的，则对结果进行四舍五入。
//
// 格式fmt是以下格式之一
// 'b' (-ddddp±ddd, 2进制指数),
// 'e' (-d.dddde±dd, 10进制指数),
// 'E' (-d.ddddE±dd, 10进制指数),
// 'f' (-ddd.dddd, 无指数),
// 'g' (对于大指数，为'e'，否则为'f'),
// 'G' (对于大指数，为'E'，否则为'F'),
// 'x' (-0xd.ddddp±ddd, 十六进制分数和二进制指数), 或
// 'X' (-0Xd.ddddP±ddd, 十六进制分数和二进制指数).
//
// 精度prec控制以'e'，'E'，'f'，'g'，'G'，'x'和'X'格式打印的位数（不包括指数）。
// 对于'e'，'E'，'f'，'x'和'X'，它是小数点后的位数。
// 对于'g'和'G'，它是有效数字的最大数量（删除零位）。
// 特殊精度-1使用所需的最小位数，以使ParseFloat准确返回f。
func FormatFloat(f float64, fmt byte, prec, bitSize int) string {
	return string(genericFtoa(make([]byte, 0, max(prec+4, 24)), f, fmt, prec, bitSize))
}

// AppendFloat 将由FormatFloat生成的浮点数f的字符串形式附加到dst并返回扩展缓冲区。
func AppendFloat(dst []byte, f float64, fmt byte, prec, bitSize int) []byte {
	return genericFtoa(dst, f, fmt, prec, bitSize)
}

func genericFtoa(dst []byte, val float64, fmt byte, prec, bitSize int) []byte {
	// dst：destination
	// val：要转换的数据
	// fmt：
	// prec：精度
	// bitSize：位数
	var bits uint64 //注：val的二进制数据
	var flt *floatInfo
	switch bitSize {
	case 32:
		bits = uint64(math.Float32bits(float32(val))) //注：截取为32位二进制转为uint64
		flt = &float32info
	case 64:
		bits = math.Float64bits(val) //注：二进制转为uint64
		flt = &float64info
	default:
		panic("strconv: illegal AppendFloat/FormatFloat bitSize") //恐慌："非法的AppendFloat/FormatFloat位数大小"
	}

	neg := bits>>(flt.expbits+flt.mantbits) != 0          //注：获取符号位（例：类型为float32，mant为23，exp为8，32位>>31位，得出符号位）
	exp := int(bits>>flt.mantbits) & (1<<flt.expbits - 1) //注：获取指数，(1<<flt.expbits - 1)得出小数位数，bits右移得出小数，索引0开始为小数部分
	mant := bits & (uint64(1)<<flt.mantbits - 1)          //注：获取尾数，(uint64(1)<<flt.mantbits - 1)得出整数位数，索引0开始为整数部分

	switch exp {
	case 1<<flt.expbits - 1: //注：指数为最大值
		// 无限或NanN
		var s string
		switch {
		case mant != 0:
			s = "NaN"
		case neg:
			s = "-Inf"
		default:
			s = "+Inf"
		}
		return append(dst, s...)

	case 0:
		// 非正规化
		exp++

	default:
		// 添加隐式最高位
		mant |= uint64(1) << flt.mantbits //注：整数的最高位设置为1
	}
	exp += flt.bias

	// 选择简单的二进制，十六进制格式。
	if fmt == 'b' {
		return fmtB(dst, neg, mant, exp, flt) // 注：将尾数mant与指数exp结合，根据neg判断整数是否为负数，将数据存储在dst中并返回
	}
	if fmt == 'x' || fmt == 'X' {
		return fmtX(dst, prec, fmt, neg, mant, exp, flt)
	}

	if !optimize {
		return bigFtoa(dst, prec, fmt, neg, mant, exp, flt)
	}

	var digs decimalSlice
	ok := false
	// 负精度意味着“只需要精确的数量”。
	shortest := prec < 0
	if shortest {
		// 尝试使用Grisu3算法。
		f := new(extFloat)
		lower, upper := f.AssignComputeBounds(mant, exp, neg, flt)
		var buf [32]byte
		digs.d = buf[:]
		ok = f.ShortestDecimal(&digs, &lower, &upper)
		if !ok {
			return bigFtoa(dst, prec, fmt, neg, mant, exp, flt)
		}
		// Precision for shortest representation mode.
		switch fmt {
		case 'e', 'E':
			prec = max(digs.nd-1, 0)
		case 'f':
			prec = max(digs.nd-digs.dp, 0)
		case 'g', 'G':
			prec = digs.nd
		}
	} else if fmt != 'f' {
		// Fixed number of digits.
		digits := prec
		switch fmt {
		case 'e', 'E':
			digits++
		case 'g', 'G':
			if prec == 0 {
				prec = 1
			}
			digits = prec
		}
		if digits <= 15 {
			// try fast algorithm when the number of digits is reasonable.
			var buf [24]byte
			digs.d = buf[:]
			f := extFloat{mant, exp - int(flt.mantbits), neg}
			ok = f.FixedDecimal(&digs, digits)
		}
	}
	if !ok {
		return bigFtoa(dst, prec, fmt, neg, mant, exp, flt)
	}
	return formatDigits(dst, shortest, neg, digs, prec, fmt)
}

// bigFtoa uses multiprecision computations to format a float.
func bigFtoa(dst []byte, prec int, fmt byte, neg bool, mant uint64, exp int, flt *floatInfo) []byte {
	d := new(decimal)
	d.Assign(mant)
	d.Shift(exp - int(flt.mantbits))
	var digs decimalSlice
	shortest := prec < 0
	if shortest {
		roundShortest(d, mant, exp, flt)
		digs = decimalSlice{d: d.d[:], nd: d.nd, dp: d.dp}
		// Precision for shortest representation mode.
		switch fmt {
		case 'e', 'E':
			prec = digs.nd - 1
		case 'f':
			prec = max(digs.nd-digs.dp, 0)
		case 'g', 'G':
			prec = digs.nd
		}
	} else {
		// Round appropriately.
		switch fmt {
		case 'e', 'E':
			d.Round(prec + 1)
		case 'f':
			d.Round(d.dp + prec)
		case 'g', 'G':
			if prec == 0 {
				prec = 1
			}
			d.Round(prec)
		}
		digs = decimalSlice{d: d.d[:], nd: d.nd, dp: d.dp}
	}
	return formatDigits(dst, shortest, neg, digs, prec, fmt)
}

func formatDigits(dst []byte, shortest bool, neg bool, digs decimalSlice, prec int, fmt byte) []byte {
	switch fmt {
	case 'e', 'E':
		return fmtE(dst, neg, digs, prec, fmt)
	case 'f':
		return fmtF(dst, neg, digs, prec)
	case 'g', 'G':
		// trailing fractional zeros in 'e' form will be trimmed.
		eprec := prec
		if eprec > digs.nd && digs.nd >= digs.dp {
			eprec = digs.nd
		}
		// %e is used if the exponent from the conversion
		// is less than -4 or greater than or equal to the precision.
		// if precision was the shortest possible, use precision 6 for this decision.
		if shortest {
			eprec = 6
		}
		exp := digs.dp - 1
		if exp < -4 || exp >= eprec {
			if prec > digs.nd {
				prec = digs.nd
			}
			return fmtE(dst, neg, digs, prec-1, fmt+'e'-'g')
		}
		if prec > digs.dp {
			prec = digs.nd
		}
		return fmtF(dst, neg, digs, max(prec-digs.dp, 0))
	}

	// unknown format
	return append(dst, '%', fmt)
}

// roundShortest rounds d (= mant * 2^exp) to the shortest number of digits
// that will let the original floating point value be precisely reconstructed.
func roundShortest(d *decimal, mant uint64, exp int, flt *floatInfo) {
	// If mantissa is zero, the number is zero; stop now.
	if mant == 0 {
		d.nd = 0
		return
	}

	// 计算上限和下限，以使上限和下限之间的任何十进制数字（可能包含在内）都将舍入到原始浮点数。

	// 我们可能会立即看到该数字已经最短了。
	//
	// 假设d不是非正规的，那么2^exp <= d < 10^dp。
	// 最接近的较短数字至少相距10^(dp-nd)。
	// 下面计算的上下限距离最大为2^(exp-mantbits)。
	//
	// 因此，如果10^(dp-nd) > 2^(exp-mantbits)，
	// 或者等效地log2(10)*(dp-nd) > exp-mantbits，则该数字已经是最短的。
	// 如果 332/100*(dp-nd) >= exp-mantbits (log2(10) > 3.32)，则为true。
	minexp := flt.bias + 1 // 最小可能指数
	if exp > minexp && 332*(d.dp-d.nd) >= 100*(exp-int(flt.mantbits)) {
		// The number is already shortest.
		return
	}

	// d = mant << (exp - mantbits)
	// Next highest floating point number is mant+1 << exp-mantbits.
	// Our upper bound is halfway between, mant*2+1 << exp-mantbits-1.
	upper := new(decimal)
	upper.Assign(mant*2 + 1)
	upper.Shift(exp - int(flt.mantbits) - 1)

	// d = mant << (exp - mantbits)
	// Next lowest floating point number is mant-1 << exp-mantbits,
	// unless mant-1 drops the significant bit and exp is not the minimum exp,
	// in which case the next lowest is mant*2-1 << exp-mantbits-1.
	// Either way, call it mantlo << explo-mantbits.
	// Our lower bound is halfway between, mantlo*2+1 << explo-mantbits-1.
	var mantlo uint64
	var explo int
	if mant > 1<<flt.mantbits || exp == minexp {
		mantlo = mant - 1
		explo = exp
	} else {
		mantlo = mant*2 - 1
		explo = exp - 1
	}
	lower := new(decimal)
	lower.Assign(mantlo*2 + 1)
	lower.Shift(explo - int(flt.mantbits) - 1)

	// The upper and lower bounds are possible outputs only if
	// the original mantissa is even, so that IEEE round-to-even
	// would round to the original mantissa and not the neighbors.
	inclusive := mant%2 == 0

	// As we walk the digits we want to know whether rounding up would fall
	// within the upper bound. This is tracked by upperdelta:
	//
	// If upperdelta == 0, the digits of d and upper are the same so far.
	//
	// If upperdelta == 1, we saw a difference of 1 between d and upper on a
	// previous digit and subsequently only 9s for d and 0s for upper.
	// (Thus rounding up may fall outside the bound, if it is exclusive.)
	//
	// If upperdelta == 2, then the difference is greater than 1
	// and we know that rounding up falls within the bound.
	var upperdelta uint8

	// Now we can figure out the minimum number of digits required.
	// Walk along until d has distinguished itself from upper and lower.
	for ui := 0; ; ui++ {
		// lower, d, and upper may have the decimal points at different
		// places. In this case upper is the longest, so we iterate from
		// ui==0 and start li and mi at (possibly) -1.
		mi := ui - upper.dp + d.dp
		if mi >= d.nd {
			break
		}
		li := ui - upper.dp + lower.dp
		l := byte('0') // lower digit
		if li >= 0 && li < lower.nd {
			l = lower.d[li]
		}
		m := byte('0') // middle digit
		if mi >= 0 {
			m = d.d[mi]
		}
		u := byte('0') // upper digit
		if ui < upper.nd {
			u = upper.d[ui]
		}

		// Okay to round down (truncate) if lower has a different digit
		// or if lower is inclusive and is exactly the result of rounding
		// down (i.e., and we have reached the final digit of lower).
		okdown := l != m || inclusive && li+1 == lower.nd

		switch {
		case upperdelta == 0 && m+1 < u:
			// Example:
			// m = 12345xxx
			// u = 12347xxx
			upperdelta = 2
		case upperdelta == 0 && m != u:
			// Example:
			// m = 12345xxx
			// u = 12346xxx
			upperdelta = 1
		case upperdelta == 1 && (m != '9' || u != '0'):
			// Example:
			// m = 1234598x
			// u = 1234600x
			upperdelta = 2
		}
		// Okay to round up if upper has a different digit and either upper
		// is inclusive or upper is bigger than the result of rounding up.
		okup := upperdelta > 0 && (inclusive || upperdelta > 1 || ui+1 < upper.nd)

		// If it's okay to do either, then round to the nearest one.
		// If it's okay to do only one, do it.
		switch {
		case okdown && okup:
			d.Round(mi + 1)
			return
		case okdown:
			d.RoundDown(mi + 1)
			return
		case okup:
			d.RoundUp(mi + 1)
			return
		}
	}
}

type decimalSlice struct {
	d      []byte
	nd, dp int
	neg    bool
}

// %e: -d.ddddde±dd
func fmtE(dst []byte, neg bool, d decimalSlice, prec int, fmt byte) []byte {
	// sign
	if neg {
		dst = append(dst, '-')
	}

	// first digit
	ch := byte('0')
	if d.nd != 0 {
		ch = d.d[0]
	}
	dst = append(dst, ch)

	// .moredigits
	if prec > 0 {
		dst = append(dst, '.')
		i := 1
		m := min(d.nd, prec+1)
		if i < m {
			dst = append(dst, d.d[i:m]...)
			i = m
		}
		for ; i <= prec; i++ {
			dst = append(dst, '0')
		}
	}

	// e±
	dst = append(dst, fmt)
	exp := d.dp - 1
	if d.nd == 0 { // special case: 0 has exponent 0
		exp = 0
	}
	if exp < 0 {
		ch = '-'
		exp = -exp
	} else {
		ch = '+'
	}
	dst = append(dst, ch)

	// dd or ddd
	switch {
	case exp < 10:
		dst = append(dst, '0', byte(exp)+'0')
	case exp < 100:
		dst = append(dst, byte(exp/10)+'0', byte(exp%10)+'0')
	default:
		dst = append(dst, byte(exp/100)+'0', byte(exp/10)%10+'0', byte(exp%10)+'0')
	}

	return dst
}

// %f: -ddddddd.ddddd
func fmtF(dst []byte, neg bool, d decimalSlice, prec int) []byte {
	// sign
	if neg {
		dst = append(dst, '-')
	}

	// integer, padded with zeros as needed.
	if d.dp > 0 {
		m := min(d.nd, d.dp)
		dst = append(dst, d.d[:m]...)
		for ; m < d.dp; m++ {
			dst = append(dst, '0')
		}
	} else {
		dst = append(dst, '0')
	}

	// fraction
	if prec > 0 {
		dst = append(dst, '.')
		for i := 0; i < prec; i++ {
			ch := byte('0')
			if j := d.dp + i; 0 <= j && j < d.nd {
				ch = d.d[j]
			}
			dst = append(dst, ch)
		}
	}

	return dst
}

// %b: -ddddddddp±ddd
func fmtB(dst []byte, neg bool, mant uint64, exp int, flt *floatInfo) []byte { //注：将尾数mant与指数exp结合，根据neg判断整数是否为负数，将数据存储在dst中并返回
	// 符号
	if neg {
		dst = append(dst, '-') //注：如果neg == true，输出-
	}

	// 数字
	dst, _ = formatBits(dst, mant, 10, false, true) //注：输出整数

	// p
	dst = append(dst, 'p') //注：输出p

	// ±指数
	exp -= int(flt.mantbits)
	if exp >= 0 {
		dst = append(dst, '+') //注：如果exp > 0，输出+
	}
	dst, _ = formatBits(dst, uint64(exp), 10, exp < 0, true) //注：输出指数

	return dst
}

// %x: -0x1.yyyyyyyyp±ddd or -0x0p+0. （y是十六进制数字，d是十进制数字）
func fmtX(dst []byte, prec int, fmt byte, neg bool, mant uint64, exp int, flt *floatInfo) []byte {
	if mant == 0 {
		exp = 0
	}

	// 移位数字，因此前导1（如果有）位于位1 << 60。
	mant <<= 60 - flt.mantbits
	for mant != 0 && mant&(1<<60) == 0 {
		mant <<= 1
		exp--
	}

	// Round if requested.
	if prec >= 0 && prec < 15 {
		shift := uint(prec * 4)
		extra := (mant << shift) & (1<<60 - 1)
		mant >>= 60 - shift
		if extra|(mant&1) > 1<<59 {
			mant++
		}
		mant <<= 60 - shift
		if mant&(1<<61) != 0 {
			// Wrapped around.
			mant >>= 1
			exp++
		}
	}

	hex := lowerhex
	if fmt == 'X' {
		hex = upperhex
	}

	// sign, 0x, leading digit
	if neg {
		dst = append(dst, '-')
	}
	dst = append(dst, '0', fmt, '0'+byte((mant>>60)&1))

	// .fraction
	mant <<= 4 // remove leading 0 or 1
	if prec < 0 && mant != 0 {
		dst = append(dst, '.')
		for mant != 0 {
			dst = append(dst, hex[(mant>>60)&15])
			mant <<= 4
		}
	} else if prec > 0 {
		dst = append(dst, '.')
		for i := 0; i < prec; i++ {
			dst = append(dst, hex[(mant>>60)&15])
			mant <<= 4
		}
	}

	// p±
	ch := byte('P')
	if fmt == lower(fmt) {
		ch = 'p'
	}
	dst = append(dst, ch)
	if exp < 0 {
		ch = '-'
		exp = -exp
	} else {
		ch = '+'
	}
	dst = append(dst, ch)

	// dd or ddd or dddd
	switch {
	case exp < 100:
		dst = append(dst, byte(exp/10)+'0', byte(exp%10)+'0')
	case exp < 1000:
		dst = append(dst, byte(exp/100)+'0', byte((exp/10)%10)+'0', byte(exp%10)+'0')
	default:
		dst = append(dst, byte(exp/1000)+'0', byte(exp/100)%10+'0', byte((exp/10)%10)+'0', byte(exp%10)+'0')
	}

	return dst
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
