/*
	文件：
		builder.go										实现StringBuilder
		compare.go										实现字符串比较，但不建议使用
		reader.go										为字符串提供io.Reader相关功能
		replace.go										提供字符串替换功能
		search.go										提供字符串搜索功能（正序遍历，倒序比较）
		strings.go										为字符串提供Split、Contain、Index、Trim、转换、比较、重复、替换等多种函数
	结构体与接口：
		type Builder struct								缓冲区
		type Reader struct 								提供对于s的各种读取方法
		type Replacer struct							根据oldnew获取不同类型的替换器
		type replacer interface							替换器需要实现的方法
		type trieNode struct							#
		type stringWriter struct						#
		type genericReplacer struct						通用替换器
		type singleStringReplacer struct				单字符串替换器，只替换一组old与new
		type byteStringReplacer struct					字节字符串替换器，将字节替换为字符串
		type stringFinder struct						#
	函数与方法：
		--导出方法
		Compare(a, b string) int						比较字符串，不建议使用
		Join(elems []string, sep string) string			将elems合并为字符串，每个元素之间使用sep分隔
		Map(mapping func(rune) rune, s string) string	获取通过mapping映射后的s
		Repeat(s string, count int) string				获取重复count次的s
		ToValidUTF8(s, replacement string) string		获取将无效rune替换为replacement的s
		Replace(s, old, new string, n int) string		获取将前n次old替换为new的s
		ReplaceAll(s, old, new string) string			获取将old替换为new的s
		EqualFold(s, t string) bool						获取s与t是否相等

		--未导出方法
		noescape(p unsafe.Pointer) unsafe.Pointer		在逃逸分析中隐藏指针p
		longestCommonSuffix(a, b string) (i int)		返回a与b相同的后缀长度i
		max(a, b int) int								返回a与b中的最大值
		makeASCIISet(...)								获取chars中连续的ASCII的编码集合
			(as *asciiSet) contains(c byte) bool		获取c是否在as内
		makeCutsetFunc(cutset string) func(rune) bool	返回cutset是否包含rune的方法

		--split
		explode(s string, n int) []string				将s拆分为n个rune
		genSplit(...)									将s根据sep拆分为n份数据，每份数据额外读取sepSave字节数据
		SplitN(s, sep string, n int) []string			将s根据sep拆分为n份数据
		SplitAfterN(s, sep string, n int) []string		将s根据sep拆分为n份数据（保留sep）
		Split(s, sep string) []string					将s根据sep拆分
		SplitAfter(s, sep string) []string				将s根据sep拆分（保留sep）
		Fields(s string) []string						将s根据一个或多个连续的空白字符进行拆分
		FieldsFunc(...)									将s根据f(rune)拆分
		isSeparator(r rune) bool						获取r是否为分隔符

		--Rabin-Karp算法
		hashStr(sep string) (uint32, uint32)			获取sep的哈希与乘法因子（算法：Rabin-Karp）
		hashStrRev(sep string) (uint32, uint32)			获取sep的倒数的哈希与乘法因子（算法：Rabin-Karp）

		--contains
		Count(s, substr string) int						获取s中substr出现的次数
		Contains(s, substr string) bool					获取s中是否包括substr
		ContainsAny(s, chars string) bool				获取s是否包含chars的任意元素
		ContainsRune(s string, r rune) bool				获取s是否包含r

		--index
		Index(s, substr string) int						获取s中第一个substr的索引
		LastIndex(s, substr string) int					获取s中最后一次出现substr的索引
		IndexByte(s string, c byte) int					获取s中第第一次出现c的索引
		IndexRune(s string, r rune) int					获取s中第一次出现r的索引
		IndexAny(s, chars string) int					获取s中出现chars的任意元素的位置
		LastIndexAny(s, chars string) int				获取s中最后一次出现charts的任意元素的位置
		LastIndexByte(s string, c byte) int				获取s中最后一个c出现的位置
		IndexFunc(s string, f func(rune) bool) int						获取s中第一个符合f(r)为true的索引
		LastIndexFunc(s string, f func(rune) bool) int					获取s中最后一个符合f(r)为true的索引
		indexFunc(s string, f func(rune) bool, truth bool) int			获取s中第一个符合f(r) == truth的索引
		lastIndexFunc(s string, f func(rune) bool, truth bool) int		获取s中最后一个符合f(r) == truth的索引
		indexRabinKarp(s, substr string) int							使用Rabin-Karp搜索s中substr的索引

		--前缀/后缀
		HasPrefix(s, prefix string) bool				获取s的前缀是否为prefix
		HasSuffix(s, suffix string) bool				获取s的后缀是否为suffix

		--大写/小写/标题大小写
		ToUpper(s string) string						将s转为大写
		ToLower(s string) string						将s转为小写
		ToTitle(s string) string						将s转为标题大小写
		ToUpperSpecial(...)								将s通过c的特殊映射转为大写
		ToLowerSpecial(...)								将s通过c的特殊映射转为小写
		ToTitleSpecial(...)								将s通过c的特殊映射转为标题大小写
		Title(s string) string							获取s的标题格式

		---trim
		Trim(s string, cutset string) string							获取截取符合cutset包含的前缀与后缀的s
		TrimLeftFunc(s string, f func(rune) bool) string				获取截断符合f(r)为false的前缀的s
		TrimRightFunc(s string, f func(rune) bool) string				获取截断符合f(r)为false的后缀的s
		TrimFunc(s string, f func(rune) bool) string					获取截断符合f(r)为false的前缀与后缀的s
		TrimLeft(s string, cutset string) string						获取截取符合cutset包含的前缀的s（前缀为一个rune）
		TrimRight(s string, cutset string) string						获取截取符合cutset包含的后缀的s（后缀为一个rune）
		TrimSpace(s string) string										获取截取前后空格的s
		TrimPrefix(s, prefix string) string								获取去掉prefix前缀的s
		TrimSuffix(s, suffix string) string								获取去掉suffix后缀的s

		--base
		(b *Builder) copyCheck()						检查拷贝，保证b.addr == b
		(b *Builder) String() string					返回b的字符串表现形式
		(b *Builder) Len() int							获取b的数据长度
		(b *Builder) Cap() int							获取b的数据容量
		(b *Builder) Reset()							重置b
		(b *Builder) grow(n int)						保证b至少还可以容纳长度为n的数据
		(b *Builder) Grow(n int)						保证b至少还可以容纳长度为n的数据
		--write
		(b *Builder) Write(p []byte) (int, error)		将p附加到b
		(b *Builder) WriteByte(c byte) error			将c附加到b
		(b *Builder) WriteRune(r rune) (int, error)		将r附加到b
		(b *Builder) WriteString(s string) (int, error)	将s附加到b

		NewReader(s string) *Reader											工厂函数，生成一个数据为s的Reader结构体
			--base
			(r *Reader) Len() int											获取r未读取的字节数
			(r *Reader) Size() int64										获取r的数据长度
			(r *Reader) Seek(offset int64, whence int) (int64, error)		修改r的索引为相对位置whence偏移offset
			(r *Reader) Reset(s string)							 			重置r，数据为s

			--write
			(r *Reader) WriteTo(w io.Writer) (n int64, err error)			将r的数据写入w中

			--read
			(r *Reader) Read(b []byte) (n int, err error)					从r中读取数据到b中
			(r *Reader) ReadAt(b []byte, off int64) (n int, err error)		从r偏移off字节处读取数据到b中
			(r *Reader) ReadByte() (byte, error)							从r中读取一字节数据
			(r *Reader) ReadRune() (ch rune, size int, err error)			从r中读取一个rune

			--unread
			(r *Reader) UnreadByte() error									撤回上次ReadByte()
			(r *Reader) UnreadRune() error									撤回上次ReadRune()

		NewReplacer(oldnew ...string) *Replacer								工厂函数，生成一个替换字符为oldnew的Replacer结构体
			(r *Replacer) buildOnce()										根据oldnew创建Replacer
			(b *Replacer) build() replacer									根据oldnew创建对应的Replacer
			(r *Replacer) Replace(s string) string 							将s的old替换为new

		makeGenericReplacer(oldnew []string) *genericReplacer 				获取通用替换器
			(r *genericReplacer) lookup(...)								#
			(r *genericReplacer) Replace(s string) string					#
			(r *genericReplacer) WriteString(...)							#

		makeSingleStringReplacer(...)										获取单字符串替换器
			(r *singleStringReplacer) Replace(s string) string				将s中所有old替换为new
			(r *singleStringReplacer) WriteString(...)						将s中所有old替换为new写入w中

		getStringWriter(w io.Writer) io.StringWriter						将w转为stringWriter
			(w stringWriter) WriteString(s string) (int, error)				将s写入w

		(r *byteReplacer) Replace(s string) string 							将s中所有old替换为new
		(r *byteReplacer) WriteString(...)									将s中所有为old的位替换为new写入w中
		(r *byteStringReplacer) Replace(s string) string					将s中所有old替换为new
		(r *byteStringReplacer) WriteString(...)							将s的old替换为new写入w中
		(w *appendSliceWriter) Write(p []byte) (int, error) 				将p写入w
		(w *appendSliceWriter) WriteString(s string) (int, error)			将s写入w
		(t *trieNode) add(...)												#

		makeStringFinder(pattern string) *stringFinder						生成一个stringFinder结构体
			(f *stringFinder) next(text string) int							获取text中第一次出现pattern的索引
	用法：
		Replacer：将"abcde"中的"a"替换为"1"，将"b"替换为"2"
			rp := strings.NewReplacer("a", "1", "b", "2")
			rp.WriteString(os.Stdout, "abcde") // 将结果"12cde"写入os.Stdout
			s := rp.Replace("abcde")           // 将结果"12cde"返回

		Reader：为操作字符串"abcde"提供各种方法
			r := strings.NewReader("abcde")
			r1, _, _ := r.ReadRune() // 读取一个rune，'a'
			r2, _, _ := r.ReadRune() // 读取一个rune，'b'
			r3, _, _ := r.ReadRune() // 读取一个rune，'c'
			r.UnreadRune()           // 撤回上次ReadRune()
			r4, _, _ := r.ReadRune() // 读取一个rune，'c'
			r5, _, _ := r.ReadRune() // 读取一个rune，'d'
*/
package strings
