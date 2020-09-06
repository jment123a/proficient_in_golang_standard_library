//版权所有2011 The Go Authors。版权所有。
//此源代码的使用受BSD样式的约束
//可以在LICENSE文件中找到的许可证。

// Package atomic 提供了用于实现同步算法的低级原子内存基元。
//
// 这些功能需要格外小心才能正确使用。
// 除特殊的低层应用程序外，最好通过通道或sync软件包的功能来完成同步。
// 通过通信共享内存；
// 不要通过共享内存进行通信。
//
// 由SwapT函数实现的交换操作在原子上等效于：
//
//	old = *addr
//	*addr = new
//	return old
//
// 由CompareAndSwapT函数实现的比较交换操作在原子上等效于：
//
//	if *addr == old {
//		*addr = new
//		return true
//	}
//	return false
//
//由AddT函数实现的加法操作在原子上等效于：
//
//	*addr += delta
//	return *addr
//
// 由LoadT和StoreT函数实现的加载和存储操作是"return *addr" 和 "*addr = val".的原子等效项。
//
package atomic

import (
	"unsafe"
)

// BUG（rsc）：在x86-32上，64位函数使用奔腾MMX之前不可用的指令。
//
// 在非Linux ARM上，64位函数使用ARMv6k内核之前不可用的指令。
//
// 在ARM，x86-32和32位MIPS上，调用者负责安排原子访问的64位字的64位对齐。 变量或已分配的结构，数组或片中的第一个单词可以依赖于64位对齐。

// SwapInt32 原子性地将new存储到*addr，返回*addr以前的值
func SwapInt32(addr *int32, new int32) (old int32)

// SwapInt64 原子性地将new存储到*addr，返回*addr以前的值
func SwapInt64(addr *int64, new int64) (old int64)

// SwapUint32 原子性地将new存储到*addr，返回*addr以前的值
func SwapUint32(addr *uint32, new uint32) (old uint32)

// SwapUint64 原子性地将new存储到*addr，返回*addr以前的值
func SwapUint64(addr *uint64, new uint64) (old uint64)

// SwapUintptr 原子性地将new存储到*addr，返回*addr以前的值
func SwapUintptr(addr *uintptr, new uintptr) (old uintptr)

// SwapPointer 原子性地将new存储到*addr，返回*addr以前的值
func SwapPointer(addr *unsafe.Pointer, new unsafe.Pointer) (old unsafe.Pointer)

// CompareAndSwapInt32 对int32值执行比较和交换操作。
func CompareAndSwapInt32(addr *int32, old, new int32) (swapped bool)

// CompareAndSwapInt64 对int64值执行比较和交换操作。
func CompareAndSwapInt64(addr *int64, old, new int64) (swapped bool)

// CompareAndSwapUint32 对uint32值执行比较和交换操作。
func CompareAndSwapUint32(addr *uint32, old, new uint32) (swapped bool)

// CompareAndSwapUint64 对uint64值执行比较和交换操作。
func CompareAndSwapUint64(addr *uint64, old, new uint64) (swapped bool)

// CompareAndSwapUintptr 对uintptr值执行比较和交换操作。
func CompareAndSwapUintptr(addr *uintptr, old, new uintptr) (swapped bool)

// CompareAndSwapPointer 对unsafe.Pointer值执行比较和交换操作。
func CompareAndSwapPointer(addr *unsafe.Pointer, old, new unsafe.Pointer) (swapped bool)

// AddInt32 原子性地将*addr加delta，返回新值
func AddInt32(addr *int32, delta int32) (new int32)

// AddUint32 原子性地将*addr加delta，返回新值
// 要从x中减去一个有符号的正常数值c，请执行AddUint32(&x, ^uint32(c-1)).
// 特别是要减少x，请执行AddUint32(&x, ^uint32(0)).
// 例：x偏移100，99（0110 0011）^uint32(99) = 1111 1111 1111 1111 1111 1111 1001 1100
func AddUint32(addr *uint32, delta uint32) (new uint32)

// AddInt64 原子性地将*addr加delta，返回新值
func AddInt64(addr *int64, delta int64) (new int64)

// AddUint64 原子性地将*addr加delta，返回新值
// 要从x中减去一个有符号的正常数值c，请执行AddUint64(&x, ^uint64(c-1)).
// 特别是要减少x，请执行AddUint64(&x, ^uint64(0)).
func AddUint64(addr *uint64, delta uint64) (new uint64)

// AddUintptr 原子性地将*addr加delta，返回新值
func AddUintptr(addr *uintptr, delta uintptr) (new uintptr)

// LoadInt32 原子性地加载*addr
func LoadInt32(addr *int32) (val int32)

// LoadInt64 原子性地加载*addr
func LoadInt64(addr *int64) (val int64)

// LoadUint32 原子性地加载*addr
func LoadUint32(addr *uint32) (val uint32)

// LoadUint64 原子性地加载*addr
func LoadUint64(addr *uint64) (val uint64)

// LoadUintptr 原子性地加载*addr
func LoadUintptr(addr *uintptr) (val uintptr)

// LoadPointer 原子性地加载*addr
func LoadPointer(addr *unsafe.Pointer) (val unsafe.Pointer)

// StoreInt32 原子性地加载*addr
func StoreInt32(addr *int32, val int32)

// StoreInt64 原子性地加载*addr
func StoreInt64(addr *int64, val int64)

// StoreUint32 原子性地加载*addr
func StoreUint32(addr *uint32, val uint32)

// StoreUint64 原子性地加载*addr
func StoreUint64(addr *uint64, val uint64)

// StoreUintptr 原子性地加载*addr
func StoreUintptr(addr *uintptr, val uintptr)

// StorePointer 原子性地加载*addr
func StorePointer(addr *unsafe.Pointer, val unsafe.Pointer)

// ARM的助手。 链接器将在其他系统上丢弃
func panic64() {
	panic("sync/atomic: broken 64-bit atomic operations (buggy QEMU)") // 恐慌："损坏的64位原子操作（错误的QEMU）"
}
