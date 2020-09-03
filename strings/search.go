// 版权所有2015 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package strings

// stringFinder 有效地在源文本中查找字符串。
// 它是使用Boyer-Moore字符串搜索算法实现的：
// https://en.wikipedia.org/wiki/Boyer-Moore_string_search_algorithm
// https://www.cs.utexas.edu/~moore/publications/fstrpos.pdf（注意：该老化文档使用基于1的索引）
type stringFinder struct { // 注：#
	// pattern是我们在文本中搜索的字符串。
	pattern string // 注：要搜索的字符串

	// badCharSkip[b]包含pattern的最后一个字节与pattern中最右边的b之间的距离。 如果b不在pattern中，则badCharSkip[b]为len(pattern)。
	//
	// 只要发现文本中的字节b不匹配，我们就可以安全地将匹配帧至少移至badCharSkip[b]，直到下次匹配的char可以对齐为止。
	badCharSkip [256]int // 注：#

	// goodSuffixSkip [i]定义给定后缀pattern[i+1:]可以将匹配帧移动多远，而字节pattern[i]不可以。 有两种情况需要考虑：
	//
	// 1. 匹配的后缀出现在pattern中的其他位置（我们可能会匹配一个不同的字节）。
	// 在这种情况下，我们可以移动匹配的框架以使其与下一个后缀块对齐。
	// 例如，pattern "mississi"在索引1处出现了后缀"issi" （从右到左顺序），
	// 因此goodSuffixSkip[3] == shift+len(suffix) == 3+4 == 7.
	//
	// 2. 如果匹配的后缀未在pattern中的其他位置出现，则匹配的帧可能会与匹配的后缀的末尾共享部分前缀。
	// 在这种情况下，goodSuffixSkip[i]将包含将帧移动多远以使前缀的此部分与后缀对齐。
	// 例如，在PATTERN "abcxxxabc"中，当发现从背面开始的第一个不匹配项位于位置3时，在pattern的其他位置找不到匹配的后缀"xxabc"。
	// 但是，它最右边的"abc"（在位置6）是整个PATTERN的前缀，因此goodSuffixSkip[3] == shift + len(suffix)== 6 + 5 == 11。
	goodSuffixSkip []int
}

func makeStringFinder(pattern string) *stringFinder { // 注：生成一个stringFinder结构体
	// 例1：pattern = "abcde"
	// f.pattern = "abcde"
	// f.badCharSkip = [256]int{5 ... 5}
	// last = 4
	// i = 0	f.badCharSkip[97] = 4
	// i = 1	f.badCharSkip[98] = 3
	// i = 2	f.badCharSkip[99] = 2
	// i = 3	f.badCharSkip[100] = 1
	//
	// 例2：pattern = "aaa"
	// f.pattern = "aaa"
	// f.badCharSkip = [256]int{3 ... 3}
	// last = 2
	// i = 0	f.badCharSkip[97] = 2
	// i = 1	f.badCharSkip[97] = 1
	//
	// 注：除了pattern包含的字符全比默认值5要小，
	// 注：pattern的最后一个元素不处理

	f := &stringFinder{ // 注：生成一个stringFinder
		pattern:        pattern,
		goodSuffixSkip: make([]int, len(pattern)),
	}
	// last是pattern中最后一个字符的索引。
	last := len(pattern) - 1

	// 建立错误字符表。
	// 不在pattern中的字节可以跳过一个pattern的长度。
	for i := range f.badCharSkip { // 注：遍历badCharSkip，每个元素都赋值为pattern的长度
		f.badCharSkip[i] = len(pattern)
	}
	// 循环条件是<而不是<=，因此最后一个字节与其自身之间的距离不为零。 如果发现此字节不正确，则表明它不在最后一个位置。
	for i := 0; i < last; i++ { // 注：在badCharSkip中设置pattern出现的序号
		f.badCharSkip[pattern[i]] = last - i
	}

	// 例1：pattern = "abcde"，last = 4
	// i = 4	判断abcde是否以""作为前缀			true	lastPrefix = 5	f.goodSuffixSkip[4] = 5+4-4 = 5
	// i = 3	判断abcde是否以"e"作为前缀			false	lastPrefix = 5	f.goodSuffixSkip[3] = 5+4-3 = 6
	// i = 2	判断abcde是否以"de"作为前缀			false	lastPrefix = 5	f.goodSuffixSkip[2] = 5+4-2 = 7
	// i = 1	判断abcde是否以"cde"作为前缀		false	lastPrefix = 5	f.goodSuffixSkip[1] = 5+4-1 = 8
	// i = 0	判断abcde是否以"bcde"作为前缀		false	lastPrefix = 5	f.goodSuffixSkip[0] = 5+4-0 = 9
	//
	// 例2：pattern = "aaa"，last = 2
	// i = 2	判断aaa是否以""作为前缀		true	lastPrefix = 3	f.goodSuffixSkip[2] = 3+2-2 = 3
	// i = 1	判断aaa是否以"a"作为前缀	true	lastPrefix = 2	f.goodSuffixSkip[1] = 2+2-1 = 3
	// i = 0	判断aaa是否以"aa"作为前缀	true	lastPrefix = 1	f.goodSuffixSkip[0] = 1+2-0 = 3

	// 建立良好的后缀表。
	// 第一遍：将每个值设置为下一个索引，该索引以开头
	// pattern。
	lastPrefix := last
	for i := last; i >= 0; i-- { // 注：倒序遍历pattern
		if HasPrefix(pattern, pattern[i+1:]) {
			lastPrefix = i + 1
		}
		// lastPrefix是移位, 而(last-i) 是 len(suffix).
		f.goodSuffixSkip[i] = lastPrefix + last - i
	}

	// 例1：pattern = "abcde"，last = 4
	// i = 0	"abcde"与""相同的后缀长度为 = 0			lenSuffix = 0	"a" != "e"	f.goodSuffixSkip[4] = 0+4-0 = 4
	// i = 1	"abcde"与"b"相同的后缀长度为 = 0		lenSuffix = 0	"b" != "e"	f.goodSuffixSkip[4] = 0+4-1 = 3
	// i = 2	"abcde"与"bc"相同的后缀长度为 = 0		lenSuffix = 0	"c" != "e"	f.goodSuffixSkip[4] = 0+4-2 = 2
	// i = 3	"abcde"与"bcd"相同的后缀长度为 = 0		lenSuffix = 0	"d" != "e"	f.goodSuffixSkip[4] = 0+4-3 = 1
	//
	// 例2：pattern = "aaa"，last = 2
	// i = 0	"aaa"与""相同的后缀长度为 = 0		lenSuffix = 0	"a" == "a"
	// i = 1	"aaa"与"a"相同的后缀长度为 = 1		lenSuffix = 1	"a" == "a"

	// 第二遍：从前面开始查找重复的pattern后缀。
	for i := 0; i < last; i++ {
		lenSuffix := longestCommonSuffix(pattern, pattern[1:i+1])
		if pattern[i-lenSuffix] != pattern[last-lenSuffix] {
			// (last-i)是偏移量，lenSuffix是len(suffix).
			f.goodSuffixSkip[last-lenSuffix] = lenSuffix + last - i
		}
	}

	return f
}

func longestCommonSuffix(a, b string) (i int) { // 注：返回a与b相同的后缀长度i
	for ; i < len(a) && i < len(b); i++ { // 注：倒序遍历a与b
		if a[len(a)-1-i] != b[len(b)-1-i] { // 注：如果a与b的元素不同，结束循环
			break
		}
	}
	return
}

// next 返回pattern第一次出现时的文本索引。 如果找不到pattern，则返回-1。
func (f *stringFinder) next(text string) int { // 注：获取text中第一次出现pattern的索引
	// 例：f.pattern = "abcde"，text = "123abcde123"
	// i = 4	j = 4	4 >= 0 && "b" == "e"	false	text[i] = "b"	f.badCharSkip[98] = 3	f.goodSuffixSkip[4] = 1
	// i = 7	j = 4	4 >= 0 && "e" == "e"	true
	// 进入子循环
	// i = 6	j = 3	3 >= 0 && "d" == "d"	true
	// i = 5	j = 2	2 >= 0 && "c" == "c"	true
	// i = 4	j = 1	1 >= 0 && "b" == "b"	true
	// i = 3	j = 0	0 >= 0 && "a" == "a"	true
	// i = 2	j = -1	返回3

	i := len(f.pattern) - 1
	for i < len(text) { // 注：正序遍历，倒序比较
		// 从末尾开始比较直到第一个不匹配的字符。
		j := len(f.pattern) - 1
		for j >= 0 && text[i] == f.pattern[j] {
			i--
			j--
		}
		if j < 0 {
			return i + 1 // match
		}
		i += max(f.badCharSkip[text[i]], f.goodSuffixSkip[j])
	}
	return -1
}

func max(a, b int) int { // 注：返回a与b中的最大值
	if a > b {
		return a
	}
	return b
}
