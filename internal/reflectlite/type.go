//版权所有2009 The Go Authors。 版权所有。
//此源代码的使用受BSD样式的约束
//可以在LICENSE文件中找到的许可证。

// Package reflectlite 实现轻量级的reflect，不使用"runtime"和"unsafe"以外的任何软件包。
package reflectlite

import (
	"unsafe"
)

// Type 是Go类型的表示。
// 并非所有方法都适用于所有类型。 在每种方法的文档中都注明了限制（如果有）。
// 在调用特定于种类的方法之前，使用Kind方法找出类型。 调用不适合该类型的方法会导致运行时恐慌。
// Type值是可比较的，例如==运算符，因此它们可用作映射键。
// 如果两个Type值表示相同的类型，则它们相等。
type Type interface {
	// 适用于所有类型的方法。

	// Name 返回其包中已定义类型的类型名称。
	// 对于其他（未定义）类型，它返回空字符串。
	Name() string //注：返回类型名称

	// PkgPath 返回定义的类型的程序包路径，即唯一标识程序包的导入路径，例如"encoding/base64"。
	// 如果类型是预先声明的（字符串，错误）或未定义（*T, struct{}, []int或A，其中A是未定义类型的别名），则包路径将为空字符串 。
	PkgPath() string //注：返回定义类型的包路径，例如"encoding/base64"。

	// Size 返回存储给定类型的值所需的字节数； 它类似于unsafe.Sizeof。
	Size() uintptr //注：返回值所需的字节数

	// Kind 返回此类型的特定种类。
	Kind() Kind //注：返回类型的枚举

	// Implements 报告类型是否实现接口类型u。
	Implements(u Type) bool //注：是否可以实现接口u

	// AssignableTo 报告该类型的值是否可分配给u类型。
	AssignableTo(u Type) bool //注：是否可以分配或实现u

	//Comparable 报告此类型的值是否可比较。
	Comparable() bool //注：是否有equal方法，是否可以比较

	// String 返回该类型的字符串表示形式。
	// 字符串表示形式可以使用缩短的包名称（例如，使用base64代替"encoding/base64"），并且不能保证类型之间的唯一性。
	// 要测试类型标识，请直接比较类型。
	String() string //注：#

	// Elem 返回类型的元素类型。
	// 如果类型的Kind不是Ptr，则会出现恐慌。
	Elem() Type //注：#

	common() *rtype          //注：#
	uncommon() *uncommonType //注：#
}

/*
 * 这些数据结构是编译器已知的（../../cmd/internal/gc/reflect.go）。
 * ../runtime/type.go已知有一些可以传达给调试器。
 * 他们也以../runtime/type.go着称。
 */

// Kind 代表类型所代表的特定类型。
// 零种类不是有效种类。
type Kind uint //注：26种go自带数据类型

const (
	//Invalid 0	无效类型
	Invalid Kind = iota
	//Bool 1
	Bool
	//Int 2
	Int
	//Int8 3
	Int8
	//Int16 4
	Int16
	//Int32 5
	Int32
	//Int64 6
	Int64
	//Uint 7
	Uint
	//Uint8 8
	Uint8
	//Uint16 9
	Uint16
	//Uint32 10
	Uint32
	//Uint64 11
	Uint64
	//Uintptr 12
	Uintptr
	//Float32 13
	Float32
	//Float64 14
	Float64
	//Complex64 15
	Complex64
	//Complex128 16
	Complex128
	//Array 17
	Array
	//Chan 18
	Chan
	//Func 19
	Func
	//Interface 20
	Interface
	//Map 21
	Map
	//Ptr 22
	Ptr
	//Slice 23
	Slice
	//String 24
	String
	//Struct 25
	Struct
	//UnsafePointer 26
	UnsafePointer
)

// tflag 由rtype使用//来指示紧随rtype值之后在内存中还有哪些额外的类型信息。
//
// tflag值必须与以下位置的副本保持同步：
// cmd/compile/internal/gc/reflect.go
// cmd/link/internal/ld/decodesym.go
// runtime/type.go
type tflag uint8 //注：rtype中额外的信息标志位

const (
	// tflagUncommon 表示在外部类型结构的正上方有一个指针*uncommonType。
	// 例如，如果t.Kind() == Struct且t.tflag&tflagUncommon = 0，则t具有uncommonType数据，可以按以下方式访问它：
	//
	// type tUncommon struct {
	//     structType
	//     u uncommonType
	// }
	// u := &(*tUncommon)(unsafe.Pointer(t)).u
	tflagUncommon tflag = 1 << 0 //注：0000 0001，掩码位，是否有uncommonType

	// tflagExtraStar 表示str字段中的名称带有多余的"*"前缀。
	// 这是因为对于程序中的大多数T类型，*T类型也存在，并且重新使用str数据可节省二进制大小。
	tflagExtraStar tflag = 1 << 1 //注：0000 0010，掩码位，str是否有多余的"*"前缀。

	// tflagNamed 表示类型具有名称。
	tflagNamed tflag = 1 << 2 //注：0000 0100，掩码位，是否具有名称

	// tflagRegularMemory 意味着equal和hash函数可以将此类型视为t.size字节的单个区域。
	tflagRegularMemory tflag = 1 << 3 //注：0000 1000，掩码位，是否equal和hash函数可以将此类型视为t.size字节的单个区域。
)

// rtype 是大多数值的通用实现。
// 它嵌入在其他结构类型中。
//
// rtype必须与../runtime/type.go:/^type._type保持同步。
// tflag[0]：
// tflag[1]：
// tflag[2]：
// tflag[3]：
// tflag[4]：是否equal和hash函数可以将此类型视为t.size字节的单个区域。
// tflag[5]：类型是否有名称
// tflag[6]：str中是否带有多余的*
// tflag[7]：外部类型结构是否有uncommonType
type rtype struct {
	size       uintptr //注：大小
	ptrdata    uintptr // 类型中可以包含指针的字节数
	hash       uint32  // 类型的哈希； 避免在哈希表中进行计算
	tflag      tflag   // 额外类型信息标志
	align      uint8   // 将此类型的变量对齐
	fieldAlign uint8   // 将此类型的结构字段对齐
	kind       uint8   // C的枚举，注：Kind&Kindmask计算得出26种内部类型
	// 比较此类对象的函数
	//（将ptr指向对象A，将ptr指向对象B）-> ==
	equal     func(unsafe.Pointer, unsafe.Pointer) bool //注：是否可以比较，不为nil则可以比较
	gcdata    *byte                                     // 垃圾收集数据
	str       nameOff                                   // 字符串形式
	ptrToThis typeOff                                   // 指向此类型的指针的类型，可以为零
}

//非接口类型的方法
type method struct {
	name nameOff // 方法名称
	mtyp typeOff // 方法类型（无接收者）
	ifn  textOff // 接口调用中使用的fn（单字接收器）
	tfn  textOff // fn用于常规方法调用
}

// uncommonType 仅对定义的类型或带有方法的类型存在（如果T是定义的类型，则T和*T的uncommonTypes具有方法）。
// 使用指向此结构的指针可减少描述没有方法的未定义类型所需的总体大小。
type uncommonType struct {
	pkgPath nameOff // 导入路径； 对于内置类型（如int，string）为空
	mcount  uint16  // 方法数量
	xcount  uint16  // 导出方法的数量，注：第一个字母为大写的方法
	moff    uint32  // 从此uncommonType到达第1个方法的偏移量
	_       uint32  // 没用过
}

// chanDir 表示通道类型的方向。
type chanDir int //注：表示通道的方向

const (
	recvDir chanDir             = 1 << iota // <-chan
	sendDir                                 // chan<-
	bothDir = recvDir | sendDir             // chan
)

// arrayType 表示固定的数组类型。
type arrayType struct {
	rtype
	elem  *rtype // 数组元素类型
	slice *rtype // 切片类型
	len   uintptr
}

// chanType 表示通道类型。
type chanType struct {
	rtype
	elem *rtype  // 通道元素类型
	dir  uintptr // 通道方向（chanDir）
}

// funcType 表示函数类型。
//
// 每个in和out参数的*rtype存储在一个数组中，该数组紧随funcType（可能还有其uncommonType）。
// 因此，具有一个方法，一个输入和一个输出的函数类型为：
//
// struct {
//     funcType
//     uncommonType
//     [2] * rtype // [0]为in，[1]为out
// }
type funcType struct {
	rtype
	inCount  uint16
	outCount uint16 //如果最后一个输入参数为...，则设置最高位
}

// imethod 表示接口类型上的方法
type imethod struct {
	name nameOff // 方法名称
	typ  typeOff // .(*FuncType) underneath
}

// interfaceType 表示接口类型。
type interfaceType struct {
	rtype
	pkgPath name      // 导入路径
	methods []imethod // 按哈希排序，注：接口的方法集合
}

// mapType 表示集合类型。
type mapType struct {
	rtype
	key        *rtype // 集合键类型
	elem       *rtype // 集合元素（值）类型
	bucket     *rtype // 内部存储桶结构函数，用于散列键（从ptr到键，种子）->散列
	hasher     func(unsafe.Pointer, uintptr) uintptr
	keysize    uint8  // key slot的大小
	valuesize  uint8  // value slot的大小
	bucketsize uint16 // 桶的大小
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

// Struct 字段
type structField struct {
	name        name    //名称始终为非空
	typ         *rtype  // 领域类型
	offsetEmbed uintptr // 字段的字节偏移量<<1 | isEmbedded
}

func (f *structField) offset() uintptr {
	return f.offsetEmbed >> 1
}

func (f *structField) embedded() bool {
	return f.offsetEmbed&1 != 0
}

// structType 表示结构类型。
type structType struct {
	rtype
	pkgPath name
	fields  []structField // 按偏移量排序
}

// name 是带有可选的额外数据的编码类型名称。
// 第1个字节是1个位字段，其中包含：
// 1 << 0 导出名称
// 1 << 1 标签数据跟随名称
// 1 << 2 pkgPath nameOff跟随名称和标记
//
// 接下来的2个字节是数据长度：
// l := uint16(data[1]) << 8 | uint16(data[2])
//
// 字节[3 : 3+l]是字符串数据。
// 如果跟随标签数据，则字节3 + l和3 + l + 1是标签长度，其后跟随数据。
// 如果遵循导入路径，则数据末尾的4个字节形成nameOff。 仅为在与包类型不同的包中定义的具体方法设置导入路径。
//
// 如果名称以"*"开头，则导出的位表示是否导出了所指向的类型。
//注：
// offset：0
// name.bytes[0]：
// name.bytes[1]：
// name.bytes[2]：
// name.bytes[3]：
// name.bytes[4]：
// name.bytes[5]：是否遵循导入路径
// name.bytes[6]：是否有标签数据
// name.bytes[7]：是否导出
// offset：1
// name.bytes：数据长度高位
// offset：2
// name.bytes：数据长度低位，与数据长度高位组合为l，是数据长度
// offset：3 - 3+l
// name.bytes：字符串数据
// offset：3+l
// name.bytes：标签长度高位
// offset：3+l+1
// name.bytes：标签长度低位，与标签长度高位组合为l1，是标签长度
// offset：3+l+2 - 3+l+2+l1
// name.bytes：标签数据
type name struct {
	bytes *byte
}

func (n name) data(off int, whySafe string) *byte { //注：返回*n.bytes + off的指针，以及whySafe为什么这个操作是安全的
	return (*byte)(add(unsafe.Pointer(n.bytes), uintptr(off), whySafe))
}

func (n name) isExported() bool { //注：返回n是否为导出（首字母是否大写，外部是否可以调用）
	return (*n.bytes)&(1<<0) != 0 //注：返回*n.bytes的最后1位是否 != 0
}

func (n name) nameLen() int { //注：返回数据长度
	return int(uint16(*n.data(1, "name len field"))<<8 | uint16(*n.data(2, "name len field"))) //注：取出n.bytes指针位置的下2个字节，组合为数据长度并返回
}

func (n name) tagLen() int { //注：返回标签长度
	if *n.data(0, "name flag field")&(1<<1) == 0 { //注：如果*n.bytes的倒数第2个字节 == 1，返回0
		return 0
	}
	off := 3 + n.nameLen() //注：标签数据长度
	return int(uint16(*n.data(off, "name taglen field"))<<8 | uint16(*n.data(off+1, "name taglen field")))
}

func (n name) name() (s string) { //注：返回n的数据指针与长度
	if n.bytes == nil {
		return
	}
	b := (*[4]byte)(unsafe.Pointer(n.bytes)) //注：获取前4位

	hdr := (*stringHeader)(unsafe.Pointer(&s))
	hdr.Data = unsafe.Pointer(&b[3])   //注：将数据起始指针赋值给hdr.Data
	hdr.Len = int(b[1])<<8 | int(b[2]) //注：将数据长度赋值给hdr.Len
	return s
}

func (n name) tag() (s string) { //注：返回n的标签指针与长度
	tl := n.tagLen() //注：获取n的标签长度
	if tl == 0 {     //注：如果没有标签则返回
		return ""
	}
	nl := n.nameLen() //注：获取n的数据长度
	hdr := (*stringHeader)(unsafe.Pointer(&s))
	hdr.Data = unsafe.Pointer(n.data(3+nl+2, "non-empty string")) //注：将标签数据起始指针赋值给hdr.Data
	hdr.Len = tl                                                  //注：将标签长度赋值给hdr.Data
	return s
}

func (n name) pkgPath() string { //注：返回n的包路径
	if n.bytes == nil || *n.data(0, "name flag field")&(1<<2) == 0 { //注：是否跟随导入路径
		return ""
	}
	off := 3 + n.nameLen()        //注：获取数据尾部偏移量
	if tl := n.tagLen(); tl > 0 { //注：获取标签数据尾部偏移量
		off += 2 + tl
	}
	var nameOff int32
	//请注意，此字段可能未在内存中对齐，因此我们在此处不能使用直接的int32分配。
	copy((*[4]byte)(unsafe.Pointer(&nameOff))[:], (*[4]byte)(unsafe.Pointer(n.data(off, "name offset field")))[:]) //注：获取4位nameoff数据
	pkgPathName := name{(*byte)(resolveTypeOff(unsafe.Pointer(n.bytes), nameOff))}
	return pkgPathName.name()
}

/*
 * 编译器知道上面所有数据结构的确切布局。
 * 编译器不了解以下数据结构和方法。
 */

const (
	kindDirectIface = 1 << 5       //注：是否间接存储再接口值中
	kindGCProg      = 1 << 6       //Type.gc指向GC程序
	kindMask        = (1 << 5) - 1 //注：11111，类型掩码
)

func (t *uncommonType) methods() []method { //注：返回t的所有方法
	if t.mcount == 0 { //注：如果t没有方法，返回nil
		return nil
	}
	return (*[1 << 16]method)(add(unsafe.Pointer(t), uintptr(t.moff), "t.mcount > 0"))[:t.mcount:t.mcount] //注：指针偏移t.moff，到达方法处
}

func (t *uncommonType) exportedMethods() []method { //注：返回t的所有导出方法
	if t.xcount == 0 { //注：如果t没有导出方法，返回nil
		return nil
	}
	return (*[1 << 16]method)(add(unsafe.Pointer(t), uintptr(t.moff), "t.xcount > 0"))[:t.xcount:t.xcount] //注：指针偏移t.moff，到达方法处
}

// resolveNameOff 解析与基本指针的名称偏移量。
// (*rtype).nameOff方法是此函数的便捷包装。
// 在runtime包中实现。
func resolveNameOff(ptrInModule unsafe.Pointer, off int32) unsafe.Pointer

// resolveTypeOff 从基本类型解析*rtype偏移量。
// (*rtype).typeOff方法是此函数的便捷包装。
// 在runtime包中实现。
func resolveTypeOff(rtype unsafe.Pointer, off int32) unsafe.Pointer

type nameOff int32 // 偏移名称
type typeOff int32 // 偏移到*rtype
type textOff int32 // 与文字部分顶部的偏移量

func (t *rtype) nameOff(off nameOff) name { //注：获取t+偏移量off的name
	return name{(*byte)(resolveNameOff(unsafe.Pointer(t), int32(off)))}
}

func (t *rtype) typeOff(off typeOff) *rtype { //注：获取t+偏移量off的rtype
	return (*rtype)(resolveTypeOff(unsafe.Pointer(t), int32(off)))
}

func (t *rtype) uncommon() *uncommonType { //注：返回t的uncommonType
	if t.tflag&tflagUncommon == 0 { //注：t的外部类型结构不拥有指针*uncommonType
		return nil
	}
	switch t.Kind() { //注：获取t的类型
	case Struct:
		return &(*structTypeUncommon)(unsafe.Pointer(t)).u //注：t的外部类型结构中的u为结构体
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

func (t *rtype) String() string { //注：返回t的字符串形式
	s := t.nameOff(t.str).name()     //注：获取t的字符串形式指针
	if t.tflag&tflagExtraStar != 0 { //注：如果带有*前缀，则丢弃
		return s[1:]
	}
	return s
}

func (t *rtype) Size() uintptr { return t.size } //注：返回t的大小

func (t *rtype) Kind() Kind { return Kind(t.kind & kindMask) } //注：根据t.kind获取t的类型

func (t *rtype) pointers() bool { return t.ptrdata != 0 } //注：#

func (t *rtype) common() *rtype { return t } //注：返回t本身

func (t *rtype) exportedMethods() []method { //注：返回t的所有导出方法
	ut := t.uncommon() //注：获取t的uncommonType
	if ut == nil {
		return nil
	}
	return ut.exportedMethods() //注：返回t的所有导出方法
}

func (t *rtype) NumMethod() int {
	if t.Kind() == Interface { //注：如果t是接口
		tt := (*interfaceType)(unsafe.Pointer(t))
		return tt.NumMethod() //注：返回t接口中的所有方法数
	}
	return len(t.exportedMethods()) //注：否则返回t中的所有导出方法
}

func (t *rtype) PkgPath() string {
	if t.tflag&tflagNamed == 0 { //注：类型是否有名称
		return ""
	}
	ut := t.uncommon() //注：获取t的uncommonType
	if ut == nil {
		return ""
	}
	return t.nameOff(ut.pkgPath).name() //注：获取包导入路径的名字
}

func (t *rtype) hasName() bool { //注：t是否有名字
	return t.tflag&tflagNamed != 0
}

func (t *rtype) Name() string { //注：返回t的'.'之后的字符串形式
	if !t.hasName() { //注：t是否有名字
		return ""
	}
	s := t.String() //获取t的字符串形式
	i := len(s) - 1
	for i >= 0 && s[i] != '.' { //注：找到'.'的起始位置
		i--
	}
	return s[i+1:]
}

func (t *rtype) chanDir() chanDir { //注：返回管道类型t的通道方向
	if t.Kind() != Chan {
		panic("reflect: chanDir of non-chan type") //注：非chan类型的频道
	}
	tt := (*chanType)(unsafe.Pointer(t)) //注：转位管道类型
	return chanDir(tt.dir)               //注：返回管道方向
}

func (t *rtype) Elem() Type { //注：获取t的元素类型
	switch t.Kind() { //注：将t的转换为以下元素
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
	panic("reflect: Elem of invalid type") //恐慌："无效类型的元素"
}

func (t *rtype) In(i int) Type { //注：返回t的第i个输出参数类型
	if t.Kind() != Func { //注：如果t不是函数类型
		panic("reflect: In of non-func type") //恐慌："非函数类型"
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return toType(tt.in()[i]) //注：返回t的第i个输出参数类型
}

func (t *rtype) Key() Type {
	if t.Kind() != Map { //注：如果t的类型不是集合
		panic("reflect: Key of non-map type") //恐慌："非集合类型的键"
	}
	tt := (*mapType)(unsafe.Pointer(t))
	return toType(tt.key) //注：返回集合的键
}

func (t *rtype) Len() int {
	if t.Kind() != Array {
		panic("reflect: Len of non-array type")
	}
	tt := (*arrayType)(unsafe.Pointer(t))
	return int(tt.len)
}

func (t *rtype) NumField() int {
	if t.Kind() != Struct {
		panic("reflect: NumField of non-struct type")
	}
	tt := (*structType)(unsafe.Pointer(t))
	return len(tt.fields)
}

func (t *rtype) NumIn() int { //注：返回t的输入参数数量
	if t.Kind() != Func { //注：t的类型必须为方法
		panic("reflect: NumIn of non-func type") //恐慌："非函数类型的NumIn"
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return int(tt.inCount) //注：返回t的输入参数数量
}

func (t *rtype) NumOut() int { //注：返回t的输出参数数量
	if t.Kind() != Func { //注：t的类型必须为方法
		panic("reflect: NumOut of non-func type") //恐慌："非函数类型的NumOut"
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return len(tt.out()) //注：返回t的输出参数数量
}

func (t *rtype) Out(i int) Type {
	if t.Kind() != Func {
		panic("reflect: Out of non-func type")
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return toType(tt.out()[i])
}

func (t *funcType) in() []*rtype { //注：获取函数类型的t的所有输入参数（形参）
	uadd := unsafe.Sizeof(*t)       //注：uadd赋值为t的大小
	if t.tflag&tflagUncommon != 0 { //注：如果有uncommonType
		uadd += unsafe.Sizeof(uncommonType{}) //注：uadd += 空uncommonType大小
	}
	if t.inCount == 0 { //注：输入参数（形参）的数量为0，返回nil
		return nil
	}
	return (*[1 << 20]*rtype)(add(unsafe.Pointer(t), uadd, "t.inCount > 0"))[:t.inCount:t.inCount]
}

func (t *funcType) out() []*rtype { //注：获取函数类型的t的所有输出参数（返回值）
	uadd := unsafe.Sizeof(*t)       //注：uadd赋值为t的大小
	if t.tflag&tflagUncommon != 0 { //注：如果有uncommonType
		uadd += unsafe.Sizeof(uncommonType{}) //注：uadd += 空uncommonType大小
	}
	outCount := t.outCount & (1<<15 - 1) //注：0111 1111 1111 1111
	if outCount == 0 {                   //注：输出参数（返回值）的数量为0，返回nil
		return nil
	}
	return (*[1 << 20]*rtype)(add(unsafe.Pointer(t), uadd, "outCount > 0"))[t.inCount : t.inCount+outCount : t.inCount+outCount]
}

// add 返回p + x。
//
// whySafe字符串将被忽略，因此该函数仍可以像p + x一样有效地内联，
// 但是所有调用站点都应使用该字符串来记录为什么加法是安全的，
// 这就是为什么加法不会导致x前进到p分配的末尾，并因此错误地指向内存中的下一个块。
func add(p unsafe.Pointer, x uintptr, whySafe string) unsafe.Pointer { //注：利用unsafe.Pointer实现c中指针的计算将p+x的指针返回，需要在参数whySafe中写入为什么这是安全的
	return unsafe.Pointer(uintptr(p) + x)
}

// NumMethod 返回类型的方法集中的接口方法数。
func (t *interfaceType) NumMethod() int { return len(t.methods) } //注：返回接口的接口方法数

// TypeOf 返回表示i的动态类型的反射类型。
// 如果i是一个nil接口值，则TypeOf返回nil。
func TypeOf(i interface{}) Type { //注：返回i的动态反射类型
	eface := *(*emptyInterface)(unsafe.Pointer(&i))
	return toType(eface.typ)
}

func (t *rtype) Implements(u Type) bool {
	if u == nil { //注：u不能为nil
		panic("reflect: nil type passed to Type.Implements") //恐慌："nil类型传递给Type.Implements"
	}
	if u.Kind() != Interface { //注：u必须为接口类型
		panic("reflect: non-interface type passed to Type.Implements") //恐慌："非接口类型传递给Type.Implements"
	}
	return implements(u.(*rtype), t)
}

func (t *rtype) AssignableTo(u Type) bool { //注：t是否可以分配或实现u
	if u == nil {
		panic("reflect: nil type passed to Type.AssignableTo") //恐慌："无类型传递给Type.AssignableTo"
	}
	uu := u.(*rtype)
	return directlyAssignable(uu, t) || implements(uu, t) //注：t是否可以分配或实现u
}

func (t *rtype) Comparable() bool { //注：是否有equal方法，是否可以进行比较
	return t.equal != nil
}

// implements 报告类型V是否实现接口类型T。
func implements(T, V *rtype) bool { //注：检查V作为接口或作为类型，是否可以实现T
	if T.Kind() != Interface { //注：如果T不是接口，返回false
		return false
	}
	t := (*interfaceType)(unsafe.Pointer(T))
	if len(t.methods) == 0 { //注：如果T没有方法，返回true
		return true
	}

	//两种情况下都使用相同的算法，但是接口类型和具体类型的方法表不同，因此代码重复。
	//在两种情况下，该算法都是同时对两个列表（T方法和V方法）进行线性扫描。
	//由于方法表是以唯一的排序顺序存储的（字母顺序，没有重复的方法名称），因此对V的方法进行扫描必须沿途对每个T的方法进行匹配，否则V不会实现T。
	//这样，我们就可以在整个线性时间内运行扫描，而不是天真的搜索所需的二次时间。
	//另请参阅../runtime/iface.go。
	if V.Kind() == Interface { //注：如果V是接口
		v := (*interfaceType)(unsafe.Pointer(V))
		i := 0
		for j := 0; j < len(v.methods); j++ { //注：遍历V的方法，每次都遍历一次T的方法
			tm := &t.methods[i]
			tmName := t.nameOff(tm.name)
			vm := &v.methods[j]
			vmName := V.nameOff(vm.name)
			if vmName.name() == tmName.name() && V.typeOff(vm.typ) == t.typeOff(tm.typ) { //注：T的方法与V的方法名字与typ
				if !tmName.isExported() { //注：如果不是导出
					tmPkgPath := tmName.pkgPath() //注：获取方法的包路径
					if tmPkgPath == "" {
						tmPkgPath = t.pkgPath.name() //注：如果为空，获取接口的包路径
					}
					vmPkgPath := vmName.pkgPath()
					if vmPkgPath == "" {
						vmPkgPath = v.pkgPath.name()
					}
					if tmPkgPath != vmPkgPath { //注：如果不相等，则跳过本次循环，遍历T的下一个方法
						continue
					}
				}
				if i++; i >= len(t.methods) { //注：如果V已经满足了T的所有方法，返回true
					return true
				}
			}
		}
		return false //注：遍历完V也没有满足T，返回false
	}

	v := V.uncommon() //注：将V转为类型
	if v == nil {
		return false
	}
	i := 0
	vmethods := v.methods()              //注：获取类型V的所有方法
	for j := 0; j < int(v.mcount); j++ { //注：遍历
		tm := &t.methods[i]
		tmName := t.nameOff(tm.name)
		vm := vmethods[j]
		vmName := V.nameOff(vm.name)
		if vmName.name() == tmName.name() && V.typeOff(vm.mtyp) == t.typeOff(tm.typ) { //注：同上
			if !tmName.isExported() {
				tmPkgPath := tmName.pkgPath()
				if tmPkgPath == "" {
					tmPkgPath = t.pkgPath.name()
				}
				vmPkgPath := vmName.pkgPath()
				if vmPkgPath == "" {
					vmPkgPath = V.nameOff(v.pkgPath).name()
				}
				if tmPkgPath != vmPkgPath {
					continue
				}
			}
			if i++; i >= len(t.methods) {
				return true
			}
		}
	}
	return false
}

// directAssignable 报告是否可以将V类型的值x直接（使用记忆）分配给T类型的值。
// https://golang.org/doc/go_spec.html#Assignability忽略接口规则（在其他地方实现）和理想常量规则（运行时没有理想常量）。
func directlyAssignable(T, V *rtype) bool {
	// x的类型V == T？
	if T == V {
		return true
	}

	//否则，不得定义T和V中的至少一个，并且它们必须具有相同的种类。
	if T.hasName() && V.hasName() || T.Kind() != V.Kind() { //注：如果T和V同时有名字或类型不同，返回false
		return false
	}

	// x的类型T和V必须具有相同的基础类型。
	return haveIdenticalUnderlyingType(T, V, true)
}

func haveIdenticalType(T, V Type, cmpTags bool) bool { //注：返回T和V的名称、类型与元素是否相同,cmpTags标识是否可以进行直接比较
	if cmpTags { //注：是否可以直接比较
		return T == V
	}

	if T.Name() != V.Name() || T.Kind() != V.Kind() { //注：名字或类型不同，返回true
		return false
	}

	return haveIdenticalUnderlyingType(T.common(), V.common(), false) //注：#
}

func haveIdenticalUnderlyingType(T, V *rtype, cmpTags bool) bool { //注：返回T和V是否相同,cmpTags标识是否可以进行直接比较
	if T == V { //注：完全相同，返回true
		return true
	}

	kind := T.Kind()
	if kind != V.Kind() { //注：如果类型不同，返回false
		return false
	}

	// 相同种类的非复合类型具有相同的基础类型（该类型的预定义实例）。
	if Bool <= kind && kind <= Complex128 || kind == String || kind == UnsafePointer { //注：如果类型为基础类型，返回true
		return true
	}

	//复合类型。
	switch kind { //注：复合类型
	case Array: //注：对比长度与元素
		return T.Len() == V.Len() && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Chan: //注：对比管道方向与元素
		// 特殊情况：
		// x是双向通道值，T是通道类型，
		// 和x的类型V和T具有相同的元素类型。
		if V.chanDir() == bothDir && haveIdenticalType(T.Elem(), V.Elem(), cmpTags) { //注：如果是双向管道且元素相同，返回true
			return true
		}

		//否则继续测试相同的基础类型。
		return V.chanDir() == T.chanDir() && haveIdenticalType(T.Elem(), V.Elem(), cmpTags) //注：否则对比管道方向

	case Func: //注：对比输入、输出参数
		t := (*funcType)(unsafe.Pointer(T))
		v := (*funcType)(unsafe.Pointer(V))
		if t.outCount != v.outCount || t.inCount != v.inCount { //注：如果输出参数不同或输入参数不同，返回false
			return false
		}
		for i := 0; i < t.NumIn(); i++ { //注：如果输入参数不相等，返回false
			if !haveIdenticalType(t.In(i), v.In(i), cmpTags) {
				return false
			}
		}
		for i := 0; i < t.NumOut(); i++ { //注：如果输出参数不相等，返回false
			if !haveIdenticalType(t.Out(i), v.Out(i), cmpTags) {
				return false
			}
		}
		return true

	case Interface: //注：如果方法数量相同，返回true
		t := (*interfaceType)(unsafe.Pointer(T))
		v := (*interfaceType)(unsafe.Pointer(V))
		if len(t.methods) == 0 && len(v.methods) == 0 { //注：如果接口没有方法，返回true
			return true
		}

		//可能具有相同的方法，但仍需要运行时转换。
		return false

	case Map: //注：对比键类型与值类型
		return haveIdenticalType(T.Key(), V.Key(), cmpTags) && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Ptr, Slice: //注：对比值类型
		return haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Struct: //注：对比字段数量、包路径名称，字段的名称、类型、标签、偏移量（大小）
		t := (*structType)(unsafe.Pointer(T))
		v := (*structType)(unsafe.Pointer(V))
		if len(t.fields) != len(v.fields) { //注： 如果字段数量不同，返回false
			return false
		}
		if t.pkgPath.name() != v.pkgPath.name() { //注：如果包路径名称不同，返回false
			return false
		}
		for i := range t.fields { //注：遍历t的字段
			tf := &t.fields[i]
			vf := &v.fields[i]
			if tf.name.name() != vf.name.name() { //注：如果字段名称不同，返回false
				return false
			}
			if !haveIdenticalType(tf.typ, vf.typ, cmpTags) { //注：如果字段类型不同，返回false
				return false
			}
			if cmpTags && tf.name.tag() != vf.name.tag() { //注：如果字段标签不同，返回false
				return false
			}
			if tf.offsetEmbed != vf.offsetEmbed { //注：如果偏移量不同（长度），返回false
				return false
			}
		}
		return true
	}

	return false
}

type structTypeUncommon struct {
	structType
	u uncommonType
}

// toType 从*rtype转换为Type，可以将其返回给package Reflection的客户端。
// 在gc中，唯一需要注意的是必须将nil *rtype替换为nil Type，但是在gccgo中，此函数将确保确保将同一类型的多个*rtype合并为单个Type。
func toType(t *rtype) Type { //注：将t转为Type格式返回
	if t == nil {
		return nil
	}
	return t
}

// ifaceIndir 报告t是否间接存储在接口值中。
func ifaceIndir(t *rtype) bool { //注：报告t是否间接存储在接口值中。
	return t.kind&kindDirectIface == 0
}
