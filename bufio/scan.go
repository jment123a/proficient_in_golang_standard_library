// 版权所有2013 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package bufio

import (
	"bytes"
	"errors"
	"io"
	"unicode/utf8"
)

// Scanner 提供了一个方便的界面来读取数据，例如用换行符分隔的文本行的文件。
// 连续调用Scan方法将逐步浏览文件的"令牌"，跳过令牌之间的字节。
// 令牌的规范是由SplitFunc类型的split函数定义的；
// 默认的split功能将输入分成几行，剥去了行终端。
// 在此软件包中定义了拆分功能，用于将文件扫描成行，字节，UTF-8编码的符文和空格分隔的单词。
// 客户端可以改为提供自定义拆分功能。
//
// 扫描在EOF，第一个I/O错误或令牌太大而无法容纳在缓冲区中时停止恢复。
// 当扫描停止时，reader可能会任意超越最后一个令牌。
// 需要对错误处理或较大令牌进行更多控制的程序，或必须在reader上运行顺序扫描的程序，应改用bufio.Reader。
type Scanner struct {
	r            io.Reader // 客户端提供的reader。
	split        SplitFunc // 拆分令牌的函数。
	maxTokenSize int       // 令牌的最大大小； 通过测试修改。
	token        []byte    // 拆分返回的最后一个token。注：#
	buf          []byte    // 缓冲区用作拆分参数。
	start        int       // buf中的第一个未处理字节。
	end          int       // buf中的数据结束。
	err          error     // 粘性错误。
	empties      int       // 连续空令牌的计数。
	scanCalled   bool      // 扫描已被调用； 缓冲区正在使用中。
	done         bool      // 扫描已完成。
}

// SplitFunc 是用于对输入进行标记化的split函数的签名。
// 参数是剩余的未处理数据的初始子字符串，以及标志atEOF，该标志报告Reader是否没有更多数据可提供。
// 返回值是推进输入的字节数，以及返回给用户的下一个令牌（如果有）以及错误（如果有）。
//
// 如果函数返回错误，扫描将停止，在这种情况下，某些输入可能会被丢弃。
//
// 否则，扫描仪将前进输入。如果令牌不为零，则扫描程序会将其返回给用户。
// 如果令牌为零，则扫描程序会读取更多数据并继续扫描；
// 如果没有更多数据-如果atEOF为true-扫描程序将返回。
// 如果数据还没有完整的令牌，例如，如果在扫描行时没有换行符，则SplitFunc可以返回（0，nil，nil），
// 以指示扫描程序将更多数据读入切片，然后尝试更长的时间再试一次切片从输入中的同一点开始。
//
// 除非atEOF为true，否则永远不要使用空数据片调用该函数。
// 但是，如果atEOF为true，则数据可能是非空的，并且像往常一样包含未处理的文本。
type SplitFunc func(data []byte, atEOF bool) (advance int, token []byte, err error)

// Scanner返回的错误。
var (
	ErrTooLong         = errors.New("bufio.Scanner: token too long")                               // 错误："令牌太长"
	ErrNegativeAdvance = errors.New("bufio.Scanner: SplitFunc returns negative advance count")     // 错误："SplitFunc返回负提前计数"
	ErrAdvanceTooFar   = errors.New("bufio.Scanner: SplitFunc returns advance count beyond input") // 错误："SplitFunc返回超出输入的提前计数"
)

const (
	// MaxScanTokenSize 是用于缓冲令牌的最大大小，除非用户使用Scanner.Buffer提供显式缓冲区。
	//由于缓冲区可能需要包含例如换行符，因此实际最大令牌大小可能会更小。
	MaxScanTokenSize = 64 * 1024

	startBufSize = 4096 // 缓冲区的初始分配大小。
)

// NewScanner 返回一个新的Scanner以从r读取。
// 拆分功能默认为ScanLines。
func NewScanner(r io.Reader) *Scanner { // 工厂函数，用于生成Scanner结构体
	return &Scanner{
		r:            r,
		split:        ScanLines,
		maxTokenSize: MaxScanTokenSize,
	}
}

// Err 返回扫描程序遇到的第一个非EOF错误。
func (s *Scanner) Err() error { // 注：获取s的非EOF错误
	if s.err == io.EOF {
		return nil
	}
	return s.err
}

// Bytes 返回通过调用Scan生成的最新令牌。
// 基础数组可能指向将被后续Scan调用覆盖的数据。 它不分配。
func (s *Scanner) Bytes() []byte { // 注：获取s的令牌
	return s.token
}

// Text 返回调用Scan生成的最新令牌，作为新分配的包含其字节的字符串。
func (s *Scanner) Text() string { // 注：获取s的令牌的字符串表现形式
	return string(s.token)
}

// ErrFinalToken 是特殊的前哨错误值。 它打算由Split函数返回，以指示带有错误的传递的令牌是最后一个令牌，扫描应在此之后停止。
// 扫描收到ErrFinalToken后，扫描将停止且没有错误。
// 该值对于尽早停止处理或在有必要交付最终空令牌时很有用。 可以通过自定义错误值实现相同的行为，但在此处提供一个更整洁的方法。
// 有关此值的用法，请参见emptyFinalToken示例。
var ErrFinalToken = errors.New("final token") // 错误："最终令牌"

// Scan 将扫描程序前进到下一个令牌，然后可以通过Bytes或Text方法使用该令牌。 当扫描停止时（到达输入末尾或出现错误），它返回false。
// 在Scan返回false之后，Err方法将返回扫描期间发生的任何错误，但如果是io.EOF，则Err将返回nil。
// 如果split函数在不提前输入的情况下返回了太多的空令牌，请扫描恐慌。 这是扫描仪的常见错误模式。
func (s *Scanner) Scan() bool { // 注：#获取从s中获取令牌
	// 注：
	// 1. 如果缓冲区中有数据，检查缓冲区中是否存在令牌
	// 2. 如果缓冲区没有数据，缓冲区中不存在令牌，清理缓冲区中的已处理数据，如果缓冲区已满，扩容缓冲区，从reader中读取数据到缓冲区

	if s.done { // 注：#如果scan已经完成，返回false
		return false
	}
	s.scanCalled = true // 注：正在进行scan，缓冲区正在使用
	// 循环直到我们有了令牌。
	for {
		// 看看是否可以使用已有的令牌获取令牌。
		// 如果数据用完了但是有错误，请给split函数一个机会，以恢复所有剩余的，可能为空的令牌。
		if s.end > s.start || s.err != nil { // 注：如果缓冲区中存在数据，拆分缓冲区，检查是否有令牌
			advance, token, err := s.split(s.buf[s.start:s.end], s.err != nil) // 注：获取缓冲区拆分的令牌数
			if err != nil {
				if err == ErrFinalToken { // 注：如果是最终令牌，返回true
					s.token = token
					s.done = true
					return true
				}
				s.setErr(err) // 注：记录错误，返回false
				return false
			}
			if !s.advance(advance) { // 注：如果s无法前进advance个字节，返回false
				return false
			}
			s.token = token   // 注：赋值令牌
			if token != nil { // 注：如果令牌为空
				if s.err == nil || advance > 0 { // 注：#如果不是空令牌
					s.empties = 0
				} else { // 注：如果是空令牌
					// 返回的令牌在EOF处未提前输入。
					s.empties++
					if s.empties > maxConsecutiveEmptyReads { // 注：如果执行次数超过了容忍次数，引发恐慌
						panic("bufio.Scan: too many empty tokens without progressing") // 恐慌："太多的空令牌没有进展"
					}
				}
				return true
			}
		}
		// 我们无法使用持有的内容生成令牌。
		// 如果我们已经遇到EOF或I/O错误，那么就完成了。
		if s.err != nil { // 注：如果s出现错误，返回false
			// 关掉它。
			s.start = 0
			s.end = 0
			return false
		}
		// 必须读取更多数据。
		// 首先，如果需要大量空白空间，则将数据移至缓冲区的开头。
		if s.start > 0 && (s.end == len(s.buf) || s.start > len(s.buf)/2) { // 注：如果已处理数据占用缓冲区过大，清空已处理数据
			copy(s.buf, s.buf[s.start:s.end])
			s.end -= s.start
			s.start = 0
		}
		// 缓冲区是否已满？ 如果是这样，请调整大小。
		if s.end == len(s.buf) { // 注：如果缓冲区装满了，对缓冲区进行扩容
			// 确保下面的乘法没有溢出。
			const maxInt = int(^uint(0) >> 1)                          // 注：uint表示的最大值
			if len(s.buf) >= s.maxTokenSize || len(s.buf) > maxInt/2 { // 注：如果缓冲区空间过大，返回错误
				s.setErr(ErrTooLong)
				return false
			}
			newSize := len(s.buf) * 2
			if newSize == 0 {
				newSize = startBufSize
			}
			if newSize > s.maxTokenSize {
				newSize = s.maxTokenSize
			}
			newBuf := make([]byte, newSize) // 注：扩容缓冲区
			copy(newBuf, s.buf[s.start:s.end])
			s.buf = newBuf
			s.end -= s.start
			s.start = 0
		}
		// 最后，我们可以读取一些输入。
		// 确保我们不会被行为不端的读者所困扰。
		// 正式而言，我们不需要这样做，但是请格外小心：Scanner用于安全，简单的工作。
		for loop := 0; ; { // 注：自旋，从reader中获取数据
			n, err := s.r.Read(s.buf[s.end:len(s.buf)]) // 注：从reader中读取数据到缓冲区
			s.end += n
			if err != nil {
				s.setErr(err)
				break
			}
			if n > 0 { // 注：如果读取到数据，跳出循环
				s.empties = 0
				break
			}
			loop++
			if loop > maxConsecutiveEmptyReads { // 注：如果循环次数超过了容忍值，返回错误
				s.setErr(io.ErrNoProgress)
				break
			}
		}
	}
}

// advance 消耗n个字节的缓冲区。 它报告前进是否合法。.
func (s *Scanner) advance(n int) bool { // 注：跳过n字节的未处理数据
	if n < 0 { // 注：n必须大于0
		s.setErr(ErrNegativeAdvance) // 错误："SplitFunc返回负提前计数"
		return false
	}
	if n > s.end-s.start { // 注：如果n大于缓冲区剩余未处理数据大小，返回错误
		s.setErr(ErrAdvanceTooFar) // 错误："SplitFunc返回超出输入的提前计数"
		return false
	}
	s.start += n
	return true
}

// setErr 记录遇到的第一个错误。
func (s *Scanner) setErr(err error) { // 注：记录s遇到的第一个错误
	if s.err == nil || s.err == io.EOF {
		s.err = err
	}
}

// Buffer 设置扫描时要使用的初始缓冲区以及扫描过程中可能分配的最大缓冲区大小。
// 最大令牌大小是max和cap(buf)中的较大者。
// 如果max <= cap(buf)，Scan将仅使用此缓冲区，并且不进行分配。
//
// 默认情况下，Scan使用内部缓冲区并将最大令牌大小设置为MaxScanTokenSize。
//
// 如果在扫描开始后调用了panic，则将其缓冲。
func (s *Scanner) Buffer(buf []byte, max int) { // 注：设置s的缓冲区大小与令牌大小
	if s.scanCalled {
		panic("Buffer called after Scan") // 恐慌："Scan后调用的Buffer"
	}
	s.buf = buf[0:cap(buf)]
	s.maxTokenSize = max
}

// Split 设置Scanner的分割功能。
// 默认的拆分功能是ScanLines。
//
// 如果在扫描开始后调用了panic，则将其拆分。
func (s *Scanner) Split(split SplitFunc) { // 注：设置s的拆分令牌函数
	if s.scanCalled {
		panic("Split called after Scan") // 恐慌："Scan后调用Split"
	}
	s.split = split
}

// 分割功能

// ScanBytes 是Scanner的拆分功能，它返回每个字节作为令牌。
func ScanBytes(data []byte, atEOF bool) (advance int, token []byte, err error) { // 注：#
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	return 1, data[0:1], nil
}

var errorRune = []byte(string(utf8.RuneError)) // 注：错误rune，U+FFFD

// ScanRunes 是Scanner的拆分功能，它会将每个UTF-8编码的rune作为令牌返回。
// 返回的rune序列与输入中的范围循环中的字符串相同，这意味着错误的UTF-8编码转换为U+FFFD = "\xef\xbf\xbd".
// 由于具有Scan接口，因此客户端无法将正确编码的替换符文与编码错误区分开。
func ScanRunes(data []byte, atEOF bool) (advance int, token []byte, err error) { // 注：获取data中的第一个rune
	if atEOF && len(data) == 0 { // 注：如果data为空，返回0
		return 0, nil, nil
	}

	// 快速路径1： ASCII.
	if data[0] < utf8.RuneSelf { // 注：如果data是单字节rune，返回rune
		return 1, data[0:1], nil
	}

	// 快速路径2：正确的UTF-8解码没有错误。
	_, width := utf8.DecodeRune(data)
	if width > 1 { // 注：如果data是多字节rune，返回rune
		// 这是有效的编码。 对于正确编码的非ASCII符文，宽度不能为1。
		return width, data[0:width], nil
	}

	// 我们知道这是一个错误：我们有width == 1且隐含了r == utf8.RuneError。
	// 错误是因为没有完整的rune需要解码吗？
	// FullRune可以正确区分错误编码和不完整编码。
	if !atEOF && !utf8.FullRune(data) { // 注：如果data不是一个完整的rune，返回0
		// 不完整； 获得更多字节。
		return 0, nil, nil
	}

	// 我们有一个真正的UTF-8编码错误。
	// 返回正确编码的错误rune，但仅提前一个字节。
	// 这与编码错误的字符串上的范围循环的行为相匹配。
	return 1, errorRune, nil // 注：如果data是U+FFFD（错误rune），返回错误rune
}

// dropCR 从数据中删除一个终端 \r。
func dropCR(data []byte) []byte { // 注：如果data的最后一个字节是\r，则截取
	if len(data) > 0 && data[len(data)-1] == '\r' { // 注：如果data的最后一个字节为\r，截取
		return data[0 : len(data)-1]
	}
	return data
}

// ScanLines 是Scanner的拆分功能，它返回文本的每一行，并删除所有行尾标记。
// 返回的行可能为空。
// 行尾标记是一个可选的回车符，后跟一个强制换行符。
// 在正则表达式中，它是`\r?\n`.
// 即使没有换行符，也将返回输入的最后一个非空行。
func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) { // 注：如果data中出现\n，返回\n出现之前的数据，否则返回data本身
	if atEOF && len(data) == 0 { // 注：如果data为空，返回空
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 { // 注：如果data中出现了\n
		// 我们有一个完整的以换行符结尾的行。
		return i + 1, dropCR(data[0:i]), nil // 注：返回\n之前的data
	}
	// 如果我们是在EOF上，那么我们有一条最终的，不终止的生产线。 把它返还。
	if atEOF { // 注：如果data中没有出现\n，返回data
		return len(data), dropCR(data), nil
	}
	// 请求更多数据。
	return 0, nil, nil // 注：返回nil
}

// isSpace 报告该字符是否为Unicode空格字符。
// 我们避免依赖unicode包，但在测试中检查实现的有效性。
func isSpace(r rune) bool { // 注：获取r是否为空格
	if r <= '\u00FF' { // 注：
		// 明显的ASCII码：\t到\r加上空格。 再加上两个Latin-1怪胎
		switch r {
		case ' ', '\t', '\n', '\v', '\f', '\r': // 注：如果是ASCII换行符，返回true
			return true
		case '\u0085', '\u00A0': // 注：Latin-1换行符
			return true
		}
		return false
	}
	// 高价值的。
	if '\u2000' <= r && r <= '\u200a' { // 注：标点符号
		return true
	}
	switch r {
	case '\u1680', '\u2028', '\u2029', '\u202f', '\u205f', '\u3000': // 注：其他比字符集
		return true
	}
	return false
}

// ScanWords 是扫描仪的拆分功能，它返回每个空格分隔的文本单词，并删除周围的空格。
// 它永远不会返回空字符串。 空间的定义由unicode.IsSpace设置。
func ScanWords(data []byte, atEOF bool) (advance int, token []byte, err error) { // 注：获取跳过空格前缀之后，再次遇到空格之前的data
	// 例：data = "   123   321"，返回123
	// 跳过前导空格。
	start := 0
	for width := 0; start < len(data); start += width { // 注：截取data的空格前缀
		var r rune
		r, width = utf8.DecodeRune(data[start:]) // 注：获取data的第一个rune
		if !isSpace(r) {                         // 注：如果不是空格，跳出循环
			break
		}
	}
	// 扫描至空格，标记单词的结尾。
	for width, i := 0, start; i < len(data); i += width { // 注：遍历data
		var r rune
		r, width = utf8.DecodeRune(data[i:]) // 注：获取data的第一个rune
		if isSpace(r) {                      // 注：如果是空格，返回之前的数据
			return i + width, data[start:i], nil
		}
	}
	// 如果我们是在EOF，我们会有一个最终的，非空的，不终止的单词。 把它返还。
	if atEOF && len(data) > start { // 注：如果data中没有空格，返回data
		return len(data), data[start:], nil
	}
	// 请求更多数据。
	return start, nil, nil
}
