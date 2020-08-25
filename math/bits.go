// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package math

const (
	uvnan    = 0x7FF8000000000001 //注：非数字，0111 1111 1111 1000 ... 0001
	uvinf    = 0x7FF0000000000000 //注：正无穷大，0111 1111 1111...
	uvneginf = 0xFFF0000000000000 //注：负无穷大，1111 1111 1111...
	uvone    = 0x3FF0000000000000
	mask     = 0x7FF
	shift    = 64 - 11 - 1
	bias     = 1023
	signMask = 1 << 63
	fracMask = 1<<shift - 1
)

// Inf 如果符号 >= 0，则返回正无穷大；如果符号 < 0，则返回负无穷大。
func Inf(sign int) float64 { //注：根据sign >=0，返回正无穷大与负无穷大
	var v uint64
	if sign >= 0 {
		v = uvinf
	} else {
		v = uvneginf
	}
	return Float64frombits(v)
}

// NaN 返回IEEE 754的``非数字''值。
func NaN() float64 { return Float64frombits(uvnan) } //注：返回非数字

// IsNaN 报告f是否为IEEE 754``非数字''值。
func IsNaN(f float64) (is bool) { //注：返回f是否为非数字
	// IEEE 754说，只有NaN满足f != f。
	// 为避免浮点硬件，可以使用：
	//	x := Float64bits(f);
	//	return uint32(x>>shift)&mask == mask && x != uvinf && x != uvneginf
	return f != f
}

// IsInf 根据sign报告f是否为无穷大。
// 如果sign > 0，则IsInf报告f是否为正无穷大。
// 如果sign < 0，则IsInf报告f是否为负无穷大。
// 如果sign == 0，则IsInf报告f是否为无穷大。
func IsInf(f float64, sign int) bool { //注：根据符号sign，判断f是否为正负无穷大
	// 通过与最大浮点数进行比较来测试无穷大。
	// 为避免浮点硬件，可以使用：
	//	x := Float64bits(f);
	//	return sign >= 0 && x == uvinf || sign <= 0 && x == uvneginf;
	return sign >= 0 && f > MaxFloat64 || sign <= 0 && f < -MaxFloat64
}

// normalize 返回正态数y和满足x == y×2 ** exp的指数exp。 假设x为有限且非零。
func normalize(x float64) (y float64, exp int) { //注：#
	const SmallestNormal = 2.2250738585072014e-308 // 2**-1022
	if Abs(x) < SmallestNormal {
		return x * (1 << 52), -52
	}
	return x, 0
}
