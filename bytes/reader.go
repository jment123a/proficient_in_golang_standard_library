// 版权所有2012 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package bytes

import (
	"errors"
	"io"
	"unicode/utf8"
)

// Reader 通过读取字节片来实现io.Reader，io.ReaderAt，io.WriterTo，io.Seeker，io.ByteScanner和io.RuneScanner接口。
// 与Buffer不同，Reader是只读的，并支持查找。
// Reader的零值的作用类似于空切片的Reader。
// 注：s[0:i] = 已经读取的数据，s[prevRune:i] = 上一个rune的数据，s[i:len(s)] = 未读取的数据
// 注：len(s) - i = 未读取的字节数
type Reader struct {
	s        []byte // 注：数据
	i        int64  // 当前读取的索引
	prevRune int    // 前一个rune的索引； 或者 < 0
}

// Len 返回切片的未读部分的字节数。
func (r *Reader) Len() int { // 注：获取r的未读取字节数
	if r.i >= int64(len(r.s)) { // 注：全部读取完毕，返回0
		return 0
	}
	return int(int64(len(r.s)) - r.i)
}

// Size 返回基础字节片的原始长度。
// Size是可通过ReadAt读取的字节数。
// 返回的值始终相同，并且不受任何其他方法的调用影响。
func (r *Reader) Size() int64 { return int64(len(r.s)) } // 注：获取r的数据大小

// Read 实现io.Reader接口。
func (r *Reader) Read(b []byte) (n int, err error) { // 注：从r中读取数据到b中，返回读取到的数据长度n与错误err
	if r.i >= int64(len(r.s)) { // 注：如果没有未读取的字节数，返回EOF
		return 0, io.EOF
	}
	r.prevRune = -1        // 注：上一次读取的不是rune
	n = copy(b, r.s[r.i:]) // 注：读取数据
	r.i += int64(n)        // 注：增加偏移量
	return
}

// ReadAt 实现io.ReaderAt接口。
func (r *Reader) ReadAt(b []byte, off int64) (n int, err error) { // 注：从r中偏移量为off的位置读取数据到b中，返回数据到的数据长度n与错误err
	// 无法修改状态-请参阅io.ReaderAt
	if off < 0 {
		return 0, errors.New("bytes.Reader.ReadAt: negative offset") // 恐慌："负数偏移量"
	}
	if off >= int64(len(r.s)) { // 注：如果偏移量超过了数据的长度，返回EOF
		return 0, io.EOF
	}
	n = copy(b, r.s[off:])
	if n < len(b) {
		err = io.EOF
	}
	return
}

// ReadByte 实现io.ByteReader接口。
func (r *Reader) ReadByte() (byte, error) { // 注：从r中读取一个字节并返回
	r.prevRune = -1
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	b := r.s[r.i]
	r.i++
	return b, nil
}

// UnreadByte 在实现io.ByteScanner接口时对ReadByte进行了补充。
func (r *Reader) UnreadByte() error { // 注：撤回上一次ReadByte()
	if r.i <= 0 {
		return errors.New("bytes.Reader.UnreadByte: at beginning of slice") // 恐慌："在切片的开始"
	}
	r.prevRune = -1
	r.i--
	return nil
}

// ReadRune 实现io.RuneReader接口。
func (r *Reader) ReadRune() (ch rune, size int, err error) { // 注：从r中读取一个rune，返回rune，rune的长度与错误err
	if r.i >= int64(len(r.s)) { // 注：没有未读取的数据，返回EOF
		r.prevRune = -1
		return 0, 0, io.EOF
	}
	r.prevRune = int(r.i) // 注：读取一个rune
	if c := r.s[r.i]; c < utf8.RuneSelf {
		r.i++
		return rune(c), 1, nil
	}
	ch, size = utf8.DecodeRune(r.s[r.i:])
	r.i += int64(size)
	return
}

// UnreadRune 在实现io.RuneScanner接口方面对ReadRune进行了补充。
func (r *Reader) UnreadRune() error { // 注：撤回上一次ReadRune()
	if r.i <= 0 {
		return errors.New("bytes.Reader.UnreadRune: at beginning of slice") // 恐慌："在切片的开始"
	}
	if r.prevRune < 0 {
		return errors.New("bytes.Reader.UnreadRune: previous operation was not ReadRune") // 恐慌："先前的操作不是ReadRune"
	}
	r.i = int64(r.prevRune)
	r.prevRune = -1
	return nil
}

// Seek 实现io.Seeker接口。
func (r *Reader) Seek(offset int64, whence int) (int64, error) { // 注：根据whence设置r未读取数据的索引，偏移量为ieoffset
	r.prevRune = -1
	var abs int64
	switch whence {
	case io.SeekStart: // 注：设置索引为offset
		abs = offset
	case io.SeekCurrent: // 注：设置索引为当前偏移 + offset
		abs = r.i + offset
	case io.SeekEnd: // 注：设置索引为r.s的结尾 + offset
		abs = int64(len(r.s)) + offset
	default:
		return 0, errors.New("bytes.Reader.Seek: invalid whence") // 恐慌："无效地点"
	}
	if abs < 0 {
		return 0, errors.New("bytes.Reader.Seek: negative position") // 恐慌："负数位置"
	}
	r.i = abs
	return abs, nil
}

// WriteTo 实现io.WriterTo接口。
func (r *Reader) WriteTo(w io.Writer) (n int64, err error) { // 注：将r中未读取的数据写入w，返回写入数据的长度n与错误err
	r.prevRune = -1
	if r.i >= int64(len(r.s)) { // 注：没有未读取的数据
		return 0, nil
	}
	b := r.s[r.i:]
	m, err := w.Write(b) // 注：向w写入r中为读取的数据
	if m > len(b) {
		panic("bytes.Reader.WriteTo: invalid Write count") // 恐慌："无效的写计数"
	}
	r.i += int64(m)
	n = int64(m)
	if m != len(b) && err == nil {
		err = io.ErrShortWrite
	}
	return
}

// Reset 将reader重置为从b中读取。
func (r *Reader) Reset(b []byte) { *r = Reader{b, 0, -1} } // 注：重置r，数据为b

// NewReader 从b返回一个新的Reader读数。
func NewReader(b []byte) *Reader { return &Reader{b, 0, -1} } // 工厂函数，创建一个reader结构体，数据为b
