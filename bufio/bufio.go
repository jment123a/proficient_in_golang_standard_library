//版权所有2009 The Go Authors。 版权所有。
//此源代码的使用受BSD样式的约束
//可以在LICENSE文件中找到的许可证。

// Package bufio 实现了缓冲的I/O。
// 它包装了一个io.Reader或io.Writer对象，创建了另一个对象（Reader或Writer），该对象也实现了接口，但提供了缓冲和一些文本I/O帮助。
package bufio

import (
	"bytes"
	"errors"
	"io"
	"unicode/utf8"
)

const (
	defaultBufSize = 4096 // 注：默认缓冲区大小
)

var (
	ErrInvalidUnreadByte = errors.New("bufio: invalid use of UnreadByte") // 错误："无效使用UnreadByte"
	ErrInvalidUnreadRune = errors.New("bufio: invalid use of UnreadRune") // 错误："无效使用UnreadRune"
	ErrBufferFull        = errors.New("bufio: buffer full")               // 错误："缓冲区已满"
	ErrNegativeCount     = errors.New("bufio: negative count")            // 错误："负数计数"
)

// 缓冲的输入。

// Reader 为io.Reader对象实现缓冲。
// 注：
// r, w：r为从Reader中读取了多少字节的数据，w为从Reader中写入了多少字节的数据
// 例：buf = [100]byte{'1', '2', '3', '4', '5'}，r = 3，w = 4，表示buf[0:5]有数据，buf[0:3]（"123"）已经读取过了
type Reader struct {
	buf          []byte    // 注：缓冲区
	rd           io.Reader // 客户端提供的Reader
	r, w         int       // buf读写位置
	err          error     // 注：发生的错误
	lastByte     int       // 为UnreadByte读取的最后一个字节； -1表示无效
	lastRuneSize int       // UnreadRune读取的最后一个符文的大小； -1表示无效
}

const minReadBufferSize = 16         // 注：最小缓冲区大小
const maxConsecutiveEmptyReads = 100 // 注：最大读取空数据次数，如果read()执行100次还没有数据，则返回错误"多个读取调用不返回任何数据或错误"

// NewReaderSize 返回一个新的Reader，其缓冲区至少具有指定的大小。 如果参数io.Reader已经是足够大的Reader，则它将返回基础Reader。
func NewReaderSize(rd io.Reader, size int) *Reader { // 工厂函数，生成一个缓冲区大小为size的io.Reader结构体
	// 它已经是Reader了吗？
	b, ok := rd.(*Reader)
	if ok && len(b.buf) >= size { // 注：如果rd已经是缓冲区大小为size的Reader，返回rd
		return b
	}
	if size < minReadBufferSize { // 注：最小缓冲区大小
		size = minReadBufferSize
	}
	r := new(Reader)                // 注：新建一个Reader
	r.reset(make([]byte, size), rd) // 注：重置
	return r
}

// NewReader 返回一个新的Reader，其缓冲区具有默认大小。
func NewReader(rd io.Reader) *Reader { // 工厂函数，生成一个io.Reader结构体
	return NewReaderSize(rd, defaultBufSize)
}

// Size 返回基础缓冲区的大小（以字节为单位）。
func (b *Reader) Size() int { return len(b.buf) } // 注：获取b的缓冲区大小

// Reset 丢弃所有缓冲的数据，重置所有状态，并将缓冲的读取器切换为从r读取。
func (b *Reader) Reset(r io.Reader) { // 注：重置b
	b.reset(b.buf, r) // 注：缓冲区不变，io.Reader为r
}

func (b *Reader) reset(buf []byte, r io.Reader) { // 注：重置b
	*b = Reader{ // 注：缓冲区为buf，io.Reader为r
		buf:          buf,
		rd:           r,
		lastByte:     -1,
		lastRuneSize: -1,
	}
}

var errNegativeRead = errors.New("bufio: reader returned negative count from Read") // 错误："Reader从Read中返回了负数"

// fill 将新的块读取到缓冲区中。
func (b *Reader) fill() { // 注：从b.rd中读取数据写入缓冲区（自旋直到读取到数据或尝试超过一定次数）
	// 注：截取已读取的数据，将read到的数据追加至缓冲区

	// 将现有数据滑动到开头。
	if b.r > 0 { // 注：去掉已经读取的数据
		copy(b.buf, b.buf[b.r:b.w])
		b.w -= b.r
		b.r = 0
	}

	if b.w >= len(b.buf) { // 注：缓冲区已满
		panic("bufio: tried to fill full buffer") // 恐慌："试图填充已满的缓冲区"
	}

	// 读取新数据：尝试有限次数。
	for i := maxConsecutiveEmptyReads; i > 0; i-- { // 注：尝试读取数据
		n, err := b.rd.Read(b.buf[b.w:]) // 注：从b中读取数据，写入缓冲区
		if n < 0 {
			panic(errNegativeRead) // 恐慌："Reader从Read中返回了负数"
		}
		b.w += n
		if err != nil {
			b.err = err
			return
		}
		if n > 0 {
			return
		}
	}
	b.err = io.ErrNoProgress // 错误："多个读取调用不返回任何数据或错误"
}

func (b *Reader) readErr() error { // 注：返回b的错误
	err := b.err
	b.err = nil
	return err
}

// Peek 返回下n个字节，而不会使Reader前进。
// 在下一个读取调用中，字节停止有效。
// 如果Peek返回的字节数少于n个字节，则它还会返回一个错误，解释读取短的原因。
// 如果n大于b的缓冲区大小，则错误为ErrBufferFull。
//
// 调用Peek会阻止UnreadByte或UnreadRune调用成功，直到下一次读取操作为止。
func (b *Reader) Peek(n int) ([]byte, error) { // 注：b读取n字节数据
	if n < 0 {
		return nil, ErrNegativeCount // 错误："负数计数"
	}

	b.lastByte = -1
	b.lastRuneSize = -1

	for b.w-b.r < n && b.w-b.r < len(b.buf) && b.err == nil { // 注：从reader读取数据写入缓冲区，直到缓冲区至少存在n个字节或缓冲区满
		b.fill() // b.w-b.r < len(b.buf) => 缓冲区未满
	}

	if n > len(b.buf) { // 注：如果n比缓冲区大，返回部分数据与错误
		return b.buf[b.r:b.w], ErrBufferFull
	}

	// 0 <= n <= len(b.buf)
	var err error
	if avail := b.w - b.r; avail < n { // 注：如果缓冲区已满，但读取的数据长度<n
		// 缓冲区中数据不足
		n = avail
		err = b.readErr() // 注：获取reader的错误
		if err == nil {
			err = ErrBufferFull
		}
	}
	return b.buf[b.r : b.r+n], err // 注：返回部分数据与错误
}

// Discard 跳过接下来的n个字节，返回丢弃的字节数。
//
// 如果Discard跳过少于n个字节，则它还会返回错误。
// 如果0 <= n <= b.Buffered()，则确保Discard成功执行，而无需从基础io.Reader中读取。
func (b *Reader) Discard(n int) (discarded int, err error) { // 注：将b的缓冲区读取位置跳过n个字符
	// 例：
	// b.r = 0
	// 执行b.Discard(2)后
	// b.r = 2

	if n < 0 {
		return 0, ErrNegativeCount
	}
	if n == 0 {
		return
	}
	remain := n
	for { // 注：自旋，直到b跳过了n个字节
		skip := b.Buffered() // 注：获取b的缓冲区中可以读取的字节数
		if skip == 0 {       // 注：如果没有可读取的数据，从reader中读取数据再获取
			b.fill()
			skip = b.Buffered()
		}
		if skip > remain {
			skip = remain
		}
		b.r += skip    // 注：跳过skip个字节
		remain -= skip // 注：如果reamin不为0，表示还有remain个字节没有跳过
		if remain == 0 {
			return n, nil
		}
		if b.err != nil {
			return n - remain, b.readErr()
		}
	}
}

// Read 将数据读入p。
// 返回读入p的字节数。
// 这些字节是从基础读取器上的最多一个读取中获取的，因此n可能小于len(p)。
// 要精确读取len(p)个字节，请使用io.ReadFull(b，p)。
// 在EOF处，计数为零，而err为io.EOF。
func (b *Reader) Read(p []byte) (n int, err error) { // 注：将b的缓冲区内的未读数据写入p中
	// 注：根据p的长度判断是否需要缓冲区
	// 如果缓冲区没有未读数据，并且p比缓冲区大，直接写入至p
	// 如果缓冲区没有未读数据，并且p比缓冲区小，重置缓冲区，将数据写入缓冲区，再将缓冲区内的数据尽量写入p
	// 如果缓冲区有未读数据，将缓冲区内的数据尽量写入p

	n = len(p)
	if n == 0 { // 注：如果p的长度为0，检查缓冲区是否有未读数据
		if b.Buffered() > 0 {
			return 0, nil
		}
		return 0, b.readErr()
	}

	if b.r == b.w { // 注：如果没有未读数据
		if b.err != nil {
			return 0, b.readErr()
		}
		if len(p) >= len(b.buf) { // 注：如果p的长度大于缓冲区，直接读取到p中
			// 大读取，空缓冲区。
			// 直接读入p以避免复制。
			n, b.err = b.rd.Read(p) // 注：从b读取数据写入p
			if n < 0 {
				panic(errNegativeRead)
			}
			if n > 0 { // 注：记录这次读取的最后一个字节
				b.lastByte = int(p[n-1])
				b.lastRuneSize = -1
			}
			return n, b.readErr()
		}
		// 读取一次。
		// 不要使用b.fill，它会循环。
		b.r = 0
		b.w = 0
		n, b.err = b.rd.Read(b.buf) // 注：初始化缓冲区，从b读取数据写入缓冲区
		if n < 0 {
			panic(errNegativeRead)
		}
		if n == 0 {
			return 0, b.readErr()
		}
		b.w += n
	}

	// 尽可能多地复制
	n = copy(p, b.buf[b.r:b.w]) // 注：将缓冲区的数据拷贝给p
	b.r += n
	b.lastByte = int(b.buf[b.r-1])
	b.lastRuneSize = -1
	return n, nil
}

// ReadByte 读取并返回一个字节。
// 如果没有可用的字节，则返回错误。
func (b *Reader) ReadByte() (byte, error) { // 注：获取b的缓冲区中1字节的未读数据
	// 注：自旋直到b读取到数据
	b.lastRuneSize = -1
	for b.r == b.w {
		if b.err != nil {
			return 0, b.readErr()
		}
		b.fill() // 缓冲区为空，注：自旋直到读取到数据
	}
	c := b.buf[b.r] // 注：读取一个数据
	b.r++
	b.lastByte = int(c)
	return c, nil
}

// UnreadByte 不读取最后一个字节。 只有最近读取的字节可以不被读取。
//
// 如果在Reader上调用的最新方法不是读取操作，则UnreadByte返回错误。
// 值得注意的是，Peek不被视为读取操作。
func (b *Reader) UnreadByte() error { // 注：撤回上次ReadByte()
	if b.lastByte < 0 || b.r == 0 && b.w > 0 { // 注：如果上次没有读取字节 或 没有读取过数据
		return ErrInvalidUnreadByte // 错误："无效使用UnreadByte"
	}
	// b.r > 0 || b.w == 0
	if b.r > 0 { // 注：如果读取过至少一个字节
		b.r--
	} else { // 注：如果没有未读数据，也没有读取过数据，表示已读取的数据被截取了
		// b.r == 0 && b.w == 0
		b.w = 1
	}
	b.buf[b.r] = byte(b.lastByte)
	b.lastByte = -1
	b.lastRuneSize = -1
	return nil
}

// ReadRune 读取单个UTF-8编码的Unicode字符，并返回符文及其大小（以字节为单位）。
// 如果编码的符文无效，则它消耗一个字节并返回unicode.ReplacementChar (U+FFFD)，大小为1。
func (b *Reader) ReadRune() (r rune, size int, err error) { // 注：从未读数据中获取一个rune
	// 注：未读数据中有没有容纳rune的空间 并且 未读数据中没有一个完整的rune 并且 b没有错误 并且 未读数据小于缓冲区
	for b.r+utf8.UTFMax > b.w && !utf8.FullRune(b.buf[b.r:b.w]) && b.err == nil && b.w-b.r < len(b.buf) { // 注：如果未读数据装不下rune，读取数据到缓冲区
		b.fill() // b.w-b.r < len(buf) => 缓冲区未满
	}
	b.lastRuneSize = -1
	if b.r == b.w { // 注：如果读取不到数据，返回错误
		return 0, 0, b.readErr()
	}
	r, size = rune(b.buf[b.r]), 1 // 注：获取一个字节
	if r >= utf8.RuneSelf {       // 注：如果为多字节rune，读取完整的rune
		r, size = utf8.DecodeRune(b.buf[b.r:b.w])
	}
	b.r += size
	b.lastByte = int(b.buf[b.r-1])
	b.lastRuneSize = size
	return r, size, nil
}

// UnreadRune 取消读取最后一个rune。
// 如果在Reader上调用的最新方法不是ReadRune，则UnreadRune返回错误。
// （在这方面，它比UnreadByte严格，它将从任何读取操作中读取最后一个字节。）
func (b *Reader) UnreadRune() error { // 注：撤回上次ReadRune()
	if b.lastRuneSize < 0 || b.r < b.lastRuneSize { // 注：如果没有读取过rune 或 rune的长度不对，返回错误："无效使用UnreadRune"
		return ErrInvalidUnreadRune
	}
	b.r -= b.lastRuneSize
	b.lastByte = -1
	b.lastRuneSize = -1
	return nil
}

// Buffered 返回可以从当前缓冲区读取的字节数。
func (b *Reader) Buffered() int { return b.w - b.r } // 注：获取b的缓冲区可以读取的字节数

// ReadSlice 读取直到输入中第一次出现delim为止，返回一个指向缓冲区中字节的切片。
// 字节在下一次读取时不再有效。
// 如果ReadSlice在找到定界符之前遇到错误，它将返回缓冲区中的所有数据以及错误本身（通常为io.EOF）。
// 如果缓冲区填充不带delim，则ReadSlice失败，错误ErrBufferFull。
// 因为从ReadSlice返回的数据将被下一个I/O操作覆盖，所以大多数客户端应改用ReadBytes或ReadString。
// 当且仅当行不以delim结尾时，ReadSlice返回err！= nil。
func (b *Reader) ReadSlice(delim byte) (line []byte, err error) { // 注：获取数据，直到遇到delim
	// 注：
	// 读取未读数据，如果遇到delim，返回切片
	// 读取未读数据，如果没有遇到delim，读取数据到缓冲区，如果遇到delim，返回切片
	// 读取未读数据，如果没有遇到delim，读取数据到缓冲区，如果缓冲区满了，返回错误
	s := 0 // 搜索开始索引
	for {  // 注：自旋
		// 搜索缓冲区。
		if i := bytes.IndexByte(b.buf[b.r+s:b.w], delim); i >= 0 { // 注：如果遇到delim
			i += s
			line = b.buf[b.r : b.r+i+1]
			b.r += i + 1
			break
		}

		// 待处理错误？
		if b.err != nil { // 注：如果b出现错误，返回粗恶偶
			line = b.buf[b.r:b.w]
			b.r = b.w
			err = b.readErr()
			break
		}

		// 缓冲区已满？
		if b.Buffered() >= len(b.buf) { // 注：如果缓冲区装满了未读数据，返回错误
			b.r = b.w
			line = b.buf
			err = ErrBufferFull
			break
		}

		// 注：如果没有遇到delim，获取数据，直到遇到delim 或 缓冲区装满
		s = b.w - b.r // 不要重新扫描我们之前扫描过的区域

		b.fill() // 缓冲区未满
	}

	// 处理最后一个字节（如果有）。
	if i := len(line) - 1; i >= 0 {
		b.lastByte = int(line[i])
		b.lastRuneSize = -1
	}

	return
}

// ReadLine 是低级别的行读取原语。 大多数调用者应改用ReadBytes('\n')或ReadString('\n')或使用Scanner。
//
// ReadLine尝试返回单行，不包括行尾字节。
// 如果该行对于缓冲区而言太长，则设置isPrefix并返回该行的开头。
// 该行的其余部分将从以后的呼叫中返回。
// 返回行的最后一个片段时，isPrefix将为false。
// 返回的缓冲区仅在下一次调用ReadLine之前有效。
// ReadLine返回一个非空行，或者返回一个错误，从不都返回。
//
// 从ReadLine返回的文本不包含行尾（"\r\n"或"\n"）。
// 如果输入在没有最后一行结束的情况下结束，则不会给出任何指示或错误。
// 在ReadLine之后调用UnreadByte将始终不读取最后一个读取的字节（可能是属于行尾的字符），即使该字节不属于ReadLine返回的行的一部分。
func (b *Reader) ReadLine() (line []byte, isPrefix bool, err error) { // 注：获取数据，直到遇到\n或\r\n
	line, err = b.ReadSlice('\n') // 注：读取未读数据直到遇到\n
	if err == ErrBufferFull {     // 注：如果缓冲区满了，返回缓冲区内的切片
		// 处理"\r\n"跨越缓冲区的情况。
		if len(line) > 0 && line[len(line)-1] == '\r' { // 注：如果缓冲区最后一个字符是\r
			// 将"\r"放回buf并将其从行中放下。
			// 让下一次对ReadLine的调用检查"\r\n"。
			if b.r == 0 {
				// 应该无法到达
				panic("bufio: tried to rewind past start of buffer") // 恐慌："试图倒带过去缓冲区的开始"
			}
			b.r--
			line = line[:len(line)-1] // 注：截取\r
		}
		return line, true, nil
	}

	if len(line) == 0 { // 注：如果返回数据为空 并且 出现错误，返回错误
		if err != nil {
			line = nil
		}
		return
	}
	err = nil

	if line[len(line)-1] == '\n' { // 注：如果最后一个字符是\n，截取\n或\r\n
		drop := 1
		if len(line) > 1 && line[len(line)-2] == '\r' { // 注：如果倒数第二个字符是\r
			drop = 2
		}
		line = line[:len(line)-drop]
	}
	return
}

// ReadBytes 读取直到在输入中第一次出现delim为止，并返回一个切片，该切片包含直到定界符（包括定界符）的数据。
// 如果ReadBytes在找到定界符之前遇到错误，它将返回错误之前读取的数据和错误本身（通常为io.EOF）。
// 当且仅当返回的数据未以delim结尾时，ReadBytes返回err != nil。
// 对于简单的用途，扫描仪可能更方便。
func (b *Reader) ReadBytes(delim byte) ([]byte, error) { // 注：获取数据，直到遇到delim（保证数据完整）
	// 使用ReadSlice查找数组，累积完整的缓冲区。
	var frag []byte
	var full [][]byte
	var err error
	n := 0
	for { // 注：自旋，直至数据完整
		var e error
		frag, e = b.ReadSlice(delim) // 注：获取数据，直到遇到delim
		if e == nil {                // 得到了最后的片段
			break
		}
		if e != ErrBufferFull { // 意外的错误，注：除了数据不完整会出现的错误，其他错误返回
			err = e
			break
		}

		// 复制缓冲区。
		buf := make([]byte, len(frag))
		copy(buf, frag)
		full = append(full, buf) // 注：追加至full
		n += len(buf)
	}

	n += len(frag)

	// 分配新缓冲区以容纳完整片段和片段。
	buf := make([]byte, n)
	n = 0
	// 复制完整的片段并切入。
	for i := range full { // 注：？为什么这么麻烦
		n += copy(buf[n:], full[i])
	}
	copy(buf[n:], frag) // 注：拷贝最后的片段
	return buf, err
}

// ReadString 读取直到输入中第一次出现delim为止，返回一个字符串，其中包含直到定界符（包括定界符）的数据。
// 如果ReadString在找到定界符之前遇到错误，它将返回错误之前读取的数据和错误本身（通常为io.EOF）。
// 仅当返回的数据不以delim结尾时，ReadString才返回err != nil。
// 对于简单的用途，扫描仪可能更方便。
func (b *Reader) ReadString(delim byte) (string, error) { // 注：获取数据，直到遇到delim（保证数据完整）
	bytes, err := b.ReadBytes(delim)
	return string(bytes), err
}

// WriteTo 实现io.WriterTo。
// 这可能会多次调用基础Reader的Read方法。
// 如果基础reader支持WriteTo方法，则此方法将调用基础WriteTo而不进行缓冲。
func (b *Reader) WriteTo(w io.Writer) (n int64, err error) { // 注：将b的所有数据写入w
	// 注：
	// 1. b将缓冲区内的未读数据写入w
	// 2. 尝试执行r.WriteTo(w)与w.ReadFrom(b.rd)
	// 3. 从b.rd读取数据写入w，直到b.rd读取结束

	n, err = b.writeBuf(w) // 注：将缓冲区的未读数据写入w
	if err != nil {
		return
	}

	if r, ok := b.rd.(io.WriterTo); ok { // 注：断言执行WriteTo，从b.rd读取数据写入w
		m, err := r.WriteTo(w)
		n += m
		return n, err
	}

	if w, ok := w.(io.ReaderFrom); ok { // 注：断言执行ReadFrom，从b.rd读取数据写入w
		m, err := w.ReadFrom(b.rd)
		n += m
		return n, err
	}

	if b.w-b.r < len(b.buf) { // 注：缓冲区未满，填充数据
		b.fill() // 缓冲区未满
	}

	for b.r < b.w { // 注：自旋，直到b.rd的所有数据都写入w
		// b.r < b.w => 缓冲区不为空
		m, err := b.writeBuf(w) // 注：将b的未读数据写入w
		n += m
		if err != nil {
			return n, err
		}
		b.fill() // 缓冲区为空，注：b获取数据
	}

	if b.err == io.EOF {
		b.err = nil
	}

	return n, b.readErr()
}

var errNegativeWrite = errors.New("bufio: writer returned negative count from Write") // 错误："writer从Write返回了负数"

// writeBuf 将reader的缓冲区写入writer。
func (b *Reader) writeBuf(w io.Writer) (int64, error) { // 注：将缓冲区中的未读数据写入w
	n, err := w.Write(b.buf[b.r:b.w]) // 注：将未读数据写入w
	if n < 0 {
		panic(errNegativeWrite)
	}
	b.r += n
	return int64(n), err
}

// 缓冲的输出

// Writer 为io.Writer对象实现缓冲。
// 如果在写入Writer时发生错误，将不再接受更多数据，并且所有后续写入和Flush都将返回错误。
// 写入所有数据之后，客户端应调用Flush方法以确保所有数据都已转发到基础io.Writer。
type Writer struct {
	err error     // 注：出现的错误
	buf []byte    // 注：缓冲区
	n   int       // 注：缓冲区已使用的字节数
	wr  io.Writer // 注：数据源
}

// NewWriterSize 返回一个新的Writer，其缓冲区至少具有指定的大小。
// 如果参数io.Writer已经是一个足够大的Writer，它将返回基础Writer。
func NewWriterSize(w io.Writer, size int) *Writer { // 工厂函数，生成一个Writer结构体，缓冲区大小为size
	// 已经是writer吗？
	b, ok := w.(*Writer)
	if ok && len(b.buf) >= size { // 注：如果w已经是writer，直接返回
		return b
	}
	if size <= 0 { // 注：默认缓冲区大小
		size = defaultBufSize
	}
	return &Writer{
		buf: make([]byte, size),
		wr:  w,
	}
}

// NewWriter 返回一个新的Writer，其缓冲区具有默认大小。
func NewWriter(w io.Writer) *Writer { // 工厂函数，生成一个Writer结构体
	return NewWriterSize(w, defaultBufSize)
}

// Size 返回基础缓冲区的大小（以字节为单位）。
func (b *Writer) Size() int { return len(b.buf) } // 注：获取b的缓冲区大小

// Reset 丢弃所有未刷新的缓冲数据，清除所有错误，并将b复位以将其输出写入w。
func (b *Writer) Reset(w io.Writer) { // 注：重置b的缓冲区，writer为w
	b.err = nil
	b.n = 0
	b.wr = w
}

// Flush 将所有缓冲的数据写入基础io.Writer。
func (b *Writer) Flush() error { // 注：将缓冲区中的数据写入writer
	// 注：将缓冲区中的数据写入writer，有可能只写入了部分数据
	if b.err != nil { // 注：如果b出现错误，返回错误
		return b.err
	}
	if b.n == 0 { // 注：如果缓冲区没有数据，返回nil
		return nil
	}
	n, err := b.wr.Write(b.buf[0:b.n]) // 注：向wr写入缓冲区的数据
	if n < b.n && err == nil {
		err = io.ErrShortWrite
	}
	if err != nil {
		if n > 0 && n < b.n { // 注：如果只写入了部分数据，截取缓冲区
			copy(b.buf[0:b.n-n], b.buf[n:b.n])
		}
		b.n -= n
		b.err = err
		return err
	}
	b.n = 0
	return nil
}

// Available 返回缓冲区中未使用的字节数。
func (b *Writer) Available() int { return len(b.buf) - b.n } // 注：获取缓冲区中未使用的字节数

// Buffered 返回已写入当前缓冲区的字节数。
func (b *Writer) Buffered() int { return b.n } // 注：获取缓冲区中已使用的字节数

// Write 将p的内容写入缓冲区。
// 返回写入的字节数。
// 如果nn < len(p)，它还会返回一个错误，解释为什么写入的数据少。
func (b *Writer) Write(p []byte) (nn int, err error) { // 注：将p写入writer
	for len(p) > b.Available() && b.err == nil { // 注：如果p的内容大于缓冲区
		var n int
		if b.Buffered() == 0 { // 注：如果缓冲区中没有数据，直接将p写入writer
			// 大写，空缓冲区。
			// 直接从p写入以避免复制。
			n, b.err = b.wr.Write(p)
		} else { // 注：否则按缓冲区大小拆分p，分段写入writer
			n = copy(b.buf[b.n:], p)
			b.n += n
			b.Flush()
		}
		nn += n
		p = p[n:] // 注：截取p
	}
	if b.err != nil { // 注：如果出现错误，返回错误
		return nn, b.err
	}
	n := copy(b.buf[b.n:], p) // 注：写入剩余的数据
	b.n += n
	nn += n
	return nn, nil
}

// WriteByte 写入一个字节。
func (b *Writer) WriteByte(c byte) error { // 注：flush缓冲区，将c写入缓冲区
	// 注：先flush，再将c写入缓冲区
	if b.err != nil { // 注：如果出现错误，返回错误
		return b.err
	}
	if b.Available() <= 0 && b.Flush() != nil { // 注：如果缓冲区装满了数据 并且 将数据写入writer出错，返回错误
		return b.err
	}
	b.buf[b.n] = c
	b.n++
	return nil
}

// WriteRune 写一个Unicode代码点，返回写入的字节数和任何错误。
func (b *Writer) WriteRune(r rune) (size int, err error) { // 注：将r写入缓冲区
	// 注：
	// 如果r是单字节rune，直接写入缓冲区
	// 如果r是多字节rune，如果缓冲区装不下r，flush缓冲区，如果缓冲区装的下r，将写入缓冲区
	// 如果r是多字节rune，如果缓冲区装不下r，flush缓冲区，如果缓冲区装不下r，将r flush，将剩余的数据写入缓冲区
	// 如果r是多字节rune，如果缓冲区装的下r，将写入缓冲区

	if r < utf8.RuneSelf { // 注：如果r是单字节rune，将r写入缓冲区
		err = b.WriteByte(byte(r))
		if err != nil {
			return 0, err
		}
		return 1, nil
	}
	if b.err != nil {
		return 0, b.err
	}
	n := b.Available()
	if n < utf8.UTFMax { // 注：如果缓冲区容纳不下rune
		if b.Flush(); b.err != nil { // 注：flush缓冲区
			return 0, b.err
		}
		n = b.Available()
		if n < utf8.UTFMax { // 注：如果缓冲区还是容纳不下rune
			// 仅当缓冲区很小时才会发生。
			return b.WriteString(string(r)) // 注：将r写入缓冲区
		}
	}
	size = utf8.EncodeRune(b.buf[b.n:], r)
	b.n += size
	return size, nil
}

// WriteString 写一个字符串。
// 返回写入的字节数。
// 如果计数小于len(s)，它还会返回一个错误，解释为什么写入的数据少。
func (b *Writer) WriteString(s string) (int, error) { // 注：将s写入缓冲区
	// 注：如果len(s) > b.Available()，将s根据缓冲区大小拆分后写入writer，将剩余的数据放入缓冲区
	//
	// 例：s = "23456"，b.buf = [2]byte{'1'}
	// flush [2]byte{'1', '2'}
	// flush [2]byte{'3', '4'}
	// b.buf = [2]byte{'5', '6'}，b.n = 2
	nn := 0
	for len(s) > b.Available() && b.err == nil { // 注：如果s的大小超过了缓冲区可用的字节数，将s按缓冲区大小分段flush
		n := copy(b.buf[b.n:], s)
		b.n += n
		nn += n
		s = s[n:]
		b.Flush()
	}
	if b.err != nil {
		return nn, b.err
	}
	n := copy(b.buf[b.n:], s) // 注：将剩余的s写入缓冲区
	b.n += n
	nn += n
	return nn, nil
}

// ReadFrom 实现io.ReaderFrom。
// 如果基础编写器支持ReadFrom方法，并且b尚无缓冲数据，则这将调用基础ReadFrom而不进行缓冲。
func (b *Writer) ReadFrom(r io.Reader) (n int64, err error) { // 注：从r中读取数据写入writer，直到出现错误
	if b.err != nil { // 注：如果b出现错误，返回错误
		return 0, b.err
	}
	if b.Buffered() == 0 { // 注：如果没有缓冲数据
		if w, ok := b.wr.(io.ReaderFrom); ok { // 注：Writer断言执行ReadFrom，从r中读取数据写入writer
			n, err = w.ReadFrom(r)
			b.err = err
			return n, err
		}
	}
	var m int
	for { // 注：自旋，从r中读取数据写入writer，直到出现错误
		if b.Available() == 0 { // 注：如果缓冲区没有可用空间，flush
			if err1 := b.Flush(); err1 != nil {
				return n, err1
			}
		}
		nr := 0
		for nr < maxConsecutiveEmptyReads { // 注：在接受次数内，从r中读取数据到缓冲区，直到读取到数据
			m, err = r.Read(b.buf[b.n:])
			if m != 0 || err != nil {
				break
			}
			nr++
		}
		if nr == maxConsecutiveEmptyReads { // 注：如果超过接受次数，返回错误
			return n, io.ErrNoProgress
		}
		b.n += m
		n += int64(m)
		if err != nil {
			break
		}
	}
	if err == io.EOF {
		// 如果我们完全填满了缓冲区，请先刷新。
		if b.Available() == 0 {
			err = b.Flush()
		} else {
			err = nil
		}
	}
	return n, err
}

// 缓冲	输入和输出

// ReadWriter 储指向Reader和Writer的指针。
// 实现io.ReadWriter。
type ReadWriter struct {
	*Reader
	*Writer
}

// NewReadWriter 分配一个新的ReadWriter，该ReadWriter调度到r和w。
func NewReadWriter(r *Reader, w *Writer) *ReadWriter { // 工厂函数，生成一个ReadWriter结构体
	return &ReadWriter{r, w}
}
