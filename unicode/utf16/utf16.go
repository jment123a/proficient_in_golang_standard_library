// 版权所有2010 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

// Package utf16 实现UTF-16序列的编码和解码。
package utf16

// 在测试中验证了条件replaceChar == unicode.ReplacementChar和maxRune == unicode.MaxRune。
// 在本地定义它们可以避免此软件包，具体取决于软件包的unicode。

const (
	replacementChar = '\uFFFD'     // Unicode替换字符，注：错误的unicode
	maxRune         = '\U0010FFFF' // 最大有效Unicode代码点。注：rune的最大值
)

const (
	// 0xd800-0xdc00编码一对的高10位。
	// 0xdc00-0xe000编码一对的低10位。
	// 值是那20位加0x10000。
	//
	// 注：一个utf-16字符由(utf-16高位-surr1)<<10 | (utf-16低位 - surr2) + surrSelf组成
	// 其中surr1 <= utf-16高位 < surr2 <= utf-16低位 <= surr3
	// 一个utf-16高位与utf-16低位被成为一组unicode代理对
	// utf16 = (utf-16高位 - surr1)<<10 | (utf-16低位 - surr2) + surrSelf
	surr1 = 0xd800 // 注：utf-16高位的最小值
	surr2 = 0xdc00 // 注：utf-16高位的最大值，utf-16低位的最小值
	surr3 = 0xe000 // 注：utf-16低位的最大值

	surrSelf = 0x10000 // 注：代理对的基数
)

// IsSurrogate 报告指定的Unicode代码点是否可以出现在代理对中。
func IsSurrogate(r rune) bool { // 注：获取r是否可以出现在代理对中
	return surr1 <= r && r < surr3
}

// DecodeRune 返回代理对的UTF-16解码。
// 如果该对不是有效的UTF-16代理对，则DecodeRune返回Unicode替换代码点U + FFFD。
func DecodeRune(r1, r2 rune) rune { // 注：获取r1作为代码对高位，r2作为代码对低位计算出的utf-16字符
	if surr1 <= r1 && r1 < surr2 && surr2 <= r2 && r2 < surr3 {
		return (r1-surr1)<<10 | (r2 - surr2) + surrSelf
	}
	return replacementChar
}

// EncodeRune 返回给定符文的UTF-16代理对r1，r2。
// 如果该符文不是有效的Unicode代码点或不需要编码，则EncodeRune返回U + FFFD，U + FFFD。
func EncodeRune(r rune) (r1, r2 rune) { // 注：获取utf-16编码r的代理对，高位r1，低位r2
	if r < surrSelf || r > maxRune { // 注：如果r小于最小的utf-16或r大于rune的最大值，返回替代字符
		return replacementChar, replacementChar
	}
	r -= surrSelf
	return surr1 + (r>>10)&0x3ff, surr2 + r&0x3ff // 注：0011 1111 1111，返回r的前10位与r的后10位
}

// Encode 返回Unicode代码点序列s的UTF-16编码。
func Encode(s []rune) []uint16 { // 注：将s中的utf-16拆分为utf-8编码并返回
	n := len(s)
	for _, v := range s { // 注：遍历s，获取utf-16字符的数量
		if v >= surrSelf {
			n++
		}
	}

	a := make([]uint16, n)
	n = 0
	for _, v := range s { // 注：遍历s
		switch {
		case 0 <= v && v < surr1, surr3 <= v && v < surrSelf: // 注：utf-8
			// 普通rune
			a[n] = uint16(v)
			n++
		case surrSelf <= v && v <= maxRune: // 注：utf-16
			// 需要替代序列
			r1, r2 := EncodeRune(v) // 注：拆分为代理对
			a[n] = uint16(r1)
			a[n+1] = uint16(r2)
			n += 2
		default: // 注：不是unicode
			a[n] = uint16(replacementChar)
			n++
		}
	}
	return a[:n]
}

// Decode 返回由UTF-16编码s表示的Unicode代码点序列。
func Decode(s []uint16) []rune { // 注：将s中的utf-8合并为utf-16编码并返回
	a := make([]rune, len(s))
	n := 0
	for i := 0; i < len(s); i++ { // 注：遍历s
		switch r := s[i]; {
		case r < surr1, surr3 <= r: // 注：utf-8
			// 普通rune
			a[n] = rune(r)
		case surr1 <= r && r < surr2 && i+1 < len(s) && // 注：utf-16，条件为surr1 <= utf-8高位 < surr2 <= utf-8低位 <= surc
			surr2 <= s[i+1] && s[i+1] < surr3:
			// 有效替代序列
			a[n] = DecodeRune(rune(r), rune(s[i+1]))
			i++
		default:
			// 无效的替代序列
			a[n] = replacementChar
		}
		n++
	}
	return a[:n]
}
