// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package math

// Abs 返回x的绝对值。
//
// 特殊情况是：
//	Abs(±Inf) = +Inf
//	Abs(NaN) = NaN
func Abs(x float64) float64 { //注：求绝对值
	// 例：1010 1111（-47） &^ 1000 0000 = 0010 1111
	// 注：就是把符号位变为0
	return Float64frombits(Float64bits(x) &^ (1 << 63))
}
