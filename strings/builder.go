// 版权所有2017 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package strings

import (
	"unicode/utf8"
	"unsafe"
)

// Builder 用于使用Write方法有效地构建字符串。
// 将内存复制减到最少。 零值可以使用了。
// 不要复制非零的Builder。
type Builder struct {
	addr *Builder // 接收方，按值检测副本
	buf  []byte   // 注：缓冲区
}

// noescape 在转义分析中隐藏指针。 noescape是标识函数，但是转义分析不认为输出取决于输入。 noescape是内联的，当前可编译为零指令。
// 小心使用！
// 这是从运行时复制的； 请参阅问题23382和7921。
//go:nosplit
//go:nocheckptr
func noescape(p unsafe.Pointer) unsafe.Pointer { // 注：在逃逸分析中隐藏指针p
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

func (b *Builder) copyCheck() { // 注：检查拷贝，保证b.addr == b
	if b.addr == nil { // 注：如果b的指针位空，
		// 此hack解决了Go的转义分析失败的问题，该失败导致b逃逸并被分配了堆。
		// 参见问题23382。
		// TODO：解决了问题7921之后，应将其恢复为
		// 仅"b.addr = b".
		b.addr = (*Builder)(noescape(unsafe.Pointer(b)))
	} else if b.addr != b {
		panic("strings: illegal use of non-zero Builder copied by value") // 恐慌："非法使用按值复制的非零生成器"
	}
}

// String 返回累积的字符串。
func (b *Builder) String() string { // 注：返回b的字符串表现形式
	return *(*string)(unsafe.Pointer(&b.buf))
}

// Len 返回累积的字节数；b.Len() == len(b.String())
func (b *Builder) Len() int { return len(b.buf) } // 注：获取b的数据长度

// Cap 返回构建器基础字节片的容量。 它是为正在构建的字符串分配的总空间，包括已写入的所有字节。
func (b *Builder) Cap() int { return cap(b.buf) } // 注：获取b的数据容量

// Reset 将生成器重置为空。
func (b *Builder) Reset() { // 注：重置b
	b.addr = nil
	b.buf = nil
}

// grow 将缓冲区复制到一个更大的新缓冲区中，以便在len(b.buf)之后至少保留n个字节的容量。
func (b *Builder) grow(n int) { // 注：保证b至少还可以容纳长度为n的数据
	buf := make([]byte, len(b.buf), 2*cap(b.buf)+n)
	copy(buf, b.buf)
	b.buf = buf
}

// Grow 必要时增加b的容量，以保证另外n个字节的空间。
// 在Grow(n)之后，至少可以将n个字节写入b，而无需进行其他分配。 如果n为负，则恐慌。
func (b *Builder) Grow(n int) { // 注：保证b至少还可以容纳长度为n的数据
	b.copyCheck()
	if n < 0 {
		panic("strings.Builder.Grow: negative count") // 恐慌："负数数量"
	}
	if cap(b.buf)-len(b.buf) < n {
		b.grow(n)
	}
}

// Write 将p的内容附加到b的缓冲区。
// Write操作始终返回len(p), nil。
func (b *Builder) Write(p []byte) (int, error) { // 注：将p附加到b
	b.copyCheck()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

// WriteByte 将字节c附加到b的缓冲区。
// 返回的错误始终为nil。
func (b *Builder) WriteByte(c byte) error { // 注：将c附加到b
	b.copyCheck()
	b.buf = append(b.buf, c)
	return nil
}

// WriteRune 将Unicode代码点r的UTF-8编码附加到b的缓冲区。
// 返回r的长度和nil错误。
func (b *Builder) WriteRune(r rune) (int, error) { // 注：将r附加到b
	b.copyCheck()
	if r < utf8.RuneSelf { // 注：如果r位单字节rune，直接写入单个字节
		b.buf = append(b.buf, byte(r))
		return 1, nil
	}
	l := len(b.buf)
	if cap(b.buf)-l < utf8.UTFMax {
		b.grow(utf8.UTFMax)
	}
	n := utf8.EncodeRune(b.buf[l:l+utf8.UTFMax], r)
	b.buf = b.buf[:l+n]
	return n, nil
}

// WriteString 将s的内容附加到b的缓冲区。
// 返回s的长度和nil错误。
func (b *Builder) WriteString(s string) (int, error) { // 注：将s附加到b
	b.copyCheck()
	b.buf = append(b.buf, s...)
	return len(s), nil
}
