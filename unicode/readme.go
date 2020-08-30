/*
	文件用途：
		utf8/utf8.go			提供utf-8相关处理
		utf16/export_test.go	用于测试，但进到了unicode包里
		utf16/utf16.go			提供utf-16相关处理
		caetables.go			土耳其语编码相关规则
		digit.go				提供各种编码的十进制数字判断
		graphic.go				提供判断rune属于哪种字符集的
		letter.go				提供字母相关处理
		tables.go				提供各种编码、字符集常量

	---utf8/utf8.go
		接口与结构体：
			type acceptRange struct									rune第2个字节的范围结构体
		函数与方法：
			FullRune(p []byte) bool									获取p中是否有一个完整的rune
			DecodeRune(p []byte) (r rune, size int) 				获取p的第一个rune
			DecodeLastRune(p []byte) (r rune, size int)				倒序获取第一个rune
			RuneLen(r rune) int 									获取r占用的字节数
			EncodeRune(p []byte, r rune) int						将r写入p
			RuneCount(p []byte) int									获取p中rune的数量
			RuneStart(b byte) bool									b是否为rune的第一个字节
			Valid(p []byte) bool									获取p中所有rune是否全部有效
			ValidRune(r rune) bool									获取r是否为合法rune

			--string
			FullRuneInString(s string) bool							同FullRune
			DecodeRuneInString(s string) (r rune, size int)			同DecodeRune
			DecodeLastRuneInString(s string) (r rune, size int)		同DecodeLastRune
			RuneCountInString(s string) (n int) 					同RuneCount
			ValidString(s string) bool 								同Valid

	---utf16/utf16.go
		需要了解：
			UTF-16代理对
		函数与方法：
			IsSurrogate(r rune) bool			获取r是否可以出现在代理对中
			DecodeRune(r1, r2 rune) rune		获取r1作为代码对高位，r2作为代码对低位计算出的utf-16字符
			EncodeRune(r rune) (r1, r2 rune)	获取utf-16编码r的代理对，高位r1，低位r2
			Encode(s []rune) []uint16			将s中的utf-16拆分为utf-8编码并返回
			Decode(s []uint16) []rune			将s中的utf-8合并为utf-16编码并返回

	---digit.go
		函数与方法：
			IsDigit(r rune) bool 				#获取r是否为十进制数据

	---graphic.go
		函数与方法：
			IsGraphic(r rune) bool							获取r是否为图形字符（字符集：L, M, N, P, S, Zs）
			IsPrint(r rune) bool							获取r是否为可以打印字符（字符集：L, M, N, P, S）
			In(r rune, ranges ...*RangeTable) bool			获取r是否在ranges范围内
			IsOneOf(ranges []*RangeTable, r rune) bool		获取r是否在ranges范围内（同In）
			IsControl(r rune) bool							获取r是否为控制字符
			IsLetter(r rune) bool							获取r是否为字母（字符集：L）
			IsMark(r rune) bool								获取r是否为标记（字符集：M）
			IsNumber(r rune) bool							获取r是否为标记（字符集：N）
			IsPunct(r rune) bool							获取r是否为标点（字符集：P）
			IsSpace(r rune) bool							#获取r是否为空格
			IsSymbol(r rune) bool							获取r是否为符号（字符集：S）

	---letter.go
		接口与结构体：
			type RangeTable struct									字符集范围集合
			type Range16 struct										16位范围集合
			type Range32 struct										32位范围集合
			type CaseRange struct									大小写映射
			type foldPair struct									#
		函数与方法：
			is16(ranges []Range16, r uint16) bool					获取r是否在16位范围ranges中
			is32(ranges []Range32, r uint32) bool					获取r是否在32位范围ranges中
			Is(rangeTab *RangeTable, r rune) bool					获取r是否在rangeTab的范围内
			isExcludingLatin(rangeTab *RangeTable, r rune) bool		获取r是否在字符集rangeTab内
			IsUpper(r rune) bool									获取r是否为大写字母
			IsLower(r rune) bool									获取r是否为小写字母
			IsTitle(r rune) bool									获取r是否为标题大小写字母
			ToUpper(r rune) rune									获取r的大写字母（简化To）
			ToLower(r rune) rune									获取r的小写字母（简化To）
			ToTitle(r rune) rune									获取r的标题大小写（简化To）
			To(_case int, r rune) rune								简化（to）
			to(...)													#获取r根据对应的caseRange中_case的增量计算出的rune
			SimpleFold(r rune) rune									#简单折叠rune（将大写转为小写、将小写转为大写，将unicdoe转为ASCII，将特殊符号转为unicode）
			(special SpecialCase) ToUpper(r rune) rune				将special转为大写字母
			(special SpecialCase) ToTitle(r rune) rune				将special转为标题大小写字母
			(special SpecialCase) ToLower(r rune) rune				将special转为小写字母
*/