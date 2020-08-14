// 版权所有2010 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package fmt

import (
	"errors"
	"io"
	"math"
	"os"
	"reflect"
	"strconv"
	"sync"
	"unicode/utf8"
)

// ScanState 表示传递给自定义扫描仪的扫描仪状态。
// 扫描程序可以一次扫描符文，或要求ScanState发现下一个以空格分隔的令牌。
type ScanState interface {
	// ReadRune 从输入中读取下一个符文（Unicode代码点）。
	// 如果在Scanln，Fscanln或Sscanln中调用，则ReadRune()将在返回第一个'\n'或读取超出指定宽度的值时返回EOF。
	ReadRune() (r rune, size int, err error)
	// UnreadRune 使对ReadRune的下一次调用返回相同的符文。
	UnreadRune() error
	// SkipSpace 跳过输入中的空格。 换行符适合执行的操作； 有关更多信息，请参见软件包文档。
	SkipSpace()
	// Token 如果skipSpace为true，则跳过输入中的空格，然后返回满足f(c)的Unicode代码点c的运行。
	// 如果f为nil，则使用!unicode.IsSpace(c); 也就是说，Token将包含非空格字符。
	// 换行符适合执行的操作； 有关更多信息，请参见软件包文档。
	// 返回的切片指向共享数据，共享数据可能被下一次对Token的调用，
	// 使用ScanState作为输入的Scan函数的调用或调用的Scan方法返回时所覆盖。
	Token(skipSpace bool, f func(rune) bool) (token []byte, err error)
	// Width 返回width选项的值以及是否已设置。
	// 单位是Unicode代码点。
	Width() (wid int, ok bool)
	// 由于ReadRune由该接口实现，因此扫描程序永远不应调用Read，并且ScanState的有效实现可以选择始终从Read返回错误。
	Read(buf []byte) (n int, err error)
}

// Scanner 由具有Scan方法的任何值实现，该方法将扫描输入以表示值的表示并将结果存储在接收器中，
// 接收器必须是有用的指针。 对于实现该方法的Scan，Scanf或Scanln的任何参数，调用Scan方法。
type Scanner interface {
	Scan(state ScanState, verb rune) error
}

// 注：
// Scan：默认输入源为os.Stdin
// S开头：输入源为字符串
// F开头：输入源为io.Reader

// ln结尾：Scan需要通过换行符\n结尾
// f结尾：需要传入格式化字符串

// 调用链：
// Scan（输入源为os.Stdin）、Sscan（输入源为字符串）——Fscan（）——doScan（）
// Scanln（输入源为os.Stdin）、Sscanln（输入源为字符串）——Fscanln（通过换行符\n结束输入）——doScan（）
// Scanf（输入源为os.Stdin）、Sscanf（输入源为字符串）——Fscanf（使用格式化字符串）——doScanf（）

// Scan 扫描从标准输入读取的文本，并将连续的以空格分隔的值存储到连续的参数中。
// 换行符视为空格。 它返回成功扫描的项目数。
// 如果小于参数个数，则err将报告原因。
func Scan(a ...interface{}) (n int, err error) { //注：从os.Stdin读取数据按空格分隔存储到a中，
	return Fscan(os.Stdin, a...)
}

// Scanln 与"Scan"类似，但是在换行符处停止扫描，并且在最后一项之后必须有换行符或EOF。
func Scanln(a ...interface{}) (n int, err error) { //注：从os.Stdin读取数据按空格分隔存储到a中，数据最后需要有换行符来结束输入
	return Fscanln(os.Stdin, a...)
}

// Scanf 扫描从标准输入读取的文本，将连续的空格分隔的值存储到由格式确定的连续参数中。 它返回成功扫描的项目数。
// 如果小于参数个数，则err将报告原因。
// 输入中的换行符必须与格式中的换行符匹配。
// 一个例外：%c始终扫描输入中的下一个符文，即使它是空格（或制表符等）或换行符也是如此。
func Scanf(format string, a ...interface{}) (n int, err error) { //注：从os.Stdin读取数据按空格分隔根据format格式化后存储到a中
	return Fscanf(os.Stdin, format, a...)
}

type stringReader string

func (r *stringReader) Read(b []byte) (n int, err error) { //注：从r截取b长度的数据存储到b中，返回读取到的数据长度n于错误err
	//注：
	// r=12345
	// b=make([]byte,2)
	// Read一次：r="345"	b=12	n=2
	// Read一次：r="5"		b=34	n=2
	// Read一次：r=""		b=5		n=1
	// Read一次：r=""		b=5		n=0	EOF
	n = copy(b, *r) //注：将r复制到b中
	*r = (*r)[n:]   //注：r截取掉已复制的数据
	if n == 0 {     //注：如果r没有数据拷贝给b，返回EOF
		err = io.EOF
	}
	return
}

// Sscan 扫描参数字符串，将连续的用空格分隔的值存储到连续的参数中。
// 换行符视为空格。 它返回成功扫描的项目数。 如果该数目少于参数数目，则err将报告原因。
func Sscan(str string, a ...interface{}) (n int, err error) { //注：从字符串str读取数据按空格分隔存储到a中，
	return Fscan((*stringReader)(&str), a...)
}

// Sscanln 与Sscan类似，但在换行符处停止扫描，并且在最后一个项目之后必须有换行符或EOF。
func Sscanln(str string, a ...interface{}) (n int, err error) { //注：从字符串str读取数据按空格分隔存储到a中，数据最后需要有换行符来结束输入
	return Fscanln((*stringReader)(&str), a...)
}

// Sscanf 扫描参数字符串，将连续的空格分隔的值存储到由格式确定的连续参数中。 它返回成功解析的项目数。
// 输入中的换行符必须与格式中的换行符匹配。
func Sscanf(str string, format string, a ...interface{}) (n int, err error) { //注：从字符串str读取数据按空格分隔根据format格式化后存储到a中
	return Fscanf((*stringReader)(&str), format, a...)
}

// Fscan 扫描从r读取的文本，将连续的以空格分隔的值存储到连续的参数中。
// 换行符视为空格。 它返回成功扫描的项目数。 如果该数目少于参数数目，则err将报告原因。
func Fscan(r io.Reader, a ...interface{}) (n int, err error) { //注：从r读取数据按空格分隔存储到a中
	s, old := newScanState(r, true, false)
	n, err = s.doScan(a)
	s.free(old)
	return
}

// Fscanln 与Fscan相似，但是在换行符处停止扫描，并且在最后一个项目之后必须有换行符或EOF。
func Fscanln(r io.Reader, a ...interface{}) (n int, err error) { //注：从os.Stdin读取数据按空格分隔存储到a中，数据最后需要有换行符来结束输入
	s, old := newScanState(r, false, true)
	n, err = s.doScan(a)
	s.free(old)
	return
}

// Fscanf 扫描从r读取的文本，将连续的以空格分隔的值存储到由格式确定的连续的参数中。 它返回成功解析的项目数。
// 输入中的换行符必须与格式中的换行符匹配。
func Fscanf(r io.Reader, format string, a ...interface{}) (n int, err error) { //注：从r读取数据按空格分隔根据format格式化后存储到a中
	s, old := newScanState(r, false, false)
	n, err = s.doScanf(format, a)
	s.free(old)
	return
}

// scanError 表示由扫描软件生成的错误。
// 它用作唯一的签名，以在恢复时识别此类错误。
type scanError struct {
	err error
}

const eof = -1 //注：一些方法遇到EOF时的返回值

// ss 是ScanState的内部实现。
type ss struct {
	rs    io.RuneScanner // 在哪里读取输入，注：读取数据的对象
	buf   buffer         // 代币累积器，注：缓冲区，通常在一个函数内使用与清空
	count int            // 到目前为止消耗的verb，注：从format中读取并处理了verb的数量
	atEOF bool           // 已经读过EOF，注：是否之前读取到过EOF
	ssave                //注：#
}

// ssave 包含需要在递归扫描中保存和恢复的ss部分。
type ssave struct {
	validSave bool // 是或曾经是实际ss的一部分。
	nlIsEnd   bool // 换行符是否终止扫描
	nlIsSpace bool // 换行符是否算作空格
	argLimit  int  // 此arg的ss.count的最大值； argLimit <= limit
	limit     int  // ss.count的最大值。
	maxWid    int  // 此arg的宽度，例：%1s，注：verb设置的宽度，否则为默认值hugeWid（1073741824）
}

// Read方法仅在ScanState中，以便ScanState满足io.Reader。 如果按预期使用它，则永远不会调用它，因此无需使其真正起作用。
func (s *ss) Read(buf []byte) (n int, err error) { //注：仅为满足io.Reader接口，永远不会被调用
	return 0, errors.New("ScanState's Read should not be called. Use ReadRune") //错误："不应调用ScanState的Read。 使用ReadRune"
}

func (s *ss) ReadRune() (r rune, size int, err error) { //注：从s.rs中读取下一个rune，返回rune r，rune的长度size和错误err
	if s.atEOF || s.count >= s.argLimit { //注：如果读取到EOF或操作数已经全部赋值
		err = io.EOF //注：返回错误EOF
		return
	}

	r, size, err = s.rs.ReadRune() //注：执行ReadRune，读取rune和大小
	if err == nil {
		s.count++                   //注：消耗一个verb
		if s.nlIsEnd && r == '\n' { //注：如果遇到\n，且\n等于结束扫描，返回EOF
			s.atEOF = true
		}
	} else if err == io.EOF {
		s.atEOF = true
	}
	return
}

func (s *ss) Width() (wid int, ok bool) { //注：例：fmt.Scan("%1s", a)，判断s的宽度为最大值则宽度为0，返回宽度wid与是否合理ok
	if s.maxWid == hugeWid {
		return 0, false
	}
	return s.maxWid, true
}

// public方法返回错误； 这种私人的恐慌。
// 如果getRune达到EOF，则返回值为EOF(-1)。
func (s *ss) getRune() (r rune) { //注：从s.rs中读取并返回下一个rune r
	r, _, err := s.ReadRune() //注：从s.rs中读取下一个rune
	if err != nil {
		if err == io.EOF {
			return eof //注：如果读取到错误EOF，返回-1
		}
		s.error(err) //注：否则产生恐慌err
	}
	return
}

// mustReadRune 将io.EOF变成紧急情况（io.ErrUnexpectedEOF）。
// 在诸如EOF是语法错误的字符串扫描之类的情况下被调用。
func (s *ss) mustReadRune() (r rune) { //注：返回下一个Rune，否则恐慌
	r = s.getRune() //注：获取一个rune
	if r == eof {
		s.error(io.ErrUnexpectedEOF) //注：遇到EOF引发恐慌
	}
	return
}

func (s *ss) UnreadRune() error { //注：使下一次对ReadRune的调用返回与上一次对ReadRune的调用相同的rune
	s.rs.UnreadRune() //注：使上一次ReadRune没发生过
	s.atEOF = false
	s.count--
	return nil
}

func (s *ss) error(err error) { //注：产生恐慌，内容为err
	panic(scanError{err}) //注：产生恐慌
}

func (s *ss) errorString(err string) { //注：产生恐慌，内容为err
	panic(scanError{errors.New(err)})
}

func (s *ss) Token(skipSpace bool, f func(rune) bool) (tok []byte, err error) { //注：s读取字符串skipSpace设置是否跳过空格，f判断每个字符是否有效，直至EOF或f为false，返回读取到的数据tok与错误err
	defer func() {
		if e := recover(); e != nil { //注：捕获恐慌
			if se, ok := e.(scanError); ok {
				err = se.err
			} else {
				panic(e)
			}
		}
	}()
	if f == nil { //注：f默认为要求每个字符不为空格
		f = notSpace
	}
	s.buf = s.buf[:0]           //注：清空buf
	tok = s.token(skipSpace, f) //注：获取字符串
	return
}

// space 是unicode.White_Space范围的副本，以避免依赖于软件包unicode。
var space = [][2]uint16{ //注：各种空格的范围
	{0x0009, 0x000d},
	{0x0020, 0x0020},
	{0x0085, 0x0085},
	{0x00a0, 0x00a0},
	{0x1680, 0x1680},
	{0x2000, 0x200a},
	{0x2028, 0x2029},
	{0x202f, 0x202f},
	{0x205f, 0x205f},
	{0x3000, 0x3000},
}

func isSpace(r rune) bool { //注：返回r是否是空格
	if r >= 1<<16 { //注：r超过65536
		return false
	}
	rx := uint16(r)
	for _, rng := range space {
		if rx < rng[0] {
			return false
		}
		if rx <= rng[1] {
			return true //注：space[x][0] < r && space[x][1] <= r
		}
	}
	return false
}

// notSpace 是token中使用的默认扫描功能。
func notSpace(r rune) bool { //注：返回r是否不是空格，经常将此函数用于判断读取的数据是否合法
	return !isSpace(r)
}

// readRune 是一种用于从io.Reader读取UTF-8编码的代码点的结构。 如果提供给扫描仪的阅读器尚未实现io.RuneScanner，则使用它。
type readRune struct {
	reader   io.Reader         // 注：数据读取对象
	buf      [utf8.UTFMax]byte // 仅在ReadRune内部使用
	pending  int               // pendBuf中的字节数； 对于错误的UTF-8，仅 > 0，注：reader中未读取的字节数
	pendBuf  [utf8.UTFMax]byte // 剩余字节，注：reader中未读取的数据对象
	peekRune rune              // 如果 >=0 为下一个rune; 如果 <0 为 ^(上一个Rune)，默认 < 0，执行UnreadRune撤销后 >0，再执行ReadRune时优先读取peekRune
}

// readByte 从输入返回下一个字节，如果UTF-8格式不正确，则可以从上一次读取中保留下来。
func (r *readRune) readByte() (b byte, err error) { //注：从pendBuf中获取一个字节并返回，如果pending <= 0则从reader中读取1个字节
	if r.pending > 0 { //注：还有未读取的数据，读取1个字节的数据
		b = r.pendBuf[0]
		copy(r.pendBuf[0:], r.pendBuf[1:]) //注：删除pendBuf[0]
		r.pending--
		return
	}
	n, err := io.ReadFull(r.reader, r.pendBuf[:1]) //注：从reader中读取1个字节到pendBuf
	if n != 1 {
		return 0, err
	}
	return r.pendBuf[0], err
}

// ReadRune 从r中的io.Reader返回下一个UTF-8编码的代码点。
func (r *readRune) ReadRune() (rr rune, size int, err error) { //注：从peekRune与reader中读取一个rune，返回rune rr，rune的大小size和错误err
	if r.peekRune >= 0 { //注：peekRune为下一个rune
		rr = r.peekRune
		r.peekRune = ^r.peekRune //注：peekRune为^rune
		size = utf8.RuneLen(rr)  //注：rune的长度
		return
	}
	r.buf[0], err = r.readByte() //注：获取一个字节
	if err != nil {
		return
	}
	if r.buf[0] < utf8.RuneSelf { //快速检查常见ASCII大小写，注：这个字节是单字节rune，peekRune为^rune
		rr = rune(r.buf[0])
		size = 1 // 已知为1。
		// 翻转rune的各个位，以供UnreadRune使用。
		r.peekRune = ^rr
		return
	}
	var n int
	for n = 1; !utf8.FullRune(r.buf[:n]); n++ { //注：循环读取一个字节，直到组成一个rune
		r.buf[n], err = r.readByte()
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return
		}
	}
	rr, size = utf8.DecodeRune(r.buf[:n]) //注：转码为rune
	if size < n {                         //错误，保存字节供下次读取，注:如果转码之后变短了，则扩展pending与pendBuf
		copy(r.pendBuf[r.pending:], r.buf[size:n])
		r.pending += n - size
	}
	// 翻转符文的各个位，以供UnreadRune使用。
	r.peekRune = ^rr //注：peekRune为^rune
	return
}

func (r *readRune) UnreadRune() error { //注：撤销上次ReadRune读取
	if r.peekRune >= 0 {
		return errors.New("fmt: scanning called UnreadRune with no rune available") //恐慌："没有可用rune的扫描称为UnreadRune"
	}
	// 先前读取的rune的反向位翻转以获得有效的 >= 0状态。
	r.peekRune = ^r.peekRune
	return nil
}

var ssFree = sync.Pool{ //注：ss的对象缓冲池
	New: func() interface{} { return new(ss) },
}

// newScanState 分配一个新的ss结构或获取一个缓存的结构。
func newScanState(r io.Reader, nlIsSpace, nlIsEnd bool) (s *ss, old ssave) {
	//注：工厂方法，创建一个ss结构体，将r作为ss的输入对象，nlIsSpace是否作为空格，nlIsEnd是否作为结束，返回ss对象s与空对象old

	s = ssFree.Get().(*ss)
	if rs, ok := r.(io.RuneScanner); ok { //注：如果r实现了io.RuneScanner，赋值为ss的输入对象
		s.rs = rs
	} else {
		s.rs = &readRune{reader: r, peekRune: -1} //注：否则创建一个
	}
	s.nlIsSpace = nlIsSpace //注：一些初始化
	s.nlIsEnd = nlIsEnd
	s.atEOF = false
	s.limit = hugeWid
	s.argLimit = hugeWid
	s.maxWid = hugeWid
	s.validSave = true
	s.count = 0
	return
}

// free 将使用过的ss结构保存在ssFree中； 避免每次调用分配。
func (s *ss) free(old ssave) { //注：释放ss
	// 如果以递归方式使用它，则只需恢复旧状态即可。
	if old.validSave {
		s.ssave = old
		return
	}
	// 不要使用大缓冲区的ss结构。
	if cap(s.buf) > 1024 {
		return
	}
	s.buf = s.buf[:0]
	s.rs = nil
	ssFree.Put(s)
}

// SkipSpace 使Scan方法能够跳过空格和换行符，以与格式字符串和Scan/Scanln设置的当前扫描模式保持一致。
func (s *ss) SkipSpace() { //注：跳过空格（\r\n与\n）
	for {
		r := s.getRune() //注：读取下一个rune
		if r == eof {
			return
		}
		if r == '\r' && s.peek("\n") { //注：读取到的是否为\r\n
			continue
		}
		if r == '\n' { //注：如果读取到\n
			if s.nlIsSpace { //注：如果\n算作空格
				continue
			}
			s.errorString("unexpected newline") //恐慌："意外的换行符"
			return
		}
		if !isSpace(r) { //注：如果r不是空格
			s.UnreadRune() //注：撤销这次getRune
			break
		}
	}
}

// token 返回输入中的下一个以空格分隔的字符串。 它跳过空白。 对于Scanln，它在换行符处停止。 对于扫描，换行符被视为空格。
func (s *ss) token(skipSpace bool, f func(rune) bool) []byte { //注：根据skipSpace是否跳过空格，读取rune直至EOF或f返回false
	if skipSpace {
		s.SkipSpace() //注：跳过空格
	}
	// 读取直到空白或换行符
	for {
		r := s.getRune() //注：获取一个rune
		if r == eof {    //注：遇到EOF则返回
			break
		}
		if !f(r) { //注：f返回false则撤销
			s.UnreadRune()
			break
		}
		s.buf.writeRune(r) //注：写入r
	}
	return s.buf
}

var complexError = errors.New("syntax error scanning complex number") //注：复杂类型语法错误扫描
var boolError = errors.New("syntax error scanning boolean")           //注：布尔类型语法错误扫描

func indexRune(s string, r rune) int { //注：返回s中r所在的位置
	for i, c := range s { //注：遍历s，如果包含r则返回s的索引，否则返回-1
		if c == r {
			return i
		}
	}
	return -1
}

// consume 读取输入中的下一个rune，并报告其是否在ok字符串中。
// 如果accept为true，则将字符放入输入Token中。
func (s *ss) consume(ok string, accept bool) bool { //注：s读取到的下一个rune是否在ok中并返回，根据accept是否写入s的缓冲区中
	r := s.getRune() //注：获取下一个rune
	if r == eof {
		return false
	}
	if indexRune(ok, r) >= 0 { //注：如果ok中包括这个rune
		if accept {
			s.buf.writeRune(r) //注：向缓冲区写入r
		}
		return true
	}
	if r != eof && accept {
		s.UnreadRune() //注：撤销getRune
	}
	return false
}

// peek 报告下一个字符是否在ok字符串中，而不消耗它。
func (s *ss) peek(ok string) bool { //注：返回ok中是否包含r
	r := s.getRune() //注：获取下一个rune
	if r != eof {    //注：rune不是eof
		s.UnreadRune() //注：撤销这次getRune
	}
	return indexRune(ok, r) >= 0 //注：ok中是否包含r
}

func (s *ss) notEOF() { //注：下一个rune不是EOF
	// 确保有要读取的数据。
	if r := s.getRune(); r == eof { //注：如果下一个rune是EOF则恐慌
		panic(io.EOF)
	}
	s.UnreadRune() //注：撤销getRune
}

// accept 检查输入中的下一个rune。 如果它是字符串中的字节（sic），则将其放入缓冲区并返回true。 否则返回false。
func (s *ss) accept(ok string) bool { //注：返回s的下一个rune是否在ok中，如果在ok中则写入s的缓冲区
	return s.consume(ok, true)
}

// okVerb 验证verb是否存在于列表中，如果不存在则适当设置s.err。
func (s *ss) okVerb(verb rune, okVerbs, typ string) bool { //注：返回verb是否在okVerbs中，如果不是则恐慌，恐慌类型为typ
	for _, v := range okVerbs { //注：遍历有效verb
		if v == verb { //注：如果verb有效返回true
			return true
		}
	}
	s.errorString("bad verb '%" + string(verb) + "' for " + typ) //恐慌："错误的verb %verb 于 类型"
	return false
}

// scanBool 返回下一个标记表示的布尔值。
func (s *ss) scanBool(verb rune) bool { //注：根据下一个rune，返回其布尔类型
	s.SkipSpace()                         //注：跳过空格
	s.notEOF()                            //注：下一个rune不是EOF
	if !s.okVerb(verb, "tv", "boolean") { //注：verb是否为%t或%v
		return false
	}
	// 语法检查布尔值很烦人。 我们对case并不精打细算。
	switch s.getRune() { //注：获取一个rune，接受（1，0）、不区分大小写的（true，的false）
	case '0':
		return false
	case '1':
		return true
	case 't', 'T':
		if s.accept("rR") && (!s.accept("uU") || !s.accept("eE")) { //注：如果连续4个rune不是true（不区分大小写），产生恐慌
			s.error(boolError)
		}
		return true
	case 'f', 'F':
		if s.accept("aA") && (!s.accept("lL") || !s.accept("sS") || !s.accept("eE")) { //注：如果连续4个rune不是false（不区分大小写），产生恐慌
			s.error(boolError)
		}
		return false
	}
	return false
}

// 数值元素
// 注：scan接受的float各种格式字符
const (
	binaryDigits      = "01"                     //注：2进制数字字序
	octalDigits       = "01234567"               //注：8进制数字字序
	decimalDigits     = "0123456789"             //注：10进制数字字序
	hexadecimalDigits = "0123456789aAbBcCdDeEfF" //注：16进制数字字序
	sign              = "+-"                     //注：正负符号
	period            = "."                      //注：小数点
	exponent          = "eEpP"                   //注：科学计数法，例：1e2，1p2
)

// getBase 返回由verb及其数字字符串表示的数字进制。
func (s *ss) getBase(verb rune) (base int, digits string) { //注：返回v对应的进制与数字字序
	s.okVerb(verb, "bdoUxXv", "integer") // 设置s.err，注：verb是否为bdoUxXv其中之一
	base = 10
	digits = decimalDigits
	switch verb {
	case 'b': //注：2进制
		base = 2
		digits = binaryDigits
	case 'o': //注：8进制
		base = 8
		digits = octalDigits
	case 'x', 'X', 'U': //注：16进制
		base = 16
		digits = hexadecimalDigits
	}
	return
}

// scanNumber 返回从此处开始具有指定数字的数字字符串。
func (s *ss) scanNumber(digits string, haveDigits bool) string { //注：根据!haveDigits判断digits是否包含下一个rune，如果不包含则恐慌，读取所有rune并非返回
	if !haveDigits {
		s.notEOF()             //注：下一个rune不是EOF
		if !s.accept(digits) { //注：如果digits不包含括下一个字符
			s.errorString("expected integer") //注：预期整数
		}
	}
	for s.accept(digits) { //注：读取所有rune
	}
	return string(s.buf)
}

// scanRune 返回输入中的下一个rune。
func (s *ss) scanRune(bitSize int) int64 { //注：读取下一个位数为bitSize的rune
	s.notEOF()                       //注：下一个rune不是EOF
	r := int64(s.getRune())          //注：获取下一个rune
	n := uint(bitSize)               //注：获取位数
	x := (r << (64 - n)) >> (64 - n) //注：(r << (64 - n))去掉超界的位，>> (64 - n)还原至原来的位数
	if x != r {
		s.errorString("overflow on character value " + string(r)) //注：字符值溢出
	}
	return r
}

// scanBasePrefix 报告整数是否以进制前缀开头并返回进制前缀，数字字符串以及是否找到零。
// 仅当verb为%v时才被调用。
func (s *ss) scanBasePrefix() (base int, digits string, zeroFound bool) { //注：根据下两个rune，判断是否为进制前缀，返回进制base（无效），数字字序digits与是否获取到0 zeroFound
	if !s.peek("0") { //注：下一个rune是否不包含0
		return 0, decimalDigits + "_", false
	}
	s.accept("0")
	// 0、0b，0o，0x的特殊情况。
	switch {
	case s.peek("bB"):
		s.consume("bB", true)
		return 0, binaryDigits + "_", true
	case s.peek("oO"):
		s.consume("oO", true)
		return 0, octalDigits + "_", true
	case s.peek("xX"):
		s.consume("xX", true)
		return 0, hexadecimalDigits + "_", true
	default:
		return 0, octalDigits + "_", true
	}
}

// scanInt 返回下一个标记表示的整数值，检查是否溢出。 任何错误都存储在s.err中。
func (s *ss) scanInt(verb rune, bitSize int) int64 { //注：读取最长符合verb对应字序的数据，转为bitSize位数并返回
	if verb == 'c' { //注：%c为获取ASCII对应字符，直接返回rune
		return s.scanRune(bitSize)
	}
	s.SkipSpace()                   //注：跳过空格
	s.notEOF()                      //注：不是EOF
	base, digits := s.getBase(verb) //注：获取verb对应的位数
	haveDigits := false
	if verb == 'U' { //注：获取Unicode，例：U+007B
		if !s.consume("U", false) || !s.consume("+", false) {
			s.errorString("bad unicode format ") //恐慌："错误的unicode格式"
		}
	} else {
		s.accept(sign) // 如果有符号，它将留在令牌缓冲区中。注：获取+-号
		if verb == 'v' {
			base, digits, haveDigits = s.scanBasePrefix() //注：获取数字位数，字序，是否为数字
		}
	}
	tok := s.scanNumber(digits, haveDigits)   //注：获取所有符合digits字序的数据
	i, err := strconv.ParseInt(tok, base, 64) //注： #转为数字
	if err != nil {
		s.error(err)
	}
	n := uint(bitSize)
	x := (i << (64 - n)) >> (64 - n) //注：转为对应bitSize位数
	if x != i {
		s.errorString("integer overflow on token " + tok) //注：令牌上的整数溢出
	}
	return i
}

// scanUint 返回下一个标记表示的无符号整数的值，并检查是否溢出。 任何错误都存储在s.err中。
func (s *ss) scanUint(verb rune, bitSize int) uint64 { //注：同上，读取最长符合verb对应字序的数据，转为bitSize位数并返回
	if verb == 'c' { //注：%c为获取ASCII对应字符，直接返回rune
		return uint64(s.scanRune(bitSize)) //注：直接
	}
	s.SkipSpace()                   //注：跳过空格
	s.notEOF()                      //注：不是EOF
	base, digits := s.getBase(verb) //注：获取verb对应的位数
	haveDigits := false
	if verb == 'U' { //注：获取Unicode，例：U+007B
		if !s.consume("U", false) || !s.consume("+", false) {
			s.errorString("bad unicode format ") //恐慌："错误的unicode格式"
		}
	} else if verb == 'v' {
		base, digits, haveDigits = s.scanBasePrefix() //注：获取数字位数，字序，是否为数字
	}
	tok := s.scanNumber(digits, haveDigits)    //注：获取所有符合digits字序的数据
	i, err := strconv.ParseUint(tok, base, 64) //注： #转为数字
	if err != nil {
		s.error(err)
	}
	n := uint(bitSize)
	x := (i << (64 - n)) >> (64 - n) //注：转为对应bitSize位数
	if x != i {
		s.errorString("unsigned integer overflow on token " + tok) //恐慌："令牌上的无符号整数溢出"
	}
	return i
}

// floatToken 返回从此处开始的浮点数，如果指定了宽度，则不超过swid。
// 它对语法并不严格，因为它不会检查我们是否至少有一些数字，但是Atof会做到这一点。
func (s *ss) floatToken() string { //注：读取下一个float类型字符串（格式包括nan、+-inf、+-16进制浮点数、+-10进制浮点数）
	s.buf = s.buf[:0]
	// NaN?
	if s.accept("nN") && s.accept("aA") && s.accept("nN") { //注：读取连续3个字节是否为NaN
		return string(s.buf) //注：返回NaN
	}

	// 前缀？
	s.accept(sign) //注：是否为+-号
	// Inf?
	if s.accept("iI") && s.accept("nN") && s.accept("fF") { //注：是否为inf
		return string(s.buf)
	}

	digits := decimalDigits + "_"
	exp := exponent
	if s.accept("0") && s.accept("xX") { //注：是否为0x
		digits = hexadecimalDigits + "_"
		exp = "pP"
	}
	// 数字?
	for s.accept(digits) { //注：遍历直至不符合字序要求
	}
	// 小数点？
	if s.accept(period) { //注：是否为小数点
		// 小数？
		for s.accept(digits) { //注：是否为小数点后的小数
		}
	}
	// 指数?
	if s.accept(exp) { //注：是否为科学计数法e或p
		// 前缀？
		s.accept(sign) //注：是否为+-号
		// 数字?
		for s.accept(decimalDigits + "_") { //注：是否为数字
		}
	}
	return string(s.buf) //注：返回组成的float字符串
}

// complexTokens 返回从此处开始的复数的实部和虚部。
// 该数字可能带有括号并具有（N + Ni）格式，其中N是浮点数，并且其中没有空格。
func (s *ss) complexTokens() (real, imag string) { //注：读取下一个复杂类型，返回实数与虚数
	// 注：
	// 当(12-34i)
	// 输入：complex(12.0, -34.0)

	// TODO：独立接受N和Ni？
	parens := s.accept("(") //注：下一个rune为(
	real = s.floatToken()   //注：获取下一个浮点数
	s.buf = s.buf[:0]       //注：清空buf
	// 现在必须有一个标志。
	if !s.accept("+-") { //注：如果下一个不是+-，
		s.error(complexError) //注：恐慌
	}
	// 签名现在在缓冲区中
	imagSign := string(s.buf) //注：转为字符串，(12-
	imag = s.floatToken()     //注：读取下一个浮点数
	if !s.accept("i") {       //注：下一个rune为i
		s.error(complexError)
	}
	if parens && !s.accept(")") { //注：下一个rune为)
		s.error(complexError)
	}
	return real, imagSign + imag //注：返回12.0, (12-34i)
}

func hasX(s string) bool { //注：返回s中是否存在x或X
	for i := 0; i < len(s); i++ {
		if s[i] == 'x' || s[i] == 'X' {
			return true
		}
	}
	return false
}

// convertFloat 将字符串转换为float64值。
func (s *ss) convertFloat(str string, n int) float64 { //注：将字符串str转为n（32、64）位浮点数并返回
	// strconv.ParseFloat将处理"+0x1.fp+2"，但我们必须自己实现非标准的十进制+二进制指数混合(1.2p4)。
	if p := indexRune(str, 'p'); p >= 0 && !hasX(str) { //注：如果srt中存在p并且不存在x
		// Atof不处理2的幂的指数，但易于评估。
		f, err := strconv.ParseFloat(str[:p], n) //注：#将整数转为n位浮点数
		if err != nil {
			//将完整的字符串放入错误中。
			if e, ok := err.(*strconv.NumError); ok {
				e.Num = str
			}
			s.error(err) //注：恐慌
		}
		m, err := strconv.Atoi(str[p+1:]) //注：将小数转为n位浮点数
		if err != nil {
			//将完整的字符串放入错误中。
			if e, ok := err.(*strconv.NumError); ok {
				e.Num = str
			}
			s.error(err)
		}
		return math.Ldexp(f, m)
	}
	f, err := strconv.ParseFloat(str, n) //注：直接转换为n位浮点数
	if err != nil {
		s.error(err)
	}
	return f
}

// convertComplex 将下一个标记转换为complex128值。
// atof参数是基础类型的特定于类型的读取器。
// 如果我们正在读取complex64，则atof将解析float32并将其转换为float64，以避免为每种复杂类型重现此代码。
func (s *ss) scanComplex(verb rune, n int) complex128 { //注：获取下一个n位复杂类型，根据verb格式化，
	if !s.okVerb(verb, floatVerbs, "complex") { //注：如果下一个verb不符合floatVerbs要求，返回0
		return 0
	}
	s.SkipSpace()                      //注：跳过空格
	s.notEOF()                         //注：不为EOF
	sreal, simag := s.complexTokens()  //注：获取下一个复杂类型
	real := s.convertFloat(sreal, n/2) //注：将实数转为float，如果复杂类型位数为128，实数位数则为64，如果为64，则为32
	imag := s.convertFloat(simag, n/2)
	return complex(real, imag)
}

// convertString 返回下一个输入字符表示的字符串。
// 输入的格式由verb决定。
func (s *ss) convertString(verb rune) (str string) { //注：根据verb获取对应格式的字符串str并返回
	if !s.okVerb(verb, "svqxX", "string") { //注：字符串接受这些verb
		return ""
	}
	s.SkipSpace() //注：跳过空格
	s.notEOF()    //注：下一个不是EOF
	switch verb {
	case 'q':
		str = s.quotedString() //注：返回下一个双引号或单引号包裹的字符串
	case 'x', 'X':
		str = s.hexString() //注：获取16进制字符串
	default:
		str = string(s.token(true, notSpace)) // %s 和 %v 只要返回下一个不是空格的rune
	}
	return
}

// quotedString 返回由下一个输入字符表示的双引号或反引号字符串。
func (s *ss) quotedString() string { //注：返回下一个双引号或反引号中的字符串
	s.notEOF()           //注：不是EOF
	quote := s.getRune() //注：获取一个rune
	switch quote {
	case '`':
		// 反引号：直到EOF或反引号为止。
		for {
			r := s.mustReadRune() //注：一直读取Rune直到读取到另一半反引号
			if r == quote {
				break
			}
			s.buf.writeRune(r)
		}
		return string(s.buf) //注：返回两个`之间的字符串
	case '"':
		// 双引号：包括引号并让strconv.Unquote进行反斜杠转义。
		s.buf.writeByte('"')
		for {
			r := s.mustReadRune()
			s.buf.writeRune(r)
			if r == '\\' { //注：略过反斜杠则
				// 在合法的反斜杠转义中，无论多长时间，只有转义后的字符本身都可以是反斜杠或引号。
				// 因此，我们只需要保护反斜杠后的第一个字符即可。
				s.buf.writeRune(s.mustReadRune())
			} else if r == '"' { //注：一直读取Rune直到读取到另一半双引号
				break
			}
		}
		result, err := strconv.Unquote(string(s.buf)) //注：转为字符串
		if err != nil {
			s.error(err)
		}
		return result
	default:
		s.errorString("expected quoted string") //注：预期的带引号的字符串
	}
	return ""
}

// hexDigit 返回十六进制数字的值。
func hexDigit(d rune) (int, bool) { //注：返回d是否为16进制数字
	digit := int(d)
	switch digit {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return digit - '0', true
	case 'a', 'b', 'c', 'd', 'e', 'f':
		return 10 + digit - 'a', true
	case 'A', 'B', 'C', 'D', 'E', 'F':
		return 10 + digit - 'A', true
	}
	return -1, false
}

// hexByte 返回输入中的下一个十六进制编码（两个字符）的字节。
// 如果输入中的下一个字节未编码十六进制字节，则返回ok == false。
// 如果第一个字节为十六进制，第二个字节为十六进制，则处理停止。
func (s *ss) hexByte() (b byte, ok bool) { //注：获取连续两个16进制rune组合为byte，返回byte b与是否成功
	rune1 := s.getRune() //注：获取一个rune
	if rune1 == eof {    //注：rune不是EOF
		return
	}
	value1, ok := hexDigit(rune1) //注：转为16进制数字
	if !ok {
		s.UnreadRune() //注：如果不是则撤销
		return
	}
	value2, ok := hexDigit(s.mustReadRune()) //注：再或一个rune转为16进制数字
	if !ok {
		s.errorString("illegal hex digit") //注：错误：非法的十六进制数字
		return
	}
	return byte(value1<<4 | value2), true //注：第一个rune作为前4位，第二个rune作为后4位
}

// hexString 返回以空格分隔的十六进制对编码的字符串。
func (s *ss) hexString() string { //注：以两个连续的16进制rune为1个byte，获取连续的byte组成的字符串
	s.notEOF() //注：下一个rune不是EOF
	for {
		b, ok := s.hexByte() //注：获取连续两个16进制数字
		if !ok {
			break
		}
		s.buf.writeByte(b)
	}
	if len(s.buf) == 0 {
		s.errorString("no hex data for %x string") //注：%x字符串无16进制数据
		return ""
	}
	return string(s.buf)
}

const (
	floatVerbs = "beEfFgGv" //注：合法的float verb

	hugeWid = 1 << 30 //注：宽度的最大值

	intBits     = 32 << (^uint(0) >> 63)    //注：int的位数，与操作系统位数相同
	uintptrBits = 32 << (^uintptr(0) >> 63) //注：uintptr的位数，与操作系统位数相同
)

// scanPercent 扫描文字百分比字符。
func (s *ss) scanPercent() { //注：获取下一个rune是%
	s.SkipSpace()       //注：跳过空格
	s.notEOF()          //注：确保下一个rune不是EOF
	if !s.accept("%") { //注：如果下一个rune不是%，引发恐慌
		s.errorString("missing literal %") //注：缺少%
	}
}

// scanOne 扫描一个值，从参数的类型派生扫描程序。
func (s *ss) scanOne(verb rune, arg interface{}) { //注：读取下一个arg类型的数据，根据verb格式化后存储到arg中
	s.buf = s.buf[:0] //注：清空缓冲区
	var err error
	// 如果参数具有自己的Scan方法，请使用该方法。
	if v, ok := arg.(Scanner); ok {
		err = v.Scan(s, verb) //注：如果参数实现了Scanner接口，执行Scan方法
		if err != nil {
			if err == io.EOF { //注：如果读取到了文件结尾
				err = io.ErrUnexpectedEOF
			}
			s.error(err) //注：产生恐慌
		}
		return
	}

	switch v := arg.(type) {
	case *bool:
		*v = s.scanBool(verb)
	case *complex64:
		*v = complex64(s.scanComplex(verb, 64))
	case *complex128:
		*v = s.scanComplex(verb, 128)
	case *int:
		*v = int(s.scanInt(verb, intBits))
	case *int8:
		*v = int8(s.scanInt(verb, 8))
	case *int16:
		*v = int16(s.scanInt(verb, 16))
	case *int32:
		*v = int32(s.scanInt(verb, 32))
	case *int64:
		*v = s.scanInt(verb, 64)
	case *uint:
		*v = uint(s.scanUint(verb, intBits))
	case *uint8:
		*v = uint8(s.scanUint(verb, 8))
	case *uint16:
		*v = uint16(s.scanUint(verb, 16))
	case *uint32:
		*v = uint32(s.scanUint(verb, 32))
	case *uint64:
		*v = s.scanUint(verb, 64)
	case *uintptr:
		*v = uintptr(s.scanUint(verb, uintptrBits))
	// 浮点数很棘手，因为您想以结果的精度进行扫描，而不是以高精度进行扫描并进行转换，以便保留正确的错误条件。
	case *float32:
		if s.okVerb(verb, floatVerbs, "float32") { //注：如果verb合法
			s.SkipSpace()                                    //注：跳过空格
			s.notEOF()                                       //注：下一个rune不是EOF
			*v = float32(s.convertFloat(s.floatToken(), 32)) //注：获取下一个float，转为float32
		}
	case *float64:
		if s.okVerb(verb, floatVerbs, "float64") { //注：如果verb合法
			s.SkipSpace()                           //注：跳过空格
			s.notEOF()                              //注：下一个rune不是EOF
			*v = s.convertFloat(s.floatToken(), 64) //注：获取下一个float，转为float64
		}
	case *string:
		*v = s.convertString(verb)
	case *[]byte:
		// 我们扫描到字符串并进行转换，以便获得数据的副本。
		// 如果我们扫描到字节，则切片将指向缓冲区。
		*v = []byte(s.convertString(verb))
	default: //注：其他类型，通过反射赋值
		val := reflect.ValueOf(v)
		ptr := val
		if ptr.Kind() != reflect.Ptr {
			s.errorString("type not a pointer: " + val.Type().String()) //注：不是一个指针
			return
		}
		switch v := ptr.Elem(); v.Kind() {
		case reflect.Bool:
			v.SetBool(s.scanBool(verb))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v.SetInt(s.scanInt(verb, v.Type().Bits()))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			v.SetUint(s.scanUint(verb, v.Type().Bits()))
		case reflect.String:
			v.SetString(s.convertString(verb))
		case reflect.Slice:
			// 目前，只能处理（重命名）[]byte。
			typ := v.Type()
			if typ.Elem().Kind() != reflect.Uint8 { //注：必须为[]byte切片
				s.errorString("can't scan type: " + val.Type().String())
			}
			str := s.convertString(verb)
			v.Set(reflect.MakeSlice(typ, len(str), len(str))) //注：创建一个类型为type，长度与空间为str的切片
			for i := 0; i < len(str); i++ {
				v.Index(i).SetUint(uint64(str[i])) //注：将v复制给切片
			}
		case reflect.Float32, reflect.Float64:
			s.SkipSpace()                                               //注：跳过空格
			s.notEOF()                                                  //注：下一个rune不是EOF
			v.SetFloat(s.convertFloat(s.floatToken(), v.Type().Bits())) //注：获取下一个浮点数赋值
		case reflect.Complex64, reflect.Complex128:
			v.SetComplex(s.scanComplex(verb, v.Type().Bits()))
		default:
			s.errorString("can't scan type: " + val.Type().String())
		}
	}
}

// errorHandler 将本地恐慌变成错误返回。
func errorHandler(errp *error) { //注：捕获恐慌，将实现scanError接口与EOF恐慌变为错误
	if e := recover(); e != nil {
		if se, ok := e.(scanError); ok { // 捕获本地错误
			*errp = se.err
		} else if eof, ok := e.(error); ok && eof == io.EOF { // 输入不足
			*errp = eof
		} else {
			panic(e)
		}
	}
}

// doScan 在没有格式字符串的情况下进行真正的扫描工作。
func (s *ss) doScan(a []interface{}) (numProcessed int, err error) { //注：遍历a，根据操作数读取对应类型的数据存储到操作数中，返回已处理的操作数数量numProcessed与错误err
	defer errorHandler(&err) //注：捕获恐慌
	for _, arg := range a {  //注：遍历操作数组
		s.scanOne('v', arg) //注：读取arg类型的数据格式化为默认类型存储到arg中
		numProcessed++
	}
	// 如果需要，请检查换行符（或EOF）（Scanln等）。
	if s.nlIsEnd { //注：如果换行符可以终止扫描
		for {
			r := s.getRune()           //注：获取一个rune
			if r == '\n' || r == eof { //注：换行或EOF则结束
				break
			}
			if !isSpace(r) { //注：如果r不是空格
				s.errorString("expected newline") //恐慌："预期换行符"
				break
			}
		}
	}
	return
}

// advance 确定输入中的下一个字符是否与该格式的字符匹配。 它返回格式消耗的字节数（sic）。
// 输入或格式的所有空格字符行都表现为单个空格。 换行符是特殊的：
// 格式中的换行符必须与输入中的换行符匹配，反之亦然。
// 此例程还处理%%情况。 如果返回值为零，则两种格式都以%开头（后面没有%）或输入为空。 如果为负，则输入与字符串不匹配。
func (s *ss) advance(format string) (i int) { //注：跳过format与reader中的换行符与空格，直到遇到verb，返回verb出现的位置i
	for i < len(format) { //注：遍历format
		fmtc, w := utf8.DecodeRuneInString(format[i:]) //注：#

		// 空格处理。
		// 在此注释的其余部分，"空格"表示换行符以外的空格。
		// 格式的换行符匹配零个或多个空格的输入，然后匹配换行符或输入的结尾。
		// 将换行符之前的格式的空格折叠到换行符中。
		// 换行符后的格式空格与对应的输入换行符后的零个或多个空格匹配。
		// 格式中的其他空格与一个或多个空格的输入或输入结尾匹配。

		//注：如果format："\n\n..."，则s.rs="\n\n..."
		if isSpace(fmtc) { //注：是否是空格
			newlines := 0
			trailingSpace := false
			for isSpace(fmtc) && i < len(format) { //注：遍历format直到不是空格，得出换行符的数量newlines
				if fmtc == '\n' { //注：如果为换行符
					newlines++            //注：计数+1
					trailingSpace = false //注：format最后一个字符是换行符
				} else {
					trailingSpace = true //注：format最后一个字符不是换行符
				}
				i += w
				fmtc, w = utf8.DecodeRuneInString(format[i:])
			}
			for j := 0; j < newlines; j++ { //注：遍历newlines，获取rune直到遇到newlins个换行符或EOF
				inputc := s.getRune()                   //注：获取一个rune
				for isSpace(inputc) && inputc != '\n' { //注：如果rune是空格且不是换行符，继续获取
					inputc = s.getRune()
				}
				if inputc != '\n' && inputc != eof { //注：如果rune不是换行符且不是EOF
					s.errorString("newline in format does not match input") //恐慌："格式的换行符与输入不匹配"
				}
			}

			if trailingSpace { //注：format最后一个字符不是换行符，再遍历出一个换行符
				inputc := s.getRune()
				if newlines == 0 {
					// 如果尾随的空格是单独的（未跟随换行符），则必须至少找到一个要使用的空格。
					if !isSpace(inputc) && inputc != eof { //注：如果不是空格且不是EOF
						s.errorString("expected space in input to match format") //恐慌："输入中的预期空间以匹配格式"
					}
					if inputc == '\n' { //注：如果是换行符
						s.errorString("newline in input does not match format") //恐慌："输入的换行符与格式不匹配"
					}
				}
				for isSpace(inputc) && inputc != '\n' { //注：遍历到换行符或其他非空格rune
					inputc = s.getRune()
				}
				if inputc != eof { //注：如果读取到EOF则撤回
					s.UnreadRune()
				}
			}
			continue
		}

		// Verbs
		if fmtc == '%' { //注：format出现verb则返回%出现的位置
			// 字符串末尾的%是错误。
			if i+w == len(format) {
				s.errorString("missing verb: % at end of format string") //恐慌："verb丢失：格式字符串末尾的%"
			}
			// %%的行为像真实的百分比
			nextc, _ := utf8.DecodeRuneInString(format[i+w:]) // 如果字符串为空，将不匹配%
			if nextc != '%' {
				return
			}
			i += w // 跳过第一个%
		}

		// 文字
		inputc := s.mustReadRune() //注：不是空格，不是verb，则可能是常量
		if fmtc != inputc {        //注：如果format与s.rs的常量不同，撤回操作
			s.UnreadRune()
			return -1
		}
		i += w
	}
	return
}

// doScanf 使用格式字符串进行扫描时，确实可以完成工作。
// 目前，它仅处理指向基本类型的指针。
func (s *ss) doScanf(format string, a []interface{}) (numProcessed int, err error) { //注：遍历format格式化s.reader将数据写入a中，返回处理过的操作数数量numProcessed与错误err
	defer errorHandler(&err)
	end := len(format) - 1
	// 我们会以非平凡的格式处理一项
	for i := 0; i <= end; { //注：遍历format
		w := s.advance(format[i:])
		if w > 0 { //注：搜索到了verb位置，跳转
			i += w
			continue
		}

		// 要么我们无法前进，要么是字符百分号，要么是输入用完了。
		if format[i] != '%' { //注：format中找不到verb
			// 无法推进格式。 为什么不？
			if w < 0 {
				s.errorString("input does not match format") //恐慌："输入的格式不匹配"
			}
			// 否则在EOF； 下面处理"操作数过多"错误
			break
		}
		i++ // % 是一个字节

		// 我们有20个（宽度）吗？
		var widPresent bool                                //注：例：%1s，获取宽度1
		s.maxWid, widPresent, i = parsenum(format, i, end) //注：获取format[i: end]中第一个最长的数字
		if !widPresent {                                   //注：如果没要求宽度，则宽度最大值默认为1073741824
			s.maxWid = hugeWid
		}

		c, w := utf8.DecodeRuneInString(format[i:]) //注：例：%1s，获取verb s
		i += w

		if c != 'c' { //注：不是ASCII，跳过空格
			s.SkipSpace()
		}
		if c == '%' { //注：如果format为%%，则s.rs需要读取一个%
			s.scanPercent()
			continue // 不要消耗参数。
		}
		s.argLimit = s.limit
		if f := s.count + s.maxWid; f < s.argLimit { //注：第count个verb+操作数的宽度如果小于最大参数
			s.argLimit = f
		}

		if numProcessed >= len(a) { // 操作数不足
			s.errorString("too few operands for format '%" + format[i-w:] + "'") //恐慌："verb格式的操作数太少"
			break
		}
		arg := a[numProcessed] //注：取出第numProcessed个操作数

		s.scanOne(c, arg) //注：将输入根据verb c格式化后存储到操作数中
		numProcessed++    //注：处理过的verb增加
		s.argLimit = s.limit
	}
	if numProcessed < len(a) {
		s.errorString("too many operands") //注：操作数太多
	}
	return
}
