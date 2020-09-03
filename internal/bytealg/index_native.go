// 版权所有2018 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

// +build amd64 arm64 s390x

package bytealg

//go:noescape

// Index 返回a中b的第一个实例的索引；如果a中不存在b，则返回-1。
// 需要2 <= len(b) <= MaxLen。
func Index(a, b []byte) int // 注：获取a中第1次出现b时的索引

//go:noescape

// IndexString 返回a中b的第一个实例的索引；如果a中不存在b，则返回-1。
// 需要2 <= len(b) <= MaxLen。
func IndexString(a, b string) int // 注：获取a中第1次出现b时的索引
