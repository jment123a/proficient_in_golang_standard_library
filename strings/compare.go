// 版权所有2015 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package strings

// Compare 返回一个按字典顺序比较两个字符串的整数。
// 如果a == b，结果将为0，如果a < b，结果将为-1，如果a > b，结果将为+1。
//
// 仅包含比较，以便与bytes包对称。
// 使用内置的字符串比较运算符==，<，>等通常更清晰，总是更快。
func Compare(a, b string) int { // 注：比较字符串，不建议使用
	// NOTE（rsc）：此函数不会调用运行时cmpstring函数，因为我们不想为使用string.Compare提供任何性能证明。
	// 基本上没有人应该使用strings.Compare。
	// 如上面的评论所述，这里仅用于与bytes包对称。
	// 如果性能很重要，则应更改编译器以识别模式，以便所有进行三向比较的代码，而不仅仅是使用string.Compare的代码都可以受益。
	if a == b {
		return 0
	}
	if a < b {
		return -1
	}
	return +1
}
