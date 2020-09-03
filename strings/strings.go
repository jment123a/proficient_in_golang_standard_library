//版权所有2009 The Go Authors。 版权所有。
//此源代码的使用受BSD样式的约束
//可以在LICENSE文件中找到的许可证。

// Package strings 实现简单的函数来处理UTF-8编码的字符串。
//
// 有关Go中UTF-8字符串的信息，请参见https://blog.golang.org/strings。
package strings

import (
	"internal/bytealg"
	"unicode"
	"unicode/utf8"
)

// explode 将s拆分为UTF-8字符串的一部分，每个Unicode字符一个字符串，最大为n（n < 0表示无限制）。
// 无效的UTF-8序列成为U + FFFD的正确编码。
func explode(s string, n int) []string { // 注：将s拆分为n个rune
	// 例：s = "abc123456"，n = 3，返回[]string{a, b, c123456}

	l := utf8.RuneCountInString(s) // 注：获取s中的rune个数
	if n < 0 || n > l {
		n = l
	}
	a := make([]string, n)
	for i := 0; i < n-1; i++ { // 注：遍历s，获取n-1个rune，最后一个rune包括剩下的字符串数据
		ch, size := utf8.DecodeRuneInString(s) // 注：获取下一个rune，暂存
		a[i] = s[:size]
		s = s[size:]
		if ch == utf8.RuneError {
			a[i] = string(utf8.RuneError)
		}
	}
	if n > 0 { // 注：处理其余的字符串
		a[n-1] = s
	}
	return a
}

// primeRK 是Rabin-Karp算法中使用的主要基础。
const primeRK = 16777619

// hashStr 返回哈希值和在Rabin-Karp算法中使用的适当乘法因子。
func hashStr(sep string) (uint32, uint32) { // 注：获取sep的哈希与乘法因子（算法：Rabin-Karp）
	hash := uint32(0)
	for i := 0; i < len(sep); i++ { // 注：计算sep的哈希
		hash = hash*primeRK + uint32(sep[i])
	}
	var pow, sq uint32 = 1, primeRK
	for i := len(sep); i > 0; i >>= 1 { // 注：根据sep的每一位，计算乘法因子
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash, pow
}

// hashStrRev 返回sep倒数的哈希值和在Rabin-Karp算法中使用的适当乘法因子。
func hashStrRev(sep string) (uint32, uint32) { // 注：获取sep的倒数的哈希与乘法因子（算法：Rabin-Karp）
	hash := uint32(0)
	for i := len(sep) - 1; i >= 0; i-- { // 注：倒序计算sep的哈希
		hash = hash*primeRK + uint32(sep[i])
	}
	var pow, sq uint32 = 1, primeRK
	for i := len(sep); i > 0; i >>= 1 { // 注：根据sep的每一位，计算乘法因子
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash, pow
}

// Count 计算substr中不重叠的实例数。
// 如果substr是一个空字符串，则Count返回1 + s中的Unicode代码点数。
func Count(s, substr string) int { // 注：获取s中substr出现的次数
	// 特殊情况
	if len(substr) == 0 { // 注：如果substr是空的，返回s中的rune数量+1
		return utf8.RuneCountInString(s) + 1
	}
	if len(substr) == 1 { // 注：如果substr是一个字节
		return bytealg.CountString(s, substr[0])
	}
	n := 0
	// 例：s = "121314"，substr = "1"
	// i = 1，s = "21314"
	// i = 2，s = "314"
	// i = 3，s = "4"
	// i = 3，s = ""，返回3
	for { // 注：遍历s中第一个substr的索引，删除s中第一个substr，直到遍历结束
		i := Index(s, substr)
		if i == -1 {
			return n
		}
		n++
		s = s[i+len(substr):]
	}
}

// Contains 报告substr是否在s之内。
func Contains(s, substr string) bool { // 注：获取s中是否包括substr
	return Index(s, substr) >= 0
}

// ContainsAny 报告char中的任何Unicode代码点是否在s之内。
func ContainsAny(s, chars string) bool { // 注：获取s是否包含chars的任意元素
	return IndexAny(s, chars) >= 0
}

// ContainsRune 报告Unicode代码点r是否在s之内。
func ContainsRune(s string, r rune) bool { // 注：获取s是否包含r
	return IndexRune(s, r) >= 0
}

// LastIndex 返回s中substr的最后一个实例的索引；如果s中不存在substr，则返回-1。
func LastIndex(s, substr string) int { // 注：获取s中最后一次出现substr的索引
	n := len(substr)
	switch {
	case n == 0:
		return len(s)
	case n == 1:
		return LastIndexByte(s, substr[0]) // 注：获取最后一次出现substr的索引
	case n == len(s):
		if substr == s { // 注：直接比较
			return 0
		}
		return -1
	case n > len(s):
		return -1
	}
	// 使用Rabin-Karp算法从字符串末尾搜索
	hashss, pow := hashStrRev(substr) // 注：获取substr的哈希与乘法因子
	last := len(s) - n                // 注：
	var h uint32
	for i := len(s) - 1; i >= last; i-- { // 注：计算s[len(s) - 1:len(s) - 1 + len(substr)]的哈希
		h = h*primeRK + uint32(s[i])
	}
	if h == hashss && s[last:] == substr { // 注：直接比较两者
		return last
	}
	// 例：s = "123456"，substr = "345"
	// 456的哈希									与345的哈希比较，false
	// 456的哈希 - 6的哈希 + 3的哈希 = 345的哈希	与345的哈希比较，true
	for i := last - 1; i >= 0; i-- {
		h *= primeRK
		h += uint32(s[i])
		h -= pow * uint32(s[i+n])
		if h == hashss && s[i:i+n] == substr {
			return i
		}
	}
	return -1
}

// IndexByte 返回s中c的第一个实例的索引；如果s中不存在c，则返回-1。
func IndexByte(s string, c byte) int { // 注：获取s中第第一次出现c的索引
	return bytealg.IndexByteString(s, c)
}

// IndexRune 返回Unicode代码点r的第一个实例的索引；如果s中不存在符文，则返回-1。
// 如果r为utf8.RuneError，它将返回任何无效UTF-8字节序列的第一个实例。
func IndexRune(s string, r rune) int { // 注：获取s中第一次出现r的索引
	switch {
	case 0 <= r && r < utf8.RuneSelf: // 注：如果r是单字节rune，直接比较
		return IndexByte(s, byte(r))
	case r == utf8.RuneError: // 注：如果r时错误rune，直接遍历
		for i, r := range s {
			if r == utf8.RuneError {
				return i
			}
		}
		return -1
	case !utf8.ValidRune(r): // 注：如果r不是合法rune，返回-1
		return -1
	default:
		return Index(s, string(r))
	}
}

// IndexAny 返回s中chars中任何Unicode代码点的第一个实例的索引；如果s中不存在chars中的Unicode代码点，则返回-1。
func IndexAny(s, chars string) int { // 注：获取s中出现chars的任意元素的位置
	if chars == "" {
		// 避免扫描所有。
		return -1
	}
	if len(s) > 8 {
		if as, isASCII := makeASCIISet(chars); isASCII { // 注：获取chars中连续的ASCII集合
			for i := 0; i < len(s); i++ { // 注：遍历s
				if as.contains(s[i]) { // 注：s中是否出现chars
					return i
				}
			}
			return -1
		}
	}
	for i, c := range s { // 注：如果s较小，直接遍历
		for _, m := range chars {
			if c == m {
				return i
			}
		}
	}
	return -1
}

// LastIndexAny 返回s中chars中任何Unicode代码点的最后一个实例的索引；如果s中不存在chars中的Unicode代码点，则返回-1。
func LastIndexAny(s, chars string) int { // 注：获取s中最后一次出现charts的任意元素的位置
	if chars == "" {
		// 避免扫描所有。
		return -1
	}
	if len(s) > 8 {
		if as, isASCII := makeASCIISet(chars); isASCII { // 注：获取chars中连续的ASCII集合
			for i := len(s) - 1; i >= 0; i-- { // 注：倒序遍历s
				if as.contains(s[i]) { // 注：s中是否出现chars
					return i
				}
			}
			return -1
		}
	}
	for i := len(s); i > 0; { // 注：如果s较小，直接遍历
		r, size := utf8.DecodeLastRuneInString(s[:i]) // 注：获取s的最后一个rune
		i -= size
		for _, c := range chars {
			if r == c {
				return i
			}
		}
	}
	return -1
}

// LastIndexByte 返回s中c的最后一个实例的索引；如果s中不存在c，则返回-1。
func LastIndexByte(s string, c byte) int { // 注：获取s中最后一个c出现的位置
	for i := len(s) - 1; i >= 0; i-- { // 注：倒序遍历，比较c
		if s[i] == c {
			return i
		}
	}
	return -1
}

// Generic split：在sep的每个实例之后进行拆分，包括在子数组中的sepSepSave字节。
func genSplit(s, sep string, sepSave, n int) []string { // 注：将s根据sep拆分为n份数据，每份数据额外读取sepSave字节数据
	// 例1：s = "111,222,333,444"，sep = ","，sepSave = 1
	// 返回[]string{"111,", "222,", "333,", "444"}
	//
	// 例2：s = "111,222,333,444"，sep = ","，sepSave = 2
	// 返回[]string{"111,2", "222,3", "333,4", "444"}
	if n == 0 {
		return nil
	}
	if sep == "" {
		return explode(s, n) // 注：将s拆分为n个rune
	}
	if n < 0 { // 注：默认n为s中所有rune的数量
		n = Count(s, sep) + 1
	}

	a := make([]string, n)
	n--
	i := 0
	for i < n { // 注：n - 1次
		m := Index(s, sep) // 注：获取s中第一次出现sep的索引
		if m < 0 {         // 注：如果s中没有sep，直接返回
			break
		}
		a[i] = s[:m+sepSave] // 注：暂存s分割后额外sepSave字节的数据
		s = s[m+len(sep):]
		i++
	}
	a[i] = s // 注：最后一次，暂存其余字符串
	return a[:i+1]
}

// SplitN 将s切片成由sep分隔的子字符串，并返回这些分隔符之间的子字符串的切片。
//
// 计数确定要返回的子字符串数：
// 	n > 0：最多n个子字符串； 最后一个子字符串将是未拆分的余数。
// 	n == 0：结果为nil（零子字符串）
// 	n < 0：所有子字符串
//
// s和sep的边沿大小写（例如，空字符串）的处理如Split文档中所述。
func SplitN(s, sep string, n int) []string { return genSplit(s, sep, 0, n) } // 注：将s根据sep拆分为n份数据

// SplitAfterN 在sep的每个实例之后将s切片为子字符串，并返回这些子字符串的切片。
//
// 计数确定要返回的子字符串数：
// 	n > 0：最多n个子字符串； 最后一个子字符串将是未拆分的余数。
// 	n == 0：结果为nil（零子字符串）
// 	n < 0：所有子字符串
//
// s和sep的边沿大小写（例如，空字符串）的处理方式如SplitAfter文档中所述。
func SplitAfterN(s, sep string, n int) []string { // 注：将s根据sep拆分为n份数据（保留sep）
	return genSplit(s, sep, len(sep), n)
}

// Split 将s切片为所有由sep分隔的子字符串，并返回这些分隔符之间的子字符串的切片。
// 如果s不包含sep且sep不为空，则Split返回长度为1的切片，其唯一元素为s。
// 如果sep为空，则Split在每个UTF-8序列之后拆分。 如果s和sep均为空，则Split返回一个空切片。
// 等于SplitN，计数为-1。
func Split(s, sep string) []string { return genSplit(s, sep, 0, -1) } // 注：将s根据sep拆分

// SplitAfter 在sep的每个实例之后将s切片为所有子字符串，并返回这些子字符串的切片。
// 如果s不包含sep且sep不为空，则SplitAfter返回长度为1的切片，其唯一元素为s。
// 如果sep为空，则SplitAfter在每个UTF-8序列之后拆分。 如果s和sep均为空，则SplitAfter返回一个空切片。
// 等于SplitAfterN，计数为-1。
//
// 例：fmt.Println(strings.SplitAfter("11,22",","))，返回[]string{"11,", "22"]
func SplitAfter(s, sep string) []string { // 注：将s根据sep拆分（保留sep）
	return genSplit(s, sep, len(sep), -1)
}

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1} // 注：ASCII中的空格字符

// Fields 根据unicode.IsSpace的定义，将字符串s围绕一个或多个连续的空白字符的每个实例进行拆分。
// IsSpace，返回s的子字符串切片，如果s仅包含空白，则返回空切片。
func Fields(s string) []string { // 注：将s根据一个或多个连续的空白字符进行拆分
	// 首先计算字段。
	// 如果s为ASCII，则为精确计数，否则为近似值。
	n := 0
	wasSpace := 1
	// setBits用于跟踪在s字节中设置了哪些位。
	setBits := uint8(0)
	for i := 0; i < len(s); i++ { // 注：遍历s
		r := s[i]
		setBits |= r
		isSpace := int(asciiSpace[r]) // 注：是否是空格
		n += wasSpace & ^isSpace      // 注：如果这个字符不是空格，计数+1
		wasSpace = isSpace
	}

	if setBits >= utf8.RuneSelf { // 注：多字节rune，？只需要判断setBits>>7 == 1
		// 输入字符串中的某些rune不是ASCII。
		return FieldsFunc(s, unicode.IsSpace) // 注：将s根据空格拆分
	}
	// ASCII快速路径
	a := make([]string, n)
	na := 0
	fieldStart := 0
	i := 0
	// 跳过输入前面的空格。
	for i < len(s) && asciiSpace[s[i]] != 0 {
		i++
	}
	fieldStart = i
	for i < len(s) { // 注：遍历s，根据空格分割的字符串
		if asciiSpace[s[i]] == 0 { // 注：不是空格，i+1
			i++
			continue
		}
		a[na] = s[fieldStart:i]
		na++
		i++
		// 在字段之间跳过空格。
		for i < len(s) && asciiSpace[s[i]] != 0 { // 注：跳过多个空格
			i++
		}
		fieldStart = i
	}
	if fieldStart < len(s) { // 最后一个字段可能以EOF结尾。
		a[na] = s[fieldStart:]
	}
	return a
}

// FieldsFunc 在满足f(c)的每次Unicode代码点c处分割字符串s，并返回s的切片数组。
// 如果s中的所有代码点都满足f(c)或字符串为空，则返回一个空切片。
// FieldsFunc不保证调用f(c)的顺序。
// 如果f对于给定的c没有返回一致的结果，则FieldsFunc可能会崩溃。
func FieldsFunc(s string, f func(rune) bool) []string { // 注：将s根据f(rune)拆分
	// span用于记录形式为s[start:end]的s的一部分。
	// 开始索引是包含的，结束索引是排他的。
	type span struct {
		start int
		end   int
	}
	spans := make([]span, 0, 32)

	// 例：s = "11 22 3"，f = unicode.IsSpace（判断rune是否为空格）
	// i = 0	s[i] = 1	f = false	fromIndex = 0			wasField = true
	// i = 1	s[i] = 1	f = false
	// i = 2	s[i] = ' '	f = true	span{start: 0, end: 2}	wasField = false
	// i = 3	s[i] = 2	f = false	fromIndex = 3			wasField = true
	// i = 4	s[i] = 2	f = false
	// i = 5	s[i] = ' '	f = true	span{start: 3, end: 5}	wasField = false
	// i = 6	s[i] = 3	f = false	span{start: 6, end: 6}	wasField = true
	// 查找字段的开始和结束索引。
	wasField := false
	fromIndex := 0
	for i, rune := range s { // 注：遍历s
		if f(rune) { // 注：执行f(rune)
			if wasField {
				spans = append(spans, span{start: fromIndex, end: i})
				wasField = false
			}
		} else {
			if !wasField {
				fromIndex = i
				wasField = true
			}
		}
	}

	// 最后一个字段可能以EOF结尾。
	if wasField {
		spans = append(spans, span{fromIndex, len(s)})
	}

	// 从记录的字段索引创建字符串。
	a := make([]string, len(spans))
	for i, span := range spans { // 注：遍历spans
		a[i] = s[span.start:span.end]
	}

	return a
}

// Join 连接其第一个参数的元素以创建单个字符串。 分隔符字符串sep放置在结果字符串中的元素之间。
func Join(elems []string, sep string) string { // 注：将elems合并为字符串，每个元素之间使用sep分隔
	switch len(elems) {
	case 0: // 注：如果没有元素，直接返回""
		return ""
	case 1: // 注：如果只有一个元素，返回第一个元素
		return elems[0]
	}
	n := len(sep) * (len(elems) - 1) // 注：计算空间
	for i := 0; i < len(elems); i++ {
		n += len(elems[i])
	}

	var b Builder
	b.Grow(n)                     // 注：保证b可以容纳n个字节
	b.WriteString(elems[0])       // 注：输出elems[0]
	for _, s := range elems[1:] { // 注：输出其他元素与分隔符
		b.WriteString(sep)
		b.WriteString(s)
	}
	return b.String()
}

// HasPrefix 测试字符串s是否以前缀开头。
func HasPrefix(s, prefix string) bool { // 注：获取s的前缀是否为prefix
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

// HasSuffix 测试字符串s是否以后缀结尾。
func HasSuffix(s, suffix string) bool { // 注：获取s的后缀是否为suffix
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// Map 返回字符串s的副本，其所有字符都根据映射函数进行了修改。 如果映射返回负值，则从字符串中删除该字符且不进行替换。
func Map(mapping func(rune) rune, s string) string { // 注：获取通过mapping映射后的s
	// 在最坏的情况下，字符串在映射时会增长，使事情变得不愉快。
	// 但是，我们很少敢假设它很好，所以很少见。 它也可以缩小，但这自然而然地消失了。

	// 输出缓冲区b根据需要进行初始化，这是字符第一次不同。
	var b Builder

	for i, c := range s { // 注：遍历s，获取第一个rune
		r := mapping(c)                    // 注：执行mapping
		if r == c && c != utf8.RuneError { // 注：如果映射后没有变化，略过
			continue
		}

		var width int
		if c == utf8.RuneError { // 注：如果c是错误rune，可能是多字节rune
			c, width = utf8.DecodeRuneInString(s[i:]) // 注：获取完整的rune
			if width != 1 && r == c {                 // 注：如果还是错误rune，略过
				continue
			}
		} else {
			width = utf8.RuneLen(c)
		}

		b.Grow(len(s) + utf8.UTFMax)
		b.WriteString(s[:i]) // 注：缓冲区写入字符串
		if r >= 0 {          // 注：缓冲区写入rune
			b.WriteRune(r)
		}

		s = s[i+width:] // 注：截取s
		break
	}

	// 快速路径，保持输入不变
	if b.Cap() == 0 { // 以上没有调用 b.Grow
		return s
	}

	for _, c := range s { // 注：遍历s，获取所有rune
		r := mapping(c) // 注：执行mapping

		if r >= 0 {
			// 常见情况由于有内联，因此确定是否应调用WriteByte而不是始终调用WriteRune的性能更高
			if r < utf8.RuneSelf {
				b.WriteByte(byte(r))
			} else {
				// r不是ASCII符文。
				b.WriteRune(r)
			}
		}
	}

	return b.String()
}

// Repeat 返回一个由字符串s的计数副本组成的新字符串。
//
// 如果count为负或(len(s) * count)的结果溢出，则表示恐慌。
func Repeat(s string, count int) string { // 注：获取重复count次的s
	// 例：s = "abc"，count = "5"
	// n = 15
	// 第1次循环："abcabc" // b += b
	// 第2次循环："abcabcabcabc" // b += b
	// 第2次循环："abcabcabcabcabc" // 截断
	if count == 0 {
		return ""
	}
	// 由于我们无法在溢出时返回错误，因此如果重复操作会产生溢出，我们应该惊慌。
	// 参见问题golang.org/issue/16237
	if count < 0 {
		panic("strings: negative Repeat count") // 恐慌："重复计数为负"
	} else if len(s)*count/count != len(s) {
		panic("strings: Repeat count causes overflow") // 恐慌："重复计数导致溢出"
	}

	n := len(s) * count
	var b Builder
	b.Grow(n)
	b.WriteString(s)  // 注：写入s
	for b.Len() < n { // 注：遍历count次
		if b.Len() <= n/2 { // 注：如果b填充不足一半，直接赋b += b
			b.WriteString(b.String())
		} else { // 注：否则截断填充
			b.WriteString(b.String()[:n-b.Len()])
			break
		}
	}
	return b.String()
}

// ToUpper 返回s所有Unicode字母均映射为大写。
func ToUpper(s string) string { // 注：将s转为大写
	isASCII, hasLower := true, false
	for i := 0; i < len(s); i++ { // 注：遍历s
		c := s[i]
		if c >= utf8.RuneSelf { // 注：是否全是ASCII
			isASCII = false
			break
		}
		hasLower = hasLower || ('a' <= c && c <= 'z') // 注：是否均为小写字母
	}

	if isASCII { // 针对仅ASCII字符串进行优化。
		if !hasLower { // 注：如果均为小写字母，直接返回
			return s
		}
		var b Builder
		b.Grow(len(s))
		for i := 0; i < len(s); i++ { // 注：遍历s，每个元素-32，97 - 65 = 32
			c := s[i]
			if 'a' <= c && c <= 'z' {
				c -= 'a' - 'A'
			}
			b.WriteByte(c)
		}
		return b.String()
	}
	return Map(unicode.ToUpper, s) // 注：多字节rune进行映射
}

// ToLower 返回，其中所有Unicode字母均映射为小写。
func ToLower(s string) string { // 注：将s转为小写（同上）
	isASCII, hasUpper := true, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= utf8.RuneSelf {
			isASCII = false
			break
		}
		hasUpper = hasUpper || ('A' <= c && c <= 'Z')
	}

	if isASCII { // optimize for ASCII-only strings.
		if !hasUpper {
			return s
		}
		var b Builder
		b.Grow(len(s))
		for i := 0; i < len(s); i++ {
			c := s[i]
			if 'A' <= c && c <= 'Z' {
				c += 'a' - 'A'
			}
			b.WriteByte(c)
		}
		return b.String()
	}
	return Map(unicode.ToLower, s)
}

// ToTitle 返回字符串s的副本，其中所有Unicode字母都映射到其Unicode标题大小写。
func ToTitle(s string) string { return Map(unicode.ToTitle, s) } // 注：将s转为标题大小写

// ToUpperSpecial 返回字符串s的副本，其中所有Unicode字母都使用c指定的大小写映射映射到其大写字母。
func ToUpperSpecial(c unicode.SpecialCase, s string) string { // 注：将s通过c的特殊映射转为大写
	return Map(c.ToUpper, s)
}

// ToLowerSpecial 返回字符串s的副本，其中所有Unicode字母均使用c指定的大小写映射映射到其小写。
func ToLowerSpecial(c unicode.SpecialCase, s string) string { // 注：将s通过c的特殊映射转为小写
	return Map(c.ToLower, s)
}

// ToTitleSpecial 返回字符串s的副本，其中所有Unicode字母都映射到其Unicode标题大小写，并优先使用特殊的大小写规则。
func ToTitleSpecial(c unicode.SpecialCase, s string) string { // 注：将s通过c的特殊映射转为标题大小写
	return Map(c.ToTitle, s)
}

// ToValidUTF8 返回字符串s的副本，其中每次运行的无效UTF-8字节序列都被替换字符串替换，替换字符串可能为空。
func ToValidUTF8(s, replacement string) string { // 注：获取将无效rune替换为replacement的s
	var b Builder

	for i, c := range s { // 注：遍历s
		if c != utf8.RuneError { // 注：略过所有错误rune
			continue
		}

		_, wid := utf8.DecodeRuneInString(s[i:]) // 注：如果是错误rune，可能是多字节rune，获取rune
		if wid == 1 {                            // 注：如果确实是错误rune，略过
			b.Grow(len(s) + len(replacement))
			b.WriteString(s[:i])
			s = s[i:]
			break
		}
	}

	// 快速路径，保持输入不变
	if b.Cap() == 0 { // 没有调用b.Grow
		return s
	}

	invalid := false          // 前一个字节来自无效的UTF-8序列
	for i := 0; i < len(s); { // 注：遍历s
		c := s[i]
		if c < utf8.RuneSelf { // 注：单字节rune
			i++
			invalid = false
			b.WriteByte(c)
			continue
		}
		_, wid := utf8.DecodeRuneInString(s[i:]) // 注：多字节rune
		if wid == 1 {                            // 注：如果多字节rune占用1字节，出现无效rune
			i++
			if !invalid { // 注：使用replacement替换连续的无效rune
				invalid = true
				b.WriteString(replacement)
			}
			continue
		}
		invalid = false
		b.WriteString(s[i : i+wid]) // 注：写入正常字符串
		i += wid
	}

	return b.String()
}

// isSeparator 报告rune是否可以标记单词边界。
// TODO：在程序包unicode捕获更多属性时更新。
func isSeparator(r rune) bool { // 注：获取r是否为分隔符
	// 注：除字母、数字、下划线外的ASCII均为分隔符
	// 注：空格为分隔符

	// ASCII字母数字和下划线不是分隔符
	if r <= 0x7F { // 注：r是ASCII
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

// Title 返回带有所有Unicode字母的字符串s的副本，这些字母以映射到其Unicode标题大小写的单词开头。
// BUG（rsc）：Title用于单词边界的规则不能正确处理Unicode标点符号。
func Title(s string) string { // 注：获取s的标题格式
	// 例：s = "1 234 abc"
	// prev = ' '	r = 1	返回标题大小写
	// prev = '1'	r = ' '	返回r
	// prev = ' '	r = 2	返回标题大小写
	// prev = '2'	r = 3	返回标题大小写
	// prev = '3'	r = 4	返回标题大小写
	// prev = '4'	r = ' '	返回r
	// prev = ' '	r = a	返回标题大小写
	// prev = 'a'	r = b	返回标题大小写
	// prev = 'b'	r = c	返回标题大小写
	// 返回"1 234 Abc"

	// 在此处使用闭包来记住状态。
	// 骇人但有效。 顺序依赖于地图扫描，每个rune调用一次关闭。
	prev := ' '
	return Map(
		func(r rune) rune {
			if isSeparator(prev) { // 注：如果prev是分隔符，转为标题大小写
				prev = r
				return unicode.ToTitle(r)
			}
			prev = r
			return r
		},
		s)
}

// TrimLeftFunc 返回字符串s的一部分，并删除所有满足f(c)的前导Unicode代码点c。
func TrimLeftFunc(s string, f func(rune) bool) string { // 注：获取截断符合f(r)为false的前缀的s
	// 例：s = "123456"，f = 参数为2返回false
	// 返回"23456"
	i := indexFunc(s, f, false) // 注：获取s中第一个f(r)为false的索引
	if i == -1 {
		return ""
	}
	return s[i:] // 注：截断索引之前的数据
}

// TrimRightFunc 返回字符串s的一部分，并删除所有满足f(c)的尾随Unicode代码点c。
func TrimRightFunc(s string, f func(rune) bool) string { // 注：获取截断符合f(r)为false的后缀的s
	// 例：s = "123456"，f = 参数为5返回false
	// 返回"12345"
	i := lastIndexFunc(s, f, false)      // 注：获取s中最后一个符合f(r) == false的索引
	if i >= 0 && s[i] >= utf8.RuneSelf { // 注：如果为false的rune是多字节rune，截断之后的数据
		_, wid := utf8.DecodeRuneInString(s[i:])
		i += wid
	} else {
		i++
	}
	return s[0:i]
}

// TrimFunc 返回字符串s的一部分，并删除所有满足f(c)的前导和尾随Unicode代码点c。
func TrimFunc(s string, f func(rune) bool) string { // 注：获取截断符合f(r)为false的前缀与后缀的s
	// 例：s = "12345654321"，f = 参数为2返回false
	// 返回"234565432"
	return TrimRightFunc(TrimLeftFunc(s, f), f)
}

// IndexFunc 返回满足f（c）的第一个Unicode代码点的s的索引；如果不满足，则返回-1。
func IndexFunc(s string, f func(rune) bool) int { // 注：获取s中第一个符合f(r)为true的索引
	return indexFunc(s, f, true)
}

// LastIndexFunc 将索引的最后一个满足f(c)的Unicode代码点的s返回；如果没有，则返回-1。
func LastIndexFunc(s string, f func(rune) bool) int { // 注：获取s中最后一个符合f(r)为true的索引
	return lastIndexFunc(s, f, true)
}

// indexFunc 与IndexFunc相同，不同之处在于如果true == false，则谓词函数的含义相反。
func indexFunc(s string, f func(rune) bool, truth bool) int { // 注：获取s中第一个符合f(r) == truth的索引
	for i, r := range s { // 注：遍历s，返回第一个f(r) == truth的索引
		if f(r) == truth {
			return i
		}
	}
	return -1
}

// lastIndexFunc 与LastIndexFunc相同，不同之处在于如果true == false，则谓词函数的含义相反。
func lastIndexFunc(s string, f func(rune) bool, truth bool) int { // 注：获取s中最后一个符合f(r) == truth的索引
	for i := len(s); i > 0; {
		r, size := utf8.DecodeLastRuneInString(s[0:i])
		i -= size
		if f(r) == truth {
			return i
		}
	}
	return -1
}

// asciiSet 是一个32字节的值，其中每个位代表该集中存在给定的ASCII字符。
// 从最低字的最低有效位到最高字的最高有效位，低16个字节的128位映射到所有128个ASCII字符的整个范围。
// 高16个字节的128位将被清零，以确保任何非ASCII字符都将报告为不在集合中。
//
// 注：将ASCII编码根据前3位分组，可以分出4组，将后5位格式化为每组的对应元素位置
// 注：ASCII >> 5只能得出00、01、10、11，asciiSet[ASCII>>5]只能索引4个元素
// 注：asciiSet的4个元素 * uint32的32位 = 128，可以覆盖所有ASCII编码
// 注：注释错了还是我错了？
type asciiSet [8]uint32

// makeASCIISet 创建一组ASCII字符并报告char中的所有字符是否均为ASCII。
func makeASCIISet(chars string) (as asciiSet, ok bool) { // 注：获取chars中连续的ASCII的编码集合
	// 例：charts = "abc"
	// c = "a"	97（0110 0001）	31（0001 1111）	as[3] |= 1 << 1		as[3] = 2（0000 0010）
	// c = "b"	98（0110 0010）	31（0001 1111）	as[3] |= 1 << 2		as[3] = 6（0000 0110）
	// c = "c"	99（0110 0011）	31（0001 1111）	as[3] |= 1 << 3		as[3] = 15（0000 1110）

	for i := 0; i < len(chars); i++ { // 注：遍历chars
		c := chars[i]
		if c >= utf8.RuneSelf { // 注：如果大于单字节rune，肯定不是ASCII，直接返回
			return as, false
		}
		as[c>>5] |= 1 << uint(c&31)
	}
	return as, true
}

// contains 报告c是否在集合内。
func (as *asciiSet) contains(c byte) bool { // 注：获取c是否在as内
	return (as[c>>5] & (1 << uint(c&31))) != 0 // 注：算法参见asciiSet说明
}

func makeCutsetFunc(cutset string) func(rune) bool { // 注：返回cutset是否包含rune的方法
	if len(cutset) == 1 && cutset[0] < utf8.RuneSelf { // 注：如果cutset是ASCII，返回直接比较r的方法
		return func(r rune) bool {
			return r == rune(cutset[0])
		}
	}
	if as, isASCII := makeASCIISet(cutset); isASCII { // 注：如果是多个ASCII，返回cutset是否包含rune的方法
		return func(r rune) bool {
			return r < utf8.RuneSelf && as.contains(byte(r))
		}
	}
	return func(r rune) bool { return IndexRune(cutset, r) >= 0 } // 注：如果是多字节rune，返回cutset是否包含rune的方法
}

// Trim 返回字符串s的一部分，并删除所有满足f(c)的前导和尾随Unicode代码点c。...
func Trim(s string, cutset string) string { // 注：获取截取符合cutset包含的前缀与后缀的s
	if s == "" || cutset == "" {
		return s
	}
	return TrimFunc(s, makeCutsetFunc(cutset)) // 注：s去掉符合 cutset是否包含rune 的前缀与后缀
}

// TrimLeft 返回字符串s的一部分，其中cutset中包含的所有前导Unicode代码点均已删除。
//
// 要删除前缀，请改用TrimPrefix。
func TrimLeft(s string, cutset string) string { // 注：获取截取符合cutset包含的前缀的s（前缀为一个rune）
	if s == "" || cutset == "" {
		return s
	}
	return TrimLeftFunc(s, makeCutsetFunc(cutset))
}

// TrimRight 返回字符串s的一部分，其中cutset中包含的所有尾随Unicode代码点均被删除。
//
// 要删除后缀，请改用TrimSuffix。
func TrimRight(s string, cutset string) string { // 注：获取截取符合cutset包含的后缀的s（后缀为一个rune）
	if s == "" || cutset == "" {
		return s
	}
	return TrimRightFunc(s, makeCutsetFunc(cutset))
}

// TrimSpace 返回字符串s的一部分，其中所有前导和尾随空格都已删除，这是由Unicode定义的。
func TrimSpace(s string) string { // 注：获取截取前后空格的s
	// ASCII的快速路径：查找第一个ASCII非空格字节
	start := 0
	for ; start < len(s); start++ { // 注：遍历s，找到不为空格的索引
		c := s[start]
		if c >= utf8.RuneSelf { // 注：多字节rune
			// 如果遇到非ASCII字节，请在剩下的字节上使用较慢的unicode-aware方法
			return TrimFunc(s[start:], unicode.IsSpace) // 注：返回截取前后缀的空格的s
		}
		if asciiSpace[c] == 0 {
			break
		}
	}

	// 现在从末尾查找第一个ASCII非空格字节
	stop := len(s)
	for ; stop > start; stop-- { // 注：倒序遍历s，找到不为空格的索引
		c := s[stop-1]
		if c >= utf8.RuneSelf { // 注：多字节rune
			return TrimFunc(s[start:stop], unicode.IsSpace) // 注：返回截取后缀空格的s
		}
		if asciiSpace[c] == 0 {
			break
		}
	}

	// 至此，s[start:stop]以ASCII非空格字节开始和结束，到此完成。 上面已经处理了非ASCII情况。
	return s[start:stop]
}

// TrimPrefix 返回s，但不提供提供的前导前缀字符串。
// 如果s不以prefix开头，则s不变返回。
func TrimPrefix(s, prefix string) string { // 注：获取去掉prefix前缀的s
	if HasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

// TrimSuffix 返回s，但不提供尾随的后缀字符串。
// 如果s不以suffix结尾，则s保持不变。
func TrimSuffix(s, suffix string) string { // 注：获取去掉suffix后缀的s
	if HasSuffix(s, suffix) {
		return s[:len(s)-len(suffix)]
	}
	return s
}

// Replace 返回字符串s的副本，其中前n个不重叠的旧实例被new替换。
// 如果old为空，则它在字符串的开头和每个UTF-8序列之后匹配，最多产生k个符文字符串的k + 1个替换。
// 如果n < 0，则替换次数没有限制。
func Replace(s, old, new string, n int) string { // 注：获取将前n次old替换为new的s
	// 例1：strings.Replace("123456", "", "a", 3)
	// i = 0	j = 0	t = "a"			start = 0
	// i = 1	j = 1	t = "a1a"		start = 1
	// i = 2	j = 2	t = "a1a2a"		start = 2
	// 返回"a1a2a3456"
	//
	// 例2：strings.Replace("aaaaaa", "a", "b", 3)
	// i = 0	j = 0	t = "b"			start = 0
	// i = 1	j = 0	t = "bb"		start = 1
	// i = 2	j = 0	t = "bbb"		start = 2
	// 返回"bbbaaa"

	if old == new || n == 0 {
		return s // 避免分配
	}

	// 计算替换次数。
	if m := Count(s, old); m == 0 { // 注：s中没有old，直接返回s
		return s // 避免分配
	} else if n < 0 || m < n {
		n = m
	}

	// 将替换应用于缓冲区。
	t := make([]byte, len(s)+n*(len(new)-len(old)))
	w := 0
	start := 0
	for i := 0; i < n; i++ { // 注：遍历n次
		j := start
		if len(old) == 0 { // 注：如果old为空，跳过这个rune
			if i > 0 {
				_, wid := utf8.DecodeRuneInString(s[start:])
				j += wid
			}
		} else { // 注：如果old不为空，跳过old之前的数据
			j += Index(s[start:], old)
		}
		w += copy(t[w:], s[start:j]) // 注：输出old，输出的是上一个rune
		w += copy(t[w:], new)        // 注：输出new
		start = j + len(old)
	}
	w += copy(t[w:], s[start:])
	return string(t[0:w])
}

// ReplaceAll 返回字符串s的副本，其中所有旧的非重叠实例都被new替换。
// 如果old为空，则它在字符串的开头和每个UTF-8序列之后匹配，最多产生k个符文字符串的k + 1个替换。
func ReplaceAll(s, old, new string) string { // 注：获取将old替换为new的s
	return Replace(s, old, new, -1)
}

// EqualFold 报告在Unicode大小写折叠下s和t（解释为UTF-8字符串）是否相等，这是不区分大小写的更通用形式。
func EqualFold(s, t string) bool { // 注：获取s与t是否相等
	for s != "" && t != "" {
		// 从每个字符串中提取第一个rune。
		var sr, tr rune
		if s[0] < utf8.RuneSelf { // 注：获取s的第一个rune
			sr, s = rune(s[0]), s[1:]
		} else {
			r, size := utf8.DecodeRuneInString(s)
			sr, s = r, s[size:]
		}
		if t[0] < utf8.RuneSelf { // 注：获取t的第一个rune
			tr, t = rune(t[0]), t[1:]
		} else {
			r, size := utf8.DecodeRuneInString(t)
			tr, t = r, t[size:]
		}

		// 如果他们匹配，继续前进； 如果不是，则返回false。

		// 简单的情况。
		if tr == sr { // 注：rune相等，略过
			continue
		}

		// 使sr < tr简化以下内容。
		if tr < sr {
			tr, sr = sr, tr
		}
		// 快速检查ASCII。
		if tr < utf8.RuneSelf { // 注：比较ASCII，不区分大小写
			// 仅ASCII，sr/tr必须为大写/小写
			if 'A' <= sr && sr <= 'Z' && tr == sr+'a'-'A' {
				continue
			}
			return false
		}

		// 一般情况。 SimpleFold(x)返回下一个等效的rune > x或环绕到较小的值。
		r := unicode.SimpleFold(sr)
		for r != sr && r < tr { // 注：折叠
			r = unicode.SimpleFold(r)
		}
		if r == tr {
			continue
		}
		return false
	}

	// 一个字符串为空。 都是吗
	return s == t
}

// Index 返回s中第一个substr实例的索引；如果s中不存在substr，则返回-1。
func Index(s, substr string) int { // 注：获取s中第一个substr的索引
	// substr字符串较小时，使用蛮力搜索
	// substr字符串较大时，使用Rabin-Karp算法搜索
	n := len(substr)
	switch {
	case n == 0:
		return 0
	case n == 1: // 注：如果substr是一个字节
		return IndexByte(s, substr[0]) // 注：获取s中第一个substr的索引
	case n == len(s): // 注：如果s与substr长度相等，直接判断
		if substr == s {
			return 0
		}
		return -1
	case n > len(s): // 注：如果substr比s长，不可能找到索引
		return -1
	case n <= bytealg.MaxLen: // 注：小于最大搜索长度
		// 当s和substr都较小时使用蛮力
		if len(s) <= bytealg.MaxBruteForce { // 注：字符串较小时使用蛮力进行搜索
			return bytealg.IndexString(s, substr)
		}
		c0 := substr[0]
		c1 := substr[1]
		i := 0
		t := len(s) - n + 1
		fails := 0
		// 例：s = "123456"，substr = "789"，只需要比较123、234、345、456，不需要比较56
		for i < t { // 注：遍历s
			if s[i] != c0 { // 注：如果第一个字符不匹配，找到匹配第一个字符的索引，偏移到对应位置
				// IndexByte比bytealg.IndexString快，因此只要我们没有太多误报，就可以使用它。
				o := IndexByte(s[i:t], c0)
				if o < 0 {
					return -1
				}
				i += o
			}
			if s[i+1] == c1 && s[i:i+n] == substr { // 注：第1、2个字符相同，比较所有字符
				return i
			}
			fails++ // 注：失败次数+1
			i++
			// 当IndexByte产生过多的误报时，请切换到bytealg.IndexString。
			if fails > bytealg.Cutover(i) { // 注：如果失败次数超过容忍次数，进行最后一次全文搜索
				r := bytealg.IndexString(s[i:], substr)
				if r >= 0 {
					return r + i
				}
				return -1
			}
		}
		return -1
	}
	// 大于最小搜索长度，不使用蛮力进行搜索
	c0 := substr[0]
	c1 := substr[1]
	i := 0
	t := len(s) - n + 1
	fails := 0
	// 例：s = "123456"，substr = "789"，只需要比较123、234、345、456，不需要比较56
	for i < t { // 注：遍历s
		if s[i] != c0 { // 注：如果第一个字符不匹配，找到匹配第一个字符的索引，偏移到对应位置
			o := IndexByte(s[i:t], c0)
			if o < 0 {
				return -1
			}
			i += o
		}
		if s[i+1] == c1 && s[i:i+n] == substr { // 注：第1、2个字符相同，比较所有字符
			return i
		}
		i++
		fails++
		if fails >= 4+i>>4 && i < t { // 注：如果失败次数超过容忍次数，使用Rabin-Karp算法进行搜索
			// See comment in ../bytes/bytes.go.
			j := indexRabinKarp(s[i:], substr)
			if j < 0 {
				return -1
			}
			return i + j
		}
	}
	return -1
}

func indexRabinKarp(s, substr string) int { // 注：使用Rabin-Karp搜索s中substr的索引
	// Rabin-Karp搜索
	hashss, pow := hashStr(substr) // 注：计算substr的哈希与乘法因子
	n := len(substr)
	var h uint32
	for i := 0; i < n; i++ { // 注：计算s[0:len(substr)]的哈希
		h = h*primeRK + uint32(s[i])
	}
	if h == hashss && s[:n] == substr { // 注：比较两者的哈希
		return 0
	}

	// 例：s = "123456"，substr = "345"
	// 123的哈希									与345的哈希比较，false
	// 123的哈希 - 1的哈希 + 4的哈希 = 234的哈希	与345的哈希比较，false
	// 234的哈希 - 2的哈希 + 5的哈希 = 345的哈希	与345的哈希比较，true
	for i := n; i < len(s); {
		h *= primeRK
		h += uint32(s[i])
		h -= pow * uint32(s[i-n])
		i++
		if h == hashss && s[i-n:i] == substr {
			return i - n
		}
	}
	return -1
}
