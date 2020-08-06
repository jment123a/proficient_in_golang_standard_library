// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

// Package io 提供了与I/O原语的基本接口。
// 它的主要工作是包装此类原语的现有实现，（例如os套件中的程式码），对功能进行抽象，再加上其他一些相关的原语。
// 因为这些接口和原语使用各种实现来包装较低级的操作，除非另行通知，否则客户机不应认为它们对于并行执行是安全的。
package io

import "errors"

// 求值。
// 注：以下三个常量为枚举
const (
	SeekStart   = 0 //相对于文件的原点查找
	SeekCurrent = 1 //相对于当前偏移量的搜索
	SeekEnd     = 2 //相对于终点寻求
)

// ErrShortWrite 表示写入接受的字节数少于请求的字节数
// 但未返回显式错误。
var ErrShortWrite = errors.New("short write") //注：因"已写入数据的长度<要写入数据的长度"返回的错误

// ErrShortBuffer 表示读取所需的缓冲区比提供的缓冲区更长。
var ErrShortBuffer = errors.New("short buffer") //注：因"要读取数据的长度>缓冲的长度"返回的错误

// EOF 是没有更多输入可用时由Read返回的错误。
// 函数应仅返回EOF以表示输入正常结束。
// 如果EOF在结构化数据流中意外发生，
// 适当的错误是ErrUnexpectedEOF或其他错误
// 提供更多细节。
var EOF = errors.New("EOF") //注：End Of File，因"文件读取到最后"返回的错误

// ErrUnexpectedEOF 表示在
// 读取固定大小的块或数据结构的中间。
var ErrUnexpectedEOF = errors.New("unexpected EOF") //注：因"文件读取到最后，但缓冲没有装满"返回的错误

// ErrNoProgress 表示在读取固定大小的块或数据结构的中间遇到了EOF。
var ErrNoProgress = errors.New("multiple Read calls return no data or error") //注：#

// Reader 是包装基本Read方法的接口。
//
// 读取最多将len(p)个字节读入p。 它返回读取的字节数(0 <= n <= len(p))和遇到的任何错误。
// 即使Read返回n < len(p)，也可能在调用期间将所有p用作暂存空间。
// 如果某些数据可用但不是len(p)个字节，则按常规方式，Read将返回可用数据，而不是等待更多数据。
//
// 当成功读取n > 0个字节后，Read遇到错误或文件结束条件时，它将返回读取的字节数。
// 它可能从同一调用返回（非nil）错误，或者从后续调用返回错误（n == 0）。
// 这种一般情况的一个实例是，读取器在输入流的末尾返回非零字节数的情况下，可能返回err == EOF或err == nil。 下一次读取应返回0，EOF。
//
// 在考虑错误err之前，调用者应始终处理返回的n > 0个字节。
// 这样做可以正确处理在读取某些字节后发生的I/O错误，以及两种允许的EOF行为。
//
// 不鼓励使用Read的实现返回零字节计数且错误为nil的错误，除非len(p)== 0。
// 调用者应将返回0和nil视为没有任何反应； 特别是它并不表示EOF。
//
// 实现不得保留p。
type Reader interface {
	Read(p []byte) (n int, err error)
}

// Writer 是包装基本Write方法的接口。
//
// 写操作将p的len(p)个字节写入基础数据流。
// 它返回从p(0 <= n <= len(p))写入的字节数，以及遇到的任何导致写入提前停止的错误。
// 如果写入返回n < len(p)，则必须返回一个非nil错误。
// 写操作不得修改切片数据，即使是临时的。
//
// 实现不得保留p。
type Writer interface {
	Write(p []byte) (n int, err error)
}

// Closer 是包装基本Close方法的接口。
//
// 首次调用后关闭行为是不确定的。
// 特定的实现可能会记录自己的行为。
type Closer interface {
	Close() error
}

// Seeker 是包装基本Seek方法的接口。
//
// Seek将下一次"Read"或"Write"的offset设置为偏移量，根据情况解释：
// SeekStart表示相对于文件开始的位置，
// SeekCurrent表示相对于当前偏移量，以及
// SeekEnd表示相对于末端。
// Seek返回相对于文件开头的新偏移量，如果有错误，则返回错误。
//
// 在文件开始之前寻找偏移量是一个错误。
// 寻求任何正偏移都是合法的，但是对基础对象的后续I/O操作的行为取决于实现。
type Seeker interface {
	Seek(offset int64, whence int) (int64, error)
}

// ReadWriter 是将基本的Read和Write方法分组的接口。
type ReadWriter interface {
	Reader
	Writer
}

// ReadCloser 是对基本Read和Close方法进行分组的接口。
type ReadCloser interface {
	Reader
	Closer
}

// WriteCloser 是对基本Write和Close方法进行分组的接口。
type WriteCloser interface {
	Writer
	Closer
}

// ReadWriteCloser 是对基本的Read，Write和Close方法进行分组的接口。
type ReadWriteCloser interface {
	Reader
	Writer
	Closer
}

// ReadSeeker 是将基本的Read和Seek方法分组的接口。
type ReadSeeker interface {
	Reader
	Seeker
}

// WriteSeeker 是对基本Write和Seek方法进行分组的接口。
type WriteSeeker interface {
	Writer
	Seeker
}

// ReadWriteSeeker 是对基本的Read，Write和Seek方法进行分组的接口。
type ReadWriteSeeker interface {
	Reader
	Writer
	Seeker
}

// ReaderFrom 是包装ReadFrom方法的接口。
//
// ReadFrom从r读取数据，直到EOF或错误。
// 返回值n是读取的字节数。
// 读取期间还遇到除io.EOF以外的任何错误。
//
// 复制功能使用ReaderFrom（如果可用）。
type ReaderFrom interface {
	ReadFrom(r Reader) (n int64, err error)
}

// WriterTo 是包装WriteTo方法的接口。
//
// WriteTo将数据写入w，直到没有更多数据可写或发生错误为止。
// 返回值n是写入的字节数。 写入期间遇到的任何错误也将返回。
//
// 复制功能使用WriterTo（如果有）。
type WriterTo interface {
	WriteTo(w Writer) (n int64, err error)
}

// ReaderAt 是包装基本ReadAt方法的接口。
// ReadAt从基础输入源中的偏移量off处读取len(p)个字节到p中。
// 它返回读取的字节数(0 <= n <= len(p))和遇到的任何错误。
// 当ReadAt返回n < len(p)时，它返回一个非nil错误，解释了为什么不返回更多字节。在这方面，ReadAt比Read更严格。
// 即使ReadAt返回n < len(p)，也可能在调用过程中将所有p用作临时空间。如果某些数据可用但不是len(p)字节，则ReadAt会阻塞，直到所有数据可用或发生错误为止。
// 在这方面，ReadAt与Read不同。
// 如果ReadAt返回的n = len(p)个字节位于输入源的末尾，则ReadAt可能返回err == EOF或err == nil。
// 如果ReadAt正在从具有寻道偏移量的输入源中进行读取，则ReadAt不应影响也不受底层寻道偏移量的影响。
// ReadAt的客户端可以在同一输入源上执行并行ReadAt调用。
// 实现不得保留p。
type ReaderAt interface {
	ReadAt(p []byte, off int64) (n int, err error)
}

// WriterAt 是包装基本WriteAt方法的接口。
// WriteAt将偏移量为off的len(p)个字节从p写入基础数据流。
// 它返回从p(0 <= n <= len(p))写入的字节数。
// 和遇到的任何导致写入提前停止的错误。
// 如果WriteAt返回n < len(p)，则必须返回一个非nil错误。
// 如果WriteAt正在使用寻道偏移量写入目标，则WriteAt不应影响也不受底层寻道偏移量的影响。
// 如果范围不重叠，WriteAt的客户端可以在同一目标上执行并行WriteAt调用。
// 实现不得保留p。
type WriterAt interface {
	WriteAt(p []byte, off int64) (n int, err error)
}

// ByteReader 是包装ReadByte方法的接口。
//
// ReadByte从输入或遇到的任何错误中读取并返回下一个字节。
// 如果ReadByte返回错误，则不消耗任何输入字节，并且返回的字节值不确定。
type ByteReader interface {
	ReadByte() (byte, error)
}

// ByteScanner 是将UnreadByte方法添加到基本ReadByte方法的接口。
//
// UnreadByte导致下一次对ReadByte的调用返回与上一次对ReadByte的调用相同的字节。
// 在没有干预的情况下两次调用UnreadByte可能是错误的
// 调用ReadByte。
type ByteScanner interface {
	ByteReader
	UnreadByte() error
}

// ByteWriter 是包装WriteByte方法的接口。
type ByteWriter interface {
	WriteByte(c byte) error
}

// RuneReader 是包装ReadRune方法的接口。
//
// ReadRune读取单个UTF-8编码的Unicode字符，并返回符文及其大小（以字节为单位）。 如果没有可用字符，将设置err。
type RuneReader interface {
	ReadRune() (r rune, size int, err error)
}

// RuneScanner 是将UnreadRune方法添加到基本ReadRune方法的接口。
// UnreadRune导致下一次对ReadRune的调用返回与上一次对ReadRune的调用相同的符文。
// 两次调用UnreadRune而不进行中间调用ReadRune可能是错误的。
type RuneScanner interface {
	RuneReader
	UnreadRune() error
}

// StringWriter 是包装WriteString方法的接口。
type StringWriter interface {
	WriteString(s string) (n int, err error)
}

// WriteString 将字符串s的内容写入w，w接受字节的一部分。
// 如果w实现StringWriter，则直接调用其WriteString方法。
// 否则，w.Write只会被调用一次。
func WriteString(w Writer, s string) (n int, err error) { //注：调用w.WriteString或w.Write写入s，返回写入的长度与错误err
	if sw, ok := w.(StringWriter); ok {
		return sw.WriteString(s)
	}
	return w.Write([]byte(s))
}

// ReadAtLeast 从r读取到buf，直到它至少读取了min字节为止。
// 返回复制的字节数，如果读取的字节数少则返回错误。
// 仅当未读取任何字节时，错误才是EOF。
// 如果在读取少于min字节后发生EOF，则ReadAtLeast返回ErrUnexpectedEOF。
// 如果min大于buf的长度，则ReadAtLeast返回ErrShortBuffer。
// 返回时，当且仅当err == nil时，n >= min。
// 如果r返回至少读取了min字节的错误，则丢弃该错误。
func ReadAtLeast(r Reader, buf []byte, min int) (n int, err error) { //注：调用r.Read至少min字节数据存到buf中，返回读取到的数据长度n与错误err
	//注：如果缓冲buf的长度小于min，返回缓冲区过短错误
	if len(buf) < min {
		return 0, ErrShortBuffer
	}

	//注：如果读取到的数据长度n不满足要求长度min并且没有出错，则一直循环
	for n < min && err == nil {
		var nn int
		nn, err = r.Read(buf[n:])
		n += nn
	}

	if n >= min {
		err = nil
	} else if n > 0 && err == EOF { //注：如果读取到数据又发生EOF错误，返回意外的文件结尾错误
		err = ErrUnexpectedEOF
	}
	return
}

// ReadFull 将r中的len(buf)个字节准确地读取到buf中。
// 返回复制的字节数，如果读取的字节数少则返回错误。
// 仅当未读取任何字节时，错误才是EOF。
// 如果在读取了一些而非全部字节后发生EOF，则ReadFull返回ErrUnexpectedEOF。
// 返回时，当且仅当err == nil时，n == len(buf)。
// 如果r返回读取至少len(buf)个字节的错误，则该错误将被丢弃。
func ReadFull(r Reader, buf []byte) (n int, err error) { //注：调用r.Read至少min字节数据存到buf中，返回读取到的数据长度n与错误err
	return ReadAtLeast(r, buf, len(buf))
}

// CopyN 从src复制d个字节（或直到出错）到dst。
// 返回复制的字节数以及复制时遇到的最早错误。
// 返回时，当且仅当err == nil时written == n。
// 如果dst实现了ReaderFrom接口，则使用该接口实现副本。
func CopyN(dst Writer, src Reader, n int64) (written int64, err error) { //注：从src中读取长度为n的数据写入dst，缓冲区为内部生成，返回拷贝的数据长度written与错误err
	written, err = Copy(dst, LimitReader(src, n))
	if written == n {
		return n, nil
	}
	if written < n && err == nil {
		// src提前停止； 一定是EOF。
		err = EOF
	}
	return
}

// Copy 从src复制到dst，直到src达到EOF或发生错误。
// 它返回复制的字节数和复制时遇到的第一个错误（如果有）。
// 成功的Copy返回err == nil，而不是err == EOF。
// 因为复制被定义为从src读取直到EOF，所以它不会将读取的EOF视为要报告的错误。
// 如果src实现WriterTo接口，则通过调用src.WriteTo(dst)实现该副本。
// 否则，如果dst实现了ReaderFrom接口，则通过调用dst.ReadFrom(src)实现该副本。
func Copy(dst Writer, src Reader) (written int64, err error) { //注：从src中读取数据写入dst，缓冲区为内部生成，返回拷贝的数据长度written与错误err
	return copyBuffer(dst, src, nil)
}

// CopyBuffer 与Copy相同，除了它通过提供的缓冲区（如果需要）而不是分配一个临时缓冲区，来逐步进行。
// 如果buf为nil，则分配一个；否则，为0。 否则，如果长度为零，则CopyBuffer会发生混乱。
// 如果src实现WriterTo或dst实现ReaderFrom，则buf将不用于执行复制。
func CopyBuffer(dst Writer, src Reader, buf []byte) (written int64, err error) { //注：从src中读取数据写入dst，缓冲区为buf，返回拷贝的数据长度written与错误err
	if buf != nil && len(buf) == 0 {
		panic("empty buffer in io.CopyBuffer")
	}
	return copyBuffer(dst, src, buf)
}

// copyBuffer 是Copy和CopyBuffer的实际实现。
// 如果buf为nil，则分配一个。
func copyBuffer(dst Writer, src Reader, buf []byte) (written int64, err error) { //注：从src中读取数据写入dst，缓冲区为buf，返回拷贝的数据长度written与错误err
	//如果src具有WriteTo方法，请使用该方法进行复制。
	//避免分配和复制。
	if wt, ok := src.(WriterTo); ok { //注：尝试执行src.WriterTo自定义写入dst
		return wt.WriteTo(dst)
	}
	//同样，如果编写者具有ReadFrom方法，则使用它来执行复制。
	if rt, ok := dst.(ReaderFrom); ok { //注：尝试执行dst.ReaderFrom自定义读取src
		return rt.ReadFrom(src)
	}

	if buf == nil { //注：缓冲区为空则创建一个缓冲区
		size := 32 * 1024                                           //注：缓冲区默认长度为32*1024
		if l, ok := src.(*LimitedReader); ok && int64(size) > l.N { //注：如果src实现了LimitedReader，则将缓冲区大小设置为l.N
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf = make([]byte, size)
	}

	for {
		nr, er := src.Read(buf) //注：从src读取，长度为nr
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr]) //注：写入dst，长度为nw
			if nw > 0 {                    //注：更新已经拷贝的长度
				written += int64(nw)
			}
			if ew != nil { //注：如果写入错误，跳出循环
				err = ew
				break
			}
			if nr != nw { //注：如果读取长度不等于写入长度，跳出循环
				err = ErrShortWrite
				break
			}
		}
		if er != nil { //注：如果读取错误，跳出循环
			if er != EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

// LimitReader 返回一个从r读取但在n字节后以EOF停止的Reader。
// 基础实现是*LimitedReader。
func LimitReader(r Reader, n int64) Reader { return &LimitedReader{r, n} } //注：工厂函数

// LimitedReader 从R读取但将返回的数据量限制为N个字节。
// 每次对Read的调用都会更新N以反映剩余的新数量。
// 当N <= 0或底层R返回EOF时，Read返回EOF。
type LimitedReader struct { //注：一共要读取N个字节（例:假如N为100，第一次Read读取到10个，N会变为90，第二次读取到5个，N会变为85，需要一直Read直至N变为0）
	R Reader //基础Reader
	N int64  //剩余的最大字节数
}

func (l *LimitedReader) Read(p []byte) (n int, err error) { //注：从l.R读取l.N个数据存到缓冲区p中，返回读取到的数据长度n与错误err
	if l.N <= 0 { //注：缓冲区长度不可以小于0
		return 0, EOF
	}
	if int64(len(p)) > l.N { //注：如果缓冲区p的长度>l.N，改变p的长度为N
		p = p[0:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n) //注：更新l.N
	return
}

// NewSectionReader 返回一个SectionReader，它从r开始读取，偏移量为off，并在n个字节后以EOF停止。
func NewSectionReader(r ReaderAt, off int64, n int64) *SectionReader { //工厂函数
	return &SectionReader{r, off, off, off + n}
}

// SectionReader 在基础ReaderAt的一部分上实现Read，Seek和ReadAt。
type SectionReader struct { //注：读取r的一部分数据，起始位置为base，已经读到的位置为off，截至位置为limit
	// （例：SectionReader{r, 50, 55, 100}，已经从r的第50个字节开始读了5个字节，读取到第100个字节结束）
	r     ReaderAt
	base  int64
	off   int64
	limit int64
}

func (s *SectionReader) Read(p []byte) (n int, err error) {
	if s.off >= s.limit { //注：如果偏移量超过上限，返回EOF
		return 0, EOF
	}
	if max := s.limit - s.off; int64(len(p)) > max { //注：如果偏移量p的长度大于剩余要读取的数据长度(limit - off)，则修剪一下
		p = p[0:max]
	}
	n, err = s.r.ReadAt(p, s.off)
	s.off += int64(n) //注：更新off
	return
}

var errWhence = errors.New("Seek: invalid whence") //注：非法的枚举值
var errOffset = errors.New("Seek: invalid offset") //注：非法的偏移量

// Seek 假设s.base=50，s.offect=60，s.limit=100，当s.Seek(10, x)时
func (s *SectionReader) Seek(offset int64, whence int) (int64, error) { //注：重新定位s.off至s.base与s.limit之间的某个相对位置，可以向前也可以向后
	switch whence {
	default:
		return 0, errWhence
	case SeekStart:
		offset += s.base //注：offset=60
	case SeekCurrent:
		offset += s.off //注：offset=70
	case SeekEnd:
		offset += s.limit //注：offset=110
	}
	if offset < s.base {
		return 0, errOffset
	}
	s.off = offset              //注：o.off=60，70，110
	return offset - s.base, nil //注：60-50=10，70-50=20，110-50=60
}

// ReadAt 从s.r中的off位置读取数据到p中，返回读取到的数据长度n与错误err
func (s *SectionReader) ReadAt(p []byte, off int64) (n int, err error) { //注：修剪缓冲区p后从s.r中off位置读取数据到p中，返回读取到的数据长度n与错误err
	if off < 0 || off >= s.limit-s.base { //注：off超出限界了则返回EOF
		return 0, EOF
	}
	off += s.base
	if max := s.limit - off; int64(len(p)) > max {
		p = p[0:max]
		n, err = s.r.ReadAt(p, off)
		if err == nil {
			err = EOF
		}
		return n, err
	}
	return s.r.ReadAt(p, off)
}

// Size 返回节的大小（以字节为单位）。
func (s *SectionReader) Size() int64 { return s.limit - s.base } //注：返回一共要读取的数据的长度

// TeeReader 返回一个Reader，该Reader向w写入从r读取的内容。
// 通过r执行的所有r读取都与对w的相应写入匹配。
// 没有内部缓冲-写入必须在读取完成之前完成。
// 写入时遇到的任何错误均报告为读取错误。
func TeeReader(r Reader, w Writer) Reader { //工厂函数
	return &teeReader{r, w}
}

type teeReader struct {
	r Reader
	w Writer
}

func (t *teeReader) Read(p []byte) (n int, err error) { //注：从t.r中读取数据存到缓冲区p中，再写入t.w中，返回写入数据的长度n与错误err
	n, err = t.r.Read(p)
	if n > 0 {
		if n, err := t.w.Write(p[:n]); err != nil {
			return n, err
		}
	}
	return
}
