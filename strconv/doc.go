// 版权所有2015 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

// Package strconv 实现基本数据类型的字符串表示形式之间的转换。
//
// 数值转换
//
// 最常见的数字转换是Atoi（从字符串到int）和Itoa（从int到字符串）。
//
//	i, err := strconv.Atoi("-42")
//	s := strconv.Itoa(-42)
//
// 这些假定为十进制和Go int类型。
//
// ParseBool，ParseFloat，ParseInt和ParseUint将字符串转换为值：
//
//	b, err := strconv.ParseBool("true")
//	f, err := strconv.ParseFloat("3.1415", 64)
//	i, err := strconv.ParseInt("-42", 10, 64)
//	u, err := strconv.ParseUint("42", 10, 64)
//
// 解析函数返回最宽的类型（float64，int64和uint64），但是如果size参数指定了更窄的宽度，则结果可以转换为该更窄的类型而不会丢失数据：
//
//	s := "2147483647" // 大于 int32
//	i64, err := strconv.ParseInt(s, 10, 32)
//	...
//	i := int32(i64)
//
// FormatBool, FormatFloat, FormatInt与FormatUint将值转换为字符串：
//
//	s := strconv.FormatBool(true)
//	s := strconv.FormatFloat(3.1415, 'E', -1, 64)
//	s := strconv.FormatInt(-42, 16)
//	s := strconv.FormatUint(42, 16)
//
// AppendBool, AppendFloat, AppendInt, and AppendUint 相似，但将格式化后的值附加到目标切片。
//
// 字符串转换
//
// Quote和QuoteToASCII将字符串转换为带引号的Go字符串文字。
// 后者通过转义来保证结果是ASCII字符串
// 任何带有\u的非ASCII Unicode：
//
//	q := strconv.Quote("Hello, 世界")
//	q := strconv.QuoteToASCII("Hello, 世界")
//
// QuoteRune和QuoteRuneToASCII相似，但是接受符文并返回带引号的Go符文文字。
//
// Unquote和UnquoteChar unquote Go字符串和符文文字。
//
package strconv
