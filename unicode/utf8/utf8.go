//版权所有2009 The Go Authors。 版权所有。
//此源代码的使用受BSD样式的约束
//可以在LICENSE文件中找到的许可证。

// Package utf8 实现了函数和常量以支持以UTF-8编码的文本。 它包括在rune和UTF-8字节序列之间转换的功能。
// 参见https://en.wikipedia.org/wiki/UTF-8
package utf8

// 在测试中验证了条件RuneError == unicode.ReplacementChar和MaxRune == unicode.MaxRune。
// 在本地定义它们可以避免此软件包，具体取决于软件包的unicode。

// 编码基本的数字。
// 注：rune共有4种类型：1字节rune，2字节rune，3字节rune，4字节rune
// 1字节rune：常用的是ASCII
const (
	RuneError = '\uFFFD'     // 错误rune或"Unicode替换字符"，注：解析错误的rune用RuneError代替
	RuneSelf  = 0x80         // RuneSelf下的字符在一个字节中表示为自己。注：小于RuneSelf的rune均为单字节rune
	MaxRune   = '\U0010FFFF' // 最大有效Unicode代码点。注：大于MaxRune的rune均为无效字符
	UTFMax    = 4            // UTF-8编码的Unicode字符的最大字节数。注：UTF-8编码最大占用4字节
)

// 替代范围内的代码点对UTF-8无效。
const (
	surrogateMin = 0xD800 // 注：替代范围，在此范围内的rune均为无效rune
	surrogateMax = 0xDFFF // 注：0 <= rune < surrogateMin < rune < surrogateMax <= MaxRune，在此范围内的rune为有效rune
)

const (
	t1 = 0b00000000
	tx = 0b10000000
	t2 = 0b11000000
	t3 = 0b11100000
	t4 = 0b11110000
	t5 = 0b11111000

	// 注：
	// 1字节rune：
	// 例：
	// 2字节rune：rune(p0&mask2)<<6 | rune(b1&maskx)
	// 最大值：000 000000 011111 111111
	// 3字节rune：rune(p0&mask3)<<12 | rune(b1&maskx)<<6 | rune(b2&maskx)
	// 最大值：000 001111 111111 111111
	// 4字节rune：rune(s0&mask4)<<18 | rune(s1&maskx)<<12 | rune(s2&maskx)<<6 | rune(s3&maskx)
	// 最大值：111 111111 111111 111111
	maskx = 0b00111111 // 注：rune的其余字节的掩码
	mask2 = 0b00011111 // 注：rune字节长度为2时，第一个字节的掩码
	mask3 = 0b00001111 // 注：rune字节长度为3时，第一个字节的掩码
	mask4 = 0b00000111 // 注：rune字节长度为4时，第一个字节的掩码

	rune1Max = 1<<7 - 1  // 注：1字节rune的最大值
	rune2Max = 1<<11 - 1 // 注：2字节rune的最大值
	rune3Max = 1<<16 - 1 // 注：3字节rune的最大值

	// 默认的最低和最高连续字节。
	locb = 0b10000000 // 注：rune的第3个字节的下限
	hicb = 0b10111111 // 注：rune的第3个字节的上限

	// 选择这些常量的这些名称以在下表中提供良好的对齐方式。
	// 第一个半字节是特殊的单字节情况下acceptRanges或F的索引。
	// 第二个半字节是rune长度或特殊一字节大小写的状态。
	xx = 0xF1 // 无效: rune占用1字节
	as = 0xF0 // ASCII: rune占用1字节
	s1 = 0x02 // 接受范围索引为0, rune占用2字节
	s2 = 0x13 // 接受范围索引为1, rune占用3字节
	s3 = 0x03 // 接受范围索引为0, rune占用3字节
	s4 = 0x23 // 接受范围索引为2, rune占用3字节
	s5 = 0x34 // 接受范围索引为3, rune占用4字节
	s6 = 0x04 // 接受范围索引为0, rune占用4字节
	s7 = 0x44 // 接受范围索引为4, rune占用4字节
)

// first 是有关UTF-8序列中第一个字节的信息。
// 注：2、3、4字节rune的第1个字节记录了整个rune的信息，包括取值范围与rune的字节数
var first = [256]uint8{ // 注：rune的第1个字节通过first可以得到该rune的取值范围与字节数
	//   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x00-0x0F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x10-0x1F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x20-0x2F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x30-0x3F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x40-0x4F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x50-0x5F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x60-0x6F
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x70-0x7F
	//   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, // 0x80-0x8F
	xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, // 0x90-0x9F
	xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, // 0xA0-0xAF
	xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, // 0xB0-0xBF
	xx, xx, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, // 0xC0-0xCF
	s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, // 0xD0-0xDF
	s2, s3, s3, s3, s3, s3, s3, s3, s3, s3, s3, s3, s3, s4, s3, s3, // 0xE0-0xEF
	s5, s6, s6, s6, s7, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, // 0xF0-0xFF
}

// acceptRange 给出UTF-8序列中第二个字节的有效值范围。
type acceptRange struct { // 注：rune第二个字节的接受范围的结构体，包括上限hi与下限lo
	lo uint8 // 第二个字节的最小值。
	hi uint8 // 第二个字节的最大值。
}

// acceptRanges的大小为16，以避免在使用它的代码中进行边界检查。
var acceptRanges = [16]acceptRange{ // 注：rune第二个字节的接受范围，根据接受范围索引获取每个rune字节的不同接受范围
	0: {locb, hicb},
	1: {0xA0, hicb},
	2: {locb, 0x9F},
	3: {0x90, hicb},
	4: {locb, 0x8F},
}

// FullRune 报告p中的字节是否以rune的完整UTF-8编码开头。
// 无效的编码被视为完整的rune，因为它将转换为宽度为1的错误rune。
func FullRune(p []byte) bool { // 注：获取p中是否有一个完整的rune
	n := len(p)
	if n == 0 { // 注：p是空的，直接返回false
		return false
	}
	x := first[p[0]]   // 注：获取第一个rune的信息
	if n >= int(x&7) { // 注：获取后3位，记录rune的长度，如果p可以存放rune的长度，返回true
		return true // ASCII，无效或有效。
	}
	// 必须简短或无效。
	// 注：当为rune为3字节或以上时，既检查第2个字节也检查第3个字节
	accept := acceptRanges[x>>4]                         // 注：获取第一个rune的接受范围
	if n > 1 && (p[1] < accept.lo || accept.hi < p[1]) { // 注：如果rune长度 > 1，检查第2个字节是否合法
		return true
	} else if n > 2 && (p[2] < locb || hicb < p[2]) { // 注：如果rune长度 > 1，检查第3个字节是否合法
		return true
	}
	return false
}

// FullRuneInString 类似于FullRune，但其输入是字符串。
func FullRuneInString(s string) bool { // 注：获取p中是否有一个完整的rune（同FullRune）
	n := len(s)
	if n == 0 { // 注：p是空的，直接返回false
		return false
	}
	x := first[s[0]]
	if n >= int(x&7) {
		return true // ASCII, 无效或有效。
	}
	// 必须简短或无效。
	accept := acceptRanges[x>>4]                         // 注：获取第一个rune的接受范围
	if n > 1 && (s[1] < accept.lo || accept.hi < s[1]) { // 注：如果rune长度 > 1，检查第2个字节是否合法
		return true
	} else if n > 2 && (s[2] < locb || hicb < s[2]) { // 注：如果rune长度 > 2，检查第3个字节是否合法
		return true
	}
	return false
}

// DecodeRune 解压缩p中的第一个UTF-8编码，并返回rune及其宽度（以字节为单位）。
// 如果p为空，则返回（RuneError，0）。
// 否则，如果编码无效，则返回（RuneError，1）。
// 对于正确的非空UTF-8，这都是不可能的结果。
//
// 如果编码不正确，则它是无效的UTF-8，对超出范围或不是该值的最短UTF-8编码的符文进行编码。
// 不执行其他任何验证。
func DecodeRune(p []byte) (r rune, size int) { // 注：获取p的第一个rune，返回rune r与大小size
	n := len(p)
	if n < 1 { // 注：p是空的，返回错误rune
		return RuneError, 0
	}
	p0 := p[0]     // 注：获取rune的第1个字节
	x := first[p0] // 注：获取第一个rune的信息
	if x >= as {   // 注：如果rune是ASCII或在替换范围
		// 以下代码模拟对x == xx的附加检查，并相应地处理ASCII和无效情况。 这种掩盖和/或方法防止了额外的分支。
		mask := rune(x) << 31 >> 31                 // 创建0x0000 or 0xFFFF.注：#
		return rune(p[0])&^mask | RuneError&mask, 1 // 注：#
	}
	sz := int(x & 7)
	accept := acceptRanges[x>>4]
	if n < sz { // 注：如果p无法存放rune的长度，返回错误rune
		return RuneError, 1
	}
	b1 := p[1]                            // 注：获取rune的第2个字节
	if b1 < accept.lo || accept.hi < b1 { // 注：如果第2个字节不在接受范围内，返回错误rune
		return RuneError, 1
	}
	if sz <= 2 { // <= 代替 == 来帮助编译器消除一些边界检查，注：如果rune长度为2，返回rune
		return rune(p0&mask2)<<6 | rune(b1&maskx), 2
	}
	b2 := p[2]                  // 注：获取rune的第3个字节
	if b2 < locb || hicb < b2 { // 注：如果第3个字节不在接受范围内，返回错误rune
		return RuneError, 1
	}
	if sz <= 3 { // 注：如果rune长度为3，返回rune
		return rune(p0&mask3)<<12 | rune(b1&maskx)<<6 | rune(b2&maskx), 3
	}
	b3 := p[3]                  // 注：获取rune的第4个字节
	if b3 < locb || hicb < b3 { // 注：如果第4个字节不在接受范围内，返回错误rune
		return RuneError, 1
	}
	return rune(p0&mask4)<<18 | rune(b1&maskx)<<12 | rune(b2&maskx)<<6 | rune(b3&maskx), 4
}

// DecodeRuneInString 类似于DecodeRune，但其输入是字符串。
// 如果s为空，则返回（RuneError，0）。
// 否则，如果编码无效，则返回（RuneError，1）。
// 对于正确的非空UTF-8，这都是不可能的结果。
//
// 如果编码不正确，则它是无效的UTF-8，对超出范围或不是该值的最短UTF-8编码的符文进行编码。
// 不执行其他任何验证。
func DecodeRuneInString(s string) (r rune, size int) { // 注：获取s的第一个rune（同DecodeRune）
	n := len(s)
	if n < 1 {
		return RuneError, 0
	}
	s0 := s[0]
	x := first[s0]
	if x >= as {
		// 以下代码模拟对x == xx的附加检查，并相应地处理ASCII和无效情况。 这种掩盖和/或方法防止了额外的分支。
		mask := rune(x) << 31 >> 31 // Create 0x0000 or 0xFFFF.
		return rune(s[0])&^mask | RuneError&mask, 1
	}
	sz := int(x & 7)
	accept := acceptRanges[x>>4]
	if n < sz {
		return RuneError, 1
	}
	s1 := s[1]
	if s1 < accept.lo || accept.hi < s1 {
		return RuneError, 1
	}
	if sz <= 2 { // <= instead of == to help the compiler eliminate some bounds checks
		return rune(s0&mask2)<<6 | rune(s1&maskx), 2
	}
	s2 := s[2]
	if s2 < locb || hicb < s2 {
		return RuneError, 1
	}
	if sz <= 3 {
		return rune(s0&mask3)<<12 | rune(s1&maskx)<<6 | rune(s2&maskx), 3
	}
	s3 := s[3]
	if s3 < locb || hicb < s3 {
		return RuneError, 1
	}
	return rune(s0&mask4)<<18 | rune(s1&maskx)<<12 | rune(s2&maskx)<<6 | rune(s3&maskx), 4
}

// DecodeLastRune 解压缩p中的最后一个UTF-8编码，并返回符文及其宽度（以字节为单位）。
// 如果p为空，则返回（RuneError，0）。
// 否则，如果编码无效，则返回（RuneError，1）。
// 对于正确的非空UTF-8，这都是不可能的结果。
//
// 如果编码不正确，则它是无效的UTF-8，对超出范围或不是该值的最短UTF-8编码的符文进行编码。
// 不执行其他任何验证。
func DecodeLastRune(p []byte) (r rune, size int) { // 注：倒序获取第一个rune，返回rune r与大小size
	end := len(p)
	if end == 0 { // 注：如果p为空，返回错误rune
		return RuneError, 0
	}
	start := end - 1
	r = rune(p[start])
	if r < RuneSelf { // 注：最后一个字节是单字节rune，返回rune
		return r, 1
	}
	// 向后遍历具有无效UTF-8较长序列的字符串时，防止O(n^2)行为。
	lim := end - UTFMax
	if lim < 0 {
		lim = 0
	}
	for start--; start >= lim; start-- { // 注：p倒序遍历最多4个字节
		if RuneStart(p[start]) { // 注：如果是rune的第一个字节
			break
		}
	}
	if start < 0 {
		start = 0
	}
	r, size = DecodeRune(p[start:end]) // 注：获取rune
	if start+size != end {             // 注：如果rune不完整，返回错误rune
		return RuneError, 1
	}
	return r, size
}

// DecodeLastRuneInString 类似于DecodeLastRune，但其输入是字符串。
// 如果s为空，则返回（RuneError，0）。
// 否则，如果编码无效，则返回（RuneError，1）。
// 对于正确的非空UTF-8，这都是不可能的结果。
//
// 如果编码不正确，则它是无效的UTF-8，对超出范围或不是该值的最短UTF-8编码的符文进行编码。
// 不执行其他任何验证。
func DecodeLastRuneInString(s string) (r rune, size int) { // 注：倒序获取第一个rune（同DecodeLastRune）
	end := len(s)
	if end == 0 {
		return RuneError, 0
	}
	start := end - 1
	r = rune(s[start])
	if r < RuneSelf {
		return r, 1
	}
	// guard against O(n^2) behavior when traversing
	// backwards through strings with long sequences of
	// invalid UTF-8.
	lim := end - UTFMax
	if lim < 0 {
		lim = 0
	}
	for start--; start >= lim; start-- {
		if RuneStart(s[start]) {
			break
		}
	}
	if start < 0 {
		start = 0
	}
	r, size = DecodeRuneInString(s[start:end])
	if start+size != end {
		return RuneError, 1
	}
	return r, size
}

// RuneLen 返回编码rune所需的字节数。
// 如果该rune不是要在UTF-8中编码的有效值，则返回-1。
func RuneLen(r rune) int { // 注：获取r占用的字节数
	switch {
	case r < 0:
		return -1
	case r <= rune1Max:
		return 1
	case r <= rune2Max:
		return 2
	case surrogateMin <= r && r <= surrogateMax: // 注：r在替代范围内
		return -1
	case r <= rune3Max:
		return 3
	case r <= MaxRune:
		return 4
	}
	return -1
}

// EncodeRune 将rune的UTF-8编码写入p（必须足够大）。
// 返回写入的字节数。
func EncodeRune(p []byte, r rune) int { // 注：将r写入p，返回r的长度
	// 负值是错误的。 将其设置为未签名即可解决该问题。
	switch i := uint32(r); {
	case i <= rune1Max: // 注：如果r是单字节rune，直接赋值给p
		p[0] = byte(r)
		return 1
	case i <= rune2Max: // 注：如果r是2字节rune，每6位存储到一个字节中
		_ = p[1] // 消除界限检查
		p[0] = t2 | byte(r>>6)
		p[1] = tx | byte(r)&maskx
		return 2
	case i > MaxRune, surrogateMin <= i && i <= surrogateMax: // 注：如果r在替代范围内，赋值为错误rune
		r = RuneError
		fallthrough
	case i <= rune3Max: // 注：如果r是3字节rune，每6位存储到一个字节中
		_ = p[2] // 消除界限检查
		p[0] = t3 | byte(r>>12)
		p[1] = tx | byte(r>>6)&maskx
		p[2] = tx | byte(r)&maskx
		return 3
	default: // 注：如果r是4字节rune，每6位存储到一个字节中
		_ = p[3] // 消除界限检查
		p[0] = t4 | byte(r>>18)
		p[1] = tx | byte(r>>12)&maskx
		p[2] = tx | byte(r>>6)&maskx
		p[3] = tx | byte(r)&maskx
		return 4
	}
}

// RuneCount 返回p中的rune数量。 错误和短编码被视为宽度为1字节的单个rune。
func RuneCount(p []byte) int { // 注：获取p中rune的数量
	np := len(p)
	var n int
	for i := 0; i < np; { // 注：遍历p
		n++
		c := p[i]
		if c < RuneSelf { // 注：单字节rune
			// ASCII 快速路径
			i++
			continue
		}
		x := first[c] // 注：获取rune的信息
		if x == xx {  // 注：rune无效，计数+1
			i++ // 无效
			continue
		}
		size := int(x & 7) // 注：获取rune的大小
		if i+size > np {   // 注：如果rune占用4字节，但p还有1字节遍历完成，说明该rune无效，计数+1
			i++ // 过短或无效
			continue
		}
		accept := acceptRanges[x>>4] // 注：rune接受的范围
		if c := p[i+1]; c < accept.lo || accept.hi < c {
			size = 1
		} else if size == 2 {
		} else if c := p[i+2]; c < locb || hicb < c {
			size = 1
		} else if size == 3 {
		} else if c := p[i+3]; c < locb || hicb < c {
			size = 1
		}
		i += size
	}
	return n
}

// RuneCountInString 就像RuneCount一样，但是它的输入是一个字符串。
func RuneCountInString(s string) (n int) { // 注：获取p中rune的数量（同RuneCount）
	ns := len(s)
	for i := 0; i < ns; n++ {
		c := s[i]
		if c < RuneSelf {
			// ASCII fast path
			i++
			continue
		}
		x := first[c]
		if x == xx {
			i++ // invalid.
			continue
		}
		size := int(x & 7)
		if i+size > ns {
			i++ // Short or invalid.
			continue
		}
		accept := acceptRanges[x>>4]
		if c := s[i+1]; c < accept.lo || accept.hi < c {
			size = 1
		} else if size == 2 {
		} else if c := s[i+2]; c < locb || hicb < c {
			size = 1
		} else if size == 3 {
		} else if c := s[i+3]; c < locb || hicb < c {
			size = 1
		}
		i += size
	}
	return n
}

// RuneStart 报告该字节是否可能是编码的，可能无效的rune的第一个字节。 第二个和后续字节始终将高两位设置为10。
// 注：C0（1100 0000）， 80（1000 0000）
// b的高2位 != 10即为rune的第1个字节，rune的第一个字节的前两位为11、00、01，不可能为10
// xx = 0xF1	1111 0001
// as = 0xF0	1111 0000
// s1 = 0x02	0000 0010
// s2 = 0x13	0001 0011
// s3 = 0x03	0000 0011
// s4 = 0x23	0010 0011
// s5 = 0x34	0011 0100
// s6 = 0x04	0000 0100
// s7 = 0x44	0100 0100
func RuneStart(b byte) bool { return b&0xC0 != 0x80 } // 注：b是否为rune的第一个字节

// Valid 报告p是否完全由有效的UTF-8编码的rune组成。
func Valid(p []byte) bool { // 注：获取p中所有rune是否全部有效
	n := len(p)
	for i := 0; i < n; { // 注：遍历p
		pi := p[i]
		if pi < RuneSelf { // 注：如果为单字节rune，遍历下一个字节
			i++
			continue
		}
		x := first[pi]
		if x == xx { // 注：如果rune的第一个字节无效，返回false
			return false // 起始字节非法。
		}
		size := int(x & 7)
		if i+size > n { // 注：如果rune不完整，返回false
			return false // 短或无效。
		}
		accept := acceptRanges[x>>4] // 注：如果rune的某个字节超出对应返回，返回false
		if c := p[i+1]; c < accept.lo || accept.hi < c {
			return false
		} else if size == 2 {
		} else if c := p[i+2]; c < locb || hicb < c {
			return false
		} else if size == 3 {
		} else if c := p[i+3]; c < locb || hicb < c {
			return false
		}
		i += size
	}
	return true
}

// ValidString 报告s是否完全由有效的UTF-8编码的符文组成。
func ValidString(s string) bool { // 注：获取p中所有rune是否全部有效（同Valid）
	n := len(s)
	for i := 0; i < n; {
		si := s[i]
		if si < RuneSelf {
			i++
			continue
		}
		x := first[si]
		if x == xx {
			return false // Illegal starter byte.
		}
		size := int(x & 7)
		if i+size > n {
			return false // Short or invalid.
		}
		accept := acceptRanges[x>>4]
		if c := s[i+1]; c < accept.lo || accept.hi < c {
			return false
		} else if size == 2 {
		} else if c := s[i+2]; c < locb || hicb < c {
			return false
		} else if size == 3 {
		} else if c := s[i+3]; c < locb || hicb < c {
			return false
		}
		i += size
	}
	return true
}

// ValidRune 报告r是否可以合法编码为UTF-8。
// 超出范围或替代一半的代码点是非法的。
func ValidRune(r rune) bool { // 注：获取r是否为合法rune
	switch {
	case 0 <= r && r < surrogateMin: // 注：r是否小于替代范围
		return true
	case surrogateMax < r && r <= MaxRune: // 注：r是否大于替代范围 并且 小于最大rune值
		return true
	}
	return false
}
