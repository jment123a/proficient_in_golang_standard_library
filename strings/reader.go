// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package strings

import (
	"errors"
	"io"
	"unicode/utf8"
)

// Reader 通过从字符串中读取来实现io.Reader，io.ReaderAt，io.Seeker，io.WriterTo，io.ByteScanner和io.RuneScanner接口。
// Reader的零值类似于空字符串的Reader。
type Reader struct { // 注：提供对于s的各种读取方法
	s        string
	i        int64 // 当前读取到的索引
	prevRune int   // 上一个读取到的rune的索引，或 < 0
}

// Len 返回字符串的未读部分的字节数。
func (r *Reader) Len() int { // 注：获取r未读取的字节数
	if r.i >= int64(len(r.s)) {
		return 0
	}
	return int(int64(len(r.s)) - r.i)
}

// Size 返回基础字符串的原始长度。
// Size是可通过ReadAt读取的字节数。
// 返回的值始终相同，并且不受任何其他方法的调用影响。
func (r *Reader) Size() int64 { return int64(len(r.s)) } // 注：获取r的数据长度

func (r *Reader) Read(b []byte) (n int, err error) { // 注：从r中读取数据到b中
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	r.prevRune = -1
	n = copy(b, r.s[r.i:])
	r.i += int64(n)
	return
}

func (r *Reader) ReadAt(b []byte, off int64) (n int, err error) { // 注：从r偏移off字节处读取数据到b中
	// 无法修改状态-请参阅io.ReaderAt
	if off < 0 {
		return 0, errors.New("strings.Reader.ReadAt: negative offset") // 注：负数偏移量
	}
	if off >= int64(len(r.s)) {
		return 0, io.EOF
	}
	n = copy(b, r.s[off:])
	if n < len(b) {
		err = io.EOF
	}
	return
}

func (r *Reader) ReadByte() (byte, error) { // 注：从r中读取一字节数据
	r.prevRune = -1
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	b := r.s[r.i]
	r.i++
	return b, nil
}

func (r *Reader) UnreadByte() error { // 注：撤回上次ReadByte()
	if r.i <= 0 {
		return errors.New("strings.Reader.UnreadByte: at beginning of string") // 错误："字符串开头"
	}
	r.prevRune = -1
	r.i--
	return nil
}

func (r *Reader) ReadRune() (ch rune, size int, err error) { // 注：从r中读取一个rune
	if r.i >= int64(len(r.s)) {
		r.prevRune = -1
		return 0, 0, io.EOF
	}
	r.prevRune = int(r.i)
	if c := r.s[r.i]; c < utf8.RuneSelf {
		r.i++
		return rune(c), 1, nil
	}
	ch, size = utf8.DecodeRuneInString(r.s[r.i:])
	r.i += int64(size)
	return
}

func (r *Reader) UnreadRune() error { // 注：撤回上次ReadRune()
	if r.i <= 0 {
		return errors.New("strings.Reader.UnreadRune: at beginning of string") // 错误："字符串开头"
	}
	if r.prevRune < 0 {
		return errors.New("strings.Reader.UnreadRune: previous operation was not ReadRune") // 错误："先前的操作不是ReadRune"
	}
	r.i = int64(r.prevRune)
	r.prevRune = -1
	return nil
}

// Seek 实现io.Seeker接口。
func (r *Reader) Seek(offset int64, whence int) (int64, error) { // 注：修改r的索引为相对位置whence偏移offset
	r.prevRune = -1
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.i + offset
	case io.SeekEnd:
		abs = int64(len(r.s)) + offset
	default:
		return 0, errors.New("strings.Reader.Seek: invalid whence") // 错误："无效地点"
	}
	if abs < 0 {
		return 0, errors.New("strings.Reader.Seek: negative position") // 错误："无效索引"
	}
	r.i = abs
	return abs, nil
}

// WriteTo 实现io.WriterTo接口。
func (r *Reader) WriteTo(w io.Writer) (n int64, err error) { // 注：将r的数据写入w中
	r.prevRune = -1
	if r.i >= int64(len(r.s)) {
		return 0, nil
	}
	s := r.s[r.i:]
	m, err := io.WriteString(w, s)
	if m > len(s) {
		panic("strings.Reader.WriteTo: invalid WriteString count") // 恐慌："无效的WriteString计数"
	}
	r.i += int64(m)
	n = int64(m)
	if m != len(s) && err == nil {
		err = io.ErrShortWrite
	}
	return
}

// Reset 将Reader重置为从s读取。
func (r *Reader) Reset(s string) { *r = Reader{s, 0, -1} } // 注：重置r，数据为s

// NewReader 返回从读取的新Reader。
// 它类似于bytes.NewBufferString，但效率更高且只读。
func NewReader(s string) *Reader { return &Reader{s, 0, -1} } // 工厂函数，生成一个数据为s的Reader结构体
