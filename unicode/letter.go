// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package unicode provides data and functions to test some properties of
// Unicode code points.
package unicode

const (
	MaxRune         = '\U0010FFFF' // 最大有效Unicode代码点。
	ReplacementChar = '\uFFFD'     // 表示无效的代码点。
	MaxASCII        = '\u007F'     // 最大ASCII值。
	MaxLatin1       = '\u00FF'     // Latin-1的最大值。
)

// RangeTable 通过列出一组Unicode代码点的范围来定义它。
// 为了节省空间，在两个切片中列出了范围：切片的16位范围和切片的32位范围。
// 这两个切片必须按排序顺序且不重叠。
// 同样，R32应该只包含>= 0x10000 (1<<16)的值
type RangeTable struct { // 注：字符集范围集合
	R16         []Range16
	R32         []Range32
	LatinOffset int // Hi <= MaxLatin1的R16中的条目数
}

// Range16 表示一系列16位Unicode代码点。
// 范围从Lo到Hi（含），并具有指定的跨度。
type Range16 struct { // 注：字符集16位范围集合
	Lo     uint16
	Hi     uint16
	Stride uint16
}

// Range32 表示一系列Unicode代码点，并且当一个或多个值不适合16位时使用。
// 范围从Lo到Hi（含），并具有指定的跨度。 Lo和Hi必须始终为> = 1 << 16。
type Range32 struct { // 注：字符集32位范围集合
	Lo     uint32
	Hi     uint32
	Stride uint32
}

// CaseRange 表示一系列Unicode代码点，用于简单（一个代码点到一个代码点）的大小写转换。
// 范围从Lo到Hi（包括两端），固定跨度为1。Delta是要添加到代码点的数字，以达到该字符在不同情况下的代码点。
// 他们可能是负面的。 如果为零，则表示字符在相应的情况下。 有一种特殊情况表示交替的相应的上对和下对对的序列。
// 它以{UpperLower，UpperLower，UpperLower}的固定Delta出现。常数UpperLower具有否则不可能的增量值。
type CaseRange struct { // 注：大小写映射结构
	Lo    uint32
	Hi    uint32
	Delta d
}

// SpecialCase 代表特定语言的案例映射，例如土耳其语。
// SpecialCase的方法自定义（通过覆盖）标准映射。
type SpecialCase []CaseRange

// BUG（r）：没有用于全大小写折叠的机制，也就是说，对于在输入或输出中涉及多个rune的字符而言。
// 指向CaseRanges内部的Delta数组以进行案例映射。
const (
	UpperCase = iota // 注：转为大写ASCII
	LowerCase        // 注：转为小写ASCII
	TitleCase        // 注：转为标题
	MaxCase          // 注：转为最大值
)

type d [MaxCase]rune // 使CaseRanges文本更短，注：d[0]：大写，d[1]：小写，d[2]：标题

// 如果CaseRange的Delta字段为UpperLower，则表示此CaseRange表示形式为（例如）Upper Lower Upper Lower的序列。
const (
	UpperLower = MaxRune + 1 // （不能是有效的增量。）
)

// linearMax 是线性搜索非Latin1 rune的最大尺寸表。
// 通过运行"go test -calibrate"派生。
const linearMax = 18

// is16 报告r是否在16位范围的排序切片中。
func is16(ranges []Range16, r uint16) bool { // 注：获取r是否在16位范围ranges中
	if len(ranges) <= linearMax || r <= MaxLatin1 { // 注：如果ranges小于Latin1的最大尺寸 或者 r在Latin1范围内
		for i := range ranges { // 注：遍历ranges
			range_ := &ranges[i]
			if r < range_.Lo { // 注：如果r小于ranges的最小值，返回false
				return false
			}
			if r <= range_.Hi { // 注：如果r处于当前range的范围内
				return range_.Stride == 1 || (r-range_.Lo)%range_.Stride == 0 // 注：#
			}
		}
		return false
	}

	// 范围内的二分搜索
	lo := 0
	hi := len(ranges)
	for lo < hi { // 注：二分搜索ranges
		m := lo + (hi-lo)/2
		range_ := &ranges[m]
		if range_.Lo <= r && r <= range_.Hi {
			return range_.Stride == 1 || (r-range_.Lo)%range_.Stride == 0 // 注：#
		}
		if r < range_.Lo {
			hi = m
		} else {
			lo = m + 1
		}
	}
	return false
}

// is32 报告r是否在32位范围的排序切片中。
func is32(ranges []Range32, r uint32) bool { // 注：获取r是否在32位范围ranges中
	if len(ranges) <= linearMax { // 注：如果ranges小于Latin1的最大尺寸
		for i := range ranges { // 注：遍历ranges
			range_ := &ranges[i]
			if r < range_.Lo { // 注：如果r小于ranges的最小值，返回false
				return false
			}
			if r <= range_.Hi { // 注：如果r处于当前range的范围内
				return range_.Stride == 1 || (r-range_.Lo)%range_.Stride == 0
			}
		}
		return false
	}

	// 范围内的二分搜索
	lo := 0
	hi := len(ranges)
	for lo < hi { // 注：二分搜索ranges
		m := lo + (hi-lo)/2
		range_ := ranges[m]
		if range_.Lo <= r && r <= range_.Hi {
			return range_.Stride == 1 || (r-range_.Lo)%range_.Stride == 0 // 注：#
		}
		if r < range_.Lo {
			hi = m
		} else {
			lo = m + 1
		}
	}
	return false
}

// Is 报告rune是否在指定的范围表中。
func Is(rangeTab *RangeTable, r rune) bool { // 注：获取r是否在rangeTab的范围内
	r16 := rangeTab.R16
	if len(r16) > 0 && r <= rune(r16[len(r16)-1].Hi) { // 注：如果r <= rangeTab中UTF-16编码最大值
		return is16(r16, uint16(r)) // 注：#
	}
	r32 := rangeTab.R32
	if len(r32) > 0 && r >= rune(r32[0].Lo) { // 注：如果r >= rangeTab中UTF-32编码最小值
		return is32(r32, uint32(r)) // 注：#
	}
	return false
}

func isExcludingLatin(rangeTab *RangeTable, r rune) bool { // 注：获取r是否在字符集rangeTab内
	r16 := rangeTab.R16
	if off := rangeTab.LatinOffset; len(r16) > off && r <= rune(r16[len(r16)-1].Hi) { // 注：如果r <= rangeTab中UTF-16编码最大值
		return is16(r16[off:], uint16(r)) // 注：r是否在rangeTab的16位范围内
	}
	r32 := rangeTab.R32
	if len(r32) > 0 && r >= rune(r32[0].Lo) { // 注：如果r >= rangeTab中UTF-32编码最小值
		return is32(r32, uint32(r)) // 注：r是否在rangeTab的32位范围内
	}
	return false
}

// IsUpper 报告该rune是否为大写字母。
func IsUpper(r rune) bool { // 注：获取r是否为大写字母
	// 请参见IsGraphic中的评论。
	if uint32(r) <= MaxLatin1 {
		return properties[uint8(r)]&pLmask == pLu
	}
	return isExcludingLatin(Upper, r)
}

// IsLower 报告该rune是否为小写字母。
func IsLower(r rune) bool { // 注：获取r是否为小写字母
	// 请参见IsGraphic中的评论。
	if uint32(r) <= MaxLatin1 {
		return properties[uint8(r)]&pLmask == pLl
	}
	return isExcludingLatin(Lower, r)
}

// IsTitle 报告该rune是否为标题大小写字母。
func IsTitle(r rune) bool { // 注：获取r是否为标题大小写字母
	if r <= MaxLatin1 {
		return false
	}
	return isExcludingLatin(Title, r)
}

// to 使用指定的大小写映射来映射rune。
// 它另外报告caseRange是否包含r的映射。
func to(_case int, r rune, caseRange []CaseRange) (mappedRune rune, foundMapping bool) { // 注：#获取r根据对应的caseRange中_case的增量计算出的rune
	if _case < 0 || MaxCase <= _case { // 注：如果_case超界，返回错误
		return ReplacementChar, false // 任何合理的错误
	}
	// 范围内的二分查找
	lo := 0
	hi := len(caseRange)
	for lo < hi { // 注：遍历caseRange，二分查找
		m := lo + (hi-lo)/2
		cr := caseRange[m]
		if rune(cr.Lo) <= r && r <= rune(cr.Hi) { // 注：如果找到与r对应的caseRange
			delta := cr.Delta[_case] // 注：获取caseRange对应_case的增量
			if delta > MaxRune {
				// 在始终以UpperCase字母开头的Upper-Lower序列中，实际增量始终如下所示：
				// 	{0, 1, 0} UpperCase （下一个是小写）
				// 	{-1, 0, -1} LowerCase（大写，标题为上一个）
				// 从序列开始处偶数偏移的字符为大写； 那些在奇数偏移处的值较低。
				// 可以通过清除或设置序列偏移量的低位来完成正确的映射。
				// 常量UpperCase和TitleCase为偶数，而LowerCase为奇数，因此我们取_case中的低位。
				return rune(cr.Lo) + ((r-rune(cr.Lo))&^1 | rune(_case&1)), true // 注：#
			}
			return r + delta, true
		}
		if r < rune(cr.Lo) { // 注：二分查找
			hi = m
		} else {
			lo = m + 1
		}
	}
	return r, false
}

// To 将rune映射到指定的大小写：UpperCase，LowerCase或TitleCase。
func To(_case int, r rune) rune { // 注：获取r根据_case转换后的rune
	r, _ = to(_case, r, CaseRanges)
	return r
}

// ToUpper 将rune映射为大写。
func ToUpper(r rune) rune { // 注：获取r的大写字母，先判断r是否为ASCII
	if r <= MaxASCII { // 注：如果r是ASCII，并且是小写字母，转为大写字母
		if 'a' <= r && r <= 'z' {
			r -= 'a' - 'A'
		}
		return r
	}
	return To(UpperCase, r)
}

// ToLower 将rune映射为小写。
func ToLower(r rune) rune { // 注：获取r的小写字母，先判断r是否为ASCII
	if r <= MaxASCII {
		if 'A' <= r && r <= 'Z' {
			r += 'a' - 'A'
		}
		return r
	}
	return To(LowerCase, r)
}

// ToTitle 将rune映射到标题大小写。
func ToTitle(r rune) rune { // 注：获取r的标题大小写，先判断r是否为ASCII
	if r <= MaxASCII {
		if 'a' <= r && r <= 'z' { // 标题大小写是ASCII的大写
			r -= 'a' - 'A'
		}
		return r
	}
	return To(TitleCase, r)
}

// ToUpper 将rune映射为大写，并优先使用特殊映射。
func (special SpecialCase) ToUpper(r rune) rune { // 注：将special转为大写字母
	r1, hadMapping := to(UpperCase, r, []CaseRange(special))
	if r1 == r && !hadMapping {
		r1 = ToUpper(r)
	}
	return r1
}

// ToTitle 将rune映射到标题大小写，并优先使用特殊映射。
func (special SpecialCase) ToTitle(r rune) rune { // 注：将special转为标题大小写字母
	r1, hadMapping := to(TitleCase, r, []CaseRange(special))
	if r1 == r && !hadMapping {
		r1 = ToTitle(r)
	}
	return r1
}

// ToLower 将rune映射为小写，并优先使用特殊映射。
func (special SpecialCase) ToLower(r rune) rune { // 注：将special转为小写字母
	r1, hadMapping := to(LowerCase, r, []CaseRange(special))
	if r1 == r && !hadMapping {
		r1 = ToLower(r)
	}
	return r1
}

// caseOrbit 在tables.go中定义为[]foldPair。
// 现在，所有条目都适合uint16，因此请使用uint16。
// 如果更改，编译将失败（复合文字中的常量将不适用于uint16），并且此处的类型可以更改为uint32。
type foldPair struct {
	From uint16
	To   uint16
}

// SimpleFold 遍历Unicode定义的简单大小写折叠下的Unicode代码点。
// 在相当于rune的代码点（包括符文本身）中，如果存在，SimpleFold返回最小的rune > r，否则返回最小的rune >= 0。
// 如果r不是有效的Unicode代码点，则SimpleFold(r)返回r。
//
// 例如：
//	SimpleFold('A') = 'a'
//	SimpleFold('a') = 'A'
//
//	SimpleFold('K') = 'k'
//	SimpleFold('k') = '\u212A' (开尔文符号, K)
//	SimpleFold('\u212A') = 'K'
//
//	SimpleFold('1') = '1'
//
//	SimpleFold(-2) = -2
//
func SimpleFold(r rune) rune { // 注：#简单折叠rune（将大写转为小写、将小写转为大写，将unicdoe转为ASCII，将特殊符号转为unicode）
	if r < 0 || r > MaxRune { // 注：如果r不是rune，返回r
		return r
	}

	if int(r) < len(asciiFold) { // 注：#
		return rune(asciiFold[r])
	}

	// 咨询caseOrbit表以了解特殊情况。
	lo := 0
	hi := len(caseOrbit)
	for lo < hi {
		m := lo + (hi-lo)/2
		if rune(caseOrbit[m].From) < r {
			lo = m + 1
		} else {
			hi = m
		}
	}
	if lo < len(caseOrbit) && rune(caseOrbit[lo].From) == r {
		return rune(caseOrbit[lo].To)
	}

	// 未指定折叠。 这是一元素或二元素等效类，如果它们与符文不同，
	// 则包含符文以及ToLower(rune)和ToUpper(rune)。
	if l := ToLower(r); l != r {
		return l
	}
	return ToUpper(r)
}
