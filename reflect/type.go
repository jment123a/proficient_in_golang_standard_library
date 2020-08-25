// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

// reflect包实现了运行时反射，从而允许程序处理任意类型的对象。
// 典型的用法是使用静态类型interface{}来获取一个值，并通过调用TypeOf来提取其动态类型信息，该类型将返回Type。
//
// 调用ValueOf返回代表运行时数据的Value。
// 零采用一个类型，并返回一个表示该类型的零值的值。
//
// 有关Go语言中反射的介绍，请参见“反射法则”：
// https://golang.org/doc/articles/laws_of_reflection.html
package reflect

import (
	"strconv"
	"sync"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

// Type是Go类型的表示。
//
// 并非所有方法都适用于所有类型。 在每种方法的文档中都注明了限制（如果有）。
// 在调用特定于种类的方法之前，使用Kind方法找出类型。 调用不适合该类型的方法会导致运行时恐慌。
//
// 类型值是可比较的，例如==运算符，因此它们可用作映射键。
// 如果两个Type值表示相同的类型，则它们相等。
type Type interface {
	// 适用于所有类型的方法。
	// 当在内存中分配时，Align返回此类型值的对齐方式（以字节为单位）。
	Align() int

	// FieldAlign 当用作结构体字段时，返回此类型值的对齐方式（以字节为单位）。
	FieldAlign() int

	// Method 返回类型的方法集中的第i个方法。
	// 如果i不在[0, NumMethod())范围内，则会发生恐慌。
	// 对于非接口类型T或*T，返回的Method的Type和Func字段描述了一个函数，其第一个参数为接收方。
	// 对于接口类型，返回的Method的Type字段给出方法签名，没有接收方，而Func字段为nil。
	// 仅可访问导出的方法，并且它们按字典顺序排序。
	Method(int) Method

	// MethodByName 返回该方法，该方法在类型的方法集中具有该名称，并带有一个布尔值，指示是否找到了该方法。
	// 对于非接口类型T或*T，返回的Method的Type和Func字段描述了一个函数，其第一个参数为接收方。
	// 对于接口类型，返回的Method的Type字段给出方法签名，没有接收方，而Func字段为nil。
	MethodByName(string) (Method, bool)

	// NumMethod 返回类型的方法集中导出的方法的数量。
	NumMethod() int

	// Name 返回其包中用于定义类型的类型名称。
	// 对于其他（未定义）类型，它返回空字符串。
	Name() string

	// PkgPath 返回定义的类型的包路径，即唯一标识包的导入路径，例如"encoding/base64".
	// 如果类型是预先声明的(string, error)或未定义(*T, struct{},  []int或A，其中A是未定义类型的别名)，则包路径将为空字符串 。
	PkgPath() string

	// Size 返回存储给定类型的值所需的字节数； 它类似于unsafe.Sizeof。
	Size() uintptr

	// String 返回该类型的字符串表示形式。
	// 字符串表示形式可以使用缩短的包名称（例如，使用base64代替"encoding/base64"），并且不能保证类型之间的唯一性。
	// 要测试类型标识，请直接比较类型。
	String() string

	// Kind 返回此类型的特定种类。
	Kind() Kind

	// Implements 报告该类型是否实现接口类型u。
	Implements(u Type) bool

	// AssignableTo 报告类型的值是否可分配给类型u。
	AssignableTo(u Type) bool

	// ConvertibleTo 报告类型的值是否可转换为类型u。
	ConvertibleTo(u Type) bool

	// Comparable 报告此类型的值是否可比较。
	Comparable() bool

	// 仅适用于某些类型的方法，具体取决于Kind。
	// 每种类型允许的方法是：
	//
	//	Int*, Uint*, Float*, Complex*: Bits
	//	Array: Elem, Len
	//	Chan: ChanDir, Elem
	//	Func: In, NumIn, Out, NumOut, IsVariadic.
	//	Map: Key, Elem
	//	Ptr: Elem
	//	Slice: Elem
	//	Struct: Field, FieldByIndex, FieldByName, FieldByNameFunc, NumField

	// Bits 返回以位为单位的类型的大小。
	// 如果类型的Kind不是大小型或未大小写的Int，Uint，Float或Complex类型之一，它会感到恐慌。
	Bits() int

	// ChanDir返回通道类型的方向。
	// 如果该类型的Kind不是Chan，则会感到恐慌。
	ChanDir() ChanDir

	// IsVariadic 报告函数类型的最终输入参数是否为"..."参数。 如果是这样，则t.In(t.NumIn() - 1)返回参数的隐式实际类型[]T。
	// 具体来说，如果t代表func（x int，y ... float64），则
	//
	//	t.NumIn() == 2
	//	t.In(0) is the reflect.Type for "int"
	//	t.In(1) is the reflect.Type for "[]float64"
	//	t.IsVariadic() == true
	//
	// IsVariadic 如果类型的Kind不是Func，则会发生混乱。
	IsVariadic() bool

	// Elem 返回类型的元素类型。
	// 如果类型的Kind不是Array，Chan，Map，Ptr或Slice，则会出现恐慌。
	Elem() Type

	// Field 返回结构类型的第i个字段。
	// 如果类型的Kind不是Struct，它会感到恐慌。
	// 如果我不在[0，NumField())范围内，则会发生恐慌。
	Field(i int) StructField

	// FieldByIndex 返回与索引序列相对应的嵌套字段。 等效于为每个索引i依次调用Field。
	// 如果类型的Kind不是Struct，它会感到恐慌。
	FieldByIndex(index []int) StructField

	// FieldByName 返回具有给定名称的struct字段和一个布尔值，指示是否找到了该字段。
	FieldByName(name string) (StructField, bool)

	// FieldByNameFunc 返回具有满足match函数名称的struct字段和一个布尔值，指示是否找到了该字段。
	// FieldByNameFunc首先考虑结构本身中的字段，然后再考虑所有嵌入结构中的字段，
	// 并以广度优先的顺序停止在最浅的嵌套深度，其中包含一个或多个满足match函数的字段。
	// 如果该深度处的多个字段满足匹配功能，则它们会相互抵消，并且FieldByNameFunc不返回匹配项。
	// 此行为反映了Go在包含嵌入式字段的结构中对名称查找的处理。
	FieldByNameFunc(match func(string) bool) (StructField, bool)

	// In 返回函数类型的第i个输入参数的类型。
	// 如果类型的Kind不是Func，它会感到恐慌。
	// 如果我不在[0, NumIn())范围内，则会发生恐慌。
	In(i int) Type

	// Key 返回地图类型的键类型。
	// 如果类型的Kind不是Map，它会感到恐慌。
	Key() Type

	// Len 返回数组类型的长度。
	// 如果类型的Kind不是Array，则会发生恐慌。
	Len() int

	// NumField 返回结构类型的字段计数。
	// 如果类型的Kind不是Struct，它会感到恐慌。
	NumField() int

	// NumIn 返回函数类型的输入参数计数。
	// 如果类型的Kind不是Func，它会感到恐慌。
	NumIn() int

	// NumOut 返回函数类型的输出参数计数。
	// 如果类型的Kind不是Func，它会感到恐慌。
	NumOut() int

	// Out 返回函数类型的第i个输出参数的类型。
	// 如果类型的Kind不是Func，它会感到恐慌。
	// 如果i不在[0, NumOut())范围内，则会发生恐慌。
	Out(i int) Type

	common() *rtype
	uncommon() *uncommonType
}

// BUG(rsc)：FieldByName和相关函数将结构字段名称视为相等，即使名称相同，
// 即使它们是源自不同包的未导出名称也是如此。
// 这样做的实际效果是，如果结构类型t包含多个名为x的字段（嵌入在不同的程序包中），
// 则t.FieldByName("x")的结果定义不明确。
// FieldByName可能返回名为x的字段之一，或者可能报告没有字段。
// 有关更多详细信息，请参见https://golang.org/issue/4876
/*
 * 这些数据结构是编译器已知的（../../cmd/internal/gc/reflect.go）。
 * ../runtime/type.go已知有一些可以传达给调试器。
 * 他们也以../runtime/type.go着称。
 */

// Kind 代表类型所代表的特定类型。
// 零种类不是有效种类。
type Kind uint // 注：rtype/Type的数据类型枚举，（应用：tflag.Kind()）

const (
	Invalid Kind = iota
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	Array
	Chan
	Func
	Interface
	Map
	Ptr
	Slice
	String
	Struct
	UnsafePointer
)

// rtype 使用tflag来指示紧随rtype值之后在内存中还有哪些额外的类型信息。
//
// tflag值必须与以下位置的副本保持同步：
//	cmd/compile/internal/gc/reflect.go
//	cmd/link/internal/ld/decodesym.go
//	runtime/type.go
type tflag uint8 //注：表示rtype之后的内存还有哪些额外的信息，这些信息可以靠将rtype转为tUncommon获取

const (
	// tflagUncommon 表示在外部类型结构的正上方有一个指针*uncommonType。
	// 例如，如果t.Kind() == Struct且t.tflag&tflagUncommon != 0，则t具有uncommonType数据，可以按以下方式访问它：
	//	type tUncommon struct {
	//		structType
	//		u uncommonType
	//	}
	//	u := &(*tUncommon)(unsafe.Pointer(t)).u
	tflagUncommon tflag = 1 << 0 // 注：紧随rtype之后的内存是否具有uncommonType数据

	// tflagExtraStar 表示str字段中的名称带有多余的"*"前缀。
	// 这是因为对于程序中的大多数T类型，*T类型也存在，并且重新使用str数据可节省二进制大小。
	tflagExtraStar tflag = 1 << 1 // 注：变量为指针时为0，否则为1

	// tflagNamed 表示类型具有名称。
	tflagNamed tflag = 1 << 2 // 注：类型是否有名称，例：int（tflag&tflagNamed == 1）、#（tflag&tflagNamed == 1）

	// tflagRegularMemory 意味着equal和hash函数可以将此类型视为t.size字节的单个区域。
	tflagRegularMemory tflag = 1 << 3 // 注：#
)

// rtype 是大多数值的通用实现。
// 它嵌入在其他结构类型中。
//
// rtype必须与../runtime/type.go:/^type._type保持同步。
type rtype struct { //注：ReflectType，反射的类型
	size    uintptr // 注：类型占用的字节数
	ptrdata uintptr // 类型中可以包含指针的字节数，注：是否为指针类型
	hash    uint32  // 类型的哈希； 避免在哈希表中进行计算，注：如果rtype为func，为输入、输出参数的hash
	// 注:

	// tflag[3]：equal和hash函数可以将此类型视为t.size字节的单个区域。
	// tflag[2]：类型是否具有名称
	// tflag[1]：str字段中的名称带有多余的"*"前缀
	// tflag[0]：是否具有uncommonType数据
	tflag      tflag // 额外类型信息标志
	align      uint8 // 变量与此类型的对齐
	fieldAlign uint8 // 结构域与该类型的对齐
	// kind[6]：#Type.gc指向GC程序
	// kind[5]：是否为间接指针，为0时rtype为间接指针
	// kind[0: 5]：类型枚举
	kind uint8 // C的枚举
	// 比较此类对象的函数（ptr与对象A，ptr与对象B）-> ==？
	equal     func(unsafe.Pointer, unsafe.Pointer) bool
	gcdata    *byte   // 垃圾收集数据
	str       nameOff // 字符串形式，注：到达名称的偏移量
	ptrToThis typeOff // 指向此类型的指针的类型，可以为零，注：指针的类型
}

// Method 在非接口类型上的方法
type method struct {
	name nameOff // 方法名称
	mtyp typeOff // 方法类型（无接收者）
	ifn  textOff // 接口调用中使用的fn（单字接收器）
	tfn  textOff // fn用于普通方法调用
}

// uncommonType 仅对于定义的类型或带有方法的类型存在（如果T是定义的类型，则T和*T的uncommonType具有方法）。
// 使用指向此结构的指针可减少描述没有方法的未定义类型所需的总体大小。
type uncommonType struct {
	pkgPath nameOff // 导入路径； 对于内置类型（如int，string）为空
	mcount  uint16  // 方法数量，注：所有方法的数量，mcount >= xcount，mcount - xcount = 未导出方法的数量
	xcount  uint16  // 导出方法的数量，注：所有已导出方法的数量
	moff    uint32  // 从uncommontype到[mcount]method的偏移量
	_       uint32  // 没用过
}

// ChanDir 表示通道类型的方向。
type ChanDir int // 注：管道的方向

const (
	RecvDir ChanDir             = 1 << iota // <-chan
	SendDir                                 // chan<-
	BothDir = RecvDir | SendDir             // chan
)

// arrayType 表示固定数组类型。
type arrayType struct { // 注：数组类型
	rtype         // 注：#
	elem  *rtype  // 数组元素类型
	slice *rtype  // 切片类型
	len   uintptr // 注：数组长度
}

// chanType 代表管道类型。
type chanType struct { // 注：管道类型
	rtype        // 注：#
	elem  *rtype // 通道元素类型，注：#
	// 注：dir = 01时只读，10时只写，11时为读写
	// dir[1]：只写
	// dir[0]：只读
	dir uintptr // 管道方向（ChanDir）
}

// funcType 表示函数类型。
// 每个in和out参数的*rtype存储在一个数组中，该数组紧随funcType（可能还有其uncommonType）。
// 因此，具有一个方法，一个输入和一个输出的函数类型为：
//	struct {
//		funcType
//		uncommonType
//		[2]*rtype    // [0] is in, [1] is out
//	}
// 注：funcType在内存中表现为：funcType_uncommonType_输入参数_输出参数

type funcType struct {
	rtype
	inCount uint16 // 注：输入参数的数量
	// 注：如果最后一个输入参数为...，outCount的最高位为1，否则为0
	outCount uint16 // 如果最后一个输入参数为...，则设置最高位，注：输出参数的数量
}

// imethod 表示接口类型上的方法
type imethod struct {
	name nameOff // 方法名称的偏移量
	typ  typeOff // .(*FuncType) underneath 注：返回类型的偏移量
}

// interfaceType 表示接口类型。
type interfaceType struct {
	rtype
	pkgPath name      // 导入路径，注：所在包位置
	methods []imethod // 按哈希排序，注：按哈希排序的方法集合
}

// mapType 	表示map类型
type mapType struct {
	rtype
	key    *rtype // key的类型
	elem   *rtype // 元素（值）的类型
	bucket *rtype // 内部桶结构
	// 散列键（点到key, seed）的函数 -> 散列
	hasher     func(unsafe.Pointer, uintptr) uintptr
	keysize    uint8  // key槽的尺寸
	valuesize  uint8  // val槽的尺寸
	bucketsize uint16 // 桶的尺寸
	flags      uint32
}

// ptrType 表示指针类型。
type ptrType struct {
	rtype
	elem *rtype // 指针元素（指向）类型
}

// sliceType 表示切片类型。
type sliceType struct {
	rtype
	elem *rtype // 切片元素类型
}

// 结构体字段
type structField struct {
	name name   // 名称始终为非空
	typ  *rtype // 字段的类型
	// offsetEmbed[1: 8]：字段在结构体内的偏移量
	// offsetEmbed[0]：是否为嵌入式字段
	offsetEmbed uintptr // （字段<<1 | isEmbedded）的字节偏移量，注：aa...ab，a为字段的偏移量，b为是否为嵌入式字段，所以要>>1

}

func (f *structField) offset() uintptr { //注：获取f在结构体内的偏移量
	return f.offsetEmbed >> 1
}

func (f *structField) embedded() bool { //注：获取f是否是嵌入式字段
	return f.offsetEmbed&1 != 0 //注：#嵌入式字段即为该字段不为基础类型
}

// structType 代表结构体类型。
type structType struct {
	rtype                 // 注：
	pkgPath name          // 注：程序包名称
	fields  []structField // 按偏移量排序，注：字段
}

// name 是带有可选的额外数据的编码类型名称。
// 第1个字节是1个位字段，其中包含：
//
// 1 << 0 是导出名称
// 1 << 1 标签数据跟随名称
// 1 << 2 pkgPath nameOff跟随名称和标记
//
// 接下来的两个字节是数据长度：
//
// l:= uint16(data[1])<<8 | uint16(data[2])
//
// bytes[3: 3 + l]是字符串数据。
// 如果跟随标签数据，则字节3 + l和3 + l + 1是标签长度，其后跟随数据。
// 如果遵循导入路径，则数据末尾的4个字节形成nameOff。 仅为在与包类型不同的包中定义的具体方法设置导入路径。
// 如果名称以"*"开头，则导出的位表示是否导出了所指向的类型。
//注：
// add(bytes, 3+l+nl)：标签数据
// add(bytes, 3+l+1):标签长度低位，最大长度65535（1111 1111 1111 1111），占用2个字节
// add(bytes, 3+l)：标签长度高位，最大长度65535（1111 1111 1111 1111），占用2个字节
// add(bytes, 3, "至3+l")：字符串数据
// add(bytes, 2, "")：数据长度低位，l:= uint16(bytes[1])<<8 | uint16(bytes[2])，最大长度65535（1111 1111 1111 1111），占用2个字节
// add(bytes, 1, "")：数据长度高位，l:= uint16(bytes[1])<<8 | uint16(bytes[2])，最大长度65535（1111 1111 1111 1111），占用2个字节

// bytes[7]：无
// bytes[6]：无
// bytes[5]：无
// bytes[4]：无
// bytes[3]：无
// bytes[2]：pkgPath
// bytes[1]：是否有标签
// bytes[0]：是否为已导出字段
type name struct {
	bytes *byte
}

func (n name) data(off int, whySafe string) *byte { // 注：获取name偏移off字节后的*byte，whySafe表示为什么这个操作是安全的
	return (*byte)(add(unsafe.Pointer(n.bytes), uintptr(off), whySafe))
}

func (n name) isExported() bool { //注：获取n是否导出
	return (*n.bytes)&(1<<0) != 0 //注：bytes[0] != 0
}

func (n name) nameLen() int { // 注：获取n的名称长度
	return int(uint16(*n.data(1, "name len field"))<<8 | uint16(*n.data(2, "name len field"))) // 注：add(1)和add(2)组成名称长度
}

func (n name) tagLen() int { // 注：获取n的标签长度
	if *n.data(0, "name flag field")&(1<<1) == 0 { // 注：bytes[1] == 0，如果标签长度为0，返回0
		return 0
	}
	off := 3 + n.nameLen()                                                                                 // 注：标志字节 + 名称长度高位字节 + 名称长度低位字节 + 名称数据字段
	return int(uint16(*n.data(off, "name taglen field"))<<8 | uint16(*n.data(off+1, "name taglen field"))) // 注：add(3 + off)和add(3 + off + 1)组合成标签长度
}

func (n name) name() (s string) { // 注：获取n的名称
	if n.bytes == nil {
		return
	}
	b := (*[4]byte)(unsafe.Pointer(n.bytes)) // 注：获取前4个字节，包括标志字节（0）、名称长度字节（1、2）、名称数据的第1个字节（3）

	hdr := (*stringHeader)(unsafe.Pointer(&s))
	hdr.Data = unsafe.Pointer(&b[3])   // 注：第3个字节开始是名称字符串
	hdr.Len = int(b[1])<<8 | int(b[2]) // 注：第1、2个字节是名称长度
	return s
}

func (n name) tag() (s string) { // 注：获取n的标签
	tl := n.tagLen() // 注：获取标签的长度
	if tl == 0 {     // 注：为0代表没有标签
		return ""
	}
	nl := n.nameLen() // 注：获取名称的长度
	hdr := (*stringHeader)(unsafe.Pointer(&s))
	hdr.Data = unsafe.Pointer(n.data(3+nl+2, "non-empty string")) // 注：标志字节 + 名称长度高位字节 + 名称长度低位字节 + 名称数据字段 + 标签长度高位字节 + 标签长度低位字节
	hdr.Len = tl
	return s
}

func (n name) pkgPath() string { // 注：#
	if n.bytes == nil || *n.data(0, "name flag field")&(1<<2) == 0 { // 注：bytes[2] == 0，如果没有程序包路径，返回""
		return ""
	}
	off := 3 + n.nameLen()        // 注：名称数据偏移量
	if tl := n.tagLen(); tl > 0 { // 注：标签数据偏移量
		off += 2 + tl
	}
	var nameOff int32
	// 请注意，此字段可能未在内存中对齐，因此我们在此处不能使用直接的int32分配。
	copy((*[4]byte)(unsafe.Pointer(&nameOff))[:], (*[4]byte)(unsafe.Pointer(n.data(off, "name offset field")))[:]) // 注：#
	pkgPathName := name{(*byte)(resolveTypeOff(unsafe.Pointer(n.bytes), nameOff))}                                 // 注：#
	return pkgPathName.name()                                                                                      // 注：#
}

func newName(n, tag string, exported bool) name { // 注：生成一个新name，名称为n，标签为tag，exported表示该name是否为已导出
	if len(n) > 1<<16-1 { // 注：65535，1111 1111 1111 1111，占用2个字节
		panic("reflect.nameFrom: name too long: " + n) // 恐慌："名称太长"
	}
	if len(tag) > 1<<16-1 { // 注：65535，1111 1111 1111 1111，占用2个字节
		panic("reflect.nameFrom: tag too long: " + tag) // 恐慌："标签太长"
	}

	var bits byte
	l := 1 + 2 + len(n)
	if exported { // 注：设置bytes[0] = 1，表示为已导出
		bits |= 1 << 0
	}
	if len(tag) > 0 {
		l += 2 + len(tag)
		bits |= 1 << 1 // 注：设置bytes[1] = 1，表示有标签数据
	}

	b := make([]byte, l)
	b[0] = bits               // 注：标志字节
	b[1] = uint8(len(n) >> 8) // 注：名称数据高位
	b[2] = uint8(len(n))      // 注：名称数据低位
	copy(b[3:], n)            // 注：名称数据
	if len(tag) > 0 {
		tb := b[3+len(n):]
		tb[0] = uint8(len(tag) >> 8) // 注：标签数据高位
		tb[1] = uint8(len(tag))      // 注：标签数据低位
		copy(tb[2:], tag)            // 注：标签数据
	}

	return name{bytes: &b[0]}
}

/*
 * 编译器知道上面所有数据结构的确切布局。
 * 编译器不了解以下数据结构和方法。
 */

// Method 表示单个方法。
type Method struct {
	// Name是方法名称。
	// PkgPath是限定小写（未导出）方法名称的程序包路径。 大写（导出）的方法名称为空。
	// PkgPath和Name的组合唯一标识方法集中的方法。
	// 参见https://golang.org/ref/spec#Uniqueness_of_identifiers
	Name    string // 注：方法名称
	PkgPath string // 注：程序包路径

	Type  Type  // 方法类型
	Func  Value // 以接收者为第一个参数的func
	Index int   // Type.Method的索引，注：表示第Index个方法
}

const (
	kindDirectIface = 1 << 5       // 注：0010 0000，是否是间接类型（指针），为0时为间接指针
	kindGCProg      = 1 << 6       // Type.gc指向GC程序，注：0100 0000
	kindMask        = (1 << 5) - 1 // 注：0001 1111，用于获取类型的掩码
)

// String 返回k的名称。
func (k Kind) String() string { // 注：获取k的名称字符串
	if int(k) < len(kindNames) { // 注：如果k在kindNames中，直接返回
		return kindNames[k]
	}
	return "kind" + strconv.Itoa(int(k)) // 注：否则转为字符串返回
}

var kindNames = []string{ // 注：kind的字符串对照
	Invalid:       "invalid",
	Bool:          "bool",
	Int:           "int",
	Int8:          "int8",
	Int16:         "int16",
	Int32:         "int32",
	Int64:         "int64",
	Uint:          "uint",
	Uint8:         "uint8",
	Uint16:        "uint16",
	Uint32:        "uint32",
	Uint64:        "uint64",
	Uintptr:       "uintptr",
	Float32:       "float32",
	Float64:       "float64",
	Complex64:     "complex64",
	Complex128:    "complex128",
	Array:         "array",
	Chan:          "chan",
	Func:          "func",
	Interface:     "interface",
	Map:           "map",
	Ptr:           "ptr",
	Slice:         "slice",
	String:        "string",
	Struct:        "struct",
	UnsafePointer: "unsafe.Pointer",
}

func (t *uncommonType) methods() []method { //注：获取t的所有方法
	if t.mcount == 0 { // 注：所有方法数量为0时，返回nil
		return nil
	}
	return (*[1 << 16]method)(add(unsafe.Pointer(t), uintptr(t.moff), "t.mcount > 0"))[:t.mcount:t.mcount]
}

func (t *uncommonType) exportedMethods() []method { //注：获取t的所有导出方法
	if t.xcount == 0 { // 注：所有导出方法数量为0时，返回nil
		return nil
	}
	//注：t指针偏移至方法的位置，获取所有的导出方法
	//注：根据methods()方法得知，xcount应该 <= t.mcount，即rtype之后的内存是导出的方法，再之后是未导出的方法，t.mcount-t.xcount就是未导出方法的数量
	return (*[1 << 16]method)(add(unsafe.Pointer(t), uintptr(t.moff), "t.xcount > 0"))[:t.xcount:t.xcount]
}

// resolveNameOff 解析与基本指针的名称偏移。
// (*rtype).nameOff 方法是此函数的便捷包装。
// 在runtime包中实现。
func resolveNameOff(ptrInModule unsafe.Pointer, off int32) unsafe.Pointer

// resolveTypeOff 从基本类型解析*rtype偏移量。
// (*rtype).typeOff 方法是此函数的便捷包装。
// 在runtime包中实现。
func resolveTypeOff(rtype unsafe.Pointer, off int32) unsafe.Pointer

// resolveTextOff 解析函数指针与基本类型的偏移量。
//(*rtype).textOff 方法是此函数的便捷包装。
// 在runtime包中实现。
func resolveTextOff(rtype unsafe.Pointer, off int32) unsafe.Pointer

// addReflectOff 在运行时中添加一个指向反射查找map的指针。
// 返回一个新的ID，该ID可以用作typeOff或textOff，并且可以正确解析。
// 在runtime包中实现。
func addReflectOff(ptr unsafe.Pointer) int32

// resolveReflectType 在运行时为反射查找图添加一个名称。
// 它返回一个新的nameOff，可以用来引用该指针。
func resolveReflectName(n name) nameOff {
	return nameOff(addReflectOff(unsafe.Pointer(n.bytes)))
}

// resolveReflectType 在运行时将*rtype添加到反射查找映射。
// 它返回一个新的typeOff，可以用来引用该指针。
func resolveReflectType(t *rtype) typeOff {
	return typeOff(addReflectOff(unsafe.Pointer(t)))
}

// resolveReflectText 在运行时向反射查找映射添加函数指针。
// 它返回一个新的textOff，可用于引用该指针。
func resolveReflectText(ptr unsafe.Pointer) textOff {
	return textOff(addReflectOff(ptr))
}

type nameOff int32 // 到达名称的偏移量
type typeOff int32 // 到达*rtype的偏移量
type textOff int32 // 与文字部分顶部的偏移量

func (t *rtype) nameOff(off nameOff) name { // 注：#
	return name{(*byte)(resolveNameOff(unsafe.Pointer(t), int32(off)))}
}

func (t *rtype) typeOff(off typeOff) *rtype { //注：#
	return (*rtype)(resolveTypeOff(unsafe.Pointer(t), int32(off)))
}

func (t *rtype) textOff(off textOff) unsafe.Pointer { // 注：#
	return resolveTextOff(unsafe.Pointer(t), int32(off))
}

func (t *rtype) uncommon() *uncommonType { //注：获取t的uncommonType
	if t.tflag&tflagUncommon == 0 { //注：t之后的内存没有uncommonType数据
		return nil
	}
	switch t.Kind() { //注：根据t的类型获取t的uncommonType数据
	case Struct:
		return &(*structTypeUncommon)(unsafe.Pointer(t)).u
	case Ptr:
		type u struct {
			ptrType
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case Func:
		type u struct {
			funcType
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case Slice:
		type u struct {
			sliceType
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case Array:
		type u struct {
			arrayType
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case Chan:
		type u struct {
			chanType
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case Map:
		type u struct {
			mapType
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case Interface:
		type u struct {
			interfaceType
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	default:
		type u struct {
			rtype
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	}
}

func (t *rtype) String() string { // 注：返回t的名字
	s := t.nameOff(t.str).name()     // 注：获取t的名称
	if t.tflag&tflagExtraStar != 0 { // 注：t不是指针，去掉*
		return s[1:]
	}
	return s
}

func (t *rtype) Size() uintptr { return t.size } // 注：获取t的数据类型占用的字节数

func (t *rtype) Bits() int { // 注：获取t的数据类型占用的位数
	if t == nil {
		panic("reflect: Bits of nil Type") // 恐慌："空类型的Bits"
	}
	k := t.Kind()
	if k < Int || k > Complex128 {
		panic("reflect: Bits of non-arithmetic Type " + t.String()) // 恐慌："非算术类型的Bits"
	}
	return int(t.size) * 8
}

func (t *rtype) Align() int { return int(t.align) } // 注：#

func (t *rtype) FieldAlign() int { return int(t.fieldAlign) } // 注：#

func (t *rtype) Kind() Kind { return Kind(t.kind & kindMask) } // 注：获取t的类型枚举

func (t *rtype) pointers() bool { return t.ptrdata != 0 } //注：t是否为指针（t.ptrdata是否不为0）

func (t *rtype) common() *rtype { return t } // 注： 获取t本身

func (t *rtype) exportedMethods() []method { //注：获取t作为类型的的导出方法
	ut := t.uncommon() //注：获取t的uncommon
	if ut == nil {
		return nil
	}
	return ut.exportedMethods() //注：返回t的已导出方法
}

func (t *rtype) NumMethod() int { //注：获取t的已导出方法数量
	if t.Kind() == Interface { //注：如果t是接口
		tt := (*interfaceType)(unsafe.Pointer(t))
		return tt.NumMethod() //注：返回方法数，接口全是导出方法
	}
	return len(t.exportedMethods()) //注：返回作为类型的导出方法数
}

func (t *rtype) Method(i int) (m Method) {
	if t.Kind() == Interface { // 注：如果t是接口类型
		tt := (*interfaceType)(unsafe.Pointer(t))
		return tt.Method(i)
	}
	methods := t.exportedMethods()
	if i < 0 || i >= len(methods) {
		panic("reflect: Method index out of range") // 恐慌："方法索引超出范围"
	}
	p := methods[i]
	pname := t.nameOff(p.name)
	m.Name = pname.name()
	fl := flag(Func)
	mtyp := t.typeOff(p.mtyp)
	ft := (*funcType)(unsafe.Pointer(mtyp))
	in := make([]Type, 0, 1+len(ft.in()))
	in = append(in, t)
	for _, arg := range ft.in() {
		in = append(in, arg)
	}
	out := make([]Type, 0, len(ft.out()))
	for _, ret := range ft.out() {
		out = append(out, ret)
	}
	mt := FuncOf(in, out, ft.IsVariadic())
	m.Type = mt
	tfn := t.textOff(p.tfn)
	fn := unsafe.Pointer(&tfn)
	m.Func = Value{mt.(*rtype), fn, fl}

	m.Index = i
	return m
}

func (t *rtype) MethodByName(name string) (m Method, ok bool) { // 注：返回t中名为name的方法m，与是否找到方法ok
	if t.Kind() == Interface { // 注：如果t是接口类型，直接返回
		tt := (*interfaceType)(unsafe.Pointer(t))
		return tt.MethodByName(name)
	}
	ut := t.uncommon()
	if ut == nil {
		return Method{}, false
	}
	// TODO(mdempsky): Binary search.
	for i, p := range ut.exportedMethods() { // 注：遍历t的已导出方法
		if t.nameOff(p.name).name() == name { // 注：如果找到了与name同名的额方法，返回
			return t.Method(i), true
		}
	}
	return Method{}, false
}

func (t *rtype) PkgPath() string {
	if t.tflag&tflagNamed == 0 {
		return ""
	}
	ut := t.uncommon()
	if ut == nil {
		return ""
	}
	return t.nameOff(ut.pkgPath).name()
}

func (t *rtype) hasName() bool { // 注：类型是否有名称
	return t.tflag&tflagNamed != 0
}

func (t *rtype) Name() string { // 注：获取t的类型名称
	// 例：
	// a := 1，返回int
	if !t.hasName() { // 注：类型没有名称，返回""
		return ""
	}
	s := t.String() // 注：获取t的类型名称
	i := len(s) - 1
	for i >= 0 && s[i] != '.' { // 注：返回.之后的字符串
		i--
	}
	return s[i+1:]
}

func (t *rtype) ChanDir() ChanDir { // 注：获取管道类型t的管道方向
	if t.Kind() != Chan { // 注：要求t是管道类型
		panic("reflect: ChanDir of non-chan type " + t.String()) // 恐慌："非管道类型的ChanDir"
	}
	tt := (*chanType)(unsafe.Pointer(t))
	return ChanDir(tt.dir)
}

func (t *rtype) IsVariadic() bool { // 注：获取t的最后一个输入参数是否为可变参数（...）
	if t.Kind() != Func { // 注：要求t是函数类型
		panic("reflect: IsVariadic of non-func type " + t.String()) // 恐慌："非函数类型的IsVariadic"
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return tt.outCount&(1<<15) != 0 // 注：返回输出参数最高位（1000 0000 0000 0000） != 0
}

func (t *rtype) Elem() Type { // 注：获取t的元素
	switch t.Kind() {
	case Array:
		tt := (*arrayType)(unsafe.Pointer(t))
		return toType(tt.elem)
	case Chan:
		tt := (*chanType)(unsafe.Pointer(t))
		return toType(tt.elem)
	case Map:
		tt := (*mapType)(unsafe.Pointer(t))
		return toType(tt.elem)
	case Ptr:
		tt := (*ptrType)(unsafe.Pointer(t))
		return toType(tt.elem)
	case Slice:
		tt := (*sliceType)(unsafe.Pointer(t))
		return toType(tt.elem)
	}
	panic("reflect: Elem of invalid type " + t.String()) // 恐慌："无效类型的元素"
}

func (t *rtype) Field(i int) StructField {
	if t.Kind() != Struct {
		panic("reflect: Field of non-struct type " + t.String()) // 恐慌："非结构体类型的字段"
	}
	tt := (*structType)(unsafe.Pointer(t))
	return tt.Field(i)
}

func (t *rtype) FieldByIndex(index []int) StructField { // 注：获取结构体类型t根据index递归查找的字段f
	if t.Kind() != Struct {
		panic("reflect: FieldByIndex of non-struct type " + t.String())
	}
	tt := (*structType)(unsafe.Pointer(t))
	return tt.FieldByIndex(index)
}

func (t *rtype) FieldByName(name string) (StructField, bool) { // 注：获取结构体类型t中名为name的字段，返回字段与是否找到该字段
	if t.Kind() != Struct { // 注：t必须是结构体类型
		panic("reflect: FieldByName of non-struct type " + t.String()) // 恐慌："非结构类型的FieldByName"
	}
	tt := (*structType)(unsafe.Pointer(t))
	return tt.FieldByName(name)
}

func (t *rtype) FieldByNameFunc(match func(string) bool) (StructField, bool) { // 注：#
	if t.Kind() != Struct {
		panic("reflect: FieldByNameFunc of non-struct type " + t.String()) // 恐慌："非结构类型的FieldByNameFunc"
	}
	tt := (*structType)(unsafe.Pointer(t))
	return tt.FieldByNameFunc(match)
}

func (t *rtype) In(i int) Type { // 注：获取方法类型t的第i个输入参数的类型
	if t.Kind() != Func { // 注：要求t是方法类型
		panic("reflect: In of non-func type " + t.String()) // 恐慌："非方法类型的In"
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return toType(tt.in()[i])
}

func (t *rtype) Key() Type { // 注：#
	if t.Kind() != Map { // 注：要求t是集合类型
		panic("reflect: Key of non-map type " + t.String()) // 恐慌："非集合类型的key"
	}
	tt := (*mapType)(unsafe.Pointer(t))
	return toType(tt.key)
}

func (t *rtype) Len() int { // 注：返回数组类型t的长度
	if t.Kind() != Array { // 注：要求t是数组类型
		panic("reflect: Len of non-array type " + t.String()) // 恐慌："非数组类型的长度"
	}
	tt := (*arrayType)(unsafe.Pointer(t))
	return int(tt.len)
}

func (t *rtype) NumField() int { // 注：获取结构体类型t的字段数量
	if t.Kind() != Struct { // 注：要求t是结构体类型
		panic("reflect: NumField of non-struct type " + t.String()) // 恐慌："非结构类型的NumField"
	}
	tt := (*structType)(unsafe.Pointer(t))
	return len(tt.fields)
}

func (t *rtype) NumIn() int { // 注：获取方法类型t的输入参数数量
	if t.Kind() != Func { // 注：要求t是方法类型
		panic("reflect: NumIn of non-func type " + t.String()) // 恐慌："非函数类型的NumIn"
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return int(tt.inCount)
}

func (t *rtype) NumOut() int { // 注：获取方法类型t的输出参数数量
	if t.Kind() != Func { // 注：要求t是方法类型
		panic("reflect: NumOut of non-func type " + t.String()) // 恐慌："非函数类型的NumOut"
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return len(tt.out())
}

func (t *rtype) Out(i int) Type { // 注：获取方法类型t的第i个输出参数
	if t.Kind() != Func { // 注：要求t是方法类型
		panic("reflect: Out of non-func type " + t.String()) // 恐慌："非函数类型的Out"
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return toType(tt.out()[i])
}

func (t *funcType) in() []*rtype { // 注：获取方法类型t的输入参数
	uadd := unsafe.Sizeof(*t)       // 注：方法占用的空间
	if t.tflag&tflagUncommon != 0 { // 注：方法有uncommonType
		uadd += unsafe.Sizeof(uncommonType{})
	}
	if t.inCount == 0 { // 注：如果输入参数 == 0，返回nil
		return nil
	}
	return (*[1 << 20]*rtype)(add(unsafe.Pointer(t), uadd, "t.inCount > 0"))[:t.inCount:t.inCount] // 注：获取方法t中t.inCount个rtype作为输入参数
}

func (t *funcType) out() []*rtype { // 注：获取方法类型t的输出参数
	uadd := unsafe.Sizeof(*t)       // 注：方法占用的空间
	if t.tflag&tflagUncommon != 0 { // 注：方法有uncommonType
		uadd += unsafe.Sizeof(uncommonType{})
	}
	outCount := t.outCount & (1<<15 - 1) // 注：第16位是标志位，所以忽略
	if outCount == 0 {                   // 注：如果输出参数 == 0，返回nil
		return nil
	}
	return (*[1 << 20]*rtype)(add(unsafe.Pointer(t), uadd, "outCount > 0"))[t.inCount : t.inCount+outCount : t.inCount+outCount] // 注：获取方法t中t.outCount个rtype作为输出参数
}

// add 返回 p+x.
//
// whySafe字符串将被忽略，因此该函数仍可以像p + x一样有效地内联，
// 但是所有调用站点都应使用该字符串来记录为什么加法是安全的，
// 也就是说加法为何不会导致x提前 到p分配的最后，因此错误地指向了内存中的下一个块。
func add(p unsafe.Pointer, x uintptr, whySafe string) unsafe.Pointer { // 注：将指针p偏移x字节并返回新指针，whySafe表示为什么这个操作是安全的
	return unsafe.Pointer(uintptr(p) + x)
}

func (d ChanDir) String() string { // 注：将管道方向转为字符串返回
	switch d {
	case SendDir:
		return "chan<-"
	case RecvDir:
		return "<-chan"
	case BothDir:
		return "chan"
	}
	return "ChanDir" + strconv.Itoa(int(d))
}

// Method 返回类型的方法集中的第i个方法。
func (t *interfaceType) Method(i int) (m Method) { // 注：#获取接口类型t的第i个方法m
	if i < 0 || i >= len(t.methods) { // 注：超出范围
		return
	}
	p := &t.methods[i]
	pname := t.nameOff(p.name) // 注：#
	m.Name = pname.name()
	if !pname.isExported() { // 注：如果是未导出方法
		m.PkgPath = pname.pkgPath()
		if m.PkgPath == "" { // 注：如果方法没有包路径，设置为接口的包路径
			m.PkgPath = t.pkgPath.name()
		}
	}
	m.Type = toType(t.typeOff(p.typ)) // 注：#
	m.Index = i
	return
}

// NumMethod 返回类型的方法集中的接口方法数。
func (t *interfaceType) NumMethod() int { return len(t.methods) } //注：获取接口类型t的方法数

// MethodByName 在类型的方法集中具有给定名称的方法。
func (t *interfaceType) MethodByName(name string) (m Method, ok bool) { // 注：#获取接口类型t中方法名为name的方法m，返回m与是否找到方法ok
	if t == nil {
		return
	}
	var p *imethod
	for i := range t.methods { // 注：遍历t的方法
		p = &t.methods[i]
		if t.nameOff(p.name).name() == name { // 注：#方法的名字与name相同，返回这个方法
			return t.Method(i), true
		}
	}
	return
}

// StructField 描述结构中的单个字段。
// 例1：包：main，结构体：structA，字段：fieldB byte `json:"123"`
// Name：fieldB，PkgPath：main，Type：uint8，Tag：json:"123"，Offset：0，Index：[0]，Anonymous：false
// 例1：包：main，结构体：structA，字段：structB `json:"321"`
// Name：structB，PkgPath：main，Type：main.structB，Tag：json:"321"，Offset：8，Index：[1]，Anonymous：true
type StructField struct { // 注：结构体字段
	// Name 是字段名称。
	Name string // 注：字段名称
	// PkgPath 是限定小写（未导出）字段名称的程序包路径。 大写（导出）字段名称为空。
	// 参见https://golang.org/ref/spec#Uniqueness_of_identifiers
	PkgPath string // 注：程序包路径

	Type      Type      // 字段类型，注：字段的类型
	Tag       StructTag // 字段标签字符串，注：字段的标签
	Offset    uintptr   // 结构中的偏移量（以字节为单位），注：结构体到当前字段的偏移量
	Index     []int     // Type.FieldByIndex的索引序列，注：结构体中的字段索引
	Anonymous bool      // 是一个嵌入式字段，注：该字段类型是否为结构体
}

// StructTag 是struct字段中的标记字符串。
// 按照惯例，标记字符串是由可选的以空格分隔的key："value"对的串联。
// 每个键都是一个非空字符串，由非控制字符组成，除了空格(U+0020 ' ')，
// 引号 (U+0022 '"')和冒号 (U+003A ':')。每个值 使用U+0022 '"'字符和Go字符串文字语法引用。
type StructTag string // 注：结构体标签

// Get 返回与标签字符串中的key关联的值。
// 如果标记中没有这样的键，则Get返回空字符串。
// 如果标记不具有常规格式，则未指定Get返回的值。 若要确定是否将标记明确设置为空字符串，请使用Lookup。
func (tag StructTag) Get(key string) string { // 注：获取结构体标签tag中key为key的value
	v, _ := tag.Lookup(key)
	return v
}

// Lookup 返回与标签字符串中的key关联的值。
// 如果关键字存在于标签中，则返回值（可能为空）。 否则，返回值将为空字符串。
// ok返回值报告该值是否在标记字符串中显式设置。 如果标记不具有常规格式，则未指定Lookup返回的值。
func (tag StructTag) Lookup(key string) (value string, ok bool) { // 注：获取tag中key为key的value，与是否找到key ok
	// 修改此代码时，还请更新cmd/vet/structtag.go中的validateStructTag代码。

	for tag != "" { // 注：遍历tag
		// 跳过前缀空格
		i := 0
		for i < len(tag) && tag[i] == ' ' { // 注：跳过前缀空格
			i++
		}
		tag = tag[i:]  // 注：丢掉空格
		if tag == "" { // 注：丢掉空格后为空，返回false
			break
		}

		// 扫描到冒号。 空格，引号或控制字符是语法错误。
		// 严格来说，控制字符包括范围[0x7f，0x9f]，而不仅仅是[0x00，0x1f]，但实际上，我们忽略了多字节控制字符，因为检查标签的字节比标签的符文更容易。
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f { // 注：遇到字符串
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i]) // 注：key
		tag = tag[i+1:]         // 注：key之后的字符串

		// 扫描带引号的字符串以查找值。
		i = 1
		for i < len(tag) && tag[i] != '"' { // 注：跳过开始"，遍历到结束"
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1]) // 注：value
		tag = tag[i+1:]             // 注：value之后的字符串

		if key == name { // 注：key和name相同
			value, err := strconv.Unquote(qvalue) // 注：#
			if err != nil {
				break
			}
			return value, true
		}
	}
	return "", false
}

// Field 返回第i个结构体字段。
func (t *structType) Field(i int) (f StructField) { // 注：获取t的第i个字段f
	if i < 0 || i >= len(t.fields) {
		panic("reflect: Field index out of bounds") // 恐慌："字段索引超出范围"
	}
	p := &t.fields[i]
	f.Type = toType(p.typ)     // 注：p.type转为Type
	f.Name = p.name.name()     // 注：根据p.name获取字段的名称
	f.Anonymous = p.embedded() // 注：p是否为嵌入式字段
	if !p.name.isExported() {  // 注：未导出字段要赋值程序包路径
		f.PkgPath = t.pkgPath.name()
	}
	if tag := p.name.tag(); tag != "" { // 注：如果有标签，赋值标签
		f.Tag = StructTag(tag)
	}
	f.Offset = p.offset() // 注：字段在结构体内的偏移量

	// NOTE（rsc）：这是reflect.Type在接口中提供的唯一分配。
	// 至少在通常情况下，最好避免这种情况，但是我们需要确保行为不当的反射客户不会影响反射的其他用途。
	// 一种可能是CL 5371098，但我们推迟了这种丑陋，直到表现出对性能的需求为止。 这是问题2320。
	f.Index = []int{i} // 注：字段索引
	return
}

// TODO（gri）：如果FieldByIndex的索引错误，是否应该有一个错误/布尔指示器？

// FieldByIndex 返回与索引对应的嵌套字段。
func (t *structType) FieldByIndex(index []int) (f StructField) { // 注：获取t根据index递归查找的字段f
	// 例：index = []int{1, 4, 5}
	// 第1次循环：f1 = t.Type.Field(1)
	// 第1次循环：f2 = f1.Type.Field(4)
	// 第1次循环：f3 = f2.Type.Field(5)
	f.Type = toType(&t.rtype)
	for i, x := range index { // 注：遍历index
		if i > 0 {
			ft := f.Type
			if ft.Kind() == Ptr && ft.Elem().Kind() == Struct {
				ft = ft.Elem()
			}
			f.Type = ft
		}
		f = f.Type.Field(x) // 注：获取第index[i]个字段
	}
	return
}

// fieldScan代表fieldByNameFunc扫描工作列表上的项目。
type fieldScan struct {
	typ   *structType
	index []int
}

// FieldByNameFunc 返回结构名称为struct的字段
// 匹配函数和一个布尔值，指示是否找到了该字段。
func (t *structType) FieldByNameFunc(match func(string) bool) (result StructField, ok bool) { // 注：#
	// 这使用了与Go语言相同的条件：在给定的深度级别，必须有唯一的匹配实例。
	// 如果在同一深度有多个匹配的实例，它们将相互消灭并在较低级别禁止任何可能的匹配。
	// 该算法是广度优先搜索，一次是一个深度级别。

	// current和next切片是工作队列：
	// current 列出该深度级别上要访问的字段，而next列出下一个较低级别上的字段。
	current := []fieldScan{}
	next := []fieldScan{{typ: t}}

	// nextCount 记录遇到嵌入式类型并考虑在“下一个”切片中进行排队的次数。
	// 我们只将第一个排队，但是我们增加每个的计数。
	// 如果在给定的深度级别可以多次到达结构类型T，那么它将消灭自身，并且在我们处理下一个深度级别时根本不需要考虑它。
	var nextCount map[*structType]int

	// visited 记录已经考虑过的结构。
	// 嵌入式指针字段可以在可达的嵌入式类型图中创建循环； 来访者避免遵循这些周期。
	// 这也避免了重复的工作：如果我们在第2级没有找到嵌入类型T的字段，在第4级也不会在一个类型中找到该字段。
	visited := map[*structType]bool{}

	for len(next) > 0 {
		current, next = next, current[:0]
		count := nextCount
		nextCount = nil

		// Process all the fields at this depth, now listed in 'current'.
		// The loop queues embedded fields found in 'next', for processing during the next
		// iteration. The multiplicity of the 'current' field counts is recorded
		// in 'count'; the multiplicity of the 'next' field counts is recorded in 'nextCount'.
		for _, scan := range current {
			t := scan.typ
			if visited[t] {
				// We've looked through this type before, at a higher level.
				// That higher level would shadow the lower level we're now at,
				// so this one can't be useful to us. Ignore it.
				continue
			}
			visited[t] = true
			for i := range t.fields {
				f := &t.fields[i]
				// Find name and (for embedded field) type for field f.
				fname := f.name.name()
				var ntyp *rtype
				if f.embedded() {
					// Embedded field of type T or *T.
					ntyp = f.typ
					if ntyp.Kind() == Ptr {
						ntyp = ntyp.Elem().common()
					}
				}

				// Does it match?
				if match(fname) {
					// Potential match
					if count[t] > 1 || ok {
						// Name appeared multiple times at this level: annihilate.
						return StructField{}, false
					}
					result = t.Field(i)
					result.Index = nil
					result.Index = append(result.Index, scan.index...)
					result.Index = append(result.Index, i)
					ok = true
					continue
				}

				// Queue embedded struct fields for processing with next level,
				// but only if we haven't seen a match yet at this level and only
				// if the embedded types haven't already been queued.
				if ok || ntyp == nil || ntyp.Kind() != Struct {
					continue
				}
				styp := (*structType)(unsafe.Pointer(ntyp))
				if nextCount[styp] > 0 {
					nextCount[styp] = 2 // exact multiple doesn't matter
					continue
				}
				if nextCount == nil {
					nextCount = map[*structType]int{}
				}
				nextCount[styp] = 1
				if count[t] > 1 {
					nextCount[styp] = 2 // exact multiple doesn't matter
				}
				var index []int
				index = append(index, scan.index...)
				index = append(index, i)
				next = append(next, fieldScan{styp, index})
			}
		}
		if ok {
			break
		}
	}
	return
}

// FieldByName 返回具有给定名称的struct字段和一个布尔值，以指示是否找到了该字段。
func (t *structType) FieldByName(name string) (f StructField, present bool) { // 注：获取结构体类型t中名为name的字段，返回字段f与是否找到该字段present
	// 快速检查顶级名称或没有嵌入字段的结构。
	hasEmbeds := false
	if name != "" {
		for i := range t.fields { // 注：遍历结构体t的字段
			tf := &t.fields[i]
			if tf.name.name() == name { // 注：如果字段名称等于name，返回字段
				return t.Field(i), true
			}
			if tf.embedded() { // 注：是嵌入式字段
				hasEmbeds = true
			}
		}
	}
	if !hasEmbeds {
		return
	}
	return t.FieldByNameFunc(func(s string) bool { return s == name }) // 注：#
}

// TypeOf 返回表示i的动态类型的反射类型。
// 如果i是一个nil接口值，则TypeOf返回nil。
func TypeOf(i interface{}) Type { // 注：获取Type类型的i
	eface := *(*emptyInterface)(unsafe.Pointer(&i)) //注：将i转为空接口反射类型
	return toType(eface.typ)                        //注：获取i的类型并返回
}

// ptrMap 是PtrTo的缓存。
var ptrMap sync.Map // map[*rtype]*ptrType

// PtrTo 返回带有元素t的指针类型。
// 例如，如果t表示类型Foo，则PtrTo(t)表示*Foo。
func PtrTo(t Type) Type { // 注：#
	return t.(*rtype).ptrTo()
}

func (t *rtype) ptrTo() *rtype { // 注：#
	if t.ptrToThis != 0 { // 注：#
		return t.typeOff(t.ptrToThis)
	}

	// 检查缓存。
	if pi, ok := ptrMap.Load(t); ok { // 注：检查缓存
		return &pi.(*ptrType).rtype
	}

	// 查找已知类型。
	s := "*" + t.String()
	for _, tt := range typesByString(s) {
		p := (*ptrType)(unsafe.Pointer(tt))
		if p.elem != t {
			continue
		}
		pi, _ := ptrMap.LoadOrStore(t, p)
		return &pi.(*ptrType).rtype
	}

	// Create a new ptrType starting with the description
	// of an *unsafe.Pointer.
	var iptr interface{} = (*unsafe.Pointer)(nil)
	prototype := *(**ptrType)(unsafe.Pointer(&iptr))
	pp := *prototype

	pp.str = resolveReflectName(newName(s, "", false))
	pp.ptrToThis = 0

	// For the type structures linked into the binary, the
	// compiler provides a good hash of the string.
	// Create a good hash for the new string by using
	// the FNV-1 hash's mixing function to combine the
	// old hash and the new "*".
	pp.hash = fnv1(t.hash, '*')

	pp.elem = t

	pi, _ := ptrMap.LoadOrStore(t, &pp)
	return &pi.(*ptrType).rtype
}

// fnv1 使用FNV-1哈希函数将字节列表合并到哈希x中。
func fnv1(x uint32, list ...byte) uint32 { // 注：FNV-1算法
	for _, b := range list {
		x = x*16777619 ^ uint32(b)
	}
	return x
}

func (t *rtype) Implements(u Type) bool { // 注：获取类型t是否实现接口u
	if u == nil {
		panic("reflect: nil type passed to Type.Implements") // 恐慌："nil类型传递给Type.Implements"
	}
	if u.Kind() != Interface {
		panic("reflect: non-interface type passed to Type.Implements") // 恐慌："非接口类型传递给Type.Implements"
	}
	return implements(u.(*rtype), t)
}

func (t *rtype) AssignableTo(u Type) bool { // 注：#
	if u == nil {
		panic("reflect: nil type passed to Type.AssignableTo") // 恐慌："nil类型传递给Type.AssignableTo"
	}
	uu := u.(*rtype)
	return directlyAssignable(uu, t) || implements(uu, t)
}

func (t *rtype) ConvertibleTo(u Type) bool { // 注：是否可以将t类型的值转换为u类型的值
	if u == nil {
		panic("reflect: nil type passed to Type.ConvertibleTo") // 恐慌："nil类型传递给Type.ConvertibleTo"
	}
	uu := u.(*rtype)
	return convertOp(uu, t) != nil
}

func (t *rtype) Comparable() bool { // 注：t类型是否可以比较
	return t.equal != nil
}

// implements 报告类型V是否实现接口类型T。
func implements(T, V *rtype) bool { // 注：获取类型V是否实现接口T
	if T.Kind() != Interface { // 注：T不是接口类型，返回false
		return false
	}
	t := (*interfaceType)(unsafe.Pointer(T)) // 注：T转为接口反射类型
	if len(t.methods) == 0 {                 // 注：如果T没有方法，则为空接口，可以被任何类型实现，返回true
		return true
	}

	// 两种情况下都使用相同的算法，但是接口类型和具体类型的方法表不同，因此代码重复。
	// 在两种情况下，该算法都是同时对两个列表（T方法和V方法）进行线性扫描。
	// 由于方法表是以唯一的排序顺序存储的（字母顺序，没有重复的方法名称），因此对V方法的扫描必须沿途对每个T方法进行匹配，否则V不会实现T。
	// 这样一来，我们就可以在整个线性时间内运行扫描，而不是天真的搜索所需的二次时间。
	// 另请参阅../runtime/iface.go。
	if V.Kind() == Interface { // 注：如果V是接口，检查v是否实现了T的所有接口
		v := (*interfaceType)(unsafe.Pointer(V)) // 注：V转为接口反射类型
		i := 0
		for j := 0; j < len(v.methods); j++ { // 注：遍历V的方法
			tm := &t.methods[i] // 注：T的方法
			tmName := t.nameOff(tm.name)
			vm := &v.methods[j] // 注：V的方法
			vmName := V.nameOff(vm.name)
			if vmName.name() == tmName.name() && V.typeOff(vm.typ) == t.typeOff(tm.typ) { // 注：#V有与T.methods[i]同名的方法
				if !tmName.isExported() { // 注：如果T.methods[i]是未导出方法
					tmPkgPath := tmName.pkgPath()
					if tmPkgPath == "" {
						tmPkgPath = t.pkgPath.name()
					}
					vmPkgPath := vmName.pkgPath()
					if vmPkgPath == "" {
						vmPkgPath = v.pkgPath.name()
					}
					if tmPkgPath != vmPkgPath { // 注：比较程序包路径，程序包路径不同则方法不相同
						continue
					}
				}
				if i++; i >= len(t.methods) { // 注：如果V实现了T的所有方法，返回true
					return true
				}
			}
		}
		return false
	}

	v := V.uncommon() // 注：获取V的uncommonType
	if v == nil {     // 注：如果没有方法，无法实现有方法的T接口，返回false
		return false
	}
	i := 0
	vmethods := v.methods()              // 注：获取v的所有方法
	for j := 0; j < int(v.mcount); j++ { // 注：遍历V的方法
		tm := &t.methods[i] // 注：T的方法
		tmName := t.nameOff(tm.name)
		vm := vmethods[j] // 注：V的方法
		vmName := V.nameOff(vm.name)
		if vmName.name() == tmName.name() && V.typeOff(vm.mtyp) == t.typeOff(tm.typ) { // 注：#V有与T.methods[i]同名的方法
			if !tmName.isExported() { // 注：如果T.methods[i]是未导出方法
				tmPkgPath := tmName.pkgPath()
				if tmPkgPath == "" {
					tmPkgPath = t.pkgPath.name()
				}
				vmPkgPath := vmName.pkgPath()
				if vmPkgPath == "" {
					vmPkgPath = V.nameOff(v.pkgPath).name()
				}
				if tmPkgPath != vmPkgPath { // 注：比较程序包路径，程序包路径不同则方法不相同
					continue
				}
			}
			if i++; i >= len(t.methods) { // 注：如果V实现了T的所有方法，返回true
				return true
			}
		}
	}
	return false
}

// specialChannelAssignability 报告是否可以直接（使用记忆）将通道类型V的值x分配给另一个通道类型T。
// https://golang.org/doc/go_spec.html#Assignability T和V必须均为Chan类型。
func specialChannelAssignability(T, V *rtype) bool { // 注：获取管道类型V的值是否可以分配给管道类型T的值
	// 特殊情况：
	// x是双向通道值，T是通道类型，x的类型V和T具有相同的元素类型，并且V或T中的至少一个不是定义的类型。
	return V.ChanDir() == BothDir && (T.Name() == "" || V.Name() == "") && haveIdenticalType(T.Elem(), V.Elem(), true)
}

// directlyAssignable 报告是否可以将V类型的值x直接（使用记忆）分配给T类型的值。
// https://golang.org/doc/go_spec.html#Assignability忽略接口规则（在其他地方实现）和理想的常量规则（运行时没有理想的常量）。
func directlyAssignable(T, V *rtype) bool { // 注：获取V类型的值是否可以分配给T类型的值
	// x的类型V等于T？
	if T == V { // 注：T与V相等，返回true
		return true
	}

	// 否则，不得定义T和V中的至少一个，并且它们必须具有相同的种类。
	if T.hasName() && V.hasName() || T.Kind() != V.Kind() { // 注：#T与V都具有名称或T与V类型不同，返回false
		return false
	}

	if T.Kind() == Chan && specialChannelAssignability(T, V) { // 注：T是管道类型并且
		return true
	}

	// x的类型T和V必须具有相同的基础类型。
	return haveIdenticalUnderlyingType(T, V, true)
}

func haveIdenticalType(T, V Type, cmpTags bool) bool { // 注：获取T和V是否有相同的类型，cmpTags表示是否可以直接比较
	if cmpTags {
		return T == V
	}

	if T.Name() != V.Name() || T.Kind() != V.Kind() { // 注：如果名称不同或类型不同，返回false
		return false
	}

	return haveIdenticalUnderlyingType(T.common(), V.common(), false)
}

func haveIdenticalUnderlyingType(T, V *rtype, cmpTags bool) bool { // 注：获取T和V是否具有相同的基础类型，cmpTags表示是否可以直接比较
	if T == V { // 注：如果相等，直接返回true
		return true
	}

	kind := T.Kind()
	if kind != V.Kind() { // 注：如果类型不同，返回false
		return false
	}

	// 相同种类的非复合类型具有相同的基础类型（该类型的预定义实例）。
	if Bool <= kind && kind <= Complex128 || kind == String || kind == UnsafePointer { // 注：如果T和V为非复合类型并且相同，返回true
		return true
	}

	// 复合类型。
	switch kind {
	case Array: // 注：比较长度与成员类型
		return T.Len() == V.Len() && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Chan: // 注：比较管道方向与成员类型
		return V.ChanDir() == T.ChanDir() && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Func: // 注：比较输入、输出参数数量与类型
		t := (*funcType)(unsafe.Pointer(T))
		v := (*funcType)(unsafe.Pointer(V))
		if t.outCount != v.outCount || t.inCount != v.inCount {
			return false
		}
		for i := 0; i < t.NumIn(); i++ {
			if !haveIdenticalType(t.In(i), v.In(i), cmpTags) {
				return false
			}
		}
		for i := 0; i < t.NumOut(); i++ {
			if !haveIdenticalType(t.Out(i), v.Out(i), cmpTags) {
				return false
			}
		}
		return true

	case Interface: // 注：没有方法才为true
		t := (*interfaceType)(unsafe.Pointer(T))
		v := (*interfaceType)(unsafe.Pointer(V))
		if len(t.methods) == 0 && len(v.methods) == 0 {
			return true
		}
		// 可能具有相同的方法，但仍需要运行时转换。
		return false

	case Map: // 注：比较key与value
		return haveIdenticalType(T.Key(), V.Key(), cmpTags) && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Ptr, Slice: // 注：比较成员类型
		return haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Struct: // 注：比较字段数量、程序包名称、字段名称、字段类型、字段标签、字段偏移量
		t := (*structType)(unsafe.Pointer(T))
		v := (*structType)(unsafe.Pointer(V))
		if len(t.fields) != len(v.fields) {
			return false
		}
		if t.pkgPath.name() != v.pkgPath.name() {
			return false
		}
		for i := range t.fields {
			tf := &t.fields[i]
			vf := &v.fields[i]
			if tf.name.name() != vf.name.name() {
				return false
			}
			if !haveIdenticalType(tf.typ, vf.typ, cmpTags) {
				return false
			}
			if cmpTags && tf.name.tag() != vf.name.tag() {
				return false
			}
			if tf.offsetEmbed != vf.offsetEmbed {
				return false
			}
		}
		return true
	}

	return false
}

// typelinks 在runtime包中实现。
// 它在每个模块中返回一部分的切片，并在每个模块中返回*rtype偏移量的切片。
// 每个模块中的类型均按字符串排序。 即，第一个模块的前两个链接类型是：
//
//	d0 := sections[0]
//	t1 := (*rtype)(add(d0, offset[0][0]))
//	t2 := (*rtype)(add(d0, offset[0][1]))
// 和
//	t1.String() < t2.String()
// 注意，字符串不是类型的唯一标识符：
// 一个给定的字符串可以有多个。
// 仅包含我们可能要查找的类型：
// 指针，通道，地图，切片和数组。
func typelinks() (sections []unsafe.Pointer, offset [][]int32)

func rtypeOff(section unsafe.Pointer, off int32) *rtype {
	return (*rtype)(add(section, uintptr(off), "sizeof(rtype) > 0"))
}

// typesByString 返回typelinks()的子片段，其元素具有给定的字符串表示形式。
// 它可能为空（该字符串没有已知类型）或可能具有多个元素（该字符串为多个类型）。
func typesByString(s string) []*rtype { // 注：#
	sections, offset := typelinks()
	var ret []*rtype

	for offsI, offs := range offset {
		section := sections[offsI]

		// We are looking for the first index i where the string becomes >= s.
		// This is a copy of sort.Search, with f(h) replaced by (*typ[h].String() >= s).
		i, j := 0, len(offs)
		for i < j {
			h := i + (j-i)/2 // avoid overflow when computing h
			// i ≤ h < j
			if !(rtypeOff(section, offs[h]).String() >= s) {
				i = h + 1 // preserves f(i-1) == false
			} else {
				j = h // preserves f(j) == true
			}
		}
		// i == j, f(i-1) == false, and f(j) (= f(i)) == true  =>  answer is i.

		// Having found the first, linear scan forward to find the last.
		// We could do a second binary search, but the caller is going
		// to do a linear scan anyway.
		for j := i; j < len(offs); j++ {
			typ := rtypeOff(section, offs[j])
			if typ.String() != s {
				break
			}
			ret = append(ret, typ)
		}
	}
	return ret
}

// lookupCache 缓存ArrayOf，ChanOf，MapOf和SliceOf查找。
var lookupCache sync.Map // map[cacheKey]*rtype

// cacheKey是在lookupCache中使用的键。
// 四个值描述了我们正在寻找的任何类型：
// 输入kind，一个或两个子类型以及一个额外的整数。
type cacheKey struct {
	kind  Kind
	t1    *rtype
	t2    *rtype
	extra uintptr
}

// funcLookupCache缓存FuncOf查找。
// FuncOf不共享公共lookupCache，因为cacheKey不足以明确表示函数。
var funcLookupCache struct {
	sync.Mutex // 守卫在m上存储（但不装载）。

	// m是一个映射map[uint32][]*rtype，由FuncOf中计算出的哈希值作为键。
	// m的元素是仅追加元素，因此对于并行读取是安全的。
	m sync.Map
}

// ChanOf 返回具有给定方向和元素类型的通道类型。
// 例如，如果t表示int，则ChanOf(RecvDir, t)表示<-chan int。
//
// gc运行时对通道元素类型施加64kB的限制。
// 如果t的大小等于或超过此限制，ChanOf会慌张。
func ChanOf(dir ChanDir, t Type) Type { // 注：#
	typ := t.(*rtype)

	// 在缓存中查找。
	ckey := cacheKey{Chan, typ, nil, uintptr(dir)}
	if ch, ok := lookupCache.Load(ckey); ok {
		return ch.(*rtype)
	}

	// 此限制由gc编译器和运行时施加。
	if typ.size >= 1<<16 {
		panic("reflect.ChanOf: element size too large") // 恐慌："元素大小太大"
	}

	// 查找已知类型。
	// TODO：构造字符串时的优先级。
	var s string
	switch dir {
	default:
		panic("reflect.ChanOf: invalid dir") // 恐慌："无效的管道方向"
	case SendDir:
		s = "chan<- " + typ.String()
	case RecvDir:
		s = "<-chan " + typ.String()
	case BothDir:
		s = "chan " + typ.String()
	}
	for _, tt := range typesByString(s) { // 注：#
		ch := (*chanType)(unsafe.Pointer(tt))
		if ch.elem == typ && ch.dir == uintptr(dir) {
			ti, _ := lookupCache.LoadOrStore(ckey, tt)
			return ti.(Type)
		}
	}

	// 设置管道类型
	var ichan interface{} = (chan unsafe.Pointer)(nil)
	prototype := *(**chanType)(unsafe.Pointer(&ichan))
	ch := *prototype
	ch.tflag = tflagRegularMemory
	ch.dir = uintptr(dir)
	ch.str = resolveReflectName(newName(s, "", false))
	ch.hash = fnv1(typ.hash, 'c', byte(dir))
	ch.elem = typ

	ti, _ := lookupCache.LoadOrStore(ckey, &ch.rtype)
	return ti.(Type)
}

// MapOf 返回具有给定键和元素类型的地图类型。
// 例如，如果k表示int而e表示字符串，则MapOf（k，e）表示map [int] string。
//
// 如果键类型不是有效的地图键类型（即，如果它不实现Go的==运算符），则MapOf会发生恐慌。
func MapOf(key, elem Type) Type { // 注：#
	ktyp := key.(*rtype)
	etyp := elem.(*rtype)

	if ktyp.equal == nil {
		panic("reflect.MapOf: invalid key type " + ktyp.String()) // 恐慌："无效的key类型"
	}

	// 在缓存中查找。
	ckey := cacheKey{Map, ktyp, etyp, 0}
	if mt, ok := lookupCache.Load(ckey); ok {
		return mt.(Type)
	}

	// 查找已知类型。
	s := "map[" + ktyp.String() + "]" + etyp.String()
	for _, tt := range typesByString(s) { // 注：#
		mt := (*mapType)(unsafe.Pointer(tt))
		if mt.key == ktyp && mt.elem == etyp {
			ti, _ := lookupCache.LoadOrStore(ckey, tt)
			return ti.(Type)
		}
	}

	// Make a map type.
	// Note: flag values must match those used in the TMAP case
	// in ../cmd/compile/internal/gc/reflect.go:dtypesym.
	var imap interface{} = (map[unsafe.Pointer]unsafe.Pointer)(nil)
	mt := **(**mapType)(unsafe.Pointer(&imap))
	mt.str = resolveReflectName(newName(s, "", false))
	mt.tflag = 0
	mt.hash = fnv1(etyp.hash, 'm', byte(ktyp.hash>>24), byte(ktyp.hash>>16), byte(ktyp.hash>>8), byte(ktyp.hash))
	mt.key = ktyp
	mt.elem = etyp
	mt.bucket = bucketOf(ktyp, etyp)
	mt.hasher = func(p unsafe.Pointer, seed uintptr) uintptr {
		return typehash(ktyp, p, seed)
	}
	mt.flags = 0
	if ktyp.size > maxKeySize {
		mt.keysize = uint8(ptrSize)
		mt.flags |= 1 // indirect key
	} else {
		mt.keysize = uint8(ktyp.size)
	}
	if etyp.size > maxValSize {
		mt.valuesize = uint8(ptrSize)
		mt.flags |= 2 // indirect value
	} else {
		mt.valuesize = uint8(etyp.size)
	}
	mt.bucketsize = uint16(mt.bucket.size)
	if isReflexive(ktyp) {
		mt.flags |= 4
	}
	if needKeyUpdate(ktyp) {
		mt.flags |= 8
	}
	if hashMightPanic(ktyp) {
		mt.flags |= 16
	}
	mt.ptrToThis = 0

	ti, _ := lookupCache.LoadOrStore(ckey, &mt.rtype)
	return ti.(Type)
}

// TODO（crawshaw）：由于这些funcTypeFixedN结构没有方法，因此可以在运行时使用StructOf函数定义它们。
type funcTypeFixed4 struct { // 注：拥有4个参数的方法类型
	funcType
	args [4]*rtype
}
type funcTypeFixed8 struct { // 注：拥有8个参数的方法类型
	funcType
	args [8]*rtype
}
type funcTypeFixed16 struct { // 注：拥有16个参数的方法类型
	funcType
	args [16]*rtype
}
type funcTypeFixed32 struct { // 注：拥有32个参数的方法类型
	funcType
	args [32]*rtype
}
type funcTypeFixed64 struct { // 注：拥有64个参数的方法类型
	funcType
	args [64]*rtype
}
type funcTypeFixed128 struct { // 注：拥有128个参数的方法类型
	funcType
	args [128]*rtype
}

// FuncOf 返回具有给定参数和结果类型的函数类型。
// 例如，如果k表示int，e表示字符串，则FuncOf([]Type{k}, []Type{e}, false)表示func(int)字符串。
//
// variadic参数控制函数是否为variadic。 如果in [len（in）-1]不代表切片并且可变参数为true，则FuncOf会发生恐慌。
func FuncOf(in, out []Type, variadic bool) Type { // 注：#
	if variadic && (len(in) == 0 || in[len(in)-1].Kind() != Slice) { // 注：如果形参可变参数不是切片类型，引发恐慌
		panic("reflect.FuncOf: last arg of variadic func must be slice") // 恐慌："可变参数函数的最后一个arg必须是切片"
	}

	// 创建一个方法类型
	var ifunc interface{} = (func())(nil)
	prototype := *(**funcType)(unsafe.Pointer(&ifunc))
	n := len(in) + len(out)

	var ft *funcType
	var args []*rtype
	switch { // 注：根据参数数量，创建不同类型的方法
	case n <= 4:
		fixed := new(funcTypeFixed4)
		args = fixed.args[:0:len(fixed.args)]
		ft = &fixed.funcType
	case n <= 8:
		fixed := new(funcTypeFixed8)
		args = fixed.args[:0:len(fixed.args)]
		ft = &fixed.funcType
	case n <= 16:
		fixed := new(funcTypeFixed16)
		args = fixed.args[:0:len(fixed.args)]
		ft = &fixed.funcType
	case n <= 32:
		fixed := new(funcTypeFixed32)
		args = fixed.args[:0:len(fixed.args)]
		ft = &fixed.funcType
	case n <= 64:
		fixed := new(funcTypeFixed64)
		args = fixed.args[:0:len(fixed.args)]
		ft = &fixed.funcType
	case n <= 128:
		fixed := new(funcTypeFixed128)
		args = fixed.args[:0:len(fixed.args)]
		ft = &fixed.funcType
	default:
		panic("reflect.FuncOf: too many arguments") // 恐慌："参数太多"
	}
	*ft = *prototype

	// 建立一个散列并最少填充ft。
	var hash uint32
	for _, in := range in { // 注：遍历输入参数
		t := in.(*rtype)
		args = append(args, t)                                                               // 注：填充参数的类型
		hash = fnv1(hash, byte(t.hash>>24), byte(t.hash>>16), byte(t.hash>>8), byte(t.hash)) // 注：计算哈希
	}
	if variadic { // 注：如果有可变参数
		hash = fnv1(hash, 'v') // 注：计算哈希
	}
	hash = fnv1(hash, '.')    // 注：计算哈希
	for _, out := range out { // 注：遍历输出参数
		t := out.(*rtype)
		args = append(args, t)                                                               // 注：填充参数的类型
		hash = fnv1(hash, byte(t.hash>>24), byte(t.hash>>16), byte(t.hash>>8), byte(t.hash)) // 注：计算哈希
	}
	if len(args) > 50 {
		panic("reflect.FuncOf does not support more than 50 arguments") // 恐慌："不支持超过50个参数"
	}
	ft.tflag = 0
	ft.hash = hash
	ft.inCount = uint16(len(in))
	ft.outCount = uint16(len(out))
	if variadic { // 注：如果最后一个输入参数是可变参数，设置输出参数的最高位为1
		ft.outCount |= 1 << 15
	}

	// 在缓存中查找。
	if ts, ok := funcLookupCache.m.Load(hash); ok {
		for _, t := range ts.([]*rtype) {
			if haveIdenticalUnderlyingType(&ft.rtype, t, true) { // 注：是否具有相同的基础类型
				return t
			}
		}
	}

	// 不在缓存中，请上锁并重试。
	funcLookupCache.Lock()
	defer funcLookupCache.Unlock()
	if ts, ok := funcLookupCache.m.Load(hash); ok {
		for _, t := range ts.([]*rtype) {
			if haveIdenticalUnderlyingType(&ft.rtype, t, true) { // 注：是否具有相同的基础类型
				return t
			}
		}
	}

	addToCache := func(tt *rtype) Type { // 注：将缓存中key为hash的value追加tt
		var rts []*rtype
		if rti, ok := funcLookupCache.m.Load(hash); ok {
			rts = rti.([]*rtype)
		}
		funcLookupCache.m.Store(hash, append(rts, tt))
		return tt
	}

	// 在已知类型中查找相同的字符串表示形式。
	str := funcStr(ft) // 注：获取方法的字符串表示形式
	for _, tt := range typesByString(str) {
		if haveIdenticalUnderlyingType(&ft.rtype, tt, true) { // 注：#
			return addToCache(tt)
		}
	}

	// 填充ft的其余字段并存储在缓存中。
	ft.str = resolveReflectName(newName(str, "", false)) // 注：#
	ft.ptrToThis = 0
	return addToCache(&ft.rtype)
}

// funcStr 构建funcType的字符串表示形式。
func funcStr(ft *funcType) string { // 注：获取方法的字符串表示形式，例func(a int, b ...int) (c int, d int)
	repr := make([]byte, 0, 64)
	repr = append(repr, "func("...) // 注：func(
	for i, t := range ft.in() {     // 注：遍历ft的输入参数
		if i > 0 {
			repr = append(repr, ", "...) // 注：每个参数之间加", "
		}
		if ft.IsVariadic() && i == int(ft.inCount)-1 { // 注：最后一个参数是可变参数
			repr = append(repr, "..."...)                                         // 注：...
			repr = append(repr, (*sliceType)(unsafe.Pointer(t)).elem.String()...) // 注：切片成员类型
		} else {
			repr = append(repr, t.String()...) // 注：参数类型
		}
	}
	repr = append(repr, ')') // 注：)
	out := ft.out()
	if len(out) == 1 { // 注：如果只有一个输出参数，" "
		repr = append(repr, ' ')
	} else if len(out) > 1 { // 注：如果有多个输出参数，" ("
		repr = append(repr, " ("...)
	}
	for i, t := range out { // 注：遍历输出参数
		if i > 0 {
			repr = append(repr, ", "...) // 注：每个参数之间加", "
		}
		repr = append(repr, t.String()...) // 注：参数类型
	}
	if len(out) > 1 {
		repr = append(repr, ')') // 注：)
	}
	return string(repr)
}

// isReflexive 报告对类型的==操作是否自反。
// 也就是说，所有t类型的值x，x == x
func isReflexive(t *rtype) bool { // 注：#
	switch t.Kind() {
	case Bool, Int, Int8, Int16, Int32, Int64, Uint, Uint8, Uint16, Uint32, Uint64, Uintptr, Chan, Ptr, String, UnsafePointer:
		return true
	case Float32, Float64, Complex64, Complex128, Interface:
		return false
	case Array: // 注：检查成员类型
		tt := (*arrayType)(unsafe.Pointer(t))
		return isReflexive(tt.elem)
	case Struct: // 注：检查每个成员的类型
		tt := (*structType)(unsafe.Pointer(t))
		for _, f := range tt.fields {
			if !isReflexive(f.typ) {
				return false
			}
		}
		return true
	default:
		// Func, Map, Slice, Invalid
		panic("isReflexive called on non-key type " + t.String()) // 恐慌："调用非key类型"
	}
}

// needKeyUpdate 报告是否覆盖映射要求复制密钥。
func needKeyUpdate(t *rtype) bool { // 注：#
	switch t.Kind() {
	case Bool, Int, Int8, Int16, Int32, Int64, Uint, Uint8, Uint16, Uint32, Uint64, Uintptr, Chan, Ptr, UnsafePointer:
		return false
	case Float32, Float64, Complex64, Complex128, Interface, String:
		// 浮动键可以从+0更新为-0。
		// 可以更新字符串键以使用较小的后备存储。
		// 接口中可能包含字符串的浮点数。
		return true
	case Array: // 注：检查成员类型
		tt := (*arrayType)(unsafe.Pointer(t))
		return needKeyUpdate(tt.elem)
	case Struct: // 注：检查每个成员的类型
		tt := (*structType)(unsafe.Pointer(t))
		for _, f := range tt.fields {
			if needKeyUpdate(f.typ) {
				return true
			}
		}
		return false
	default:
		// Func, Map, Slice, Invalid
		panic("needKeyUpdate called on non-key type " + t.String()) // 恐慌："调用非key类型"
	}
}

// hashMightPanic 报告类型为t的集合键的哈希是否可能出现紧急情况。
func hashMightPanic(t *rtype) bool { // 注：#
	switch t.Kind() {
	case Interface:
		return true
	case Array: // 注：检查成员类型
		tt := (*arrayType)(unsafe.Pointer(t))
		return hashMightPanic(tt.elem)
	case Struct: // 注：检查每个成员的类型
		tt := (*structType)(unsafe.Pointer(t))
		for _, f := range tt.fields {
			if hashMightPanic(f.typ) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// 确保这些例程与../../runtime/map.go保持同步！
// 这些类型仅适用于GC，因此我们仅填写与GC相关的信息。
// 当前，这只是大小和GC程序。 我们还填写字符串以供可能的调试使用。
const (
	bucketSize uintptr = 8
	maxKeySize uintptr = 128
	maxValSize uintptr = 128
)

func bucketOf(ktyp, etyp *rtype) *rtype {
	if ktyp.size > maxKeySize {
		ktyp = PtrTo(ktyp).(*rtype)
	}
	if etyp.size > maxValSize {
		etyp = PtrTo(etyp).(*rtype)
	}

	// 准备GC数据（如果有）。
	// 一个存储桶最多为bucketSize*(1+maxKeySize+maxValSize)+2*ptrSize字节，即2072字节，或259个指针大小的字，或33字节的指针位图。
	// 请注意，由于已知键和值 <= 128字节，因此可以确保它们具有位图而不是GC程序。
	var gcdata *byte
	var ptrdata uintptr
	var overflowPad uintptr

	size := bucketSize*(1+ktyp.size+etyp.size) + overflowPad + ptrSize      // 注：#
	if size&uintptr(ktyp.align-1) != 0 || size&uintptr(etyp.align-1) != 0 { // 注：ktyp或etyp的大小超出了最大大小
		panic("reflect: bad size computation in MapOf") // 恐慌："MapOf中错误的尺寸计算"
	}

	if ktyp.ptrdata != 0 || etyp.ptrdata != 0 {
		nptr := (bucketSize*(1+ktyp.size+etyp.size) + ptrSize) / ptrSize
		mask := make([]byte, (nptr+7)/8)
		base := bucketSize / ptrSize

		if ktyp.ptrdata != 0 {
			emitGCMask(mask, base, ktyp, bucketSize)
		}
		base += bucketSize * ktyp.size / ptrSize

		if etyp.ptrdata != 0 {
			emitGCMask(mask, base, etyp, bucketSize)
		}
		base += bucketSize * etyp.size / ptrSize
		base += overflowPad / ptrSize

		word := base
		mask[word/8] |= 1 << (word % 8)
		gcdata = &mask[0]
		ptrdata = (word + 1) * ptrSize

		// overflow word must be last
		if ptrdata != size {
			panic("reflect: bad layout computation in MapOf")
		}
	}

	b := &rtype{
		align:   ptrSize,
		size:    size,
		kind:    uint8(Struct),
		ptrdata: ptrdata,
		gcdata:  gcdata,
	}
	if overflowPad > 0 {
		b.align = 8
	}
	s := "bucket(" + ktyp.String() + "," + etyp.String() + ")"
	b.str = resolveReflectName(newName(s, "", false))
	return b
}

func (t *rtype) gcSlice(begin, end uintptr) []byte { // 注：#
	return (*[1 << 30]byte)(unsafe.Pointer(t.gcdata))[begin:end:end]
}

// emitgGCMask 将[n]typ的GC掩码从位偏移量的基数开始写入。
func emitGCMask(out []byte, base uintptr, typ *rtype, n uintptr) { // 注：#
	if typ.kind&kindGCProg != 0 {
		panic("reflect: unexpected GC program") // 恐慌："意外的GC程序"
	}
	ptrs := typ.ptrdata / ptrSize
	words := typ.size / ptrSize
	mask := typ.gcSlice(0, (ptrs+7)/8)
	for j := uintptr(0); j < ptrs; j++ {
		if (mask[j/8]>>(j%8))&1 != 0 {
			for i := uintptr(0); i < n; i++ {
				k := base + i*words + j
				out[k/8] |= 1 << (k % 8)
			}
		}
	}
}

// appendGCProg 将typ的第一个ptrdata字节的GC程序追加到dst，并返回扩展的切片。
func appendGCProg(dst []byte, typ *rtype) []byte { // 注：#
	if typ.kind&kindGCProg != 0 {
		// Element has GC program; emit one element.
		n := uintptr(*(*uint32)(unsafe.Pointer(typ.gcdata)))
		prog := typ.gcSlice(4, 4+n-1)
		return append(dst, prog...)
	}

	// Element is small with pointer mask; use as literal bits.
	ptrs := typ.ptrdata / ptrSize
	mask := typ.gcSlice(0, (ptrs+7)/8)

	// Emit 120-bit chunks of full bytes (max is 127 but we avoid using partial bytes).
	for ; ptrs > 120; ptrs -= 120 {
		dst = append(dst, 120)
		dst = append(dst, mask[:15]...)
		mask = mask[15:]
	}

	dst = append(dst, byte(ptrs))
	dst = append(dst, mask...)
	return dst
}

// SliceOf 返回元素类型为t的切片类型。
// 例如，如果t表示int，则SliceOf(t)表示[]int。
func SliceOf(t Type) Type { // 注：#
	typ := t.(*rtype)

	// 在缓存中查找。
	ckey := cacheKey{Slice, typ, nil, 0}
	if slice, ok := lookupCache.Load(ckey); ok {
		return slice.(Type)
	}

	// 查找已知类型。
	s := "[]" + typ.String()
	for _, tt := range typesByString(s) { // 注：#
		slice := (*sliceType)(unsafe.Pointer(tt))
		if slice.elem == typ {
			ti, _ := lookupCache.LoadOrStore(ckey, tt)
			return ti.(Type)
		}
	}

	// Make a slice type.
	var islice interface{} = ([]unsafe.Pointer)(nil)
	prototype := *(**sliceType)(unsafe.Pointer(&islice))
	slice := *prototype
	slice.tflag = 0
	slice.str = resolveReflectName(newName(s, "", false))
	slice.hash = fnv1(typ.hash, '[')
	slice.elem = typ
	slice.ptrToThis = 0

	ti, _ := lookupCache.LoadOrStore(ckey, &slice.rtype)
	return ti.(Type)
}

// structLookupCache 缓存StructOf
// StructOf不共享公共lookupCache，因为我们需要固定与*structTypeFixedN关联的内存。
var structLookupCache struct {
	sync.Mutex // 守卫在m上存储（但不装载）。

	// m是map[uint32][]Type，该类型由StructOf中计算出的哈希值键控。
	// m中的元素仅是追加元素，因此可以安全地进行并发读取。
	m sync.Map
}

type structTypeUncommon struct { // 注：结构体类型，内存后紧随uncommonType
	structType              // 注：结构体类型
	u          uncommonType // 注：方法
}

// isLetter 报告给定的"rune"是否归类为字母。
func isLetter(ch rune) bool { // 注：获取ch是否为字母或下划线
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

// isValidFieldName 检查字符串是否为有效的（结构体）字段名称。
// 根据语言规范，字段名称应为标识符。
// identifier = letter { letter | unicode_digit } .
// letter = unicode_letter | "_" .
func isValidFieldName(fieldName string) bool { // 注：fieldName是否是有效的字段名称（首字母必须为字母或下划线）
	for i, c := range fieldName { // 注：遍历fieldName
		if i == 0 && !isLetter(c) { // 注：第1个元素不是字母，返回false
			return false
		}

		if !(isLetter(c) || unicode.IsDigit(c)) { // 注：不是字母和数字，返回false
			return false
		}
	}

	return len(fieldName) > 0
}

// StructOf 返回包含字段的结构类型。
// 偏移量和索引字段将被忽略并按照编译器的方式进行计算。
//
// 如果传递未导出的StructFields，StructOf当前不会为嵌入式字段和紧急情况生成包装方法。
// 这些限制可能会在将来的版本中取消。
func StructOf(fields []StructField) Type { // 注：#
	var (
		hash       = fnv1(0, []byte("struct {")...) // 注：哈希
		size       uintptr
		typalign   uint8
		comparable = true
		methods    []method

		fs   = make([]structField, len(fields))
		repr = make([]byte, 0, 64)
		fset = map[string]struct{}{} // fields的名称

		hasGCProg = false // 记录结构字段类型是否具有GCProg
	)

	lastzero := uintptr(0)
	repr = append(repr, "struct {"...) // 注：计算哈希"struct {"
	pkgpath := ""
	for i, field := range fields { // 注：遍历字段
		if field.Name == "" {
			panic("reflect.StructOf: field " + strconv.Itoa(i) + " has no name") // 恐慌："字段没有名称"
		}
		if !isValidFieldName(field.Name) { // 注：检查是否是有效的字段名称
			panic("reflect.StructOf: field " + strconv.Itoa(i) + " has invalid name") // 恐慌："无效的字段名称"
		}
		if field.Type == nil {
			panic("reflect.StructOf: field " + strconv.Itoa(i) + " has no type") // 恐慌："字段没有类型"
		}
		f, fpkgpath := runtimeStructField(field) // 注：field转为内部形式
		ft := f.typ
		if ft.kind&kindGCProg != 0 { // 注：如果有gc
			hasGCProg = true
		}
		if fpkgpath != "" {
			if pkgpath == "" { // 注：赋值程序包路径
				pkgpath = fpkgpath
			} else if pkgpath != fpkgpath {
				panic("reflect.Struct: fields with different PkgPath " + pkgpath + " and " + fpkgpath) // 恐慌："具有不同PkgPath的字段"
			}
		}

		// 更新字符串和哈希
		name := f.name.name()
		hash = fnv1(hash, []byte(name)...)
		repr = append(repr, (" " + name)...) // 注：字段名
		if f.embedded() {                    // 注：是否是嵌入式字段
			// 嵌入式字段
			if f.typ.Kind() == Ptr { // 注：指针类型
				// 嵌入**和*interface{}是非法的
				elem := ft.Elem()
				if k := elem.Kind(); k == Ptr || k == Interface { // 注：如果嵌入式字段的成员类型是**或*interface{}，引发异常
					panic("reflect.StructOf: illegal embedded field type " + ft.String()) // 恐慌："非法的嵌入式字段类型"
				}
			}

			switch f.typ.Kind() {
			case Interface: // 注：如果字段是接口类型
				ift := (*interfaceType)(unsafe.Pointer(ft))
				for im, m := range ift.methods { // 注：遍历字段的方法
					if ift.nameOff(m.name).pkgPath() != "" { // 注：#
						// TODO(sbinet).  Issue 15924.
						panic("reflect: embedded interface with unexported method(s) not implemented") // 恐慌："未实现未导出方法的嵌入式接口"
					}

					var (
						mtyp    = ift.typeOff(m.typ)
						ifield  = i
						imethod = im
						ifn     Value
						tfn     Value
					)

					if ft.kind&kindDirectIface != 0 {
						tfn = MakeFunc(mtyp, func(in []Value) []Value {
							var args []Value
							var recv = in[0]
							if len(in) > 1 {
								args = in[1:]
							}
							return recv.Field(ifield).Method(imethod).Call(args)
						})
						ifn = MakeFunc(mtyp, func(in []Value) []Value {
							var args []Value
							var recv = in[0]
							if len(in) > 1 {
								args = in[1:]
							}
							return recv.Field(ifield).Method(imethod).Call(args)
						})
					} else {
						tfn = MakeFunc(mtyp, func(in []Value) []Value {
							var args []Value
							var recv = in[0]
							if len(in) > 1 {
								args = in[1:]
							}
							return recv.Field(ifield).Method(imethod).Call(args)
						})
						ifn = MakeFunc(mtyp, func(in []Value) []Value {
							var args []Value
							var recv = Indirect(in[0])
							if len(in) > 1 {
								args = in[1:]
							}
							return recv.Field(ifield).Method(imethod).Call(args)
						})
					}

					methods = append(methods, method{
						name: resolveReflectName(ift.nameOff(m.name)),
						mtyp: resolveReflectType(mtyp),
						ifn:  resolveReflectText(unsafe.Pointer(&ifn)),
						tfn:  resolveReflectText(unsafe.Pointer(&tfn)),
					})
				}
			case Ptr:
				ptr := (*ptrType)(unsafe.Pointer(ft))
				if unt := ptr.uncommon(); unt != nil {
					if i > 0 && unt.mcount > 0 {
						// Issue 15924.
						panic("reflect: embedded type with methods not implemented if type is not first field")
					}
					if len(fields) > 1 {
						panic("reflect: embedded type with methods not implemented if there is more than one field")
					}
					for _, m := range unt.methods() {
						mname := ptr.nameOff(m.name)
						if mname.pkgPath() != "" {
							// TODO(sbinet).
							// Issue 15924.
							panic("reflect: embedded interface with unexported method(s) not implemented")
						}
						methods = append(methods, method{
							name: resolveReflectName(mname),
							mtyp: resolveReflectType(ptr.typeOff(m.mtyp)),
							ifn:  resolveReflectText(ptr.textOff(m.ifn)),
							tfn:  resolveReflectText(ptr.textOff(m.tfn)),
						})
					}
				}
				if unt := ptr.elem.uncommon(); unt != nil {
					for _, m := range unt.methods() {
						mname := ptr.nameOff(m.name)
						if mname.pkgPath() != "" {
							// TODO(sbinet)
							// Issue 15924.
							panic("reflect: embedded interface with unexported method(s) not implemented")
						}
						methods = append(methods, method{
							name: resolveReflectName(mname),
							mtyp: resolveReflectType(ptr.elem.typeOff(m.mtyp)),
							ifn:  resolveReflectText(ptr.elem.textOff(m.ifn)),
							tfn:  resolveReflectText(ptr.elem.textOff(m.tfn)),
						})
					}
				}
			default:
				if unt := ft.uncommon(); unt != nil {
					if i > 0 && unt.mcount > 0 {
						// Issue 15924.
						panic("reflect: embedded type with methods not implemented if type is not first field")
					}
					if len(fields) > 1 && ft.kind&kindDirectIface != 0 {
						panic("reflect: embedded type with methods not implemented for non-pointer type")
					}
					for _, m := range unt.methods() {
						mname := ft.nameOff(m.name)
						if mname.pkgPath() != "" {
							// TODO(sbinet)
							// Issue 15924.
							panic("reflect: embedded interface with unexported method(s) not implemented")
						}
						methods = append(methods, method{
							name: resolveReflectName(mname),
							mtyp: resolveReflectType(ft.typeOff(m.mtyp)),
							ifn:  resolveReflectText(ft.textOff(m.ifn)),
							tfn:  resolveReflectText(ft.textOff(m.tfn)),
						})

					}
				}
			}
		}
		if _, dup := fset[name]; dup {
			panic("reflect.StructOf: duplicate field " + name)
		}
		fset[name] = struct{}{}

		hash = fnv1(hash, byte(ft.hash>>24), byte(ft.hash>>16), byte(ft.hash>>8), byte(ft.hash))

		repr = append(repr, (" " + ft.String())...)
		if f.name.tagLen() > 0 {
			hash = fnv1(hash, []byte(f.name.tag())...)
			repr = append(repr, (" " + strconv.Quote(f.name.tag()))...)
		}
		if i < len(fields)-1 {
			repr = append(repr, ';')
		}

		comparable = comparable && (ft.equal != nil)

		offset := align(size, uintptr(ft.align))
		if ft.align > typalign {
			typalign = ft.align
		}
		size = offset + ft.size
		f.offsetEmbed |= offset << 1

		if ft.size == 0 {
			lastzero = size
		}

		fs[i] = f
	}

	if size > 0 && lastzero == size {
		// This is a non-zero sized struct that ends in a
		// zero-sized field. We add an extra byte of padding,
		// to ensure that taking the address of the final
		// zero-sized field can't manufacture a pointer to the
		// next object in the heap. See issue 9401.
		size++
	}

	var typ *structType
	var ut *uncommonType

	if len(methods) == 0 {
		t := new(structTypeUncommon)
		typ = &t.structType
		ut = &t.u
	} else {
		// A *rtype representing a struct is followed directly in memory by an
		// array of method objects representing the methods attached to the
		// struct. To get the same layout for a run time generated type, we
		// need an array directly following the uncommonType memory.
		// A similar strategy is used for funcTypeFixed4, ...funcTypeFixedN.
		tt := New(StructOf([]StructField{
			{Name: "S", Type: TypeOf(structType{})},
			{Name: "U", Type: TypeOf(uncommonType{})},
			{Name: "M", Type: ArrayOf(len(methods), TypeOf(methods[0]))},
		}))

		typ = (*structType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ut = (*uncommonType)(unsafe.Pointer(tt.Elem().Field(1).UnsafeAddr()))

		copy(tt.Elem().Field(2).Slice(0, len(methods)).Interface().([]method), methods)
	}
	// TODO(sbinet): Once we allow embedding multiple types,
	// methods will need to be sorted like the compiler does.
	// TODO(sbinet): Once we allow non-exported methods, we will
	// need to compute xcount as the number of exported methods.
	ut.mcount = uint16(len(methods))
	ut.xcount = ut.mcount
	ut.moff = uint32(unsafe.Sizeof(uncommonType{}))

	if len(fs) > 0 {
		repr = append(repr, ' ')
	}
	repr = append(repr, '}')
	hash = fnv1(hash, '}')
	str := string(repr)

	// Round the size up to be a multiple of the alignment.
	size = align(size, uintptr(typalign))

	// Make the struct type.
	var istruct interface{} = struct{}{}
	prototype := *(**structType)(unsafe.Pointer(&istruct))
	*typ = *prototype
	typ.fields = fs
	if pkgpath != "" {
		typ.pkgPath = newName(pkgpath, "", false)
	}

	// Look in cache.
	if ts, ok := structLookupCache.m.Load(hash); ok {
		for _, st := range ts.([]Type) {
			t := st.common()
			if haveIdenticalUnderlyingType(&typ.rtype, t, true) {
				return t
			}
		}
	}

	// Not in cache, lock and retry.
	structLookupCache.Lock()
	defer structLookupCache.Unlock()
	if ts, ok := structLookupCache.m.Load(hash); ok {
		for _, st := range ts.([]Type) {
			t := st.common()
			if haveIdenticalUnderlyingType(&typ.rtype, t, true) {
				return t
			}
		}
	}

	addToCache := func(t Type) Type {
		var ts []Type
		if ti, ok := structLookupCache.m.Load(hash); ok {
			ts = ti.([]Type)
		}
		structLookupCache.m.Store(hash, append(ts, t))
		return t
	}

	// Look in known types.
	for _, t := range typesByString(str) {
		if haveIdenticalUnderlyingType(&typ.rtype, t, true) {
			// even if 't' wasn't a structType with methods, we should be ok
			// as the 'u uncommonType' field won't be accessed except when
			// tflag&tflagUncommon is set.
			return addToCache(t)
		}
	}

	typ.str = resolveReflectName(newName(str, "", false))
	typ.tflag = 0 // TODO: set tflagRegularMemory
	typ.hash = hash
	typ.size = size
	typ.ptrdata = typeptrdata(typ.common())
	typ.align = typalign
	typ.fieldAlign = typalign
	typ.ptrToThis = 0
	if len(methods) > 0 {
		typ.tflag |= tflagUncommon
	}

	if hasGCProg {
		lastPtrField := 0
		for i, ft := range fs {
			if ft.typ.pointers() {
				lastPtrField = i
			}
		}
		prog := []byte{0, 0, 0, 0} // will be length of prog
		var off uintptr
		for i, ft := range fs {
			if i > lastPtrField {
				// gcprog should not include anything for any field after
				// the last field that contains pointer data
				break
			}
			if !ft.typ.pointers() {
				// Ignore pointerless fields.
				continue
			}
			// Pad to start of this field with zeros.
			if ft.offset() > off {
				n := (ft.offset() - off) / ptrSize
				prog = append(prog, 0x01, 0x00) // emit a 0 bit
				if n > 1 {
					prog = append(prog, 0x81)      // repeat previous bit
					prog = appendVarint(prog, n-1) // n-1 times
				}
				off = ft.offset()
			}

			prog = appendGCProg(prog, ft.typ)
			off += ft.typ.ptrdata
		}
		prog = append(prog, 0)
		*(*uint32)(unsafe.Pointer(&prog[0])) = uint32(len(prog) - 4)
		typ.kind |= kindGCProg
		typ.gcdata = &prog[0]
	} else {
		typ.kind &^= kindGCProg
		bv := new(bitVector)
		addTypeBits(bv, 0, typ.common())
		if len(bv.data) > 0 {
			typ.gcdata = &bv.data[0]
		}
	}
	typ.equal = nil
	if comparable {
		typ.equal = func(p, q unsafe.Pointer) bool {
			for _, ft := range typ.fields {
				pi := add(p, ft.offset(), "&x.field safe")
				qi := add(q, ft.offset(), "&x.field safe")
				if !ft.typ.equal(pi, qi) {
					return false
				}
			}
			return true
		}
	}

	switch {
	case len(fs) == 1 && !ifaceIndir(fs[0].typ):
		// structs of 1 direct iface type can be direct
		typ.kind |= kindDirectIface
	default:
		typ.kind &^= kindDirectIface
	}

	return addToCache(&typ.rtype)
}

// runtimeStructField 接受传递给StructOf的StructField值，并返回相应的内部表示形式structField和用于该字段的pkgpath值。
func runtimeStructField(field StructField) (structField, string) { // 注：获取StructField field的structField形式（检查PkgPath）与PkgPath
	if field.Anonymous && field.PkgPath != "" { // 注：如果字段是嵌入式结构体，但字段没有程序包路径，引发恐慌
		panic("reflect.StructOf: field \"" + field.Name + "\" is anonymous but has PkgPath set") // 恐慌："字段是匿名的，但已设置PkgPath"
	}

	exported := field.PkgPath == "" // 注：如果程序包路径为空，暂时判断是已导出字段
	if exported {
		// 尽最大努力检查滥用情况。
		// 由于此字段将被视为已导出，因此如果Unicode小写漏掉，不会造成太大危害。
		c := field.Name[0]
		if 'a' <= c && c <= 'z' || c == '_' { // 注：如果是已导出字段，但第一个字母不是大写的，引发恐慌
			panic("reflect.StructOf: field \"" + field.Name + "\" is unexported but missing PkgPath") // 恐慌："字段是未导出的，但缺少PkgPath"
		}
	}

	offsetEmbed := uintptr(0)
	if field.Anonymous {
		offsetEmbed |= 1
	}

	resolveReflectType(field.Type.common()) // 在运行时安装，注：#
	f := structField{
		name:        newName(field.Name, string(field.Tag), exported),
		typ:         field.Type.common(),
		offsetEmbed: offsetEmbed,
	}
	return f, field.PkgPath
}

// typeptrdata 返回包含指针数据的t前缀的字节长度。 此偏移量之后的所有内容均为标量数据。
// 与../cmd/compile/internal/gc/reflect.go保持同步
func typeptrdata(t *rtype) uintptr { // 注：#
	switch t.Kind() {
	case Struct:
		st := (*structType)(unsafe.Pointer(t))
		// 查找具有指针的最后一个字段。
		field := -1
		for i := range st.fields {
			ft := st.fields[i].typ
			if ft.pointers() {
				field = i
			}
		}
		if field == -1 {
			return 0
		}
		f := st.fields[field]             // 注：结构体t的最后一个指针
		return f.offset() + f.typ.ptrdata // 注：#

	default:
		panic("reflect.typeptrdata: unexpected type, " + t.String()) // 恐慌："意外的类型"
	}
}

// 有关常量的派生，请参见cmd/compile/internal/gc/reflect.go
const maxPtrmaskBytes = 2048

// ArrayOf 返回具有给定计数和元素类型的数组类型。
// 例如，如果t表示int，则ArrayOf(5, t)表示[5]int。
// 如果结果类型将大于可用的地址空间，则ArrayOf会引发恐慌。
func ArrayOf(count int, elem Type) Type { // 注：#
	typ := elem.(*rtype)

	// 在缓存中查找。
	ckey := cacheKey{Array, typ, nil, uintptr(count)}
	if array, ok := lookupCache.Load(ckey); ok {
		return array.(Type)
	}

	// 查找已知类型。
	s := "[" + strconv.Itoa(count) + "]" + typ.String()
	for _, tt := range typesByString(s) { // 注：#
		array := (*arrayType)(unsafe.Pointer(tt))
		if array.elem == typ {
			ti, _ := lookupCache.LoadOrStore(ckey, tt)
			return ti.(Type)
		}
	}

	// Make an array type.
	var iarray interface{} = [1]unsafe.Pointer{}
	prototype := *(**arrayType)(unsafe.Pointer(&iarray))
	array := *prototype
	array.tflag = typ.tflag & tflagRegularMemory
	array.str = resolveReflectName(newName(s, "", false))
	array.hash = fnv1(typ.hash, '[')
	for n := uint32(count); n > 0; n >>= 8 {
		array.hash = fnv1(array.hash, byte(n))
	}
	array.hash = fnv1(array.hash, ']')
	array.elem = typ
	array.ptrToThis = 0
	if typ.size > 0 {
		max := ^uintptr(0) / typ.size
		if uintptr(count) > max {
			panic("reflect.ArrayOf: array size would exceed virtual address space")
		}
	}
	array.size = typ.size * uintptr(count)
	if count > 0 && typ.ptrdata != 0 {
		array.ptrdata = typ.size*uintptr(count-1) + typ.ptrdata
	}
	array.align = typ.align
	array.fieldAlign = typ.fieldAlign
	array.len = uintptr(count)
	array.slice = SliceOf(elem).(*rtype)

	switch {
	case typ.ptrdata == 0 || array.size == 0:
		// No pointers.
		array.gcdata = nil
		array.ptrdata = 0

	case count == 1:
		// In memory, 1-element array looks just like the element.
		array.kind |= typ.kind & kindGCProg
		array.gcdata = typ.gcdata
		array.ptrdata = typ.ptrdata

	case typ.kind&kindGCProg == 0 && array.size <= maxPtrmaskBytes*8*ptrSize:
		// Element is small with pointer mask; array is still small.
		// Create direct pointer mask by turning each 1 bit in elem
		// into count 1 bits in larger mask.
		mask := make([]byte, (array.ptrdata/ptrSize+7)/8)
		emitGCMask(mask, 0, typ, array.len)
		array.gcdata = &mask[0]

	default:
		// Create program that emits one element
		// and then repeats to make the array.
		prog := []byte{0, 0, 0, 0} // will be length of prog
		prog = appendGCProg(prog, typ)
		// Pad from ptrdata to size.
		elemPtrs := typ.ptrdata / ptrSize
		elemWords := typ.size / ptrSize
		if elemPtrs < elemWords {
			// Emit literal 0 bit, then repeat as needed.
			prog = append(prog, 0x01, 0x00)
			if elemPtrs+1 < elemWords {
				prog = append(prog, 0x81)
				prog = appendVarint(prog, elemWords-elemPtrs-1)
			}
		}
		// Repeat count-1 times.
		if elemWords < 0x80 {
			prog = append(prog, byte(elemWords|0x80))
		} else {
			prog = append(prog, 0x80)
			prog = appendVarint(prog, elemWords)
		}
		prog = appendVarint(prog, uintptr(count)-1)
		prog = append(prog, 0)
		*(*uint32)(unsafe.Pointer(&prog[0])) = uint32(len(prog) - 4)
		array.kind |= kindGCProg
		array.gcdata = &prog[0]
		array.ptrdata = array.size // overestimate but ok; must match program
	}

	etyp := typ.common()
	esize := etyp.Size()

	array.equal = nil
	if eequal := etyp.equal; eequal != nil {
		array.equal = func(p, q unsafe.Pointer) bool {
			for i := 0; i < count; i++ {
				pi := arrayAt(p, i, esize, "i < count")
				qi := arrayAt(q, i, esize, "i < count")
				if !eequal(pi, qi) {
					return false
				}

			}
			return true
		}
	}

	switch {
	case count == 1 && !ifaceIndir(typ):
		// 1个直接iface类型的数组可以是直接的
		array.kind |= kindDirectIface
	default:
		array.kind &^= kindDirectIface
	}

	ti, _ := lookupCache.LoadOrStore(ckey, &array.rtype)
	return ti.(Type)
}

func appendVarint(x []byte, v uintptr) []byte { // 注：#
	for ; v >= 0x80; v >>= 7 {
		x = append(x, byte(v|0x80))
	}
	x = append(x, byte(v))
	return x
}

// toType 从*rtype转换为Type，可以将其返回给package Reflection的客户端。
// 在gc中，唯一需要注意的是必须将nil *rtype替换为nil Type，
// 但是在gccgo中，此函数负责确保将同一类型的多个*rtype合并为单个Type。
func toType(t *rtype) Type { //注：将*rtype类型的t转为Type并返回
	if t == nil {
		return nil
	}
	return t
}

type layoutKey struct {
	ftyp *funcType // 方法签名
	rcvr *rtype    // 接收器类型，如果没有则为nil
}

type layoutType struct {
	t         *rtype
	argSize   uintptr // size of arguments
	retOffset uintptr // offset of return values.
	stack     *bitVector
	framePool *sync.Pool
}

var layoutCache sync.Map // map[layoutKey]layoutType

// funcLayout 计算一个结构体类型，该结构体类型表示函数参数的布局并返回函数类型t的值。
// 如果rcvr != nil，则rcvr指定接收方的类型。
// 返回的类型仅适用于GC，因此我们仅填写与GC相关的信息。
// 当前，这只是大小和GC程序。 我们还填写该名称，以供可能的调试使用。
func funcLayout(t *funcType, rcvr *rtype) (frametype *rtype, argSize, retOffset uintptr, stk *bitVector, framePool *sync.Pool) { // 注：#
	if t.Kind() != Func { // 注：如果t不是方法类型，引发恐慌
		panic("reflect: funcLayout of non-func type " + t.String()) // 恐慌："非功能类型的funcLayout"
	}
	if rcvr != nil && rcvr.Kind() == Interface { // 注：如果接收器不为空，并且接收器是接口类型，引发恐慌
		panic("reflect: funcLayout with interface receiver " + rcvr.String()) // 恐慌："接口接收器的funcLayout"
	}
	k := layoutKey{t, rcvr}
	if lti, ok := layoutCache.Load(k); ok { // 注：如果有缓存，直接返回
		lt := lti.(layoutType)
		return lt.t, lt.argSize, lt.retOffset, lt.stack, lt.framePool
	}

	// 计算gc程序和堆栈位图作为参数
	ptrmap := new(bitVector)
	var offset uintptr
	if rcvr != nil {
		// Reflect使用"interface"调用约定作为方法，接收者无论实际大小如何都占用一个参数空间。
		if ifaceIndir(rcvr) || rcvr.pointers() { // 注：接收器是间接寻址或为指针
			ptrmap.append(1) // 注：#
		} else {
			ptrmap.append(0)
		}
		offset += ptrSize
	}
	for _, arg := range t.in() { // 注：#
		offset += -offset & uintptr(arg.align-1)
		addTypeBits(ptrmap, offset, arg)
		offset += arg.size
	}
	argSize = offset
	offset += -offset & (ptrSize - 1)
	retOffset = offset
	for _, res := range t.out() {
		offset += -offset & uintptr(res.align-1)
		addTypeBits(ptrmap, offset, res)
		offset += res.size
	}
	offset += -offset & (ptrSize - 1)

	// build dummy rtype holding gc program
	x := &rtype{
		align:   ptrSize,
		size:    offset,
		ptrdata: uintptr(ptrmap.n) * ptrSize,
	}
	if ptrmap.n > 0 {
		x.gcdata = &ptrmap.data[0]
	}

	var s string
	if rcvr != nil {
		s = "methodargs(" + rcvr.String() + ")(" + t.String() + ")"
	} else {
		s = "funcargs(" + t.String() + ")"
	}
	x.str = resolveReflectName(newName(s, "", false))

	// cache result for future callers
	framePool = &sync.Pool{New: func() interface{} {
		return unsafe_New(x)
	}}
	lti, _ := layoutCache.LoadOrStore(k, layoutType{
		t:         x,
		argSize:   argSize,
		retOffset: retOffset,
		stack:     ptrmap,
		framePool: framePool,
	})
	lt := lti.(layoutType)
	return lt.t, lt.argSize, lt.retOffset, lt.stack, lt.framePool
}

// ifaceIndir 报告t是否间接存储在接口值中。
func ifaceIndir(t *rtype) bool { //注：t是否为间接存储在接口值中（t是否是指向（指向数据的指针）指针），为0时为间接指针
	return t.kind&kindDirectIface == 0
}

// 注意：此类型必须与runtime.bitvector一致。
type bitVector struct {
	n    uint32 // 位数
	data []byte
}

// append 在位图上附加bit
func (bv *bitVector) append(bit uint8) { // 注：#
	if bv.n%8 == 0 { // 注：如果bv的位数是8的倍数，追加1个0
		bv.data = append(bv.data, 0)
	}
	bv.data[bv.n/8] |= bit << (bv.n % 8)
	bv.n++
}

func addTypeBits(bv *bitVector, offset uintptr, t *rtype) { // 注：#
	if t.ptrdata == 0 {
		return
	}

	switch Kind(t.kind & kindMask) {
	case Chan, Func, Map, Ptr, Slice, String, UnsafePointer:
		// 表示开始时有1个指针
		for bv.n < uint32(offset/uintptr(ptrSize)) {
			bv.append(0)
		}
		bv.append(1)

	case Interface:
		// 2 pointers
		for bv.n < uint32(offset/uintptr(ptrSize)) {
			bv.append(0)
		}
		bv.append(1)
		bv.append(1)

	case Array:
		// repeat inner type
		tt := (*arrayType)(unsafe.Pointer(t))
		for i := 0; i < int(tt.len); i++ {
			addTypeBits(bv, offset+uintptr(i)*tt.elem.size, tt.elem)
		}

	case Struct:
		// apply fields
		tt := (*structType)(unsafe.Pointer(t))
		for i := range tt.fields {
			f := &tt.fields[i]
			addTypeBits(bv, offset+f.offset(), f.typ)
		}
	}
}
