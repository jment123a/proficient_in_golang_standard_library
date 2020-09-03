// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package strings

import (
	"io"
	"sync"
)

// Replacer 用替换项替换字符串列表。
// 对于多个goroutine并发使用是安全的。
// 注：
// 只有一对替换字符：		SignleStringReplacer	单字符串替换器
// old为字符串：			GenericReplacer			通用替换器
// old和new均为字节：		byteReplacer			字节替换器
// old为字节，new为字符串：	byteStringReplacer		字节字符串替换器
type Replacer struct { // 注：根据oldnew获取不同类型的replacer
	once   sync.Once // 守护buildOnce方法
	r      replacer  // 注：替换器，执行build后生成
	oldnew []string  // 注：替换字符对，执行build后清空
}

// replacer 是替换算法需要实现的接口。
type replacer interface {
	Replace(s string) string
	WriteString(w io.Writer, s string) (n int, err error)
}

// NewReplacer 从旧的新字符串对列表中返回新的Replacer。
// 替换按照它们在目标字符串中出现的顺序执行，没有重叠的匹配项。
// 旧的字符串比较按参数顺序进行。
//
// 如果给定了奇数个参数，NewReplacer会出现恐慌。
func NewReplacer(oldnew ...string) *Replacer { // 工厂函数，生成一个替换字符为oldnew的Replacer结构体
	if len(oldnew)%2 == 1 {
		panic("strings.NewReplacer: odd argument count") // 恐慌："奇数参数"
	}
	return &Replacer{oldnew: append([]string(nil), oldnew...)}
}

func (r *Replacer) buildOnce() { // 注：根据oldnew创建Replacer
	r.r = r.build()
	r.oldnew = nil // 注：清空oldnew
}

func (b *Replacer) build() replacer { // 注：根据oldnew创建对应的Replacer
	// 注：
	// 只有一对替换字符：		SignleStringReplacer	单字符串替换器
	// old为字符串：			GenericReplacer			通用替换器
	// old和new均为字节：		byteReplacer			字节替换器
	// old为字节，new为字符串：	byteStringReplacer		字节字符串替换器
	oldnew := b.oldnew
	if len(oldnew) == 2 && len(oldnew[0]) > 1 { // 注：只有一对替换字符
		return makeSingleStringReplacer(oldnew[0], oldnew[1]) // 注：返回单字符串替换器
	}

	allNewBytes := true
	for i := 0; i < len(oldnew); i += 2 { // 注：遍历替换字符对
		if len(oldnew[i]) != 1 { // 注：如果old的长度>1，返回通用替换器
			return makeGenericReplacer(oldnew)
		}
		if len(oldnew[i+1]) != 1 { // 注：如果new的长度>1
			allNewBytes = false
		}
	}

	// 例：old = 'a'，new = 'b'，byteReplacer[97] = 'a'
	// byteReplacer[97] = 'b'
	if allNewBytes { // 注：如果new的长度>1
		r := byteReplacer{} // 注：字节替换器
		for i := range r {  // 注：初始化
			r[i] = byte(i)
		}
		// 第一次出现的old-> new映射优先于具有相同旧字符串的其他映射。
		for i := len(oldnew) - 2; i >= 0; i -= 2 { // 注：将每个old对应的值修改为new
			o := oldnew[i][0]
			n := oldnew[i+1][0]
			r[o] = n
		}
		return &r
	}

	r := byteStringReplacer{toReplace: make([]string, 0, len(oldnew)/2)} // 注：字节字符串替换器
	// 第一次出现的old -> new映射优先于具有相同旧字符串的其他映射。
	for i := len(oldnew) - 2; i >= 0; i -= 2 { // 注：倒序遍历替换字符对
		o := oldnew[i][0] // 注：old
		n := oldnew[i+1]  // 注：new
		// 为了避免多次计算重复次数。
		if r.replacements[o] == nil { // 注：如果replacements中old的位置为空
			// 我们需要使用string([]byte{o})而不是string(o)，以避免对o进行utf8编码。
			// 例如 byte(150)产生长度为2的字符串。
			r.toReplace = append(r.toReplace, string([]byte{o})) // 注：toReplace添加old
		}
		r.replacements[o] = []byte(n) // 注：将new赋值到replacements中old的位置

	}
	return &r
}

// Replace 返回s的副本，并执行所有替换操作。
func (r *Replacer) Replace(s string) string { // 注：将s的old替换为new
	r.once.Do(r.buildOnce) // 注：执行build
	return r.r.Replace(s)
}

// WriteString 将s写入w并执行所有替换操作。
func (r *Replacer) WriteString(w io.Writer, s string) (n int, err error) { // 注：将s的old替换为new写入w中
	r.once.Do(r.buildOnce) // 注：执行build
	return r.r.WriteString(w, s)
}

// trieNode 是优先级键/值对的查找树中的节点。
// 键和值可能为空。 例如，包含键"ax", "ay",  "bcbc", "x" h和 "xy"的trie可能有八个节点：
//
//  n0  -
//  n1  a-
//  n2  .x+
//  n3  .y+
//  n4  b-
//  n5  .cbc+
//  n6  x+
//  n7  .y+
//
// n0是根节点，其子节点为n1，n4和n6；
// n1的孩子是n2和n3；
// n4的孩子是n5；
// n6的孩子是n7。
// 节点n0，n1和n4（用结尾的"-"标记）是部分密钥，节点n2，n3，n5，n6和n7（用结尾的"+"标记）是完整密钥。
type trieNode struct {
	// value 是trie节点的键/值对的值。 如果此节点不是完整密钥，则为空。
	value string // 注：new
	// priority 是trie节点的键/值对的优先级（越高，则越重要）；
	// 键不一定最短或最长先匹配。
	// 如果此节点是完整密钥，则优先级为正，否则为零。
	// 在上面的示例中，正/零优先级标有尾随的"+"或"-"。
	priority int // 注：当前通用替换器的第几对oldnew

	// 一个trie节点可能有零个，一个或多个子节点：
	// 	*如果其余字段为零，则没有子级。
	// 	*如果prefix和next不为零，则next中有一个孩子。
	// 	*如果table不为零，则定义所有子项。
	//
	// 当有一个孩子时，前缀优先于表，但根节点始终使用表来提高查找效率。

	// prefix是此Trie节点与下一个Trie节点之间的键差。
	// 在上面的示例中，节点n4的前缀为"cbc"，n4的下一个节点为n5。
	// 节点n5没有子节点，因此前缀，next和table字段为零。
	prefix string
	next   *trieNode

	// table 是一个查找表，该索引由key中的下一个字节索引，
	// 然后通过genericReplacer.mapping重新映射该字节以创建密集索引。
	// 在上面的示例中，键仅使用 'a', 'b', 'c', 'x'和'y'，它们重新映射为0、1、2、3和4。
	// 所有其他字节都重新映射为5和。
	// genericReplacer.tableSize将为5。
	// 节点n0的表将为[]*trieNode{ 0:n1, 1:n4, 3:n6 }，其中0、1和3是重新映射的'a', 'b' 和 'x'.
	table []*trieNode
}

func (t *trieNode) add(key, val string, priority int, r *genericReplacer) { // 注：#
	if key == "" {
		if t.priority == 0 {
			t.value = val         // 注：new
			t.priority = priority // 注：第几对oldnew
		}
		return
	}

	if t.prefix != "" {
		// 需要在多个节点之间分割前缀。
		var n int // 最长公共前缀的长度
		for ; n < len(t.prefix) && n < len(key); n++ {
			if t.prefix[n] != key[n] {
				break
			}
		}
		if n == len(t.prefix) {
			t.next.add(key[n:], val, priority, r)
		} else if n == 0 {
			// First byte differs, start a new lookup table here. Looking up
			// what is currently t.prefix[0] will lead to prefixNode, and
			// looking up key[0] will lead to keyNode.
			var prefixNode *trieNode
			if len(t.prefix) == 1 {
				prefixNode = t.next
			} else {
				prefixNode = &trieNode{
					prefix: t.prefix[1:],
					next:   t.next,
				}
			}
			keyNode := new(trieNode)
			t.table = make([]*trieNode, r.tableSize)
			t.table[r.mapping[t.prefix[0]]] = prefixNode
			t.table[r.mapping[key[0]]] = keyNode
			t.prefix = ""
			t.next = nil
			keyNode.add(key[1:], val, priority, r)
		} else {
			// Insert new node after the common section of the prefix.
			next := &trieNode{
				prefix: t.prefix[n:],
				next:   t.next,
			}
			t.prefix = t.prefix[:n]
			t.next = next
			next.add(key[n:], val, priority, r)
		}
	} else if t.table != nil { // 注：当前将old按字节添加至table
		// 例：key = "abc"，value = "123"，mapping[97~99] = 012
		// mapping[a] = 0，table[0]初始化
		// mapping[b] = 1，table[1]初始化
		// mapping[c] = 2，table[2]初始化
		// 插入现有表。
		m := r.mapping[key[0]]
		if t.table[m] == nil {
			t.table[m] = new(trieNode)
		}
		t.table[m].add(key[1:], val, priority, r)
	} else {
		t.prefix = key
		t.next = new(trieNode)
		t.next.add("", val, priority, r)
	}
}

func (r *genericReplacer) lookup(s string, ignoreRoot bool) (val string, keylen int, found bool) { // 注：#
	// 遍历trie到最后，并以最高优先级获取值和keylen。
	bestPriority := 0
	node := &r.root
	n := 0
	for node != nil {
		if node.priority > bestPriority && !(ignoreRoot && node == &r.root) {
			bestPriority = node.priority
			val = node.value
			keylen = n
			found = true
		}

		if s == "" {
			break
		}
		if node.table != nil {
			index := r.mapping[s[0]]
			if int(index) == r.tableSize {
				break
			}
			node = node.table[index]
			s = s[1:]
			n++
		} else if node.prefix != "" && HasPrefix(s, node.prefix) {
			n += len(node.prefix)
			s = s[len(node.prefix):]
			node = node.next
		} else {
			break
		}
	}
	return
}

// genericReplacer 是完全通用的算法。
// 当无法使用更快的速度时，它将用作备用。
type genericReplacer struct { // 注：通用替换器
	root trieNode
	// tableSize 是trieNode的查找表的大小。 它是唯一密钥字节的数量。
	tableSize int // 注：root.table的大小
	// mapping 从关键字节映射到trieNode.table的密集索引。
	mapping [256]byte // 注：old的映射
}

func makeGenericReplacer(oldnew []string) *genericReplacer { // 注：返回通用替换器
	// 注：根据替换字符
	//
	// 例1：oldnew = []string{"abc", "123", "ddd", "321"}
	// 步骤1：将mapping的每个值赋值为1
	// 步骤2：将abcd赋值为0123，其余赋值为4

	r := new(genericReplacer)
	// 查找使用的每个字节，然后为每个索引分配一个索引。
	for i := 0; i < len(oldnew); i += 2 { // 注：遍历oldnew，将需要替换的字符标记为1
		key := oldnew[i]
		for j := 0; j < len(key); j++ {
			r.mapping[key[j]] = 1
		}
	}

	for _, b := range r.mapping { // 注：遍历映射，计算tableSize
		r.tableSize += int(b)
	}

	var index byte
	for i, b := range r.mapping { // 注：遍历映射，将需要替换的字符标记为序号，其余赋值为tableSize
		if b == 0 {
			r.mapping[i] = byte(r.tableSize)
		} else {
			r.mapping[i] = index
			index++
		}
	}
	// 确保根节点使用查找表（以提高性能）。
	r.root.table = make([]*trieNode, r.tableSize)

	for i := 0; i < len(oldnew); i += 2 {
		r.root.add(oldnew[i], oldnew[i+1], len(oldnew)-i, r)
	}
	return r
}

type appendSliceWriter []byte

// Write 写入缓冲区以满足io.Writer。
func (w *appendSliceWriter) Write(p []byte) (int, error) { // 注：将p写入w
	*w = append(*w, p...)
	return len(p), nil
}

// WriteString 在没有分配string->[]byte->string 的情况下写入缓冲区。
func (w *appendSliceWriter) WriteString(s string) (int, error) { // 注：将s写入w
	*w = append(*w, s...)
	return len(s), nil
}

type stringWriter struct {
	w io.Writer
}

func (w stringWriter) WriteString(s string) (int, error) { // 注：将s写入w
	return w.w.Write([]byte(s))
}

func getStringWriter(w io.Writer) io.StringWriter { // 注：将w转为stringWriter
	sw, ok := w.(io.StringWriter)
	if !ok {
		sw = stringWriter{w}
	}
	return sw
}

func (r *genericReplacer) Replace(s string) string {
	buf := make(appendSliceWriter, 0, len(s))
	r.WriteString(&buf, s)
	return string(buf)
}

func (r *genericReplacer) WriteString(w io.Writer, s string) (n int, err error) {
	sw := getStringWriter(w)
	var last, wn int
	var prevMatchEmpty bool
	for i := 0; i <= len(s); {
		// Fast path: s[i] is not a prefix of any pattern.
		if i != len(s) && r.root.priority == 0 {
			index := int(r.mapping[s[i]])
			if index == r.tableSize || r.root.table[index] == nil {
				i++
				continue
			}
		}

		// Ignore the empty match iff the previous loop found the empty match.
		val, keylen, match := r.lookup(s[i:], prevMatchEmpty)
		prevMatchEmpty = match && keylen == 0
		if match {
			wn, err = sw.WriteString(s[last:i])
			n += wn
			if err != nil {
				return
			}
			wn, err = sw.WriteString(val)
			n += wn
			if err != nil {
				return
			}
			i += keylen
			last = i
			continue
		}
		i++
	}
	if last != len(s) {
		wn, err = sw.WriteString(s[last:])
		n += wn
	}
	return
}

// singleStringReplacer 是仅替换一个字符串（并且该字符串具有多个字节）时使用的实现。
type singleStringReplacer struct { // 注：单字符串替换器
	finder *stringFinder // 注：被替换的字符串，oldString，支持Boyer-Moore字符串搜索算法的结构体
	// value 是在找到该模式后将替换该模式的新字符串。
	value string // 注：用于替换的字符串，newString
}

func makeSingleStringReplacer(pattern string, value string) *singleStringReplacer { // 注：生成singleStringReplacer结构体
	return &singleStringReplacer{finder: makeStringFinder(pattern), value: value}
}

func (r *singleStringReplacer) Replace(s string) string { // 注：将s中所有old替换为new
	var buf []byte
	i, matched := 0, false
	for {
		match := r.finder.next(s[i:]) // 注：获取s第一个符合pattern的索引
		if match == -1 {
			break
		}
		matched = true                     // 注：匹配
		buf = append(buf, s[i:i+match]...) // 注：写入之前的字符串
		buf = append(buf, r.value...)      // 注：写入new
		i += match + len(r.finder.pattern)
	}
	if !matched {
		return s
	}
	buf = append(buf, s[i:]...) // 注：写入之后的字符串
	return string(buf)
}

func (r *singleStringReplacer) WriteString(w io.Writer, s string) (n int, err error) { // 注：将s中所有old替换为new写入w中
	sw := getStringWriter(w)
	var i, wn int
	for {
		match := r.finder.next(s[i:]) // 注：活跃区一个匹配old的索引
		if match == -1 {
			break
		}
		wn, err = sw.WriteString(s[i : i+match]) // 注：向w写入之前的数据
		n += wn
		if err != nil {
			return
		}
		wn, err = sw.WriteString(r.value) // 注：向w写入new
		n += wn
		if err != nil {
			return
		}
		i += match + len(r.finder.pattern)
	}
	wn, err = sw.WriteString(s[i:]) // 注：向w写入之后的数据
	n += wn
	return
}

// byteReplacer 是所有"old"和"new"值均为单个ASCII字节时使用的实现。
// 数组包含由旧字节索引的替换字节。
// 注：内容将会赋值为索引值
// 例：byteReplacer[97] = 97 = 'a'
// 例：如果byteReplacer[97] = 98，则为将a替换为b
type byteReplacer [256]byte // 注：字节替换器，将字节替换为字节

func (r *byteReplacer) Replace(s string) string { // 注：将s中所有old替换为new
	var buf []byte                // 懒散地分配
	for i := 0; i < len(s); i++ { // 注：遍历s
		b := s[i]
		if r[b] != b { // 例：如果r[97] = 98，即将a替换为b
			if buf == nil {
				buf = []byte(s)
			}
			buf[i] = r[b]
		}
	}
	if buf == nil {
		return s
	}
	return string(buf)
}

func (r *byteReplacer) WriteString(w io.Writer, s string) (n int, err error) { // 注：将s中所有为old的位替换为new写入w中
	// TODO（bradfitz）：将io.WriteString与s片一起使用，避免分配。
	bufsize := 32 << 10
	if len(s) < bufsize {
		bufsize = len(s)
	}
	buf := make([]byte, bufsize)

	for len(s) > 0 { // 注：遍历s，一次处理32位
		ncopy := copy(buf, s)           // 注：拷贝32位
		s = s[ncopy:]                   // 注：截取32位
		for i, b := range buf[:ncopy] { // 注：每一位的old替换为new
			buf[i] = r[b]
		}
		wn, err := w.Write(buf[:ncopy]) // 注：向w写入
		n += wn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// byteStringReplacer 是当所有"old"值都是单个ASCII字节但"new"值的大小不同时使用的实现。
type byteStringReplacer struct { // 注：字节字符串替换器，将字节替换为字符串
	// replacements 包含由旧字节索引的替换字节片。
	// nil []byte表示不应替换旧字节。
	replacements [256][]byte // 注：记录old对应的new，256为ASCII，[]byte为对应字符串
	// toReplace 保留要替换的字节列表。 根据toReplace的长度和目标字符串的长度，使用Count或纯循环可能会更快。
	// 我们将单个字节存储为字符串，因为Count接受字符串。
	toReplace []string // 注：记录old
}

// countCutOff 控制字符串长度与替换数量之比(*byteStringReplacer).Replace切换算法。
// 对于长度比替换值大于该值的字符串，对于toReplace中的每个替换，我们称为Count。
// 由于Count开销，对于比率较低的字符串，我们使用简单循环。
// countCutOff是根据经验确定的开销乘数。
// 一旦有了基于寄存器的abi/mid-stack内联，便可以重新访问TODO（tocarip）。
const countCutOff = 8

func (r *byteStringReplacer) Replace(s string) string { // 注：将s中所有old替换为new
	newSize := len(s)
	anyChanges := false
	// 使用Count更快吗？
	if len(r.toReplace)*countCutOff <= len(s) { // 注：如果old较少，根据old计算new需要的空间
		for _, x := range r.toReplace { // 注：遍历old集合
			if c := Count(s, x); c != 0 { // 注：如果s中有old
				// -1是因为我们用len(replacements[b])个字节替换了1个字节。
				newSize += c * (len(r.replacements[x[0]]) - 1) // 注：根据old计算new需要的空间
				anyChanges = true
			}
		}
	} else { // 注：如果old较多，根据s计算new需要的空间
		for i := 0; i < len(s); i++ { // 注：遍历s
			b := s[i]
			if r.replacements[b] != nil { // 注：获取new需要的空间
				// 见上面关于-1的解释
				newSize += len(r.replacements[b]) - 1
				anyChanges = true
			}
		}
	}
	if !anyChanges { // 注：如果没有查找到任何old，直接诶返回
		return s
	}
	buf := make([]byte, newSize)
	j := 0
	for i := 0; i < len(s); i++ { // 注：遍历s
		b := s[i]
		if r.replacements[b] != nil { // 注：将old替换为new
			j += copy(buf[j:], r.replacements[b])
		} else {
			buf[j] = b
			j++
		}
	}
	return string(buf)
}

func (r *byteStringReplacer) WriteString(w io.Writer, s string) (n int, err error) { // 注：将s的old替换为new写入w中
	sw := getStringWriter(w)
	last := 0
	for i := 0; i < len(s); i++ { // 注：遍历s
		b := s[i]
		if r.replacements[b] == nil { // 注：略过不需要替换的字符
			continue
		}
		if last != i { // 注：如果有略过的字符串，写入
			nw, err := sw.WriteString(s[last:i])
			n += nw
			if err != nil {
				return n, err
			}
		}
		last = i + 1
		nw, err := w.Write(r.replacements[b]) // 注：写入替换字符
		n += nw
		if err != nil {
			return n, err
		}
	}
	if last != len(s) { // 注：如果有忽略的字符，写入
		var nw int
		nw, err = sw.WriteString(s[last:])
		n += nw
	}
	return
}
