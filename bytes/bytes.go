// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

// Package bytes 实现用于操作字节片的函数。
// 它类似于strings包的功能。
package bytes

import (
	"internal/bytealg"
	"unicode"
	"unicode/utf8"
)

// Equal 报告a和b是否长度相同并包含相同的字节。
// nil参数等效于一个空切片。
func Equal(a, b []byte) bool { // 注：返回a和b是否长度相同并包含相同的字节。
	// cmd/compile和gccgo都不分配这些字符串转换。
	return string(a) == string(b)
}

// Compare 返回一个按字典顺序比较两个字节片的整数。
// 如果a == b，结果将为0，如果a < b，结果将为-1，如果a > b，结果将为+1。
// nil参数等效于一个空切片。
func Compare(a, b []byte) int { // 注：#
	return bytealg.Compare(a, b)
}

// explode 将s拆分为UTF-8序列的一个切片，每个Unicode代码点一个（仍然是字节的切片），最多n个字节切片。 无效的UTF-8序列被切成单个字节。
func explode(s []byte, n int) [][]byte { // 注：将s拆分为n个rune
	if n <= 0 {
		n = len(s)
	}
	a := make([][]byte, n)
	var size int
	na := 0
	for len(s) > 0 {
		if na+1 >= n { // 注：如果解析出的rune数量 + 1 == n时，不解析其余数据，将其作为a的最后一个元素
			a[na] = s
			na++
			break
		}
		_, size = utf8.DecodeRune(s) // 注：s的第1个rune占用多少字节
		a[na] = s[0:size:size]       // 注：拿出这个rune放到a[na]中
		s = s[size:]                 // 注：s丢弃这个rune
		na++                         // 注：na++
	}
	return a[0:na]
}

// Count 对s中的sep的非重叠实例进行计数。
// 如果sep是一个空切片，则Count返回1 + 以s为单位的UTF-8编码的代码点数。
func Count(s, sep []byte) int { // 注：获取s中出现sep的次数
	// 特殊情况
	if len(sep) == 0 {
		return utf8.RuneCount(s) + 1
	}
	if len(sep) == 1 {
		return bytealg.Count(s, sep[0]) // 注：#
	}
	n := 0
	for {
		i := Index(s, sep) // 注：获取s中第1次出现sep的索引
		if i == -1 {
			return n
		}
		n++
		s = s[i+len(sep):]
	}
}

// Contains 报告分片是否在b之内。
func Contains(b, subslice []byte) bool { // 注：获取b是否包含subslice
	return Index(b, subslice) != -1
}

// ContainsAny 报告char中任何UTF-8编码的代码点是否在b之内。
func ContainsAny(b []byte, chars string) bool { // 注：获取chars的任一元素是否出现在b中
	return IndexAny(b, chars) >= 0
}

// ContainsRune 报告rune是否包含在UTF-8编码的字节片b中。
func ContainsRune(b []byte, r rune) bool { // 注：获取b中第1次出现r的索引
	return IndexRune(b, r) >= 0
}

// IndexByte 返回b中c的第一个实例的索引；如果b中不存在c，则返回-1。
func IndexByte(b []byte, c byte) int { // 注：#获取b中第1次出现c的索引
	return bytealg.IndexByte(b, c) // 注：#
}

func indexBytePortable(s []byte, c byte) int { // 注：获取s中第1次出现c的索引
	for i, b := range s {
		if b == c {
			return i
		}
	}
	return -1
}

// LastIndex 返回s中sep的最后一个实例的索引；如果s中不存在sep，则返回-1。
func LastIndex(s, sep []byte) int { // 注：获取s中最后1次出现sep的索引
	n := len(sep)
	switch {
	case n == 0:
		return len(s)
	case n == 1:
		return LastIndexByte(s, sep[0]) // 注：返回s中最后一次出现sep[0]的索引
	case n == len(s):
		if Equal(s, sep) { // 注：如果s与sep相等，返回0
			return 0
		}
		return -1
	case n > len(s):
		return -1
	}
	// 从字符串末尾搜索Rabin-Karp
	hashss, pow := hashStrRev(sep) // 注：倒序获取sep的哈希
	last := len(s) - n
	var h uint32
	for i := len(s) - 1; i >= last; i-- { // 注：倒序获取i的哈希值
		h = h*primeRK + uint32(s[i])
	}
	if h == hashss && Equal(s[last:], sep) { // 注：比较倒序第一位的哈希值
		return last
	}
	for i := last - 1; i >= 0; i-- { // 注：倒序遍历
		// 注：
		// 计算s[end - len(sep): end]的哈希，尝试匹配
		// 如果匹配，返回end - len(sep)
		// 否则，丢弃end的哈希，添加end - 1的哈希，继续尝试匹配
		//
		// 例：s = 12345，sep = 23
		// 1. 检查45的哈希
		// 2. 检查34的哈希
		// 3. 检查23的哈希，匹配，返回1
		h *= primeRK
		h += uint32(s[i])
		h -= pow * uint32(s[i+n])
		if h == hashss && Equal(s[i:i+n], sep) {
			return i
		}
	}
	return -1
}

// LastIndexByte 返回s中c的最后一个实例的索引；如果s中不存在c，则返回-1。
func LastIndexByte(s []byte, c byte) int { // 注：获取s中c最后一次出现的索引
	for i := len(s) - 1; i >= 0; i-- { // 注：倒序遍历
		if s[i] == c {
			return i
		}
	}
	return -1
}

// IndexRune 将s解释为UTF-8编码的代码点序列。
// 返回给定符文中s中第一次出现的字节索引。
// 如果s中不存在符文，则返回-1。
// 如果r为utf8.RuneError，它将返回任何无效UTF-8字节序列的第一个实例。
func IndexRune(s []byte, r rune) int { // 注：获取s中第一次出现r的索引
	switch {
	case 0 <= r && r < utf8.RuneSelf: // 注：如果r是单字节rune（ASCII），直接比较
		return IndexByte(s, byte(r))
	case r == utf8.RuneError: // 注：如果r时错误字符
		for i := 0; i < len(s); { // 注：遍历每个rune，直到出现错误字符
			r1, n := utf8.DecodeRune(s[i:])
			if r1 == utf8.RuneError {
				return i
			}
			i += n
		}
		return -1
	case !utf8.ValidRune(r): // 注：如果r为非法rune，返回-1
		return -1
	default: // 注：获取r中的第一个rune，直接比较
		var b [utf8.UTFMax]byte
		n := utf8.EncodeRune(b[:], r)
		return Index(s, b[:n])
	}
}

// IndexAny 将s解释为UTF-8编码的Unicode代码点的序列。
// 返回chars中任何Unicode码点中s中第一次出现的字节索引。 如果char为空或没有共同的代码点，则返回-1。
func IndexAny(s []byte, chars string) int { // 注：获取charts中任一元素出现在s中的索引
	if chars == "" {
		//避免扫描所有s。
		return -1
	}
	if len(s) > 8 {
		if as, isASCII := makeASCIISet(chars); isASCII { // 注：chars是否均是ASCII
			for i, c := range s { // 注：遍历s，如果chars的任一元素出现在s中，返回当前索引
				if as.contains(c) { // 注：如果c在as内
					return i
				}
			}
			return -1
		}
	}
	var width int
	for i := 0; i < len(s); i += width { // 注：遍历s
		r := rune(s[i])
		if r < utf8.RuneSelf { // 注：计算s中第i个rune的长度
			width = 1
		} else {
			r, width = utf8.DecodeRune(s[i:])
		}
		for _, ch := range chars { // 注：遍历chars，返回匹配的索引
			if r == ch {
				return i
			}
		}
	}
	return -1
}

// LastIndexAny 将s解释为UTF-8编码的Unicode代码点的序列。
// 它返回char中任何Unicode代码点的s中最后一次出现的字节索引。
// 如果char为空或没有共同的代码点，则返回-1。
func LastIndexAny(s []byte, chars string) int { // 注：获取charts中任一元素出现在s中的索引
	if chars == "" {
		// 避免扫描所有。
		return -1
	}
	if len(s) > 8 {
		if as, isASCII := makeASCIISet(chars); isASCII { // 注：遍历chars中所有的ASCII
			for i := len(s) - 1; i >= 0; i-- { // 注：倒序遍历s，检查匹配
				if as.contains(s[i]) {
					return i
				}
			}
			return -1
		}
	}
	for i := len(s); i > 0; { // 注：倒序遍历s，
		r, size := utf8.DecodeLastRune(s[:i]) // 注：获取最后一个rune
		i -= size
		for _, c := range chars { // 注：遍历charts，检查匹配
			if r == c {
				return i
			}
		}
	}
	return -1
}

// 通用拆分：在sep的每个实例之后拆分，包括subslice中sep的sepSave字节。
func genSplit(s, sep []byte, sepSave, n int) [][]byte { // 注：#将s按sep分割至少n份，包括索引的sepSave字节
	if n == 0 {
		return nil
	}
	if len(sep) == 0 {
		return explode(s, n) // 注：将s拆分为n个rune
	}
	if n < 0 { // 注：如果n < 0，设置初始值为s中出现sep的次数+1
		n = Count(s, sep) + 1
	}

	a := make([][]byte, n)
	n--
	i := 0
	for i < n { // 注：遍历n次
		m := Index(s, sep) // 注：s出现sep的索引
		if m < 0 {
			break
		}
		a[i] = s[: m+sepSave : m+sepSave] // 注：#
		s = s[m+len(sep):]                // 注：去掉匹配的第一个sep
		i++
	}
	a[i] = s // 注：将剩余的s赋值到a的最后一个元素
	return a[:i+1]
}

// SplitN 将s切片成由sep分隔的子切片，并返回这些分隔符之间的子切片的切片。
// 如果sep为空，则SplitN在每个UTF-8序列之后分割。
// 计数确定要返回的子切片数：
// 	n > 0：最多n个子切片； 最后一个子切片将是未拆分的剩余部分。
// 	n == 0：结果为nil（零分片）
// 	n < 0：所有子切片
func SplitN(s, sep []byte, n int) [][]byte { return genSplit(s, sep, 0, n) } // 注：将s按sep分割至少n份

// SplitAfterN 在sep的每个实例之后将s切片为子切片，并返回这些子切片的切片。
// 如果sep为空，则SplitAfterN在每个UTF-8序列之后拆分。
// 计数确定要返回的子切片数：
// 	n > 0：最多n个子切片； 最后一个子切片将是未拆分的剩余部分。
// 	n == 0：结果为nil（零分片）
// 	n < 0：所有子切片
func SplitAfterN(s, sep []byte, n int) [][]byte { // 注：将s按sep分割至少n份，包括分隔符
	return genSplit(s, sep, len(sep), n)
}

// Split 将s切片成由sep分隔的所有子切片，并返回这些分隔符之间的子切片的切片。
// 如果sep为空，则Split在每个UTF-8序列之后拆分。
// 等于SplitN，计数为-1。
func Split(s, sep []byte) [][]byte { return genSplit(s, sep, 0, -1) } // 注：将s按sep分割

// SplitAfter 在sep的每个实例之后将s切片为所有子切片，并返回这些子切片的切片。
// 如果sep为空，则SplitAfter在每个UTF-8序列之后拆分。
// 等于SplitAfterN，计数为-1。
func SplitAfter(s, sep []byte) [][]byte { // 注：将s按sep分割，包括分隔符
	return genSplit(s, sep, len(sep), -1)
}

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1} // 注：ASCII中的空格

// Fields 将s解释为UTF-8编码的代码点序列。
// 按照unicode.IsSpace的定义，它将slice s围绕一个或多个连续的空白字符的每个实例进行拆分，
// 并返回s的子切片的切片；如果s仅包含空白，则返回空切片。
func Fields(s []byte) [][]byte { // 注：#
	// 首先计算字段。
	// 如果s为ASCII，则为精确计数，否则为近似值。
	n := 0
	wasSpace := 1
	// setBits用于跟踪在s字节中设置了哪些位。
	setBits := uint8(0)
	for i := 0; i < len(s); i++ { // 注：遍历s
		r := s[i]
		setBits |= r
		isSpace := int(asciiSpace[r]) // 注：s[i]是否为空格
		n += wasSpace & ^isSpace      // 注：空格不计数
		wasSpace = isSpace
	}

	if setBits >= utf8.RuneSelf {
		// 输入片中的某些符文不是ASCII。
		return FieldsFunc(s, unicode.IsSpace)
	}

	// ASCII fast path
	a := make([][]byte, n)
	na := 0
	fieldStart := 0
	i := 0
	// Skip spaces in the front of the input.
	for i < len(s) && asciiSpace[s[i]] != 0 {
		i++
	}
	fieldStart = i
	for i < len(s) {
		if asciiSpace[s[i]] == 0 {
			i++
			continue
		}
		a[na] = s[fieldStart:i:i]
		na++
		i++
		// Skip spaces in between fields.
		for i < len(s) && asciiSpace[s[i]] != 0 {
			i++
		}
		fieldStart = i
	}
	if fieldStart < len(s) { // Last field might end at EOF.
		a[na] = s[fieldStart:len(s):len(s)]
	}
	return a
}

// FieldsFunc 将s解释为一系列UTF-8编码的代码点。
// 它将在满足f(c)的每个代码点c处分割切片s并返回s的子切片的切片。 如果s中的所有代码点均满足f(c)或len(s) = 0，则返回一个空切片。
// FieldsFunc不保证调用f(c)的顺序。
// 如果f对于给定的c没有返回一致的结果，则FieldsFunc可能会崩溃。
func FieldsFunc(s []byte, f func(rune) bool) [][]byte { // 注：#获取s中解析出的rune不满足f(r)的下一个rune的列表
	// 跨度用于记录形式为s[start:end]的s的一部分。
	// 开始索引是包含的，结束索引是排他的。
	type span struct {
		start int
		end   int
	}
	spans := make([]span, 0, 32)

	// 查找字段的开始和结束索引。
	wasField := false
	fromIndex := 0
	for i := 0; i < len(s); { // 注：遍历s
		size := 1
		r := rune(s[i])
		if r >= utf8.RuneSelf { // 注：如果r为多字节rune
			r, size = utf8.DecodeRune(s[i:])
		}
		if f(r) { // 注：满足f
			if wasField { // 注：上次未满足f
				spans = append(spans, span{start: fromIndex, end: i}) // 注：spans新增
				wasField = false
			}
		} else {
			if !wasField { // 注：上次满足f
				fromIndex = i
				wasField = true
			}
		}
		i += size
	}

	// 最后一个字段可能以EOF结尾。
	if wasField { // 注：上次未满足
		spans = append(spans, span{fromIndex, len(s)}) // 注：spans新增
	}

	// 从记录的字段索引创建子切片。
	a := make([][]byte, len(spans))
	for i, span := range spans { // 注：遍历spans，转为[][]byte
		a[i] = s[span.start:span.end:span.end]
	}

	return a
}

// Join 连接的元素以创建新的字节片。 分隔符sep放置在所得切片中的元素之间。
func Join(s [][]byte, sep []byte) []byte { // 注：将s合并为[]byte，用sep分隔
	if len(s) == 0 {
		return []byte{}
	}
	if len(s) == 1 { // 注：如果s长度为1，返回s[0]
		// 返回一个拷贝
		return append([]byte(nil), s[0]...)
	}
	n := len(sep) * (len(s) - 1) // 注：sep的个数 + s的总长度
	for _, v := range s {
		n += len(v)
	}

	b := make([]byte, n)
	bp := copy(b, s[0])
	for _, v := range s[1:] { // 注：s[0] + sep + s[1] + sep + ... + s[len(s)]
		bp += copy(b[bp:], sep)
		bp += copy(b[bp:], v)
	}
	return b
}

// HasPrefix 测试字节片s是否以prefix开头。
func HasPrefix(s, prefix []byte) bool { // 注：获取s是否以prefix开头
	return len(s) >= len(prefix) && Equal(s[0:len(prefix)], prefix)
}

// HasSuffix 测试字节片s是否以后缀结尾。
func HasSuffix(s, suffix []byte) bool { // 注：获取s是否以suffix结尾
	return len(s) >= len(suffix) && Equal(s[len(s)-len(suffix):], suffix)
}

// Map 返回字节片s的副本，其所有字符都根据映射函数进行了修改。
// 如果映射返回负值，则将字符从字节片中丢弃，并且不进行替换。
// s中的字符和输出被解释为UTF-8编码的代码点。
func Map(mapping func(r rune) rune, s []byte) []byte { // 注：遍历s中的rune，获取满足mapping(r)的rune列表
	// 在最坏的情况下，切片在映射时可能会增长，使事情变得不愉快。
	// 但是，我们很少敢假设它很好，所以很少见。 它也可以缩小，但这自然而然地消失了。
	maxbytes := len(s) // b的长度
	nbytes := 0        // b中编码的字节数
	b := make([]byte, maxbytes)
	for i := 0; i < len(s); { // 注：遍历s
		wid := 1
		r := rune(s[i])
		if r >= utf8.RuneSelf { // 注：获取一个rune
			r, wid = utf8.DecodeRune(s[i:])
		}
		r = mapping(r) // 注：执行mapping
		if r >= 0 {    // 注：映射成功
			rl := utf8.RuneLen(r) // 注：获取rune的长度
			if rl < 0 {
				rl = len(string(utf8.RuneError))
			}
			if nbytes+rl > maxbytes { // 注：如果已映射的数据长度 > 映射表的长度，扩容
				// 增加缓冲区。
				maxbytes = maxbytes*2 + utf8.UTFMax
				nb := make([]byte, maxbytes)
				copy(nb, b[0:nbytes])
				b = nb
			}
			nbytes += utf8.EncodeRune(b[nbytes:maxbytes], r) // 注：rune写入b
		}
		i += wid
	}
	return b[0:nbytes]
}

// Repeat 返回由b的计数副本组成的新字节片。
//
// 如果count为负或（len(b) * count）的结果溢出，则表示恐慌。
func Repeat(b []byte, count int) []byte { // 注：获取count个b的切片
	if count == 0 {
		return []byte{}
	}
	// 由于我们无法在溢出时返回错误，因此如果重复操作会产生溢出，我们应该惊慌。
	// 参见Issue golang.org/issue/16237。
	if count < 0 {
		panic("bytes: negative Repeat count") // 恐慌："重复计数为负"
	} else if len(b)*count/count != len(b) {
		panic("bytes: Repeat count causes overflow") // 恐慌："重复计数导致溢出"
	}

	nb := make([]byte, len(b)*count)
	bp := copy(nb, b)
	for bp < len(nb) {
		copy(nb[bp:], nb[:bp])
		bp *= 2
	}
	return nb
}

// ToUpper 返回字节片s的副本，其中所有Unicode字母都映射到它们的大写字母。
func ToUpper(s []byte) []byte { // 注：获取s的大写字母
	isASCII, hasLower := true, false
	for i := 0; i < len(s); i++ { // 注：遍历s，获取s是否全部由ASCII构成，是否包含小写字母
		c := s[i]
		if c >= utf8.RuneSelf {
			isASCII = false
			break
		}
		hasLower = hasLower || ('a' <= c && c <= 'z')
	}

	if isASCII { // 针对仅ASCII字节片进行优化。
		if !hasLower { // 注：s不包含小写字母，直接返回
			// 只需返回拷贝
			return append([]byte(""), s...)
		}
		b := make([]byte, len(s))
		for i := 0; i < len(s); i++ { // 注：遍历s，将小写字母转为大写字母
			c := s[i]
			if 'a' <= c && c <= 'z' {
				c -= 'a' - 'A'
			}
			b[i] = c
		}
		return b
	}
	return Map(unicode.ToUpper, s) // 注：映射为大写
}

// ToLower 返回字节片s的副本，其中所有Unicode字母都映射到它们的小写字母。
func ToLower(s []byte) []byte { // 注：获取s的小写字母
	isASCII, hasUpper := true, false
	for i := 0; i < len(s); i++ { // 注：遍历s，获取s是否全部由ASCII构成，是否包含大写字母
		c := s[i]
		if c >= utf8.RuneSelf {
			isASCII = false
			break
		}
		hasUpper = hasUpper || ('A' <= c && c <= 'Z')
	}

	if isASCII { // 针对仅ASCII字节片进行优化。
		if !hasUpper { // 注：s不包含大写字母，直接返回
			return append([]byte(""), s...)
		}
		b := make([]byte, len(s))
		for i := 0; i < len(s); i++ { // 注：遍历s，将大写字母转为小写字母
			c := s[i]
			if 'A' <= c && c <= 'Z' {
				c += 'a' - 'A'
			}
			b[i] = c
		}
		return b
	}
	return Map(unicode.ToLower, s) // 注：映射为小写
}

// ToTitle 将s视为UTF-8编码的字节，并返回一个副本，其中所有Unicode字母均映射到其标题大小写。
func ToTitle(s []byte) []byte { return Map(unicode.ToTitle, s) } // 注：#获取s的标题大小写

// ToUpperSpecial 将s视为UTF-8编码的字节，并返回一个副本，其中所有Unicode字母均映射为它们的大写字母，优先考虑特殊的大小写规则。
func ToUpperSpecial(c unicode.SpecialCase, s []byte) []byte { // 注：#
	return Map(c.ToUpper, s)
}

// ToLowerSpecial 将s视为UTF-8编码的字节，并返回一个副本，其中所有Unicode字母均映射为小写字母，并优先使用特殊的大小写规则。
func ToLowerSpecial(c unicode.SpecialCase, s []byte) []byte { // 注：#
	return Map(c.ToLower, s)
}

// ToTitleSpecial 将s视为UTF-8编码的字节，并返回一个副本，其中所有Unicode字母均映射到其标题大小写，并优先使用特殊的大小写规则。
func ToTitleSpecial(c unicode.SpecialCase, s []byte) []byte { // 注：#
	return Map(c.ToTitle, s)
}

// ToValidUTF8 将s视为UTF-8编码的字节，并返回一个副本，其中每次运行的字节均表示无效的UTF-8，并替换为替换中的字节，该字节可以为空。
func ToValidUTF8(s, replacement []byte) []byte { // 注：将s中无效的UTF-8字符替换为replacement并返回
	b := make([]byte, 0, len(s)+len(replacement))
	invalid := false          // 前一个字节来自无效的UTF-8序列
	for i := 0; i < len(s); { // 注：遍历s
		c := s[i]
		if c < utf8.RuneSelf { // 注：如果是单字节rune，b追加rune
			i++
			invalid = false
			b = append(b, byte(c))
			continue
		}
		_, wid := utf8.DecodeRune(s[i:]) // 注：如果是多字节rune，读取占用字节
		if wid == 1 {                    // 注：如果不是UTF8，b追加replacement
			i++
			if !invalid {
				invalid = true
				b = append(b, replacement...)
			}
			continue
		}
		invalid = false
		b = append(b, s[i:i+wid]...)
		i += wid
	}
	return b
}

// isSeparator 报告rune是否可以标记单词边界。
// TODO：在程序包unicode捕获更多属性时更新。
func isSeparator(r rune) bool { // 注：获取r是否不是字母、数字、下划线
	// ASCII字母数字和下划线不是分隔符
	if r <= 0x7F { // 注： 字母、数字、下划线返回false，其他返回true
		switch {
		case '0' <= r && r <= '9':
			return false
		case 'a' <= r && r <= 'z':
			return false
		case 'A' <= r && r <= 'Z':
			return false
		case r == '_':
			return false
		}
		return true
	}
	// 字母和数字不是分隔符
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return false
	}
	// 否则，我们现在所能做的就是将空格视为分隔符。
	return unicode.IsSpace(r)
}

// Title 将s视为UTF-8编码的字节，并返回带有所有Unicode字母的副本，该副本以单词开头的所有Unicode字母映射到其标题大小写。
//
// BUG（rsc）：标题用于单词边界的规则不能正确处理Unicode标点符号。
func Title(s []byte) []byte { // 注：#
	// 在此处使用闭包来记住状态。
	// 骇人但有效。 顺序依赖于地图扫描，每个符文调用一次关闭。
	prev := ' '
	return Map(
		func(r rune) rune {
			if isSeparator(prev) {
				prev = r
				return unicode.ToTitle(r)
			}
			prev = r
			return r
		},
		s)
}

// TrimLeftFunc 将s视为UTF-8编码的字节，并通过切分满足f(c)的所有前导UTF-8编码的代码点c来返回s的子片段。
func TrimLeftFunc(s []byte, f func(r rune) bool) []byte { // 注：遍历s中的rune，返回去掉f(r) == false的索引之前数据的s
	// 例：s = 12345，返回2345
	i := indexFunc(s, f, false) // 注：遍历s中的rune，获取f(r) == false的索引
	if i == -1 {
		return nil
	}
	return s[i:]
}

// TrimRightFunc 通过分割满足f(c)的所有尾随UTF-8编码的代码点c，返回s的子片段。
func TrimRightFunc(s []byte, f func(r rune) bool) []byte { // 注：倒序遍历s中的rune，返回去掉f(r) == false的索引之后数据的s
	// 例：s = 12345，返回1234
	i := lastIndexFunc(s, f, false)      // 注：倒序遍历s中的rune，获取f(r) == truth的索引
	if i >= 0 && s[i] >= utf8.RuneSelf { // 注：保留f(r) == truth时的r
		_, wid := utf8.DecodeRune(s[i:])
		i += wid
	} else {
		i++
	}
	return s[0:i]
}

// TrimFunc 通过分割满足f(c)的所有前导和尾随UTF-8编码的代码点c，返回s的子片段。
func TrimFunc(s []byte, f func(r rune) bool) []byte { // 注：遍历s中的rune，去掉第一次和最后一次f(r) == false的索引之前和之后的数据s
	// 例：s = 12345，返回234
	return TrimRightFunc(TrimLeftFunc(s, f), f)
}

// TrimPrefix 返回s，但不提供提供的前导前缀字符串。
// 如果s不以前缀开头，则s不变返回。
func TrimPrefix(s, prefix []byte) []byte { // 注：获取去掉前缀prefix之后的s
	if HasPrefix(s, prefix) { // 注：如果s以prefix开头，去掉前缀
		return s[len(prefix):]
	}
	return s
}

// TrimSuffix 返回s，但不提供尾随的后缀字符串。
// 如果s不以后缀结尾，则s保持不变。
func TrimSuffix(s, suffix []byte) []byte { // 注：获取去掉后缀suffix之后的s
	if HasSuffix(s, suffix) { // 注：如果s以suffix结尾，去掉后缀
		return s[:len(s)-len(suffix)]
	}
	return s
}

// IndexFunc 将s解释为UTF-8编码的代码点序列。
// 返回满足f(c)的第一个Unicode代码点的s中的字节索引，如果不满足则返回-1。
func IndexFunc(s []byte, f func(r rune) bool) int { // 注：遍历s中的rune，获取f(r) == true的索引
	return indexFunc(s, f, true)
}

// LastIndexFunc 将s解释为UTF-8编码的代码点序列。
// 返回满足f(c)的最后一个Unicode代码点的s中的字节索引，如果不满足则返回-1。
func LastIndexFunc(s []byte, f func(r rune) bool) int { // 注：倒序遍历s中的rune，获取f(r) == true的索引
	return lastIndexFunc(s, f, true)
}

// indexFunc 与IndexFunc相同，不同之处在于如果true == false，则谓词函数的含义相反。
func indexFunc(s []byte, f func(r rune) bool, truth bool) int { // 注：遍历s中的rune，获取f(r) == truth的索引
	start := 0
	for start < len(s) { // 注：遍历s
		wid := 1
		r := rune(s[start]) // 注：取出一个rune
		if r >= utf8.RuneSelf {
			r, wid = utf8.DecodeRune(s[start:])
		}
		if f(r) == truth { // 注：如果f(r) == truth，返回rune的索引
			return start
		}
		start += wid
	}
	return -1
}

// lastIndexFunc 与LastIndexFunc相同，不同之处在于如果true == false，则谓词功能的含义相反。
func lastIndexFunc(s []byte, f func(r rune) bool, truth bool) int { // 注：倒序遍历s中的rune，获取f(r) == truth的索引
	for i := len(s); i > 0; { // 注：倒序遍历s
		r, size := rune(s[i-1]), 1 // 注：倒序获取rune
		if r >= utf8.RuneSelf {
			r, size = utf8.DecodeLastRune(s[0:i])
		}
		i -= size
		if f(r) == truth { // 注：如果f(r) == truth，返回rune的索引
			return i
		}
	}
	return -1
}

// asciiSet 是一个32字节的值，其中每个位代表集合中存在给定的ASCII字符。
// 从最低字的最低有效位到最高字的最高有效位，
// 低16个字节的128位映射到所有128个ASCII字符的整个范围。
// 高16个字节的128位将被清零，以确保任何非ASCII字符都将报告为不在集合中。
type asciiSet [8]uint32

// makeASCIISet 创建一组ASCII字符并报告char中的所有字符是否均为ASCII。
func makeASCIISet(chars string) (as asciiSet, ok bool) { // 注：获取chars中的所有的ASCII as与是否全部为ASCII ok
	for i := 0; i < len(chars); i++ { // 注：遍历chars
		c := chars[i]
		if c >= utf8.RuneSelf { // 注：如果字符不是单字节rune（不是ASCII），返回false
			return as, false
		}
		as[c>>5] |= 1 << uint(c&31) // 注：#
	}
	return as, true
}

// contains 报告c是否在集合内。
func (as *asciiSet) contains(c byte) bool { // 注：获取as是否包含c
	return (as[c>>5] & (1 << uint(c&31))) != 0 // 注：#
}

func makeCutsetFunc(cutset string) func(r rune) bool { // 注：获取cutset是否包含c的方法
	if len(cutset) == 1 && cutset[0] < utf8.RuneSelf { // 注：如果cutset是一个rune，返回直接比较rune的方法
		return func(r rune) bool {
			return r == rune(cutset[0])
		}
	}
	if as, isASCII := makeASCIISet(cutset); isASCII { // 注：如果cutset全部为rune，返回ASCII集合类型cutset是否包含rune的方法
		return func(r rune) bool {
			return r < utf8.RuneSelf && as.contains(byte(r))
		}
	}
	return func(r rune) bool { // 注：返回字符串类型cutset是否包含r的方法
		for _, c := range cutset {
			if c == r {
				return true
			}
		}
		return false
	}
}

// Trim 通过切掉cutset中包含的所有前导和尾随UTF-8编码的代码点，返回s的子片段。
func Trim(s []byte, cutset string) []byte { // 注：去掉s中第一次和最后一次cutset中不包含的元素的索引之前和之后的数据
	// 例：s = 12345，cutset = 24，返回234
	return TrimFunc(s, makeCutsetFunc(cutset))
}

// TrimLeft 通过切掉cutset中包含的所有前导UTF-8编码的代码点，返回s的子片段。
func TrimLeft(s []byte, cutset string) []byte { // 注：去掉s中第一次cutset中不包含的元素的索引之前的数据
	return TrimLeftFunc(s, makeCutsetFunc(cutset))
}

// TrimRight 通过切掉cutset中包含的所有尾随UTF-8编码的代码点来返回s的子片段。
func TrimRight(s []byte, cutset string) []byte { // 注：去掉s中最后一次cutset中不包含的元素的索引之后的数据
	return TrimRightFunc(s, makeCutsetFunc(cutset))
}

// TrimSpace 通过切掉Unicode定义的所有前导和尾随空格来返回s的子片段。
func TrimSpace(s []byte) []byte { // 注：获取去掉空格后的s
	// ASCII的快速路径：查找第一个ASCII非空格字节
	start := 0
	for ; start < len(s); start++ { // 注：遍历s
		c := s[start]
		if c >= utf8.RuneSelf { // 注：如果遇到了不是ASCII的字节，返回去掉前后空格的s
			// 如果遇到非ASCII字节，请在剩余字节上回退到较慢的unicode-aware方法
			return TrimFunc(s[start:], unicode.IsSpace)
		}
		if asciiSpace[c] == 0 { // 注：如果是ASCII，直接判断
			break
		}
	}

	// 现在从末尾查找第一个ASCII非空格字节
	stop := len(s)
	for ; stop > start; stop-- { // 注：倒序遍历s
		c := s[stop-1]
		if c >= utf8.RuneSelf { // 注：如果遇到了不是ASCII的字节，返回去掉前后空格的s
			return TrimFunc(s[start:stop], unicode.IsSpace)
		}
		if asciiSpace[c] == 0 { // 注：如果是ASCII，直接判断
			break
		}
	}
	// 至此，s[start:stop]以ASCII非空格字节开始和结束，到此完成。 上面已经处理了非ASCII情况。
	if start == stop { // 注：如果去掉空格后s为空，返回nil
		// 保留以前的TrimLeftFunc行为的特殊情况，如果所有空格都返回nil而不是空片。
		return nil
	}
	return s[start:stop]
}

// Runes 将s解释为UTF-8编码的代码点序列。
// 返回等于s的一部分符文（Unicode代码点）。
func Runes(s []byte) []rune { // 注：获取s中的所有rune
	t := make([]rune, utf8.RuneCount(s)) // 注：获取s中rune的数量
	i := 0
	for len(s) > 0 { // 注：遍历s
		r, l := utf8.DecodeRune(s) // 注：解析rune
		t[i] = r
		i++
		s = s[l:]
	}
	return t
}

// Replace 返回切片s的副本，其中旧的前n个非重叠实例替换为新的。
// 如果old为空，则它在切片的开头和每个UTF-8序列之后匹配，最多产生k个符文切片的k + 1个替换。
// 如果n <0，则替换次数没有限制。
func Replace(s, old, new []byte, n int) []byte { // 注：获取将前n个old替换为new的s，如果old为空，在s中的前n个rune之间添加new
	m := 0
	if n != 0 {
		// 计算替换次数。
		m = Count(s, old) // 注：获取s中出现old的次数
	}
	if m == 0 { // 注：如果s中不包含old，直接返回s
		// 直接返回拷贝
		return append([]byte(nil), s...)
	}
	if n < 0 || m < n {
		n = m
	}

	// 将替换应用于缓冲区。
	t := make([]byte, len(s)+n*(len(new)-len(old)))
	w := 0
	start := 0
	for i := 0; i < n; i++ { // 注：在s中替换n个old
		j := start
		if len(old) == 0 { // 注：如果old是空的，在每个rune之间添加new
			if i > 0 {
				_, wid := utf8.DecodeRune(s[start:])
				j += wid
			}
		} else { // 注：如果old不为空，计算old出现的位置
			j += Index(s[start:], old)
		}
		w += copy(t[w:], s[start:j]) // 注：添加找到old之前的数据
		w += copy(t[w:], new)        // 注：添加new
		start = j + len(old)
	}
	w += copy(t[w:], s[start:]) // 注：添加完成n次替换old为new之后的数据
	return t[0:w]
}

// ReplaceAll 返回slice的副本，其中所有旧的非重叠实例都被new替换。
// 如果old为空，则它在片段的开头和每个UTF-8序列之后匹配，最多可产生k个符文片段的k + 1个替换。
func ReplaceAll(s, old, new []byte) []byte { // 注：获取将old替换为new的s
	return Replace(s, old, new, -1)
}

// EqualFold 报告在Unicode大小写折叠下s和t（解释为UTF-8字符串）是否相等，这是不区分大小写的更通用形式。
func EqualFold(s, t []byte) bool { // 注：遍历s与t中的rune比较是否相等（不区分大小写）
	for len(s) != 0 && len(t) != 0 { // 注：遍历s和t
		// 从每个中提取第一个rune
		var sr, tr rune
		if s[0] < utf8.RuneSelf { // 注：如果s的元素是是单字节rune
			sr, s = rune(s[0]), s[1:]
		} else { // 注：如果s的元素是多字节rune
			r, size := utf8.DecodeRune(s)
			sr, s = r, s[size:]
		}
		if t[0] < utf8.RuneSelf { // 注：如果t的元素是单字节rune
			tr, t = rune(t[0]), t[1:]
		} else { // 注：如果t的元素是多字节rune
			r, size := utf8.DecodeRune(t)
			tr, t = r, t[size:]
		}
		// 如果匹配，请继续； 如果不是，则返回false。
		// 简单的情况。
		if tr == sr { // 注：如果相等，检查下个rune
			continue
		}

		// 使sr <tr简化以下内容。
		if tr < sr { // 注：让sr比tr小
			tr, sr = sr, tr
		}
		// 快速检查ASCII。
		if tr < utf8.RuneSelf { // 注：如果最大的rune是单字节rune，直接比较小写ASCII
			// 仅ASCII，sr/tr必须为大写/小写
			if 'A' <= sr && sr <= 'Z' && tr == sr+'a'-'A' {
				continue
			}
			return false
		}

		// 一般情况。 SimpleFold(x)返回下一个等效的rune > x或环绕到较小的值。
		r := unicode.SimpleFold(sr) // 注：#
		for r != sr && r < tr {
			r = unicode.SimpleFold(r)
		}
		if r == tr {
			continue
		}
		return false
	}

	// One string is empty. Are both?
	return len(s) == len(t)
}

// Index 返回s中sep的第一个实例的索引；如果s中不存在sep，则返回-1。
func Index(s, sep []byte) int { // 注：获取s中第1次出现sep的索引
	// 注：
	// len(sep) <= bytealg.MaxLen：
	// 	1. 寻找匹配sep的第1个字符的索引，再匹配sep的第2个字符，再匹配全部字符
	// 	2. 超过容忍度时，使用bytealg.Index进行最后匹配
	// len(sep) > bytealg.MaxLen：
	// 	1. 同上
	// 	2. 超过容忍度时，使用Rabin-Karp字符串查找算法匹配

	n := len(sep)
	switch {
	case n == 0:
		return 0
	case n == 1:
		return IndexByte(s, sep[0]) // 注：获取s中第1次出现sep[0]的索引
	case n == len(s):
		if Equal(sep, s) { // 注：如果sep和s相等，返回0
			return 0
		}
		return -1
	case n > len(s):
		return -1
	case n <= bytealg.MaxLen:
		// 当s和sep都较小时使用蛮力
		if len(s) <= bytealg.MaxBruteForce { // 注：s的长度小于64时，使用蛮力
			return bytealg.Index(s, sep)
		}
		c0 := sep[0]
		c1 := sep[1]
		i := 0
		t := len(s) - n + 1
		fails := 0
		for i < t { // 注：滑块搜索?，遍历 len(s) - len(sep) + 1次
			if s[i] != c0 { // 注：不匹配第1个元素，寻找下一次出现sep[0]的索引
				// IndexByte比bytealg.Index快，因此只要我们不会收到很多误报，就可以使用它。
				o := IndexByte(s[i:t], c0) // 注：寻找下一次出现sep[0]的索引
				if o < 0 {                 // 注：如果完全不匹配，返回-1
					return -1
				}
				i += o
			}
			if s[i+1] == c1 && Equal(s[i:i+n], sep) { // 注：如果前2个字节匹配，比较全部字节
				return i
			}
			fails++
			i++
			// 当IndexByte产生过多的误报时，切换到bytealg.Index。
			if fails > bytealg.Cutover(i) { // 注：如果查找失败的次数超过了容忍次数，再匹配最后一次
				r := bytealg.Index(s[i:], sep) // 注：全文匹配最后一次
				if r >= 0 {
					return r + i
				}
				return -1
			}
		}
		return -1
	}
	c0 := sep[0]
	c1 := sep[1]
	i := 0
	fails := 0
	t := len(s) - n + 1
	for i < t {
		if s[i] != c0 { // 注：不匹配第1个元素，寻找下一次出现sep[0]的索引
			o := IndexByte(s[i:t], c0) // 注：寻找下一次出现sep[0]的索引
			if o < 0 {                 // 注：如果完全不匹配，返回-1
				break
			}
			i += o
		}
		if s[i+1] == c1 && Equal(s[i:i+n], sep) { // 注：如果前2个字节匹配，比较全部字节
			return i
		}
		i++
		fails++
		if fails >= 4+i>>4 && i < t {
			// 放弃IndexByte，它并没有远远超过Rabin-Karp。
			// 实验（使用IndexPeriodic）表明转换大约是16个字节的跳过。
			// TODO：如果sep的大前缀匹配，我们应该以更大的平均跳过次数进行切换，因为Equal变得更加昂贵。
			// 此代码未考虑到这种影响。
			j := indexRabinKarp(s[i:], sep) // 注：使用Rabin-Karp字符串查找算法查找s中sep出现的位置
			if j < 0 {
				return -1
			}
			return i + j
		}
	}
	return -1
}

func indexRabinKarp(s, sep []byte) int { // 注：使用Rabin-Karp字符串查找算法查找s中sep出现的位置
	// Rabin-Karp搜索
	hashsep, pow := hashStr(sep) // 注：获取哈希值与乘法因子
	n := len(sep)
	var h uint32
	for i := 0; i < n; i++ { // 注：遍历sep，计算
		h = h*primeRK + uint32(s[i])
	}
	if h == hashsep && Equal(s[:n], sep) { // 注：如果第0位开始的s与sep的哈希匹配，且数据匹配，返回0
		return 0
	}
	for i := n; i < len(s); { //遍历len(s) - len(sep)次
		// 注：
		// 计算s[start: start + len(sep)]的哈希，尝试匹配
		// 如果匹配，返回start
		// 否则，丢弃start的哈希，添加start+1的哈希，继续尝试匹配
		//
		// 例：s = 12345，sep = 34
		// 1. 检查12的哈希
		// 2. 检查23的哈希
		// 3. 检查34的哈希，匹配，返回2
		h *= primeRK
		h += uint32(s[i])
		h -= pow * uint32(s[i-n])
		i++
		if h == hashsep && Equal(s[i-n:i], sep) {
			return i - n
		}
	}
	return -1
}

// primeRK 是Rabin-Karp算法中使用的主要基础。
const primeRK = 16777619

// hashStr 返回哈希值和在Rabin-Karp算法中使用的适当乘法因子。
func hashStr(sep []byte) (uint32, uint32) { // 注：获取sep的哈希值和在Rabin-Karp算法中使用的适当乘法因子。
	hash := uint32(0)
	for i := 0; i < len(sep); i++ { // 注：遍历sep
		hash = hash*primeRK + uint32(sep[i]) // 注：计算哈希
	}
	var pow, sq uint32 = 1, primeRK
	for i := len(sep); i > 0; i >>= 1 { // 注：遍历sep长度的位数
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash, pow
}

// hashStrRev 返回sep倒数的哈希值和在Rabin-Karp算法中使用的适当乘数。
func hashStrRev(sep []byte) (uint32, uint32) { // 注：获取倒序的sep的哈希值和在Rabin-Karp算法中使用的适当乘法因子。
	hash := uint32(0)
	for i := len(sep) - 1; i >= 0; i-- { // 注：倒序遍历sep
		hash = hash*primeRK + uint32(sep[i]) // 注：计算哈希
	}
	var pow, sq uint32 = 1, primeRK
	for i := len(sep); i > 0; i >>= 1 { // 注：遍历sep长的位数
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash, pow
}
