// 版权所有2018 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package bytealg

import (
	"internal/cpu"
	"unsafe"
)

// 偏移到internal/cpu记录中以便在汇编中使用。
const (
	offsetX86HasSSE2   = unsafe.Offsetof(cpu.X86.HasSSE2)
	offsetX86HasSSE42  = unsafe.Offsetof(cpu.X86.HasSSE42)
	offsetX86HasAVX2   = unsafe.Offsetof(cpu.X86.HasAVX2)
	offsetX86HasPOPCNT = unsafe.Offsetof(cpu.X86.HasPOPCNT)

	offsetS390xHasVX = unsafe.Offsetof(cpu.S390X.HasVX)
)

// MaxLen 是要在索引中搜索的字符串的最大长度（参数b）。
var MaxLen int
