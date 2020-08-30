// 版权所有2011 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package unicode

//  U+0100下每个代码点的位掩码，用于快速查找。
const (
	pC     = 1 << iota // 控制字符。
	pP                 // 标点符号。
	pN                 // 数字。
	pS                 // 符号字符。
	pZ                 // 空格字符。
	pLu                // 大写字母。
	pLl                // 小写字母。
	pp                 // 根据Go的定义可打印的字符。注：1000 0000
	pg     = pp | pZ   // 根据Unicode定义的图形字符，注：1001 0000
	pLo    = pLl | pLu // 一个既不大写也不小写的字母。0110 0000
	pLmask = pLo       // 注：0110 0000
)

// GraphicRanges 根据Unicode定义图形字符集。
var GraphicRanges = []*RangeTable{ // 注：图形字符集
	L, M, N, P, S, Zs,
}

// PrintRanges 根据Go定义可打印字符集。
// ASCII空间，U+0020，单独处理。
var PrintRanges = []*RangeTable{ // 注：可以打印的字符集
	L, M, N, P, S,
}

// IsGraphic 报告符文是否已被Unicode定义为"图形"
// 此类字符包括字母L，M，N，P，S，Zs中的字母，标记，数字，标点符号，符号和空格。
func IsGraphic(r rune) bool { // 注：获取r是否为图形字符（字符集：L, M, N, P, S, Zs）
	// 我们转换为uint32以避免对负数进行额外测试，在索引中，我们转换为uint8以避免范围检查。
	if uint32(r) <= MaxLatin1 {
		return properties[uint8(r)]&pg != 0 // 注：10010000
	}
	return In(r, GraphicRanges...) // 注：获取r是否在图形字符集内
}

// IsPrint 报告该rune是否被Go定义为可打印。
// 这些字符包括字母，标记，数字，标点符号，符号以及ASCII空格字符，
// 它们来自类别L，M，N，P，S和ASCII空格字符。
// 此分类与IsGraphic相同，除了唯一的空格字符是ASCII空格U+0020.
func IsPrint(r rune) bool { // 注：获取r是否为可以打印字符（字符集：L, M, N, P, S）
	if uint32(r) <= MaxLatin1 {
		return properties[uint8(r)]&pp != 0 // 注：10000000
	}
	return In(r, PrintRanges...) // 注：获取r是否在打印字符集内
}

// IsOneOf 报告该rune是否为范围之一的成员。
// 函数"In"提供更好的签名，应优先于IsOneOf使用。
func IsOneOf(ranges []*RangeTable, r rune) bool { // 注：获取r是否在ranges范围内（同In）
	for _, inside := range ranges {
		if Is(inside, r) {
			return true
		}
	}
	return false
}

// In 报告该rune是否为ranges的成员。
func In(r rune, ranges ...*RangeTable) bool { // 注：获取r是否在ranges范围内
	for _, inside := range ranges { // 注：遍历ranges，返回r是否在inside范围内
		if Is(inside, r) {
			return true
		}
	}
	return false
}

// IsControl 报告rune是否为控制字符。
// C（其他）Unicode类别包含更多代码点，例如代理； 使用Is(C，r)进行测试。
func IsControl(r rune) bool { // 注：获取r是否为控制字符
	if uint32(r) <= MaxLatin1 {
		return properties[uint8(r)]&pC != 0
	}
	// 所有控制字符均为 < MaxLatin1。
	return false
}

// IsLetter 报告rune是否为字母（类别L）。
func IsLetter(r rune) bool { // 注：获取r是否为字母（字符集：L）
	if uint32(r) <= MaxLatin1 {
		return properties[uint8(r)]&(pLmask) != 0
	}
	return isExcludingLatin(Letter, r)
}

// IsMark 报告rune是否为标记字符（类别M）。
func IsMark(r rune) bool { // 注：获取r是否为标记（字符集：M）
	// Latin-1中没有标记字符。
	return isExcludingLatin(Mark, r)
}

// IsNumber 报告rune是否为数字（类别N）。
func IsNumber(r rune) bool { // 注：获取r是否为标记（字符集：N）
	if uint32(r) <= MaxLatin1 {
		return properties[uint8(r)]&pN != 0
	}
	return isExcludingLatin(Number, r)
}

// IsPunct 报告rune是否为Unicode标点字符（类别P）。
func IsPunct(r rune) bool { // 注：获取r是否为标点（字符集：P）
	if uint32(r) <= MaxLatin1 {
		return properties[uint8(r)]&pP != 0
	}
	return Is(Punct, r)
}

// IsSpace 报告符文是否为Unicode的White Space属性定义的空格字符； 在Latin-1空间中，这是
// '\t', '\n', '\v', '\f', '\r', ' ', U+0085 (NEL), U+00A0 (NBSP).
// 间隔字符的其他定义由类别Z和属性Pattern_White_Space设置。
func IsSpace(r rune) bool { // 注：#获取r是否为空格
	// 此属性与Z不同。 特例吧。
	if uint32(r) <= MaxLatin1 { // 注：如果r是Latin-1编码字符，直接判断是否为空格
		switch r {
		case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0:
			return true
		}
		return false
	}
	return isExcludingLatin(White_Space, r) // 注：#
}

// IsSymbol 报告rune是否为符号字符。
func IsSymbol(r rune) bool { // 注：获取r是否为符号（字符集：S）
	if uint32(r) <= MaxLatin1 {
		return properties[uint8(r)]&pS != 0
	}
	return isExcludingLatin(Symbol, r)
}
