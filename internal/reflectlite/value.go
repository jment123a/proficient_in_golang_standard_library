// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package reflectlite

import (
	"runtime"
	"unsafe"
)

const ptrSize = 4 << (^uintptr(0) >> 63) // unsafe.Sizeof(uintptr(0))但理想的const，注：指针的标准长度，根据操作系统位数变化而变化

// Value 是Go值的反射接口。
// 并非所有方法都适用于所有类型的值。 在每种方法的文档中都注明了限制（如果有）。
// 在调用特定于种类的方法之前，使用Kind方法找出值的种类。 调用不适合该类型的方法会导致运行时恐慌。
// 零值表示无值。
// 它的IsValid方法返回false，其Kind方法返回Invalid，其String方法返回"<invalid Value>"，所有其他方法均会出现紧急情况。
// 大多数函数和方法从不返回无效值。
// 如果是这样，则其文档会明确说明条件。
// 一个Value可以被多个goroutine并发使用，前提是可以将基础Go值同时用于等效的直接操作。
// 要比较两个值，请比较Interface方法的结果。
// 在两个值上使用==不会比较它们表示的基础值。
type Value struct {
	// typ 保留由Value表示的值的类型
	typ *rtype

	// 指针值的数据；如果设置了flagIndir，则为数据的指针。
	// 在设置flagIndir或typ.pointers()为true时有效。
	ptr unsafe.Pointer //注：指向数据的指针

	// flag 保存有关该值的元数据。
	// 最低位是标志位：
	// -flagStickyRO：通过未导出的未嵌入字段获取，因此为只读
	// -flagEmbedRO：通过未导出的嵌入式字段获取，因此为只读
	// -flagIndir：val保存指向数据的指针
	// -flagAddr：v.CanAddr为true（表示flagIndir）
	// Value不能代表方法值。
	// 接下来的五位给出Value的种类。
	// 重复typ.Kind()，方法值除外。
	// 其余的23+位给出方法值的方法编号。
	// 如果flag.kinz() != Func，则代码可以假定flagMethod未设置。
	// 如果是ifaceIndir（typ），则代码可以假定设置了flagIndir。

	//flag[len(flag) - 10]：
	//flag[len(flag) - 9]：是否可以寻址（是否分配内存）
	//flag[len(flag) - 8]：是否有指向数据的指针
	//flag[len(flag) - 7]：是否通过未导出的嵌入式字段获得的
	//flag[len(flag) - 6]：是否通过未导出的未嵌入字段获得的
	//flag[len(flag) - 1 : len(flag) - 5]：类型的枚举
	flag

	// 方法值代表某些接收者r的curd方法调用（如r.Read）
	// typ + val + flag位描述接收方r，但是标志的Kind位表示Func（方法是函数），并且该标志的高位给出r的类型的方法表中的方法编号。
}

type flag uintptr

const (
	flagKindWidth      = 5                    //有27种 //注：Value的类型枚举，在reflectlite/type.go中Kind表示
	flagKindMask  flag = 1<<flagKindWidth - 1 //注：0000 0001 1111，获取flag的倒数5位，记录类型枚举
	flagStickyRO  flag = 1 << 5               //注：0000 0010 0000，获取flag的倒数第6位，是否通过未导出的未嵌入字段获得的
	flagEmbedRO   flag = 1 << 6               //注：0000 0100 0000，获取flag的倒数第7位，是否通过未导出的嵌入式字段获得的
	// flagIndir
	// 为0时，Value.ptr是一个指向（指向数据的指针）的指针，间接指针
	// 为1时，Value.ptr是一个指向数据的指针
	flagIndir       flag = 1 << 7                     //注：0000 1000 0000，获取flag的倒数第8位，是否有指向数据的指针
	flagAddr        flag = 1 << 8                     //注：0001 0000 0000，获取flag的倒数第9位，是否可以寻址（是否分配内存）
	flagMethod      flag = 1 << 9                     //注：0010 0000 0000，获取flag的倒数第10位
	flagMethodShift      = 10                         //注：0000 0001 0011，#
	flagRO          flag = flagStickyRO | flagEmbedRO //注：0000 0110 0000，获取flag的倒数第6、7位，是否通过未导出的字段获得的
)

func (f flag) kind() Kind { //注：获取f的类型枚举
	return Kind(f & flagKindMask)
}

func (f flag) ro() flag { //注：返回f是否通过未导出的字段获得的
	if f&flagRO != 0 { //注：#如果flag的倒数第6、7位不为0，则返回32
		return flagStickyRO
	}
	return 0 //注：否则返回0
}

// pointer 返回由v表示的基础指针。
// v.Kind()必须是Ptr，Map，Chan，Func或UnsafePointer
func (v Value) pointer() unsafe.Pointer { //注：获取v指向数据的指针
	if v.typ.size != ptrSize || !v.typ.pointers() { //注：v是否为指针，v的长度是否为指针的长度
		panic("can't call pointer on a non-pointer Value") //恐慌："不能在非指针值上调用指针"
	}
	if v.flag&flagIndir != 0 { //注：v指向数据的指针不为空，返回v的指针
		return *(*unsafe.Pointer)(v.ptr)
	}
	return v.ptr
}

// packEface 将v转换为空接口。
func packEface(v Value) interface{} { //注：将v转换位为空接口
	t := v.typ
	var i interface{}
	e := (*emptyInterface)(unsafe.Pointer(&i))
	//首先，填写接口的数据部分。
	switch {
	case ifaceIndir(t): //注：t是否间接（作为指针）存储在接口值中
		if v.flag&flagIndir == 0 { //注：是否有指针
			panic("bad indir") //恐慌："错误的间接访问"
		}
		//Value是间接的，我们创建的接口也是间接的。
		ptr := v.ptr
		if v.flag&flagAddr != 0 { //注：如果v分配了地址
			// TODO：从valueInterface传递安全布尔值，因此如果safe == true，我们不需要复制？
			c := unsafe_New(t)      //注：#
			typedmemmove(t, c, ptr) //注：#将ptr作为t类型存到到c中
			ptr = c
		}
		e.word = ptr
	case v.flag&flagIndir != 0: //注：v是指针
		//Value 是间接的，但接口是直接的。 我们需要将v.ptr处的数据加载到接口数据字中。
		e.word = *(*unsafe.Pointer)(v.ptr)
	default:
		//Value是直接的，接口也是直接的。
		e.word = v.ptr
	}
	//现在，填写类型部分。 在这里，我们非常小心，不要在e.word和e.typ分配之间进行任何操作，以免垃圾回收器观察部分构建的接口值。
	e.typ = t
	return i
}

// unpackEface 将空接口i转换为Value。
func unpackEface(i interface{}) Value { //注：将空接口i转换为Value
	e := (*emptyInterface)(unsafe.Pointer(&i))
	//注意：在我们知道e.word是否真的是指针之前，不要读它。
	t := e.typ
	if t == nil { //注：如果i为nil，返回一个空Value
		return Value{}
	}
	f := flag(t.Kind())
	if ifaceIndir(t) { //注：检查t是否是指针
		f |= flagIndir
	}
	return Value{t, e.word, f}
}

// ValueError 在不支持Value的Value方法上调用Value方法时，发生ValueError。 在每种方法的说明中都记录了这种情况。
type ValueError struct {
	Method string //注：什么方法引发了错误
	Kind   Kind
}

func (e *ValueError) Error() string { //注：返回ValueError
	return "reflect: call of " + e.Method + " on zero Value" //注：返回"在空Value上调用Method"
}

// methodName 返回调用方法的名称，假定位于上面两个堆栈框架中。
func methodName() string { //注：#
	pc, _, _, _ := runtime.Caller(2)
	f := runtime.FuncForPC(pc)
	if f == nil {
		return "unknown method"
	}
	return f.Name()
}

// emptyInterface 是interface{}值的标头。
type emptyInterface struct {
	typ  *rtype
	word unsafe.Pointer
}

// mustBeExported 如果f记录该值是使用未导出的字段获得的，则表示恐慌。
func (f flag) mustBeExported() { //注：判断f是否通过使用未导出的字段获取的
	if f == 0 { //注：如果f的类型是Invalid
		panic(&ValueError{methodName(), 0}) //引发恐慌
	}
	if f&flagRO != 0 { //注：#
		panic("reflect: " + methodName() + " using value obtained using unexported field") //恐慌："使用通过未导出字段获得的值"
	}
}

// mustBeAssignable 如果f记录了该值不可分配，则表示恐慌，也就是说，该值是使用未导出的字段获取的，或者该值不可寻址。
func (f flag) mustBeAssignable() { //注：判断f是否通过未导出的字段获取的，或者是否不可寻址
	if f == 0 { //注：如果f的类型是Invalid
		panic(&ValueError{methodName(), Invalid}) //引发恐慌
	}
	//如果可寻址且不是只读则可分配。
	if f&flagRO != 0 {
		panic("reflect: " + methodName() + " using value obtained using unexported field") //恐慌："使用通过未导出字段获得的值"
	}
	if f&flagAddr == 0 {
		panic("reflect: " + methodName() + " using unaddressable value") //恐慌："使用不可寻址的值"
	}
}

// CanSet 报告v的值是否可以更改。
// 仅当值是可寻址的并且不是通过使用未导出的结构体字段获得的，才可以更改该值。
// 如果CanSet返回false，则调用Set或任何特定于类型的setter（例如SetBool，SetInt）都会感到恐慌。
func (v Value) CanSet() bool { //注：v是否可以更改（是否可寻址并且不是通过使用未导出的字段获得的）
	return v.flag&(flagAddr|flagRO) == flagAddr //注：如果flagRO位为1，则与操作后一定会也会为1，就会与flagAddr不相等
}

// Elem 返回接口v包含的值或指针v指向的值。
// 如果v的Kind不是Interface或Ptr，它会感到恐慌。
// 如果v为nil，则返回零值。
func (v Value) Elem() Value { //注：返回接口或指针v的值
	k := v.kind()
	switch k {
	case Interface: //注：如果v是接口
		var eface interface{}
		if v.typ.NumMethod() == 0 {
			eface = *(*interface{})(v.ptr)
		} else {
			eface = (interface{})(*(*interface {
				M()
			})(v.ptr))
		}
		x := unpackEface(eface)
		if x.flag != 0 {
			x.flag |= v.flag.ro()
		}
		return x
	case Ptr: //注：如果v是指针
		ptr := v.ptr
		if v.flag&flagIndir != 0 {
			ptr = *(*unsafe.Pointer)(ptr)
		}
		// The returned value's address is v's value.
		if ptr == nil {
			return Value{}
		}
		tt := (*ptrType)(unsafe.Pointer(v.typ))
		typ := tt.elem
		fl := v.flag&flagRO | flagIndir | flagAddr
		fl |= flag(typ.Kind())
		return Value{typ, ptr, fl}
	}
	panic(&ValueError{"reflectlite.Value.Elem", v.kind()}) //引发恐慌
}

func valueInterface(v Value) interface{} { //注：将v转为空接口返回
	if v.flag == 0 {
		panic(&ValueError{"reflectlite.Value.Interface", 0})
	}

	if v.kind() == Interface { //注：如果v是接口类型，直接转换
		//特例：返回接口内的元素。
		//空接口具有一种布局，所有带方法的接口具有另一种布局。
		if v.numMethod() == 0 { //注：如果v没有方法
			return *(*interface{})(v.ptr) //注：将指针转为空接口返回
		}
		return *(*interface {
			M() //注：#
		})(v.ptr)
	}

	// TODO：将安全传递给packEface，因此，如果safe == true，我们不需要复制吗？
	return packEface(v) //注：将v转为空接口
}

// IsNil 报告其参数v是否为nil。
// 参数必须是chan，func，interface，map，pointer或slice值； 如果不是，则IsNil感到恐慌。
// 请注意，IsNil并不总是等同于Go中与nil的常规比较。
// 例如，如果v是通过使用未初始化的接口变量i调用ValueOf来创建的，则i == nil为true，但v.IsNil会感到恐慌，因为v为零值。
func (v Value) IsNil() bool { //注：返回v是否为nil（v有没有分配指向数据的指针）
	k := v.kind()
	switch k {
	case Chan, Func, Map, Ptr, UnsafePointer:
		// if v.flag&flagMethod != 0 {
		// 	return false
		// }
		ptr := v.ptr
		if v.flag&flagIndir != 0 { //注：如果v指向数据的指针不为空
			ptr = *(*unsafe.Pointer)(ptr)
		}
		return ptr == nil //注：检查v的指针是否为空
	case Interface, Slice:
		//如果第一个单词为0，则interface和slice均为零。
		//两者总是大于一个单词； 假设flagIndir。
		return *(*unsafe.Pointer)(v.ptr) == nil //注：检查v的指针是否为空
	}
	panic(&ValueError{"reflectlite.Value.IsNil", v.kind()})
}

// IsValid reports whether v represents a value. 注：IsValid 报告v是否代表值。
// 如果v为零值，则返回false。
// If IsValid returns false, all other methods except String panic. 注：如果IsValid返回false，则所有其他方法（字符串恐慌除外）。
// 大多数函数和方法从不返回无效的值。
//如果是，则其文档会明确说明条件。
func (v Value) IsValid() bool { //注：验证v是否有效
	return v.flag != 0 //注：判断v的flag是否不为0
}

// Kind 返回v的Kind。
// 如果v为零值（IsValid返回false），则Kind返回Invalid。
func (v Value) Kind() Kind { //注：返回v的类型枚举
	return v.kind()
}

//在runtime实现：
func chanlen(unsafe.Pointer) int
func maplen(unsafe.Pointer) int

// Len 返回v的长度。
//如果v的Kind不是Array，Chan，Map，Slice或String，则它会感到恐慌。
func (v Value) Len() int { //注：返回v的长度
	k := v.kind()
	switch k {
	case Array:
		tt := (*arrayType)(unsafe.Pointer(v.typ))
		return int(tt.len)
	case Chan:
		return chanlen(v.pointer())
	case Map:
		return maplen(v.pointer())
	case Slice:
		// Slice 大于 word; assume flagIndir.
		return (*sliceHeader)(v.ptr).Len
	case String:
		// String 大于 word; assume flagIndir.
		return (*stringHeader)(v.ptr).Len
	}
	panic(&ValueError{"reflect.Value.Len", v.kind()})
}

// NumMethod 返回值的方法集中导出的方法的数量。
func (v Value) numMethod() int { //注：获取v的导出方法数量
	if v.typ == nil {
		panic(&ValueError{"reflectlite.Value.NumMethod", Invalid})
	}
	return v.typ.NumMethod()
}

// Set 将x赋给值v。
// 如果CanSet返回false，则会感到恐慌。
// 和Go一样，x的值必须可分配给v的类型。
func (v Value) Set(x Value) {
	v.mustBeAssignable() //注：必须是可分配的
	x.mustBeExported()   //不要让未导出的x泄漏，注：必须是导出的
	var target unsafe.Pointer
	if v.kind() == Interface {
		target = v.ptr
	}
	x = x.assignTo("reflectlite.Set", v.typ, target) //注：将x转为v的格式
	if x.flag&flagIndir != 0 {                       //注：如果x是指针，
		typedmemmove(v.typ, v.ptr, x.ptr) //注：将x的值转为v的格式赋值给v
	} else {
		*(*unsafe.Pointer)(v.ptr) = x.ptr //注：否则直接替换指针
	}
}

// Type 返回v的类型。
func (v Value) Type() Type { //注：返回v的类型（v.typ）
	f := v.flag
	if f == 0 {
		panic(&ValueError{"reflectlite.Value.Type", Invalid})
	}
	//不支持方法值。
	return v.typ
}

// stringHeader 是此包中使用的StringHeader的安全版本。
type stringHeader struct {
	Data unsafe.Pointer
	Len  int
}

// sliceHeader 是此包中使用的SliceHeader的安全版本。
type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

/*
 * constructors
 */

// 在包runtime中实现
func unsafe_New(*rtype) unsafe.Pointer

// ValueOf 返回一个新的Value，初始化为存储在接口i中的具体值。 ValueOf(nil)返回零值。
func ValueOf(i interface{}) Value { //注：将i转存到堆中，转为Value并返回
	if i == nil {
		return Value{}
	}
	// TODO：也许允许Value的内容存在于堆栈中。
	// 现在，我们使内容始终转储到堆中。
	// 它使某些地方的生活更加轻松（请参阅下面的chanrecv/mapassign评论）。
	escapes(i) //注：将i转储到堆中

	return unpackEface(i) //注：将i转为Value并返回
}

// assignTo returns a value v that can be assigned directly to typ. 返回可直接分配给typ的值v。
// 如果无法将v分配给typ，则会出现紧急情况。
// 要转换为接口类型，target是建议使用的暂存空间。
func (v Value) assignTo(context string, dst *rtype, target unsafe.Pointer) Value { //注：将v转为dst的格式，如果要分配新空间，则使用target暂存，如果引发恐慌，显示内容为context
	// if v.flag&flagMethod != 0 {
	// 	v = makeMethodValue(context, v)
	// }

	switch {
	case directlyAssignable(dst, v.typ): //注：如果v可以直接分配给dst
		//覆盖类型，使其匹配。
		//相同的内存布局，因此无害。
		fl := v.flag&(flagAddr|flagIndir) | v.flag.ro() //注：根据v的flag设置dst的flag是否分配内存，是否未导出
		fl |= flag(dst.Kind())                          //注：设置dst的flag的数据类型枚举
		return Value{dst, v.ptr, fl}                    //注：返回一个类型为dst的，数据指针指向v，是否分配内存、是否导出和v相同的Value

	case implements(dst, v.typ): //注：如果v作为接口或类型，是否可以实现dst
		if target == nil {
			target = unsafe_New(dst) //注：#
		}
		if v.Kind() == Interface && v.IsNil() { //注：接口是未分配的接口
			//传递给nil Reader的nil ReadWriter是可以的，但是在下面使用ifaceE2I会感到恐慌。
			//通过明确返回nil dst（例如Reader）来避免恐慌。
			return Value{dst, nil, flag(Interface)} //注：返回一个空的dst
		}
		x := valueInterface(v)    //注：将v转换成空接口
		if dst.NumMethod() == 0 { //注：如果dst没有方法，直接转换
			*(*interface{})(target) = x
		} else {
			ifaceE2I(dst, x, target) //注：#，将x作为dst的类型，赋值给target
		}
		return Value{dst, target, flagIndir | flag(Interface)} //注：返回一个类型为dst的，数据为新分配的，指针接口类型
	}
	// Failed.
	panic(context + ": value of type " + v.typ.String() + " is not assignable to type " + dst.String()) //恐慌："v的类型无法分配到dst"
}

// arrayAt 返回p的第i个元素，即元素为eltSize字节宽的数组。
// p指向的数组必须至少包含i + 1个元素：
// 传递i >= len是无效的（但无法在此处检查），因为结果将指向数组之外。
// whySafe必须解释为什么i < len。 （传递"i < len"是可以的；这样做的好处是可以在呼叫站点显示此假设。）
func arrayAt(p unsafe.Pointer, i int, eltSize uintptr, whySafe string) unsafe.Pointer { //注：whySafe输入为什么这个操作是安全的，将p偏移eltSize个int，返回指针
	return add(p, uintptr(i)*eltSize, "i < len")
}

func ifaceE2I(t *rtype, src interface{}, dst unsafe.Pointer)

// typedmemmove 将类型t的值从src复制到dst。
// go：不会栈逃逸
func typedmemmove(t *rtype, dst, src unsafe.Pointer)

//虚拟注释，标记x的值转义，在反射代码非常聪明以至于编译器无法遵循的情况下使用。
func escapes(x interface{}) {
	if dummy.b {
		dummy.x = x
	}
}

var dummy struct {
	b bool
	x interface{}
}
