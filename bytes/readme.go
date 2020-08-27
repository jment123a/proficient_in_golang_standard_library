/*
	结构体与接口
		type Buffer struct {
			buf      []byte // 缓冲区
			off      int    // 当前读取的索引
			lastRead readOp // 上次Read的操作
		}
		type Reader struct {
			s        []byte // 数据
			i        int64  // 当前读取的索引
			prevRune int    // 前一个rune的索引
		}
	函数与方法
		makeSlice(n int) []byte						创建一个长度为n的[]byte
		NewBuffer(buf []byte) *Buffer				工厂函数，创建一个缓冲区，缓冲区内容为buf
		NewBufferString(s string) *Buffer			工厂函数，创建一个缓冲区，缓冲区内容为s
			(b *Buffer) Bytes() []byte				获取缓冲区b的内容
			(b *Buffer) String() string				获取缓冲区b的内容的字符串形式
			(b *Buffer) empty() bool				获取缓冲区b的内容是否为空
			(b *Buffer) Len() int					获取缓冲区b的中未读取的字节数
			(b *Buffer) Cap() int					获取缓冲区b的容量
			(b *Buffer) Truncate(n int)				丢弃缓冲区b中前n个未读字节以外的所有数据
			(b *Buffer) Reset()						重置缓冲区
			(b *Buffer) tryGrowByReslice(...)		获取b是否可以容纳n个字节，返回扩容前的长度
			(b *Buffer) Grow(n int)					缓冲区b扩容n个字节的长度，返回扩容前的长度，简化grow
			(b *Buffer) grow(n int) int				缓冲区b扩容n个字节的长度，返回扩容前的长度
			(b *Buffer) Next(n int) []byte			获取缓冲区b中接下来n个字节的数据

			--read
			(b *Buffer) Read(...)					从缓冲区b中读取数据拷贝到p中，返回读取到的数据长度n与错误err
			(b *Buffer) ReadFrom(...)				从r中读取数据到缓冲区b，返回读取到数据的字节数m与错误err
			(b *Buffer) ReadByte() (byte, error)	获取缓冲区b中接下来的1个字节
			(b *Buffer) UnreadByte() error			撤回上次ReadByte
			(b *Buffer) ReadRune(...)				获取缓冲区b中接下来的1个rune
			(b *Buffer) UnreadRune() error 			撤回上次ReadRune
			(b *Buffer) ReadBytes(...)				缓冲区b中读取数据直到遇到delim，返回获取到的数据line与错误err
			(b *Buffer) readSlice(delim byte)		#
			(b *Buffer) ReadString(...)				从缓冲区b中读取数据直到遇到delim，饭那会获取到的数据line与错误err

			--write
			(b *Buffer) Write(...)					向缓冲区b写入p
			(b *Buffer) WriteString(...)			向缓冲区b写入字符串s
			(b *Buffer) WriteTo(...)				将缓冲区b的未读取数据写入w中，返回已写入的字节数n与错误err
			(b *Buffer) WriteByte(c byte) error 	向缓冲区b写入字节c
			(b *Buffer) WriteRune(...)				向缓冲区b写入rune
		---bytes.go
		Equal(a, b []byte) bool						返回a和b是否长度相同并包含相同的字节
		Compare(a, b []byte) int					#
		explode(s []byte, n int) [][]byte			将s拆分为n个rune
		Count(s, sep []byte) int					获取s中出现sep的次数
		Join(s [][]byte, sep []byte) []byte 		将s合并为[]byte，用sep分隔
		HasPrefix(s, prefix []byte) bool			获取s是否以prefix开头
		HasSuffix(s, suffix []byte) bool			获取s是否以suffix结尾
		Map(...)									遍历s中的rune，获取满足mapping(r)的rune列表
		Repeat(b []byte, count int) []byte			获取count个b的切片
		isSeparator(r rune) bool					获取r是否不是字母、数字、下划线
		Title(s []byte) []byte						#
		makeASCIISet(...)							获取chars中的所有的ASCII as与是否全部为ASCII ok
			(as *asciiSet) contains(c byte) bool	获取as是否包含c
		makeCutsetFunc(...)							获取cutset是否包含c的方法
		Runes(s []byte) []rune						获取s中的所有rune
		Replace(s, old, new []byte, n int) []byte 	获取将前n个old替换为new的s，如果old为空，在s中的前n个rune之间添加new
		ReplaceAll(s, old, new []byte) []byte		获取将old替换为new的s
		EqualFold(s, t []byte) bool					历s与t中的rune比较是否相等（不区分大小写）
		hashStr(sep []byte) (uint32, uint32			获取sep的哈希值和在Rabin-Karp算法中使用的适当乘法因子
		hashStrRev(sep []byte) (uint32, uint32)		获取倒序的sep的哈希值和在Rabin-Karp算法中使用的适当乘法因子

		--trim
		Trim(s []byte, cutset string) []byte		去掉s中第一次和最后一次cutset中不包含的元素的索引之前和之后的数据，简化TrimFunc
		TrimFunc(...)								遍历s中的rune，去掉第一次和最后一次f(r) == false的索引之前和之后的数据s
		TrimLeft(s []byte, cutset string) []byte	去掉s中第一次cutset中不包含的元素的索引之前的数据，简化TrimLeftFunc
		TrimLeftFunc(...)							遍历s中的rune，返回去掉f(r) == false的索引之前数据的s
		TrimRight(s []byte, cutset string) []byte	去掉s中最后一次cutset中不包含的元素的索引之后的数据，简化TrimRightFunc
		TrimRightFunc(...)							倒序遍历s中的rune，返回去掉f(r) == false的索引之后数据的s
		TrimPrefix(s, prefix []byte) []byte			获取去掉前缀prefix之后的s
		TrimSuffix(s, suffix []byte) []byte			获取去掉后缀suffix之后的s
		TrimSpace(s []byte) []byte 					获取去掉空格后的s

		--to
		ToUpper(s []byte) []byte					获取s的大写字母
		ToLower(s []byte) []byte					获取s的小写字母
		ToTitle(s []byte) []byte					#获取s的标题大小写
		ToUpperSpecial(...)							#
		ToLowerSpecial(...)							#
		ToTitleSpecial(...)							#
		ToValidUTF8(s, replacement []byte) []byte	将s中无效的UTF-8字符替换为replacement并返回

		--fields
		Fields(s []byte) [][]byte					#
		FieldsFunc(...)								获取s中解析出的rune不满足f(r)的下一个rune的列表

		--split
		Split(s, sep []byte) [][]byte				将s按sep分割，简化genSplit
		SplitAfter(s, sep []byte) [][]byte			将s按sep分割，包括分隔符，简化genSplit
		SplitN(s, sep []byte, n int) [][]byte		将s按sep分割至少n份，简化genSplit
		SplitAfterN(s, sep []byte, n int) [][]byt	将s按sep分割至少n份，包括分隔符，简化genSplit
		genSplit(...)								#将s按sep分割至少n份，包括索引的sepSave字节

		--contains
		Contains(b, subslice []byte) bool			获取b是否包含subslice
		ContainsAny(b []byte, chars string) bool	获取chars的任一元素是否出现在b中
		ContainsRune(b []byte, r rune) bool			获取b中第1次出现r的索引

		--index
		Index(s, sep []byte) int 							获取s中第1次出现sep的索引
		indexRabinKarp(s, sep []byte) int 					使用Rabin-Karp字符串查找算法查找s中sep出现的位置
		IndexByte(b []byte, c byte) int						#获取b中第1次出现c的索引
		LastIndexByte(s []byte, c byte) int					获取s中c最后一次出现的索引
		indexBytePortable(s []byte, c byte) int				获取s中第1次出现c的索引
		LastIndex(s, sep []byte) int						获取s中最后1次出现sep的索引
		IndexRune(s []byte, r rune) int						获取s中第一次出现r的索引
		IndexAny(s []byte, chars string) int				获取charts中任一元素出现在s中的索引
		LastIndexAny(s []byte, chars string) int			获取charts中任一元素出现在s中的索引
		IndexFunc(s []byte, f func(r rune) bool) int		遍历s中的rune，获取f(r) == true的索引，简化indexFunc
		indexFunc(...)										遍历s中的rune，获取f(r) == truth的索引
		LastIndexFunc(s []byte, f func(r rune) bool) int	倒序遍历s中的rune，获取f(r) == true的索引，简化lastIndexFunc
		lastIndexFunc(...)									倒序遍历s中的rune，获取f(r) == truth的索引
		---reader.go
		NewReader(b []byte) *Reader							工厂函数，创建一个reader结构体，数据为b
			(r *Reader) Len() int							获取r的未读取字节数
			(r *Reader) Size() int64 						获取r的数据大小
			(r *Reader) Read(b []byte) (n int, err error)	从r中读取数据到b中，返回读取到的数据长度n与错误err
			(r *Reader) ReadAt(...)							从r中偏移量为off的位置读取数据到b中，返回数据到的数据长度n与错误err
			(r *Reader) ReadByte() (byte, error)			从r中读取一个字节并返回
			(r *Reader) UnreadByte() error					撤回上一次ReadByte()
			(r *Reader) ReadRune(...)						从r中读取一个rune，返回rune，rune的长度与错误err
			(r *Reader) UnreadRune() error					撤回上一次ReadRune()
			(r *Reader) Seek(...)							根据whence设置r未读取数据的索引，偏移量为ieoffset
			(r *Reader) WriteTo(...)						将r中未读取的数据写入w，返回写入数据的长度n与错误err
			(r *Reader) Reset(b []byte)						重置r，数据为b
*/