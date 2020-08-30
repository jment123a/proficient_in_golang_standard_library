// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package unicode

// IsDigit 报告rune是否为十进制数字。
func IsDigit(r rune) bool { // 注：#获取r是否为十进制数据
	if r <= MaxLatin1 { // 注：如果r小于Latin-1最大值，比较是否为ASCII数字
		return '0' <= r && r <= '9'
	}
	return isExcludingLatin(Digit, r) // 注： #
}
