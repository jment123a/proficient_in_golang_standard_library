// 版权所有2018 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package bytealg

import "internal/cpu"

// MaxBruteForce 蛮力搜索的最大长度限制
const MaxBruteForce = 64

func init() {
	if cpu.X86.HasAVX2 {
		MaxLen = 63
	} else {
		MaxLen = 31
	}
}

// Cutover 报告在切换到索引之前我们应该容忍的索引字节失败次数。
// n是到目前为止已处理的字节数。
// 有关详细信息，请参见bytes.Index实现。
func Cutover(n int) int { // 注：根据已处理的字节数n，返回容忍索引失败的次数
	// 每8个字符1个错误，加上一些斜率开始。
	return (n + 16) / 8
}
