// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package bytes

// 用于编组数据的简单字节缓冲区。

import (
	"errors"
	"io"
	"unicode/utf8"
)

// smallBufferSize 是初始分配的最小容量。
const smallBufferSize = 64

// Buffer 是具有Read和Write方法的可变大小的字节缓冲区。
// Buffer的零值是准备使用的空缓冲区。
type Buffer struct {
	buf      []byte // 内容是字节buf[off : len(buf)]
	off      int    // 在&buf[off]读取，在&buf[len(buf)]写入
	lastRead readOp // 最后读取操作，以便未读*可以正常工作。注：记录上一次读取操作
}

// readOp 常量描述对缓冲区执行的最后一个操作，以便UnreadRune和UnreadByte可以检查无效使用。
// 选择opReadRuneX常量，以便将其转换为int，使其对应于已读取的符文大小。
type readOp int8

// 请勿使用iota，因为这些值需要与名称和注释相对应，这在使用显式时更容易看到。
const (
	opRead      readOp = -1 // 任何其他读取操作。注：读取了其他类型
	opInvalid   readOp = 0  // 非读取操作。注：不是读取操作
	opReadRune1 readOp = 1  // 读取1字节rune，注：读取了1个字节的rune
	opReadRune2 readOp = 2  // 读取2字节rune，注：读取了2个字节的rune
	opReadRune3 readOp = 3  // 读取3字节rune，注：读取了3个字节的rune
	opReadRune4 readOp = 4  // 读取4字节rune，注：读取了4个字节的rune
)

// ErrTooLarge 如果无法分配内存以将数据存储在缓冲区中，则将进入紧急状态。
var ErrTooLarge = errors.New("bytes.Buffer: too large")                                    // 错误："太大"
var errNegativeRead = errors.New("bytes.Buffer: reader returned negative count from Read") // 错误："reader从Read中返回了负数"

const maxInt = int(^uint(0) >> 1) // 注：int的最大值

// Bytes 返回长度为b.Len()的切片，其中包含缓冲区的未读部分。
// 切片仅在下一次修改缓冲区之前有效（即，仅在下一次调用诸如Read，Write，Reset或Truncate之类的方法之前）才有效。
// 至少在下一次缓冲区修改之前，切片会对缓冲区的内容起别名的作用，因此对切片的立即更改将影响将来读取的结果。
func (b *Buffer) Bytes() []byte { return b.buf[b.off:] } // 注：获取b的缓冲区内容

// String 以字符串形式返回缓冲区未读部分的内容。 如果Buffer是nil指针，则返回"<nil>".
// 要更有效地构建字符串，请参见strings.Builder类型。
func (b *Buffer) String() string { // 注：获取b的缓冲区内容的字符串形式
	if b == nil {
		// 特殊情况，在调试中很有用。
		return "<nil>"
	}
	return string(b.buf[b.off:])
}

// empty 报告缓冲区的未读部分是否为空。
func (b *Buffer) empty() bool { return len(b.buf) <= b.off } // 注：获取b的缓冲区内容是否为空

// Len 返回缓冲区未读部分的字节数；
// b.Len() == len(b.Bytes()).
func (b *Buffer) Len() int { return len(b.buf) - b.off } // 注：获取缓冲区中未读取的字节数

// Cap 返回缓冲区基础字节片的容量，即为缓冲区数据分配的总空间。
func (b *Buffer) Cap() int { return cap(b.buf) } // 注：获取缓冲区的容量

// Truncate 丢弃缓冲区中除前n个未读字节以外的所有字节，但继续使用相同的已分配存储。
// 如果n为负数或大于缓冲区的长度，则会发生恐慌。
func (b *Buffer) Truncate(n int) { // 注：丢弃缓冲区b中前n个未读字节以外的所有数据
	if n == 0 {
		b.Reset() // 注：重置缓冲区
		return
	}
	b.lastRead = opInvalid
	if n < 0 || n > b.Len() {
		panic("bytes.Buffer: truncation out of range") // 恐慌："截断超出范围"
	}
	b.buf = b.buf[:b.off+n]
}

// Reset 将缓冲区重置为空，但保留底层存储供以后的写操作使用。
// 重置与Truncate(0)相同。
func (b *Buffer) Reset() { // 注：重置缓冲区
	b.buf = b.buf[:0]
	b.off = 0
	b.lastRead = opInvalid
}

// tryGrowByReslice 是grow的可内联版本，适用于快速情况，其中仅需要对内部缓冲区进行切片。
// 它返回应该在其中写入字节的索引以及索引是否成功。
func (b *Buffer) tryGrowByReslice(n int) (int, bool) { // 注：获取b是否可以容纳n个字节，返回扩容前的长度
	if l := len(b.buf); n <= cap(b.buf)-l { // 注：如果不需要扩容，返回true
		b.buf = b.buf[:l+n]
		return l, true
	}
	return 0, false
}

// grow 增加缓冲区以保证有n个字节的空间。
// 返回要在其中写入字节的索引。
// 如果缓冲区无法增长，则会因ErrTooLarge感到恐慌。
func (b *Buffer) grow(n int) int { // 注：b扩容n个字节的空间，返回扩容前的长度
	m := b.Len()
	// 如果缓冲区为空，请重置以恢复空间。
	if m == 0 && b.off != 0 { // 注：缓冲区为空，但偏移不为0，重置缓冲区
		b.Reset()
	}
	// 尝试通过重新裁切来增长。
	if i, ok := b.tryGrowByReslice(n); ok { // 注：如果不需要扩容，返回扩容后的起始位置
		return i
	}
	if b.buf == nil && n <= smallBufferSize { // 注：如果缓冲区为nil，并且n <= 最小缓冲区大小，创建缓冲区
		b.buf = make([]byte, n, smallBufferSize)
		return 0
	}
	c := cap(b.buf)
	if n <= c/2-m { // 注：如果扩容后的长度小于容量的一半，丢弃已读取的数据
		// 我们可以向下滑动内容，而不用分配新的片段。
		// 我们只需要m + n <= c即可滑动，但是我们改为让容量增加一倍，因此我们不会花费所有时间进行复制。
		copy(b.buf, b.buf[b.off:])
	} else if c > maxInt-c-n { // 注：如果扩容后的缓冲区长度超过了int的最大值，引发恐慌
		panic(ErrTooLarge) // 恐慌："太大"
	} else {
		// 任何地方没有足够的空间，我们需要分配。
		buf := makeSlice(2*c + n) // 注：扩容一倍
		copy(buf, b.buf[b.off:])
		b.buf = buf
	}
	// 重置b.off和len(b.buf)。
	b.off = 0
	b.buf = b.buf[:m+n]
	return m
}

// Grow 必要时增加缓冲区的容量，以保证另外n个字节的空间。 在Grow(n)之后，至少可以将n个字节写入缓冲区，而无需进行其他分配。
// 如果n为负数，Grow会惊慌。
// 如果缓冲区无法增长，则会因ErrTooLarge感到恐慌。
func (b *Buffer) Grow(n int) { // 注：b保证可以容纳n个字节的空间
	if n < 0 {
		panic("bytes.Buffer.Grow: negative count") // 恐慌："负数数量"
	}
	m := b.grow(n)
	b.buf = b.buf[:m]
}

// Write 将p的内容附加到缓冲区，根据需要增大缓冲区。
// 返回值n是p的长度； 错误始终为零。 如果缓冲区太大，则Write会因ErrTooLarge感到恐慌。
func (b *Buffer) Write(p []byte) (n int, err error) { // 注：向缓冲区b写入字节数组p
	b.lastRead = opInvalid
	m, ok := b.tryGrowByReslice(len(p)) // 注：获取b是否可以容纳p
	if !ok {
		m = b.grow(len(p)) // 注：容纳不下，扩展b
	}
	return copy(b.buf[m:], p), nil // 注：追加p
}

// WriteString 将s的内容附加到缓冲区，根据需要增大缓冲区。
// 返回值n是s的长度； 错误始终为零。 如果缓冲区太大，WriteString将对ErrTooLarge感到恐慌。
func (b *Buffer) WriteString(s string) (n int, err error) { // 注：向缓冲区b写入字符串s
	b.lastRead = opInvalid
	m, ok := b.tryGrowByReslice(len(s)) // 注：获取b是否可以容纳p
	if !ok {
		m = b.grow(len(s)) // 注：容纳不下，扩展b
	}
	return copy(b.buf[m:], s), nil // 注：追加s
}

// MinRead 是Buffer.ReadFrom传递给Read调用的最小切片大小。
// 只要缓冲区至少具有至少MinRead个字节（超出保留r的内容所需的字节数），ReadFrom就不会增长底层缓冲区。
const MinRead = 512 // 注：读取一次缓冲区的最小字节数

// ReadFrom 从r读取数据，直到EOF并将其附加到缓冲区，然后根据需要增大缓冲区。
// 返回值n是读取的字节数。
// 读取期间遇到的除io.EOF之外的任何错误也将返回。
// 如果缓冲区太大，ReadFrom会因ErrTooLarge感到恐慌。
func (b *Buffer) ReadFrom(r io.Reader) (n int64, err error) { // 注：从r中读取数据到缓冲区b，返回读取到数据的字节数m与错误err
	b.lastRead = opInvalid
	for {
		i := b.grow(MinRead) // 注：扩容
		b.buf = b.buf[:i]
		m, e := r.Read(b.buf[i:cap(b.buf)]) // 注：从r中读取数据到缓冲区
		if m < 0 {
			panic(errNegativeRead) // 恐慌："Read返回了负数"
		}

		b.buf = b.buf[:i+m]
		n += int64(m) // 注：已读取的字符数
		if e == io.EOF {
			return n, nil // e为EOF，因此显式返回nil
		}
		if e != nil {
			return n, e
		}
	}
}

// makeSlice 分配大小为n的切片。 如果分配失败，则会因ErrTooLarge感到恐慌。
func makeSlice(n int) []byte { // 注：创建一个长度为n的[]byte
	// 如果制作失败，请给出一个已知的错误。
	defer func() {
		if recover() != nil { // 注：如果捕获到了异常，引发长度太大异常
			panic(ErrTooLarge)
		}
	}()
	return make([]byte, n)
}

// WriteTo 将数据写入w，直到缓冲区耗尽或发生错误。
// 返回值n是写入的字节数； 它始终适合int，但与io.WriterTo接口匹配为int64。 写入期间遇到的任何错误也将返回。
func (b *Buffer) WriteTo(w io.Writer) (n int64, err error) { // 注：将缓冲区b的未读取数据写入w中，返回已写入的字节数n与错误err
	b.lastRead = opInvalid
	if nBytes := b.Len(); nBytes > 0 { // 注：如果缓冲区的长度 > 0
		m, e := w.Write(b.buf[b.off:]) // 注：将缓冲区数据写入w
		if m > nBytes {                // 注：写入的数据比缓冲区数据多，引发恐慌
			panic("bytes.Buffer.WriteTo: invalid Write count") // 恐慌："无效的写计数"
		}
		b.off += m // 注：增加偏移量
		n = int64(m)
		if e != nil {
			return n, e
		}
		// 根据io.Writer中Write方法的定义，所有字节均应已写入
		if m != nBytes { // 注：如果写入的数据的长度不等于缓冲区数据的长度，返回错误
			return n, io.ErrShortWrite
		}
	}
	// 缓冲区现在为空；重置
	b.Reset()
	return n, nil
}

// WriteByte 将字节c附加到缓冲区，根据需要增大缓冲区。
// 返回的错误始终为nil，但包含该错误以匹配bufio.Writer的WriteByte。
// 如果缓冲区太大，WriteByte会因ErrTooLarge感到恐慌。
func (b *Buffer) WriteByte(c byte) error { // 注：向缓冲区b写入字节c
	b.lastRead = opInvalid
	m, ok := b.tryGrowByReslice(1) // 注：b是否可以容纳1个字节
	if !ok {
		m = b.grow(1) // 注：扩容1
	}
	b.buf[m] = c // 注：写入c
	return nil
}

// WriteRune 将Unicode代码点r的UTF-8编码附加到缓冲区，返回其长度和错误，
// 该错误始终为nil，但包含该错误以匹配bufio.Writer的WriteRune。 缓冲区根据需要增长；
// 如果太大，WriteRune会因ErrTooLarge感到恐慌。
func (b *Buffer) WriteRune(r rune) (n int, err error) { // 注：向缓冲区b写入rune
	if r < utf8.RuneSelf { // 注：如果r是单字节rune，直接写入
		b.WriteByte(byte(r))
		return 1, nil
	}
	b.lastRead = opInvalid
	m, ok := b.tryGrowByReslice(utf8.UTFMax) // 注：b是否儿可以容纳1个rune
	if !ok {
		m = b.grow(utf8.UTFMax) // 注：扩容
	}
	n = utf8.EncodeRune(b.buf[m:m+utf8.UTFMax], r)
	b.buf = b.buf[:m+n]
	return n, nil
}

// Read 从缓冲区中读取下一个len(p)字节，或者直到缓冲区耗尽为止。
// 返回值n是读取的字节数。 如果缓冲区没有数据要返回，则err为io.EOF（除非len(p)为零）；否则为。
// 否则为nil。
func (b *Buffer) Read(p []byte) (n int, err error) { // 注：从缓冲区b中读取数据拷贝到p中，返回读取到的数据长度n与错误err
	b.lastRead = opInvalid
	if b.empty() { // 注：如果缓冲区b为空，重置缓冲区
		//缓冲区为空，请重置以恢复空间。
		b.Reset()
		if len(p) == 0 {
			return 0, nil
		}
		return 0, io.EOF
	}
	n = copy(p, b.buf[b.off:]) // 注：从缓冲区b中读取数据复制给p
	b.off += n
	if n > 0 {
		b.lastRead = opRead
	}
	return n, nil
}

// Next 从缓冲区返回包含接下来的n个字节的切片，将缓冲区前进，就好像字节已由Read返回。
// 如果缓冲区中的字节数少于n个，则Next返回整个缓冲区。
// 切片仅在下一次调用read或write方法之前有效。
func (b *Buffer) Next(n int) []byte { // 注：获取缓冲区b中接下来n个字节的数据
	b.lastRead = opInvalid
	m := b.Len()
	if n > m {
		n = m
	}
	data := b.buf[b.off : b.off+n] // 注：读取off开始的n个字节
	b.off += n
	if n > 0 {
		b.lastRead = opRead
	}
	return data
}

// ReadByte 从缓冲区读取并返回下一个字节。
// 如果没有可用的字节，则返回错误io.EOF。
func (b *Buffer) ReadByte() (byte, error) { // 注：获取缓冲区b中接下来的1个字节
	if b.empty() {
		// 缓冲区为空，请重置以恢复空间。
		b.Reset()
		return 0, io.EOF
	}
	c := b.buf[b.off]
	b.off++
	b.lastRead = opRead
	return c, nil
}

// ReadRune 从缓冲区读取并返回下一个UTF-8编码的Unicode代码点。
// 如果没有可用的字节，则返回的错误是io.EOF。
// 如果字节是错误的UTF-8编码，则将占用一个字节并返回U + FFFD，1。
func (b *Buffer) ReadRune() (r rune, size int, err error) { // 注：获取缓冲区b中接下来的1个rune
	if b.empty() {
		// 缓冲区为空，请重置以恢复空间。
		b.Reset()
		return 0, 0, io.EOF
	}
	c := b.buf[b.off]
	if c < utf8.RuneSelf {
		b.off++
		b.lastRead = opReadRune1
		return rune(c), 1, nil
	}
	r, n := utf8.DecodeRune(b.buf[b.off:])
	b.off += n
	b.lastRead = readOp(n)
	return r, n, nil
}

// UnreadRune 取消读取ReadRune返回的最后一个符文。
// 如果缓冲区上最近的读取或写入操作未成功执行ReadRune，则UnreadRune返回错误。
// （在这方面，它比UnreadByte严格，它将从任何读取操作中读取最后一个字节。）
func (b *Buffer) UnreadRune() error { // 注：撤回上次ReadRune
	if b.lastRead <= opInvalid {
		return errors.New("bytes.Buffer: UnreadRune: previous operation was not a successful ReadRune") // 恐慌："先前的操作不是成功的ReadRune"
	}
	if b.off >= int(b.lastRead) {
		b.off -= int(b.lastRead)
	}
	b.lastRead = opInvalid
	return nil
}

var errUnreadByte = errors.New("bytes.Buffer: UnreadByte: previous operation was not a successful read") // 恐慌："先前的操作未成功读取"

// UnreadByte 取消读取最近成功读取至少一个字节的读取操作返回的最后一个字节。
// 如果自上次读取以来发生了写操作，或者如果上一次读取返回错误，或者读取的读取字节为零，则UnreadByte返回错误。
func (b *Buffer) UnreadByte() error { // 注：撤回上次ReadByte
	if b.lastRead == opInvalid {
		return errUnreadByte
	}
	b.lastRead = opInvalid
	if b.off > 0 {
		b.off--
	}
	return nil
}

// ReadBytes 读取直到在输入中第一次出现delim为止，返回一个包含数据的切片，该数据直到并包括定界符。
// 如果ReadBytes在找到定界符之前遇到错误，它将返回错误之前读取的数据和错误本身（通常为io.EOF）。
// 当且仅当返回的数据未以delim结尾时，ReadBytes返回err != nil。
func (b *Buffer) ReadBytes(delim byte) (line []byte, err error) { // 注：从缓冲区b中读取数据直到遇到delim，返回获取到的数据line与错误err
	slice, err := b.readSlice(delim)
	// 返回slice的副本。 缓冲区的支持数组可能会被以后的调用覆盖。
	line = append(line, slice...)
	return line, err
}

// readSlice 类似于ReadBytes，但是返回对内部缓冲区数据的引用。
func (b *Buffer) readSlice(delim byte) (line []byte, err error) { // 注：#
	i := IndexByte(b.buf[b.off:], delim) // 注：#
	end := b.off + i + 1
	if i < 0 {
		end = len(b.buf)
		err = io.EOF
	}
	line = b.buf[b.off:end]
	b.off = end
	b.lastRead = opRead
	return line, err
}

// ReadString 读取直到在输入中第一次出现delim为止，返回一个字符串，其中包含直到定界符（包括定界符）的数据。
// 如果ReadString在找到定界符之前遇到错误，它将返回错误之前读取的数据和错误本身（通常为io.EOF）。
// 仅当返回的数据不以delim结尾时，ReadString才返回err！= nil。
func (b *Buffer) ReadString(delim byte) (line string, err error) { // 注：从缓冲区b中读取数据直到遇到delim，饭那会获取到的数据line与错误err
	slice, err := b.readSlice(delim)
	return string(slice), err
}

// NewBuffer 使用buf作为其初始内容创建并初始化一个新的Buffer。
// 新的Buffer拥有buf的所有权，并且在此调用之后，调用方不应使用buf。
// NewBuffer旨在准备一个Buffer以读取现有数据。
// 它也可以用来设置用于写入的内部缓冲区的初始大小。 为此，buf应该具有所需的容量，但长度为零。
//
//在大多数情况下，new（Buffer）（或仅声明一个Buffer变量）足以初始化Buffer。
func NewBuffer(buf []byte) *Buffer { return &Buffer{buf: buf} } // 工厂函数，创建一个缓冲区，缓冲区内容为buf

// NewBufferString 使用字符串s作为其初始内容创建并初始化一个新的Buffer。 目的是准备一个缓冲区以读取现有的字符串。
//
// 在大多数情况下，new(Buffer)（或仅声明一个Buffer变量）足以初始化Buffer。
func NewBufferString(s string) *Buffer { // 注：工厂函数，创建一个缓冲区，缓冲区内容为s
	return &Buffer{buf: []byte(s)}
}
