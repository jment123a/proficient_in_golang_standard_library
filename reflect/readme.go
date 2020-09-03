/*
名词
	已导出的字段：该变量是否由已导出的字段获得的，通常由一个比特位控制
		例1：var Exported int		// 已导出的字段，其他包可以引用
		例2: var notExported int	// 未导出的字段，其他包不可以引用
	标签：
		例：type T struct {
				f1 string "tag"		// tag为T.f1的标签
			}
	非复合类型：
		Bool、In、Int8、Int16、Int32、Int64、Uint、Uint8、Uint16、Uint32、Uint64、Uintptr、Float32、Float64、Complex64、Complex128、String、UnsafePointer
	复合类型：
		Array、Chan、Func、Interface、Map、Ptr、Slice、Struct
结构
	type emptyInterface struct {
		typ  *rtype
		word unsafe.Pointer
	}
	type rtype struct {		// 实现Type接口
		size    uintptr		//
		ptrdata uintptr		//
		tflag   tflag		//
	}
	type Value struct {		// Go值的反射类型
		typ *rtype			//
		ptr unsafe.Pointer	// 指向数据的指针
	}
	type mapType struct {	// 集合的反射类型
		rtype
		key    *rtype 		// key的类型
		elem   *rtype		// 元素（值）的类型
		bucket *rtype 		// 内部桶结构
	}

	例：map的结构
		1. var m map[string]int									类型为map[string]int
		2. val := reflect.ValueOf(m)							类型为Value
			2.1. unpackEface(val)								类型为Value
				2.1.1 (*emptyInterface)(unsafe.Pointer(&val))	类型为emptyInterface，将typ与word组合为Value
		3. (*mapType)(unsafe.Pointer(val.typ))					类型为mapType

	结果：
	emptyInterface.typ		map类型
	emptyInterface.word		指向数据的指针

	Value.typ				map类型
	Value.ptr				指向数据的指针

	mapType.rtype			map类型
	mapType.key				string类型
	mapType.elem			int类型
	mapType.bucket			struct类型

其他：
	type Value struct
	type ValueError struct
		(e *ValueError) Error() string
	type emptyInterface struct						空接口的反射类型
	type nonEmptyInterface struct					#
	type MapIter struct								遍历map的迭代器（游标）
	type StringHeader struct						外部字符串反射类型
	type stringHeader struct						内部字符串反射类型
	type SliceHeader struct							外部切片反射类型
	type sliceHeader struct							内部切片反射类型
	type runtimeSelect struct
	type SelectCase struct
	var dummy struct								用于将变量逃逸到堆
	---type.go
	type Type interface
	type rtype struct
	type method struct
	type uncommonType struct
	type arrayType struct
	type chanType struct
	type funcType struct
	type interfaceType struct
		(t *interfaceType) Method(i int) (m Method)		#获取接口类型t的第i个方法m
		(t *interfaceType) NumMethod() int				获取接口类型t的方法数量
		(t *interfaceType) MethodByName(...)			获取接口类型t中方法名为name的方法m
		type imethod struct
	type mapType struct
	type ptrType struct
	type sliceType struct
	type structType struct
		(t *structType) Field(i int) (f StructField)	获取结构体类型t的第i个字段f
		(t *structType) FieldByIndex(...)				获取t对应index的嵌套字段
		(t *structType) FieldByNameFunc(...)			#
		(t *structType) FieldByName(...)				获取结构体类型t中名为name的字段
		type structField struct
			(tag StructTag) Get(...)				获取tag中key为key的value，简化tag.Lookup
			(tag StructTag) Lookup(...)				获取tag中key为key的value与是否找到key
	type name struct
	type Method struct
	type fieldScan struct
	type cacheKey struct
	type funcTypeFixed4 struct						拥有4个参数的方法类型
	type funcTypeFixed8 struct						拥有8个参数的方法类型
	type funcTypeFixed16 struct						拥有16个参数的方法类型
	type funcTypeFixed32 struct						拥有32个参数的方法类型
	type funcTypeFixed64 struct						拥有64个参数的方法类型
	type funcTypeFixed128 struct					拥有128个参数的方法类型
	type structTypeUncommon struct
	type layoutKey struct
	type layoutType struct
	type bitVector struct
		(bv *bitVector) append(bit uint8)			#
	---makefunc.go
	type makeFuncImpl struct
	type methodValue struct
	---deepequal.go
	type visit struct
函数与方法列表
	--已导出方法
	New(typ Type) Value								获取一个新的typ类型的值反射类型
	Append(s Value, x ...Value) Value				切片类型s附加多个元素x
	AppendSlice(s, t Value) Value					切片类型s附加切片类型t
	Copy(dst, src Value) int						将src复制给dst
	Select(...)										执行一次select，case为cases，返回执行的case的索引
	Indirect(v Value) Value							获取指针类型v指向的值
	Zero(typ Type) Value							获取typ类型的零值
	NewAt(typ Type, p unsafe.Pointer) Value			#
	---type.go
	PtrTo(t Type) Type								#
	---swapper.go
	Swapper(slice interface{}) func(i, j int)		获取使slice的第i个元素与第j个元素交换的方法
	---makefunc.go
	MakeFunc(...)									#
	---deepequal.go
	DeepEqual(x, y interface{}) bool				获取x与y是否深度相等

	--未导出方法
	methodName() string								#
	methodReceiver(...)								获取v的第methodIndex个方法的接收器信息
	packEface(v Value) interface{}					将值反射类型v包装为空接口
	unpackEface(i interface{}) Value				将空接口解包为值反射类型
	storeRcvr(v Value, p unsafe.Pointer)			#
	align(x, n uintptr) uintptr						返回x按n为对齐后的长度
	funcName(f func([]Value) []Value) string		#
	valueInterface(...)								#将v转为空接口
	copyVal(...)									分配一个新值反射类型，数据从参数中拷贝
	overflowFloat32(x float64) bool					浮点类型x是否超过float32的界限
	typesMustMatch(what string, t1, t2 Type) 		t1与t2的类型必须一致，否则引发恐慌
	arrayAt(...)									获取p偏移i*eltSize后的指针
	grow(s Value, extra int) (Value, int, int)		扩容切片s，扩容长度至少为extra
	escapes(x interface{})							#设置x的注释标记为逃逸
	---type.go
	newName(n, tag string, exported bool) name		生成一个新name
		(n name) data(...)							获取n偏移off字节后的*byte
		(n name) isExported() bool					获取n是否为已导出字段
		(n name) nameLen() int						获取n的名称长度
		(n name) tagLen() int						获取n的标签长度
		(n name) name() (s string)					获取n的名称
		(n name) tag() (s string)					获取n的标签
		(n name) pkgPath() string					#获取n的程序包路径
	add(...)										将指针p偏移x字节并返回新指针
	fnv1(x uint32, list ...byte) uint32				使用fnv-1算法计算哈希
	implements(T, V *rtype) bool					获取类型V是否可以实现接口T
	specialChannelAssignability(T, V *rtype) 		获取管道类型V的值是否可以分配给管道类型T的值
	directlyAssignable(T, V *rtype) bool			获取V类型的值死否可以分配给T类型的值
	haveIdenticalType(T, V Type, cmpTags bool) bool	获取T和V是否为相同的类型
	haveIdenticalUnderlyingType(...)				获取T和V是否具有相同的基础类型
	rtypeOff(...)									#
	typesByString(s string) []*rtype				#
	funcStr(ft *funcType) string					获取方法类型ft的字符串表示形式
	isReflexive(t *rtype) bool						#
	needKeyUpdate(t *rtype) bool					#
	hashMightPanic(t *rtype) bool					#
	bucketOf(ktyp, etyp *rtype) *rtype				#
	emitGCMask(...)									#
	appendGCProg(dst []byte, typ *rtype) []byte		#
	isLetter(ch rune) bool							获取ch是否为字母或下划线
	isValidFieldName(fieldName string) bool			获取fieldName是否为有效的字段名称
	runtimeStructField(...)							将StructField类型转为structField
	typeptrdata(t *rtype) uintptr					#
	appendVarint(x []byte, v uintptr) []byte		#
	toType(t *rtype) Type							将*rtype类型转为Type类型
	funcLayout(...)									#
	ifaceIndir(t *rtype) bool						获取t是否为间接存储在接口值中
	addTypeBits(...)								#
	---makefunc.go
	makeFuncStub()									#签名
	makeMethodValue(op string, v Value) Value		#
	methodValueCall(...)							#签名
	deepValueEqual(...)								获取v1与v2是否深度相等



	--resolveReflect
	resolveReflectName(n name) nameOff				#
	resolveReflectType(t *rtype) typeOff			#
	resolveReflectText(ptr unsafe.Pointer) textOff	#

	--sign
	rselect(...)									#签名，运行一个select，返回执行的case的索引
	unsafe_New(*rtype) unsafe.Pointer				#签名，创建一个新对象
	unsafe_NewArray(*rtype, int) unsafe.Pointer		#签名，创建一个新数组
	ifaceE2I(...)									#签名
	memmove(dst, src unsafe.Pointer, size uintptr)	#签名，将size字节的src复制到dst
	typedmemmove(t *rtype, dst, src unsafe.Pointer)	#签名，将t类型的src赋值到dst
	typedmemmovepartial(...)						#签名，
	typedmemclr(t *rtype, ptr unsafe.Pointer) 		#签名，
	typedmemclrpartial(...)							#签名，
	typedslicecopy(...)								#签名，将elemType类型的src复制给dst
	typehash(...)									#签名，
	---
	resolveNameOff(...)								#签名，
	resolveTypeOff(...)								#签名，
	resolveTextOff(...)								#签名，
	addReflectOff(ptr unsafe.Pointer) int32			#签名，
	typelinks(...)									#签名，

	--call
	call(...)										#签名
	callReflect(...)								#
	callMethod(...)									#

	--make
	MakeSlice(typ Type, len, cap int) Value			创建一个切片的值反射类型
	MakeChan(typ Type, buffer int) Value			创建一个管道的值反射类型
	MakeMap(typ Type) Value							创建一个集合的值反射类型，简化MakeMapWithSize
	MakeMapWithSize(typ Type, n int) Value			创建一个集合的值反射类型
	makeInt(f flag, bits uint64, t Type) Value		创建一个int的值反射类型
	makeFloat(f flag, v float64, t Type) Value		创建一个float的值反射类型
	makeComplex(f flag, v complex128, t Type) Value	创建一个复杂类型的值反射类型
	makeString(f flag, v string, t Type) Value		创建一个字符串类型的值反射类型
	makeBytes(f flag, v []byte, t Type) Value		创建一个字节的值的值反射类型
	makeRunes(f flag, v []rune, t Type) Value		创建一个rune的值反射类型

	--convert
	convertOp(...)									获取将src类型的值转换为dst类型的值的函数
	cvtInt(v Value, t Type) Value					int -> int
	cvtUint(v Value, t Type) Value					uint -> int
	cvtFloatInt(v Value, t Type) Value				float -> int
	cvtFloatUint(v Value, t Type) Value				float -> uint
	cvtIntFloat(v Value, t Type) Value				int -> float
	cvtUintFloat(v Value, t Type) Value				uint -> float
	cvtFloat(v Value, t Type) Value					float -> float
	cvtComplex(v Value, t Type) Value				complex -> complex
	cvtIntString(v Value, t Type) Value 			int -> string
	cvtUintString(v Value, t Type) Value			uint -> string
	cvtBytesString(v Value, t Type) Value 			[]byte -> string
	cvtStringBytes(v Value, t Type) Value			string -> []byte
	cvtRunesString(v Value, t Type) Value			[]rune -> string
	cvtStringRunes(v Value, t Type) Value			string -> []rune
	cvtDirect(v Value, typ Type) Value				#
	cvtT2I(v Value, typ Type) Value					#concrete -> interface
	cvtI2I(v Value, typ Type) Value					#interface -> interface

	--chan
	makechan(...) 									#签名，创建类型为type，缓冲区大小为size的双向通道ch
	chancap(ch unsafe.Pointer) int 					#签名，获取管道类型ch的容量
	chanlen(ch unsafe.Pointer) int 					#签名，获取管道类型ch的长度
	chanclose(ch unsafe.Pointer)   					#签名，关闭管道ch
	chanrecv(...)									#签名，接受数据
	chansend(...)									#签名，发送数据

	--map
	makemap(t *rtype, cap int) (m unsafe.Pointer)	#签名，创建成员类型为t，容量为cap的集合
	mapaccess										#签名，
	mapassign										#签名，
	mapdelete										#签名，
	maplen(m unsafe.Pointer) int					#签名，获取集合的长度

	--mapiter
	mapiterinit										#签名，
	mapiterkey										#签名，
	mapiterelem										#签名，
	mapiternext(it unsafe.Pointer)					#签名，

	--XXXOf
	TypeOf(i interface{}) Type						获将i转为类型的反射类型
	ChanOf(dir ChanDir, t Type) Type				#
	MapOf(key, elem Type) Type						#
	TypeOf(i interface{}) Type						将i转为类型的反射类型
	StructOf(fields []StructField) Type				获取fields作为字段的结构体类型的反射类型
	FuncOf(in, out []Type, variadic bool) Type		#
	SliceOf(t Type) Type							#
	StructOf(fields []StructField) Type				#
	ArrayOf(count int, elem Type) Type				#
	ValueOf(i interface{}) Value					将i转为值反射类型（转储到堆中）
		--导出方法
		(v Value) Kind() Kind						获取v的类型枚举
			(k Kind) String() string				获取k的名称字符串
		(v Value) Addr() Value						#获取指针类型v指向的数据
		(v Value) UnsafeAddr() uintptr				获取uintptr类型v的指针
		(v Value) Index(i int) Value				获取数组类型v的第i个元素
		(v Value) Elem() Value						获取v的成员
		(v Value) InterfaceData() [2]uintptr		将v转为[2]uintptr
		(v Value) MapRange() *MapIter				工厂方法，将v转换为集合迭代器
			(it *MapIter) Key() Value 				#返回集合迭代器it当前的key
			(it *MapIter) Value() Value				#返回集合迭代器it当前的value
			(it *MapIter) Next() bool 				#将集合迭代器it指向下一个条目
		(v Value) Pointer() uintptr					获取v的指针
		(v Value) Type() Type						#
		(v Value) Convert(t Type) Value				获取转换为t类型的v

		--未导出方法
		(v Value) pointer() unsafe.Pointer			获取v指向数据的指针
		(v Value) assignTo(...)						#
		--array
		(v Value) Len() int							获取v的长度
		(v Value) SetLen(n int)						设置Slice类型v的长度为n
		(v Value) Cap() int							获取v的容量
		(v Value) SetCap(n int)						设置Slice类型v的容量为n

		--overflow
		(v Value) OverflowComplex(x complex128) bool	获取x是否不能用v的类型表示
		(v Value) OverflowFloat(x float64) bool			获取x是否不能用v的类型表示
		(v Value) OverflowInt(x int64) bool				获取x是否不能用v的类型表示
		(v Value) OverflowUint(x uint64) bool			获取x是否不能用v的类型表示

		--method
		(v Value) Method(i int) Value				获取v的第i个方法
		(v Value) NumMethod() int					获取v的已导出方法的数量，调用rtype.NumMethod()
		(v Value) MethodByName(name string) Value	获取v中名称为name的方法

		--is
		(v Value) IsNil() bool						v是否未初始化（v.ptr == nil）
		(v Value) IsValid() bool					v是否合法（v.flag != 0）
		(v Value) IsZero() bool						v是否为对应类型的零值

		--field
		(v Value) Field(i int) Value					获取结构体v的第i个字段
		(v Value) FieldByName(name string) Value		#，简化v.FieldByIndex
		(v Value) FieldByIndex(index []int) Value		获取v对应index的嵌套字段（v.Field[index[0]].Field[index[1]]...）
		(v Value) FieldByNameFunc(...)					#
		(v Value) NumField() int						获取结构体v的字段数量

		--can
		(v Value) CanAddr() bool						获取v是否为间接寻址
		(v Value) CanSet() bool							获取v是否可以修改
		(v Value) CanInterface() bool					获取v是否可以使用接口（是否为已导出的字段）

		--call
		(v Value) Call(in []Value) []Value				#，简化v.call
		(v Value) CallSlice(in []Value) []Value			#，简化v.call
		(v Value) call(op string, in []Value) []Value	#

		--get
		(v Value) String() string 				获取字符串类型v的值，如果不是字符串类型，获取类型的字符串表示形式
		(v Value) Int() int64					获取int类型v的值
		(v Value) Interface() (i interface{})	获取空接口类型的v的值
		(v Value) Bool() bool					获取boo类型v的值
		(v Value) Bytes() []byte				获取[]byte类型v的值
		(v Value) runes() []rune				获取[]rune类型v的值
		(v Value) Complex() complex128			获取复杂类型v的值
		(v Value) Float() float64				获取float类型v的值
		(v Value) Slice(i, j int) Value			获取v[i: j]
		(v Value) Slice3(i, j, k int) Value		获取v[i: j: k]
		(v Value) Uint() uint64					获取uint64类型v的值

		--set
		(v Value) Set(x Value)					#将x赋值给v
		(v Value) SetFloat(x float64)			将x赋值给float类型的v
		(v Value) SetInt(x int64)				将x赋值给int类型的v
		(v Value) SetBool(x bool)				将x赋值给bool类型的v
		(v Value) SetBytes(x []byte)			将x赋值给[]byte类型的v
		(v Value) setRunes(x []rune)			将x赋值给Slice类型v
		(v Value) SetComplex(x complex128)		将x赋值给复杂类型的v
		(v Value) SetUint(x uint64)				将x赋值给uint类型的v
		(v Value) SetPointer(x unsafe.Pointer)	将x赋值给unsafe.Pointer类型的v
		(v Value) SetString(x string)			将x赋值给string类型的v

		--map
		(v Value) MapIndex(key Value) Value		#
		(v Value) MapKeys() []Value				#获取集合类型v的所有key的值反射
		(v Value) SetMapIndex(key, elem Value)	#

		--chan
		(v Value) Send(x Value)					#向管道类型v发送x，简化v.send
		(v Value) TrySend(x Value) bool			#尝试向管道类型v发送x，简化v.send
		(v Value) send(...)						#向管道类型v发送x
		(v Value) Close()						关闭管道类型v
		(v Value) Recv() (x Value, ok bool)		#从通道v接受并返回一个值，简化v.recv
		(v Value) TryRecv() (x Value, ok bool)	#尝试从通道v接受并返回一个值，简化v.recv
		(v Value) recv(...)						#从通道v接受并返回一个值

		--flag
		(f flag) kind() Kind					获取f的类型枚举
		(f flag) ro() flag						如果f是未导出字段，返回未导出的非嵌入字段掩码
		(f flag) mustBe(expected Kind)			f必须为expected类型，否则引发恐慌
		(f flag) mustBeExported()				f必须为已导出对象，否则引发恐慌，简化f.mustBeExportedSlow
		(f flag) mustBeExportedSlow()			f必须为已导出对象，否则引发恐慌
		(f flag) mustBeAssignable()				f必须为间接寻址的已导出字段，否则引发恐慌，简化f.mustBeAssignableSlow
		(f flag) mustBeAssignableSlow()			f必须为间接寻址的已导出字段，否则引发恐慌

	--rtype
	(t *rtype) Field(i int) StructField		获取结构体类型t的第i个字段
		(f *structField) offset() uintptr	获取字段f在结构体内的偏移量
			(f *structField) embedded() bool	获取字段f是否为嵌入式字段（f是否是结构体）
	(t *rtype) NumField() int				获取结构体类型t的字段数量
	(t *rtype) Elem() Type					获取t的成员类型的反射类型
	(t *rtype) Implements(u Type) bool		获取t是否实现接口u
	---
	(t *rtype) nameOff(off nameOff) name	#
	(t *rtype) typeOff(off typeOff) *rtype	#
	(t *rtype) textOff(...)					#
	(t *rtype) uncommon() *uncommonType
	(t *uncommonType) methods() []method			获取t的所有方法
	(t *uncommonType) exportedMethods() []method	获取t的所有已导出方法
	(t *rtype) String() string				#获取t的名称
	(t *rtype) Size() uintptr				获取t.size，数据类型占用的字节数
	(t *rtype) Bits() int 					获取t的数据类型占用的位数
	(t *rtype) Align() int					获取t.align
	(t *rtype) FieldAlign() int				获取t.fieldAlign
	(t *rtype) Kind() Kind					获取t的类型枚举
	(t *rtype) pointers() bool				#获取t是否是指针
	(t *rtype) common() *rtype				获取t
	(t *rtype) exportedMethods() []method	获取t类型的已导出方法
	(t *rtype) NumMethod() int				获取t的已导出方法数量
	(t *rtype) Method(i int) (m Method)		#
	(t *rtype) MethodByName(...)			获取t名为name的方法m
	(t *rtype) PkgPath() string				#
	(t *rtype) hasName() bool				获取类型t是否有名称
	(t *rtype) Name() string				获取类型t的名称
	(t *rtype) ChanDir() ChanDir			获取管道类型t的管道方向
	(d ChanDir) String() string			获取管道方向d的字符串表示形式
	(t *rtype) FieldByName(...)				获取结构体类型t中名为name的字段
	(t *rtype) FieldByNameFunc(...)			#
	(t *rtype) Key() Type					#获取集合类型t的key的类型
	(t *rtype) Len() int					获取数组类型t的长度
	(t *rtype) NumField() int				获取结构体类型t的字段数量
	(t *rtype) ptrTo() *rtype				#
	(t *rtype) Implements(u Type) bool		获取类型t是否实现接口u
	(t *rtype) AssignableTo(u Type) bool	#
	(t *rtype) ConvertibleTo(u Type) bool	是否可以将t类型的值转换为u类型的值
	(t *rtype) Comparable() bool			获取类型t是否可以比较
	(t *rtype) gcSlice(...)					#



	--func_in
	(t *rtype) NumIn() int					获取方法类型t的输入参数数量
	(t *rtype) In(i int) Type				获取方法类型t的第i个输入参数的类型
	(t *funcType) in() []*rtype				获取方法类型t的输入参数
	(t *rtype) IsVariadic() bool			获取方法类型t的最后一个输入参数是否为可变参数（...）

	--func_out
	(t *rtype) NumOut() int					获取方法类型t的输出参数数量
	(t *rtype) Out(i int) Type				获取方法类型t的第i个输出参数的类型
	(t *funcType) out() []*rtype			获取方法类型t的输出参数
*/

package reflect
