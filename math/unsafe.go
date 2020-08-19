// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。
package math

import "unsafe"

// Float32bits 返回f的IEEE 754二进制表示形式，其中f的符号位和结果位于相同的位位置。
// Float32bits(Float32frombits(x)) == x。
func Float32bits(f float32) uint32 { return *(*uint32)(unsafe.Pointer(&f)) } //注：将f的二进制数据转为uint32

// Float32frombits 返回对应于IEEE 754二进制表示形式b的浮点数，其符号位b和结果位于相同的位位置。
// Float32frombits(Float32bits(x)) == x。
func Float32frombits(b uint32) float32 { return *(*float32)(unsafe.Pointer(&b)) } //注：将b的二进制数据转为float32

// Float64bits 返回f的IEEE 754二进制表示形式，其中f的符号位和结果位于相同的位位置，
// Float64bits(Float64frombits(x)) == x.
func Float64bits(f float64) uint64 { return *(*uint64)(unsafe.Pointer(&f)) } //注：将f的二进制数据转为uint64

// Float64frombits 返回对应于IEEE 754二进制表示形式b的浮点数，其符号位b和结果位于相同的位位置。
// Float64frombits(Float64bits(x)) == x。
func Float64frombits(b uint64) float64 { return *(*float64)(unsafe.Pointer(&b)) } //注：将b的二进制数据转为float64并返回
