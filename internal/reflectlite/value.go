// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package reflectlite

import (
	"runtime"
	"unsafe"
)

const ptrSize = 4 << (^uintptr(0) >> 63) // unsafe.Sizeof(uintptr(0))但理想的const

// Value 是Go值的反射接口。
// 并非所有方法都适用于所有类型的值。 在每种方法的文档中都注明了限制（如果有）。
// 在调用特定于种类的方法之前，使用Kind方法找出值的种类。 调用不适合该类型的方法会导致运行时恐慌。
// 零值表示无值。
// 它的IsValid方法返回false，其Kind方法返回Invalid，其String方法返回"<invalid Value>"，所有其他方法均会出现紧急情况。
// 大多数函数和方法从不返回无效值。
// 如果是这样，则其文档会明确说明条件。
// 一个值可以由多个goroutine并发使用，前提是可以将基础Go值同时用于等效的直接操作。
// 要比较两个值，请比较Interface方法的结果。
// 在两个值上使用==不会比较它们表示的基础值。
type Value struct {
	// typ 保留由Value表示的值的类型
	typ *rtype

	// 指针值的数据；如果设置了flagIndir，则为数据的指针。
	// 在设置flagIndir或typ.pointers()为true时有效。
	ptr unsafe.Pointer

	// flag 保存有关该值的元数据。
	// 最低位是标志位：
	// -flagStickyRO：通过未导出的未嵌入字段获取，因此为只读
	// -flagEmbedRO：通过未导出的嵌入式字段获取，因此为只读
	// -flagIndir：val保存指向数据的指针
	// -flagAddr：v.CanAddr为true（表示flagIndir）
	// Value不能代表方法值。
	// 接下来的五位给出值的种类。
	// 重复typ.Kind()，方法值除外。
	// 其余的23+位给出方法值的方法编号。
	// 如果flag.kinz() != Func，则代码可以假定flagMethod未设置。
	// 如果是ifaceIndir（typ），则代码可以假定设置了flagIndir。
	flag

	// 方法值代表某些接收者r的curd方法调用（如r.Read）
	// typ + val + flag位描述接收方r，但是标志的Kind位表示Func（方法是函数），并且该标志的高位给出r的类型的方法表中的方法编号。
}

type flag uintptr

const (
	flagKindWidth        = 5                          //有27种 //注：#
	flagKindMask    flag = 1<<flagKindWidth - 1       //注：0001 1111	获取flag的倒数5位	标志位掩码，
	flagStickyRO    flag = 1 << 5                     //注：#
	flagEmbedRO     flag = 1 << 6                     //注：#
	flagIndir       flag = 1 << 7                     //注：#
	flagAddr        flag = 1 << 8                     //注：#
	flagMethod      flag = 1 << 9                     //注：#
	flagMethodShift      = 10                         //注：#
	flagRO          flag = flagStickyRO | flagEmbedRO //注：0110 0000	获取flag的倒数第6、7位
)

func (f flag) kind() Kind { //注：#
	return Kind(f & flagKindMask)
}

func (f flag) ro() flag {
	if f&flagRO != 0 { //注：如果flag的倒数第6、7位不为0，则返回32
		return flagStickyRO
	}
	return 0 //注：否则返回0
}

// pointer 返回由v表示的基础指针。
// v.Kind()必须是Ptr，Map，Chan，Func或UnsafePointer
func (v Value) pointer() unsafe.Pointer {
	if v.typ.size != ptrSize || !v.typ.pointers() {
		panic("can't call pointer on a non-pointer Value") //注：不能在非指针值上调用指针
	}
	if v.flag&flagIndir != 0 {
		return *(*unsafe.Pointer)(v.ptr)
	}
	return v.ptr
}

// packEface converts v to the empty interface.
func packEface(v Value) interface{} {
	t := v.typ
	var i interface{}
	e := (*emptyInterface)(unsafe.Pointer(&i))
	// First, fill in the data portion of the interface.
	switch {
	case ifaceIndir(t):
		if v.flag&flagIndir == 0 {
			panic("bad indir")
		}
		// Value is indirect, and so is the interface we're making.
		ptr := v.ptr
		if v.flag&flagAddr != 0 {
			// TODO: pass safe boolean from valueInterface so
			// we don't need to copy if safe==true?
			c := unsafe_New(t)
			typedmemmove(t, c, ptr)
			ptr = c
		}
		e.word = ptr
	case v.flag&flagIndir != 0:
		// Value is indirect, but interface is direct. We need
		// to load the data at v.ptr into the interface data word.
		e.word = *(*unsafe.Pointer)(v.ptr)
	default:
		// Value is direct, and so is the interface.
		e.word = v.ptr
	}
	// Now, fill in the type portion. We're very careful here not
	// to have any operation between the e.word and e.typ assignments
	// that would let the garbage collector observe the partially-built
	// interface value.
	e.typ = t
	return i
}

// unpackEface 将空接口i转换为Value。
func unpackEface(i interface{}) Value {
	e := (*emptyInterface)(unsafe.Pointer(&i))
	//注意：在我们知道e.word是否真的是指针之前，不要读它。
	t := e.typ
	if t == nil {
		return Value{}
	}
	f := flag(t.Kind())
	if ifaceIndir(t) {
		f |= flagIndir
	}
	return Value{t, e.word, f}
}

// A ValueError occurs when a Value method is invoked on
// a Value that does not support it. Such cases are documented
// in the description of each method.
type ValueError struct {
	Method string
	Kind   Kind
}

func (e *ValueError) Error() string {
	return "reflect: call of " + e.Method + " on zero Value"
}

// methodName returns the name of the calling method,
// assumed to be two stack frames above.
func methodName() string {
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

// mustBeExported panics if f records that the value was obtained using
// an unexported field.
func (f flag) mustBeExported() {
	if f == 0 {
		panic(&ValueError{methodName(), 0})
	}
	if f&flagRO != 0 {
		panic("reflect: " + methodName() + " using value obtained using unexported field")
	}
}

// mustBeAssignable panics if f records that the value is not assignable,
// which is to say that either it was obtained using an unexported field
// or it is not addressable.
func (f flag) mustBeAssignable() {
	if f == 0 {
		panic(&ValueError{methodName(), Invalid})
	}
	// Assignable if addressable and not read-only.
	if f&flagRO != 0 {
		panic("reflect: " + methodName() + " using value obtained using unexported field")
	}
	if f&flagAddr == 0 {
		panic("reflect: " + methodName() + " using unaddressable value")
	}
}

// CanSet reports whether the value of v can be changed.
// A Value can be changed only if it is addressable and was not
// obtained by the use of unexported struct fields.
// If CanSet returns false, calling Set or any type-specific
// setter (e.g., SetBool, SetInt) will panic.
func (v Value) CanSet() bool {
	return v.flag&(flagAddr|flagRO) == flagAddr
}

// Elem returns the value that the interface v contains
// or that the pointer v points to.
// It panics if v's Kind is not Interface or Ptr.
// It returns the zero Value if v is nil.
func (v Value) Elem() Value {
	k := v.kind()
	switch k {
	case Interface:
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
	case Ptr:
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
	panic(&ValueError{"reflectlite.Value.Elem", v.kind()})
}

func valueInterface(v Value) interface{} {
	if v.flag == 0 {
		panic(&ValueError{"reflectlite.Value.Interface", 0})
	}

	if v.kind() == Interface {
		// Special case: return the element inside the interface.
		// Empty interface has one layout, all interfaces with
		// methods have a second layout.
		if v.numMethod() == 0 {
			return *(*interface{})(v.ptr)
		}
		return *(*interface {
			M()
		})(v.ptr)
	}

	// TODO: pass safe to packEface so we don't need to copy if safe==true?
	return packEface(v)
}

// IsNil reports whether its argument v is nil. The argument must be
// a chan, func, interface, map, pointer, or slice value; if it is
// not, IsNil panics. Note that IsNil is not always equivalent to a
// regular comparison with nil in Go. For example, if v was created
// by calling ValueOf with an uninitialized interface variable i,
// i==nil will be true but v.IsNil will panic as v will be the zero
// Value.
func (v Value) IsNil() bool {
	k := v.kind()
	switch k {
	case Chan, Func, Map, Ptr, UnsafePointer:
		// if v.flag&flagMethod != 0 {
		// 	return false
		// }
		ptr := v.ptr
		if v.flag&flagIndir != 0 {
			ptr = *(*unsafe.Pointer)(ptr)
		}
		return ptr == nil
	case Interface, Slice:
		// Both interface and slice are nil if first word is 0.
		// Both are always bigger than a word; assume flagIndir.
		return *(*unsafe.Pointer)(v.ptr) == nil
	}
	panic(&ValueError{"reflectlite.Value.IsNil", v.kind()})
}

// IsValid reports whether v represents a value.
// It returns false if v is the zero Value.
// If IsValid returns false, all other methods except String panic.
// Most functions and methods never return an invalid Value.
// If one does, its documentation states the conditions explicitly.
func (v Value) IsValid() bool {
	return v.flag != 0
}

// Kind returns v's Kind.
// If v is the zero Value (IsValid returns false), Kind returns Invalid.
func (v Value) Kind() Kind {
	return v.kind()
}

// implemented in runtime:
func chanlen(unsafe.Pointer) int
func maplen(unsafe.Pointer) int

// Len returns v's length.
// It panics if v's Kind is not Array, Chan, Map, Slice, or String.
func (v Value) Len() int {
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
		// Slice is bigger than a word; assume flagIndir.
		return (*sliceHeader)(v.ptr).Len
	case String:
		// String is bigger than a word; assume flagIndir.
		return (*stringHeader)(v.ptr).Len
	}
	panic(&ValueError{"reflect.Value.Len", v.kind()})
}

// NumMethod returns the number of exported methods in the value's method set.
func (v Value) numMethod() int {
	if v.typ == nil {
		panic(&ValueError{"reflectlite.Value.NumMethod", Invalid})
	}
	return v.typ.NumMethod()
}

// Set assigns x to the value v.
// It panics if CanSet returns false.
// As in Go, x's value must be assignable to v's type.
func (v Value) Set(x Value) {
	v.mustBeAssignable()
	x.mustBeExported() // do not let unexported x leak
	var target unsafe.Pointer
	if v.kind() == Interface {
		target = v.ptr
	}
	x = x.assignTo("reflectlite.Set", v.typ, target)
	if x.flag&flagIndir != 0 {
		typedmemmove(v.typ, v.ptr, x.ptr)
	} else {
		*(*unsafe.Pointer)(v.ptr) = x.ptr
	}
}

// Type returns v's type.
func (v Value) Type() Type {
	f := v.flag
	if f == 0 {
		panic(&ValueError{"reflectlite.Value.Type", Invalid})
	}
	// Method values not supported.
	return v.typ
}

// stringHeader 是此包中使用的StringHeader的安全版本。
type stringHeader struct {
	Data unsafe.Pointer
	Len  int
}

// sliceHeader is a safe version of SliceHeader used within this package.
type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

/*
 * constructors
 */

// implemented in package runtime
func unsafe_New(*rtype) unsafe.Pointer

// ValueOf 返回一个新的Value，初始化为存储在接口i中的具体值。 ValueOf(nil)返回零值。
func ValueOf(i interface{}) Value {
	if i == nil {
		return Value{}
	}
	// TODO：也许允许Value的内容存在于堆栈中。
	// 现在，我们使内容始终转储到堆中。
	// 它使某些地方的生活更加轻松（请参阅下面的chanrecv/mapassign评论）。
	escapes(i)

	return unpackEface(i)
}

// assignTo returns a value v that can be assigned directly to typ.
// It panics if v is not assignable to typ.
// For a conversion to an interface type, target is a suggested scratch space to use.
func (v Value) assignTo(context string, dst *rtype, target unsafe.Pointer) Value {
	// if v.flag&flagMethod != 0 {
	// 	v = makeMethodValue(context, v)
	// }

	switch {
	case directlyAssignable(dst, v.typ):
		// Overwrite type so that they match.
		// Same memory layout, so no harm done.
		fl := v.flag&(flagAddr|flagIndir) | v.flag.ro()
		fl |= flag(dst.Kind())
		return Value{dst, v.ptr, fl}

	case implements(dst, v.typ):
		if target == nil {
			target = unsafe_New(dst)
		}
		if v.Kind() == Interface && v.IsNil() {
			// A nil ReadWriter passed to nil Reader is OK,
			// but using ifaceE2I below will panic.
			// Avoid the panic by returning a nil dst (e.g., Reader) explicitly.
			return Value{dst, nil, flag(Interface)}
		}
		x := valueInterface(v)
		if dst.NumMethod() == 0 {
			*(*interface{})(target) = x
		} else {
			ifaceE2I(dst, x, target)
		}
		return Value{dst, target, flagIndir | flag(Interface)}
	}

	// Failed.
	panic(context + ": value of type " + v.typ.String() + " is not assignable to type " + dst.String())
}

// arrayAt returns the i-th element of p,
// an array whose elements are eltSize bytes wide.
// The array pointed at by p must have at least i+1 elements:
// it is invalid (but impossible to check here) to pass i >= len,
// because then the result will point outside the array.
// whySafe must explain why i < len. (Passing "i < len" is fine;
// the benefit is to surface this assumption at the call site.)
func arrayAt(p unsafe.Pointer, i int, eltSize uintptr, whySafe string) unsafe.Pointer {
	return add(p, uintptr(i)*eltSize, "i < len")
}

func ifaceE2I(t *rtype, src interface{}, dst unsafe.Pointer)

// typedmemmove copies a value of type t to dst from src.
//go:noescape
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
