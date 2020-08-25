// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package reflect

import (
	"math"
	"runtime"
	"unsafe"
)

const ptrSize = 4 << (^uintptr(0) >> 63) // unsafe.Sizeof(uintptr(0)) 但是一个理想的常量，注：8

// Value 是Go值的反射接口。
//
// 并非所有方法都适用于所有类型的值。 在每种方法的文档中都注明了限制（如果有）。
// 在调用特定于种类的方法之前，使用Kind方法找出值的种类。 调用不适合该类型的方法会导致运行时恐慌。
//
// 零值表示无值。
// 它的IsValid方法返回false，其Kind方法返回Invalid，其String方法返回"<invalid Value>"，所有其他方法均会出现紧急情况。
// 大多数函数和方法从不返回无效值。
// 如果是，则其文档会明确说明条件。
//
// 一个值可以由多个goroutine并发使用，前提是可以将基础Go值同时用于等效的直接操作。
//
// 要比较两个值，请比较Interface方法的结果。
// 在两个值上使用==不会比较它们表示的基础值。
type Value struct {
	// typ保留由值表示的值的类型。
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
	// -flagMethod：v是方法值。
	// 接下来的五位给出值的种类。
	// 重复typ.Kind()，方法值除外。
	// 其余的23+位给出方法值的方法编号。
	// 如果flag.kind() = Func，则代码可以假定flagMethod未设置。
	// 如果是ifaceIndir(typ)，则代码可以假定设置了flagIndir。
	// 注：
	// flag[len(flag) - 11 : len(flag)]：已导出方法集合的最大索引
	// flag[len(flag) - 10]：是否有已导出方法
	// flag[len(flag) - 9]：是否可以寻址（是否分配内存）
	// flag[len(flag) - 8]：是否有指向数据的指针
	// flag[len(flag) - 7]：是否通过未导出的嵌入式字段获得的
	// flag[len(flag) - 6]：是否通过未导出的未嵌入字段获得的
	// flag[len(flag) - 1 : len(flag) - 5]：类型的枚举
	flag

	// 方法值表示类似于r。的接收方r的curried方法调用。
	// typ + val + flag位描述接收方r，但是标志的Kind位表示Func（方法是函数），
	// 并且该标志的高位给出r的类型的方法表中的方法编号。
}

type flag uintptr // 注：记录Value的标志
const (
	flagKindWidth        = 5                          // 注： 27种类型，类型掩码占5位
	flagKindMask    flag = 1<<flagKindWidth - 1       // 注：0000 0000 0001 1111，获取倒数5位，获取数据类型
	flagStickyRO    flag = 1 << 5                     // 注：0000 0000 0010 0000，获取倒数第6位，是否通过未导出的非嵌入字段获取
	flagEmbedRO     flag = 1 << 6                     // 注：0000 0000 0100 0000，获取倒数第7位，是否通过未导出的嵌入式字段获取
	flagIndir       flag = 1 << 7                     // 注：0000 0000 1000 0000，获取倒数第8位，是否有指向数据的指针，如果v.CanAddr为true，Indir为false，会引发恐慌
	flagAddr        flag = 1 << 8                     // 注：0000 0001 0000 0000，获取倒数第9位，是否需要寻址（此Value是一个指针，需要寻址）
	flagMethod      flag = 1 << 9                     // 注：0000 0010 0000 0000，获取倒数第10位，是否有已导出方法
	flagMethodShift      = 10                         // 注：1111 1100 0000 0000，获取倒数第11位至第0位，已导出方法集合的最大索引
	flagRO          flag = flagStickyRO | flagEmbedRO // 注：0000 0000 0110 0000，获取倒数第6、7位，是否通过未导出的字段获取
)

func (f flag) kind() Kind { //注：返回f的类型枚举
	return Kind(f & flagKindMask)
}

func (f flag) ro() flag { //注：返回f是否为通过未导出的非嵌入字段获取的
	if f&flagRO != 0 { //注：如果f是通过未导出的字段获取
		return flagStickyRO //注：返回是否通过未导出的未嵌入字段的掩码
	}
	return 0
}

// pointer 返回由v表示的基础指针。
// v.Kind()必须是Ptr，Map，Chan，Func或UnsafePointer
func (v Value) pointer() unsafe.Pointer { // 注：返回v指向数据的指针
	if v.typ.size != ptrSize || !v.typ.pointers() { //注：v不是指针类型
		panic("can't call pointer on a non-pointer Value") //恐慌："不能在非指针Value上调用指针"
	}
	if v.flag&flagIndir != 0 { //注：v是间接指针
		return *(*unsafe.Pointer)(v.ptr) //注：返回v指向数据的指针
	}
	return v.ptr //注：如果不是间接指针，直接返回指针
}

// packEface 将v转换为空接口。
func packEface(v Value) interface{} { //注：将v包装为空接口
	t := v.typ
	var i interface{}
	e := (*emptyInterface)(unsafe.Pointer(&i))
	// 首先，填写接口的数据部分。
	switch {
	case ifaceIndir(t): // 注：t是间接类型
		if v.flag&flagIndir == 0 { // 注：v没有指针
			panic("bad indir") // 恐慌："错误的指针"
		}
		// Value是间接的，我们创建的接口也是间接的。
		ptr := v.ptr
		if v.flag&flagAddr != 0 { // 注：v分配了内存地址
			// TODO：从valueInterface传递安全布尔值，因此如果safe == true，我们不需要复制？
			c := unsafe_New(t)      // 注：创建一个与v相同类型的变量
			typedmemmove(t, c, ptr) // 注：将类型为v.typ的v的指针赋值给c
			ptr = c
		}
		e.word = ptr //注：空接口的值为指针
	case v.flag&flagIndir != 0: //注：值是间接的，但接口是直接的
		// 值是间接的，但接口是直接的。 我们需要将v.ptr处的数据加载到接口数据字中。
		e.word = *(*unsafe.Pointer)(v.ptr)
	default:
		// 值是直接的，接口也是直接的。
		e.word = v.ptr
	}

	//现在，填写类型部分。 在这里，我们非常小心，不要在e.word和e.typ分配之间进行任何操作，以免垃圾回收器观察部分构建的接口值。
	e.typ = t
	return i
}

// unpackEface 将空接口i转换为Value。
func unpackEface(i interface{}) Value { //注：将空接口i解包为Value
	e := (*emptyInterface)(unsafe.Pointer(&i))
	//注意：在我们知道e.word是否真的是指针之前，不要读它。
	t := e.typ
	if t == nil {
		return Value{}
	}
	f := flag(t.Kind())
	if ifaceIndir(t) { //注：t是否是间接指针
		f |= flagIndir
	}
	return Value{t, e.word, f}
}

// 在不支持Value的Value方法上调用Value方法时，发生ValueError。
// 在每种方法的说明中都记录了这种情况。
type ValueError struct { // 注：调用Value方法时返回的错误
	Method string // 注：发生错误的函数
	Kind   Kind   // 注：发生错误的类型
}

func (e *ValueError) Error() string { // 注：返回错误
	if e.Kind == 0 {
		return "reflect: call of " + e.Method + " on zero Value"
	}
	return "reflect: call of " + e.Method + " on " + e.Kind.String() + " Value"
}

// methodName 返回调用方法的名称，假定位于上面的两个堆栈帧中。
func methodName() string { // 注：#
	pc, _, _, _ := runtime.Caller(2)
	f := runtime.FuncForPC(pc)
	if f == nil {
		return "unknown method"
	}
	return f.Name()
}

// emptyInterface 是interface{}值的标头.
type emptyInterface struct { // 注：空接口的反射类型
	typ  *rtype
	word unsafe.Pointer // 注：指向数据的指针
}

// nonEmptyInterface 是带有方法的接口值的标头。
type nonEmptyInterface struct { // 注：#
	// see ../runtime/iface.go:/Itab
	itab *struct {
		ityp *rtype // 静态接口类型
		typ  *rtype // 动态具体类型
		hash uint32 // 从typ.hash拷贝
		_    [4]byte
		fun  [100000]unsafe.Pointer // 方法表
	}
	word unsafe.Pointer
}

// mustBe 如果不期望f的种类，则必须惊慌。
// 将此方法设置为基于标志而不是基于Value的方法（并在Value中嵌入标志）意味着我们可以编写非常清晰的v.mustBe(Bool)
// 并将其编译为v.flag.mustBe(Bool)，这只会为接收者复制一个重要的单词。
func (f flag) mustBe(expected Kind) { // 注：f必须为expected类型
	// TODO(mvdan)：中间堆栈内联变得更好时，再次使用f.kind()
	if Kind(f&flagKindMask) != expected {
		panic(&ValueError{methodName(), f.kind()})
	}
}

// mustBeExported 如果f记录使用一个未导出的字段获取该值，则感到恐慌。
func (f flag) mustBeExported() { // 注：f必须为已导出的字段
	if f == 0 || f&flagRO != 0 {
		f.mustBeExportedSlow()
	}
}

func (f flag) mustBeExportedSlow() { // 产生恐慌："使用未导出字段获得的值"
	if f == 0 {
		panic(&ValueError{methodName(), Invalid})
	}
	if f&flagRO != 0 {
		panic("reflect: " + methodName() + " using value obtained using unexported field")
	}
}

// mustBeAssignable 如果f记录了该值是不可分配的，则表示感到恐慌，这意味着它是使用未导出的字段获得的，或者它是不可寻址的。
func (f flag) mustBeAssignable() { // 注：f必须为已分配地址的已导出字段
	if f&flagRO != 0 || f&flagAddr == 0 { //注：必须为已导出的字段且是可寻址的
		f.mustBeAssignableSlow() //注：产生恐慌
	}
}

func (f flag) mustBeAssignableSlow() { // 注：产生恐慌，对象已导出的字段且是可寻址的
	if f == 0 {
		panic(&ValueError{methodName(), Invalid})
	}
	// 如果可寻址且不是只读，则可分配。
	if f&flagRO != 0 {
		panic("reflect: " + methodName() + " using value obtained using unexported field") //恐慌："使用通过未导出字段获得的值"
	}
	if f&flagAddr == 0 {
		panic("reflect: " + methodName() + " using unaddressable value") //恐慌："使用不可寻址的值"
	}
}

// Addr 返回表示v地址的指针值。
// 如果CanAddr()返回false，则会感到恐慌。
// Addr通常用于获取指向struct字段或slice元素的指针，以便调用需要指针接收器的方法。
func (v Value) Addr() Value { // 注：#
	if v.flag&flagAddr == 0 { // 注：如果v无法寻址，引发恐慌
		panic("reflect.Value.Addr of unaddressable value") // 恐慌："无法寻址值的reflect.Value.Addr"
	}
	return Value{v.typ.ptrTo(), v.ptr, v.flag.ro() | flag(Ptr)}
}

// Bool 返回v的基础值。
// 如果v的种类不是Bool，则会感到恐慌。
func (v Value) Bool() bool { // 注：返回v的底层的值
	v.mustBe(Bool)
	return *(*bool)(v.ptr)
}

// Bytes 返回v的基础值。
// 如果v的基础值不是一个字节片，则会感到恐慌。
func (v Value) Bytes() []byte { // 注：返回v的底层的值
	v.mustBe(Slice) // 注：v必须是切片类型
	if v.typ.Elem().Kind() != Uint8 {
		panic("reflect.Value.Bytes of non-byte slice") // 恐慌："reflect.Value.Bytes的非字节切片"
	}
	// slice总是大于一个词； 假设flagIndir。
	return *(*[]byte)(v.ptr)
}

// runes 返回v的基础值。
// 如果v的基础值不是rune切片（int32s），则会感到恐慌。
func (v Value) runes() []rune { // 注：返回v的底层的值
	v.mustBe(Slice) // 注：v必须是切片类型
	if v.typ.Elem().Kind() != Int32 {
		panic("reflect.Value.Bytes of non-rune slice") // 恐慌："非rune切片的reflect.Value.Bytes"
	}
	// 切片总是比单词大； 假设flagIndir。
	return *(*[]rune)(v.ptr)
}

// CanAddr 报告是否可以通过Addr获取值的地址。
// 这样的值称为可寻址的。
// 如果值是切片的元素，可寻址数组的元素，则该值是可寻址的，
// 可寻址结构的字段，或取消引用指针的结果。
// 如果CanAddr返回false，调用Addr会产生恐慌。
func (v Value) CanAddr() bool { //注：v是否可寻址
	return v.flag&flagAddr != 0
}

// CanSet 报告v的值是否可以更改。
// 仅当值是可寻址的并且不是通过使用未导出的结构字段获得的，才可以更改它。
// 如果CanSet返回false，则调用Set或任何特定于类型的setter（例如SetBool，SetInt）都会感到恐慌。
func (v Value) CanSet() bool { // 注：返回值是否可以修改
	return v.flag&(flagAddr|flagRO) == flagAddr // 注：v是可寻址的且是已导出的字段
}

// Call调用带有输入参数的函数v。
// 例如，如果len(in) == 3, 则v.Call(in)表示Go调用v(in[0], in[1], in[2]).
// 如果v的Kind不是Func，则呼叫恐慌。
// 将输出结果作为值返回。
// 和Go一样，每个输入参数必须可分配给函数相应输入参数的类型。
// 如果v是可变参数函数，则Call会自己创建可变参数切片参数，并复制相应的值。
func (v Value) Call(in []Value) []Value { // 注：#
	v.mustBe(Func)     // 注：必须为方法
	v.mustBeExported() // 注：必须是已导出的方法
	return v.call("Call", in)
}

// CallSlice 调用带有输入参数in的可变参数v，将切片in[len(in)-1]配给v的最终可变参数。
// 例如，如果 len(in) == 3, 则v.CallSlice(in)表示Go调用v(in[0], in[1], in[2]...).
// 如果v的Kind不是Func或v不是可变参数，CallSlice会感到恐慌。
// 将输出结果作为值返回。
// 和Go一样，每个输入参数必须可分配给函数相应输入参数的类型。
func (v Value) CallSlice(in []Value) []Value { // 注：#
	v.mustBe(Func)     // 注：必须为方法
	v.mustBeExported() // 注：必须是已导出的方法
	return v.call("CallSlice", in)
}

var callGC bool // 用于测试; 参见TestCallMethodJump

func (v Value) call(op string, in []Value) []Value { // 注：#
	// 获取函数指针，键入。
	t := (*funcType)(unsafe.Pointer(v.typ)) // 注：v转为funcType
	var (
		fn       unsafe.Pointer
		rcvr     Value
		rcvrtype *rtype
	)
	if v.flag&flagMethod != 0 { // 注：v有已导出的方法
		rcvr = v
		rcvrtype, t, fn = methodReceiver(op, v, int(v.flag)>>flagMethodShift) // 注：获取v的方法的接收器信息
	} else if v.flag&flagIndir != 0 { // 注：如果v是间接指针，获取指针指向的数据
		fn = *(*unsafe.Pointer)(v.ptr)
	} else {
		fn = v.ptr
	}

	if fn == nil {
		panic("reflect.Value.Call: call of nil function") // 恐慌："调用nil函数"
	}

	isSlice := op == "CallSlice"
	n := t.NumIn() // 注：获取方法的形参数量
	if isSlice {
		if !t.IsVariadic() { // 注：最后一个形参不是...
			panic("reflect: CallSlice of non-variadic function") // 恐慌："非可变函数的CallSlice"
		}
		if len(in) < n {
			panic("reflect: CallSlice with too few input arguments") // 恐慌："CallSlice的输入参数太少"
		}
		if len(in) > n {
			panic("reflect: CallSlice with too many input arguments") // 恐慌："CallSlice的输入参数过多"
		}
	} else {
		if t.IsVariadic() { // 注：如果最后一个形参是...，则忽略
			n--
		}
		if len(in) < n {
			panic("reflect: Call with too few input arguments") // 恐慌："调用时输入参数太少"
		}
		if !t.IsVariadic() && len(in) > n {
			panic("reflect: Call with too many input arguments") // 恐慌："调用时输入参数过多"
		}
	}
	for _, x := range in { // 注：遍历形参
		if x.Kind() == Invalid { // 注：如果形参类型无效
			panic("reflect: " + op + " using zero Value argument") // 恐慌："使用零值参数"
		}
	}
	for i := 0; i < n; i++ { // 注：遍历形参
		if xt, targ := in[i].Type(), t.In(i); !xt.AssignableTo(targ) { // 注：获取第i个实参与第i个形参的类型，如果实参无法分配给形参，引发恐慌
			panic("reflect: " + op + " using " + xt.String() + " as type " + targ.String()) // 恐慌："类型错误"
		}
	}
	if !isSlice && t.IsVariadic() { // 注：如果不是切片类型并且最后一个形参是...
		// 为剩余值准备切片
		m := len(in) - n
		slice := MakeSlice(t.In(n), m, m) // 注：创建一个切片，用于存放变长形参
		elem := t.In(n).Elem()            // 注：获取变长形参的数据类型
		for i := 0; i < m; i++ {
			x := in[n+i]                                // 注：获取第i个变长实参
			if xt := x.Type(); !xt.AssignableTo(elem) { // 注：如果变长实参无法分配给变长形参，引发恐慌
				panic("reflect: cannot use " + xt.String() + " as type " + elem.String() + " in " + op) // 恐慌："类型错误"
			}
			slice.Index(i).Set(x) // 注：切片的第i个元素赋值为第i个实参
		}
		origIn := in
		in = make([]Value, n+1)
		copy(in[:n], origIn)
		in[n] = slice // 注：将变长实参转为切片，作为实参的最后一个元素，例：形参为：(a int, b int, c ...int)，当实参为：1, 2, 3, 4, 5时，in = 1, 2, []int{3, 4, 5}
	}

	nin := len(in)
	if nin != t.NumIn() { // 注：如果实参的数量不等于形参的数量，引发恐慌
		panic("reflect.Value.Call: wrong argument count") // 恐慌："错误的参数数"
	}
	nout := t.NumOut() // 注：获取输出参数的数量

	// 计算帧类型。
	frametype, _, retOffset, _, framePool := funcLayout(t, rcvrtype) // 注：#

	// 为帧分配一块内存。
	var args unsafe.Pointer
	if nout == 0 {
		args = framePool.Get().(unsafe.Pointer)
	} else {
		// Can't use pool if the function has return values.
		// We will leak pointer to args in ret, so its lifetime is not scoped.
		args = unsafe_New(frametype)
	}
	off := uintptr(0)

	// Copy inputs into args.
	if rcvrtype != nil {
		storeRcvr(rcvr, args)
		off = ptrSize
	}
	for i, v := range in {
		v.mustBeExported()
		targ := t.In(i).(*rtype)
		a := uintptr(targ.align)
		off = (off + a - 1) &^ (a - 1)
		n := targ.size
		if n == 0 {
			// Not safe to compute args+off pointing at 0 bytes,
			// because that might point beyond the end of the frame,
			// but we still need to call assignTo to check assignability.
			v.assignTo("reflect.Value.Call", targ, nil)
			continue
		}
		addr := add(args, off, "n > 0")
		v = v.assignTo("reflect.Value.Call", targ, addr)
		if v.flag&flagIndir != 0 {
			typedmemmove(targ, addr, v.ptr)
		} else {
			*(*unsafe.Pointer)(addr) = v.ptr
		}
		off += n
	}

	// Call.
	call(frametype, fn, args, uint32(frametype.size), uint32(retOffset))

	// For testing; see TestCallMethodJump.
	if callGC {
		runtime.GC()
	}

	var ret []Value
	if nout == 0 {
		typedmemclr(frametype, args)
		framePool.Put(args)
	} else {
		// Zero the now unused input area of args,
		// because the Values returned by this function contain pointers to the args object,
		// and will thus keep the args object alive indefinitely.
		typedmemclrpartial(frametype, args, 0, retOffset)

		// Wrap Values around return values in args.
		ret = make([]Value, nout)
		off = retOffset
		for i := 0; i < nout; i++ {
			tv := t.Out(i)
			a := uintptr(tv.Align())
			off = (off + a - 1) &^ (a - 1)
			if tv.Size() != 0 {
				fl := flagIndir | flag(tv.Kind())
				ret[i] = Value{tv.common(), add(args, off, "tv.Size() != 0"), fl}
				// Note: this does introduce false sharing between results -
				// if any result is live, they are all live.
				// (And the space for the args is live as well, but as we've
				// cleared that space it isn't as big a deal.)
			} else {
				// For zero-sized return value, args+off may point to the next object.
				// In this case, return the zero value instead.
				ret[i] = Zero(tv)
			}
			off += tv.Size()
		}
	}

	return ret
}

// callReflect 是MakeFunc返回的函数使用的调用实现。
// 在许多方面，它与上面的Value.call方法相反。
// 上面的方法将使用Values的调用转换为带有具体参数框架的函数的调用，而callReflect将具有具体参数框架的函数调用转换为使用Values的调用。
// 它在此文件中，因此可以位于上面的call方法旁边。
// MakeFunc实现的其余部分位于makefunc.go中。
//
// 注意：此函数必须在生成的代码中标记为“包装器”，以便链接器可以使其正常工作，以免发生恐慌和恢复。
// gc编译器知道这样做的名称是“ reflect.callReflect”。
//
// ctxt是MakeFunc生成的“关闭”。
// frame是指向堆栈上该闭包的参数的指针。
// retValid指向一个布尔值，当设置框架的结果部分时应设置此布尔值。
func callReflect(ctxt *makeFuncImpl, frame unsafe.Pointer, retValid *bool) { // 注：#
	ftyp := ctxt.ftyp
	f := ctxt.fn

	// Copy argument frame into Values.
	ptr := frame
	off := uintptr(0)
	in := make([]Value, 0, int(ftyp.inCount))
	for _, typ := range ftyp.in() {
		off += -off & uintptr(typ.align-1)
		v := Value{typ, nil, flag(typ.Kind())}
		if ifaceIndir(typ) {
			// value cannot be inlined in interface data.
			// Must make a copy, because f might keep a reference to it,
			// and we cannot let f keep a reference to the stack frame
			// after this function returns, not even a read-only reference.
			v.ptr = unsafe_New(typ)
			if typ.size > 0 {
				typedmemmove(typ, v.ptr, add(ptr, off, "typ.size > 0"))
			}
			v.flag |= flagIndir
		} else {
			v.ptr = *(*unsafe.Pointer)(add(ptr, off, "1-ptr"))
		}
		in = append(in, v)
		off += typ.size
	}

	// Call underlying function.
	out := f(in)
	numOut := ftyp.NumOut()
	if len(out) != numOut {
		panic("reflect: wrong return count from function created by MakeFunc")
	}

	// Copy results back into argument frame.
	if numOut > 0 {
		off += -off & (ptrSize - 1)
		for i, typ := range ftyp.out() {
			v := out[i]
			if v.typ == nil {
				panic("reflect: function created by MakeFunc using " + funcName(f) +
					" returned zero Value")
			}
			if v.flag&flagRO != 0 {
				panic("reflect: function created by MakeFunc using " + funcName(f) +
					" returned value obtained from unexported field")
			}
			off += -off & uintptr(typ.align-1)
			if typ.size == 0 {
				continue
			}
			addr := add(ptr, off, "typ.size > 0")

			// Convert v to type typ if v is assignable to a variable
			// of type t in the language spec.
			// See issue 28761.
			if typ.Kind() == Interface {
				// We must clear the destination before calling assignTo,
				// in case assignTo writes (with memory barriers) to the
				// target location used as scratch space. See issue 39541.
				*(*uintptr)(addr) = 0
				*(*uintptr)(add(addr, ptrSize, "typ.size == 2*ptrSize")) = 0
			}
			v = v.assignTo("reflect.MakeFunc", typ, addr)

			// We are writing to stack. No write barrier.
			if v.flag&flagIndir != 0 {
				memmove(addr, v.ptr, typ.size)
			} else {
				*(*uintptr)(addr) = uintptr(v.ptr)
			}
			off += typ.size
		}
	}

	// Announce that the return values are valid.
	// After this point the runtime can depend on the return values being valid.
	*retValid = true

	// We have to make sure that the out slice lives at least until
	// the runtime knows the return values are valid. Otherwise, the
	// return values might not be scanned by anyone during a GC.
	// (out would be dead, and the return slots not yet alive.)
	runtime.KeepAlive(out)

	// runtime.getArgInfo expects to be able to find ctxt on the
	// stack when it finds our caller, makeFuncStub. Make sure it
	// doesn't get garbage collected.
	runtime.KeepAlive(ctxt)
}

// methodReceiver 返回有关v描述的接收者的信息。Value v可能会或可能不会设置flagMethod位，因此不应使用v.flag中缓存的种类。
// 返回值rcvrtype给出了方法的实际接收者类型。
// 返回值t给出方法类型签名（没有接收者）。
// 返回值fn是方法代码的指针。
func methodReceiver(op string, v Value, methodIndex int) (rcvrtype *rtype, t *funcType, fn unsafe.Pointer) { // 注：获取v的第methodIndex个方法的接收器信息，返回接收器类型rcvrtype，参数类型t与方法指针fn
	i := methodIndex
	if v.typ.Kind() == Interface { // 注：如果v是接口类型
		tt := (*interfaceType)(unsafe.Pointer(v.typ)) // 注：v转为接口类型
		if uint(i) >= uint(len(tt.methods)) {         // 注：如果索引超出方法的数量，引发恐慌
			panic("reflect: internal error: invalid method index") // 恐慌："内部错误：方法索引无效"
		}
		m := &tt.methods[i]
		if !tt.nameOff(m.name).isExported() { // 注：如果v的第methodIndex个方法是未导出方法，引发恐慌
			panic("reflect: " + op + " of unexported method") // 恐慌："未导出的方法"
		}
		iface := (*nonEmptyInterface)(v.ptr)
		if iface.itab == nil { // 注：#
			panic("reflect: " + op + " of method on nil interface value") // 恐慌："nil接口值的方法"
		}
		rcvrtype = iface.itab.typ
		fn = unsafe.Pointer(&iface.itab.fun[i])
		t = (*funcType)(unsafe.Pointer(tt.typeOff(m.typ)))
	} else {
		rcvrtype = v.typ
		ms := v.typ.exportedMethods() // 注：获取v的已导出方法
		if uint(i) >= uint(len(ms)) { // 注：如果索引超出方法的数量，引发恐慌
			panic("reflect: internal error: invalid method index") // 恐慌："内部错误：方法索引无效"
		}
		m := ms[i]
		if !v.typ.nameOff(m.name).isExported() { // 注：如果v的第methodIndex个方法是未导出方法，引发恐慌
			panic("reflect: " + op + " of unexported method") // 恐慌："未导出的方法"
		}
		ifn := v.typ.textOff(m.ifn)
		fn = unsafe.Pointer(&ifn)
		t = (*funcType)(unsafe.Pointer(v.typ.typeOff(m.mtyp)))
	}
	return
}

// v是方法的接收者。 在参数列表的开头将用于对接收方进行编码的单词存储在p处。
// Reflect使用方法的“接口”调用约定，该约定始终使用一个单词来记录接收者。
func storeRcvr(v Value, p unsafe.Pointer) { // 注：#
	t := v.typ
	if t.Kind() == Interface {
		// 接口数据字成为接收方字
		iface := (*nonEmptyInterface)(v.ptr)
		*(*unsafe.Pointer)(p) = iface.word
	} else if v.flag&flagIndir != 0 && !ifaceIndir(t) {
		*(*unsafe.Pointer)(p) = *(*unsafe.Pointer)(v.ptr)
	} else {
		*(*unsafe.Pointer)(p) = v.ptr
	}
}

// align 返回将x舍入为n的倍数的结果。
// n必须是2的幂。
func align(x, n uintptr) uintptr { // 注：返回x按n位对齐后的长度
	// 例1：x = 5，n = 8
	// x + n - 1 = 12（1100），n - 1 = 7（0111），结果为8（1000）
	// 例1：x = 20，n = 8
	// x + n - 1 = 27（0001 1011），n - 1 = 7（0111），结果为24（11000）

	return (x + n - 1) &^ (n - 1) // 注：将n位的值全部置0，保留n位以上的值
}

// callMethod是 返回的函数使用的调用实现
// 通过makeMethodValue（由v.Method(i).Interface()使用）。
// 这是常规反射调用的简化版本：调用者已经为我们布置了参数框架，因此我们不必为每个参数处理单独的Values。
// 它在此文件中，因此可以位于上面的两个类似函数的旁边。
// makeMethodValue实现的其余部分位于makefunc.go中。
//
// 注意：此函数必须在生成的代码中标记为“包装器”，以便链接器可以使其正常工作，以免发生恐慌和恢复。
// gc编译器知道这样做的名称是“ reflect.callMethod”。
//
// ctxt是makeVethodValue生成的“关闭”。
// frame是指向堆栈上该闭包的参数的指针。
// retValid指向一个布尔值，当设置框架的结果部分时应设置此布尔值。
func callMethod(ctxt *methodValue, frame unsafe.Pointer, retValid *bool) { // 注：#
	rcvr := ctxt.rcvr
	rcvrtype, t, fn := methodReceiver("call", rcvr, ctxt.method)
	frametype, argSize, retOffset, _, framePool := funcLayout(t, rcvrtype)

	// Make a new frame that is one word bigger so we can store the receiver.
	// This space is used for both arguments and return values.
	scratch := framePool.Get().(unsafe.Pointer)

	// Copy in receiver and rest of args.
	storeRcvr(rcvr, scratch)
	// Align the first arg. The alignment can't be larger than ptrSize.
	argOffset := uintptr(ptrSize)
	if len(t.in()) > 0 {
		argOffset = align(argOffset, uintptr(t.in()[0].align))
	}
	// Avoid constructing out-of-bounds pointers if there are no args.
	if argSize-argOffset > 0 {
		typedmemmovepartial(frametype, add(scratch, argOffset, "argSize > argOffset"), frame, argOffset, argSize-argOffset)
	}

	// Call.
	// Call copies the arguments from scratch to the stack, calls fn,
	// and then copies the results back into scratch.
	call(frametype, fn, scratch, uint32(frametype.size), uint32(retOffset))

	// Copy return values.
	// Ignore any changes to args and just copy return values.
	// Avoid constructing out-of-bounds pointers if there are no return values.
	if frametype.size-retOffset > 0 {
		callerRetOffset := retOffset - argOffset
		// This copies to the stack. Write barriers are not needed.
		memmove(add(frame, callerRetOffset, "frametype.size > retOffset"),
			add(scratch, retOffset, "frametype.size > retOffset"),
			frametype.size-retOffset)
	}

	// Tell the runtime it can now depend on the return values
	// being properly initialized.
	*retValid = true

	// Clear the scratch space and put it back in the pool.
	// This must happen after the statement above, so that the return
	// values will always be scanned by someone.
	typedmemclr(frametype, scratch)
	framePool.Put(scratch)

	// See the comment in callReflect.
	runtime.KeepAlive(ctxt)
}

// funcName 返回f的名称，用于错误消息。
func funcName(f func([]Value) []Value) string { // 注：#
	pc := *(*uintptr)(unsafe.Pointer(&f))
	rf := runtime.FuncForPC(pc)
	if rf != nil {
		return rf.Name()
	}
	return "closure"
}

// Cap 返回v的容量。
// 如果v的Kind不是Array，Chan或Slice，它会感到恐慌。
func (v Value) Cap() int { // 注：返回v的容量（数组、管道、切片）
	k := v.kind()
	switch k {
	case Array:
		return v.typ.Len()
	case Chan:
		return chancap(v.pointer())
	case Slice:
		// 切片总是比单词大； 假设flagIndir。
		return (*sliceHeader)(v.ptr).Cap
	}
	panic(&ValueError{"reflect.Value.Cap", v.kind()})
}

// Close 关闭管道v。
// 如果v的Kind不是Chan，就会感到恐慌。
func (v Value) Close() { // 注：关闭管道v
	v.mustBe(Chan)     // 注：必须是管道类型
	v.mustBeExported() // 注：必须是已导出的变量
	chanclose(v.pointer())
}

// Complex 返回v的基础值，为complex128。
// 如果v的Kind不是Complex64或Complex128，会感到恐慌
func (v Value) Complex() complex128 { // 注：返回格式化为complex128类型的v
	k := v.kind()
	switch k {
	case Complex64:
		return complex128(*(*complex64)(v.ptr))
	case Complex128:
		return *(*complex128)(v.ptr)
	}
	panic(&ValueError{"reflect.Value.Complex", v.kind()})
}

// Elem 返回接口v包含的值或指针v指向的值。
// 如果v的Kind不是Interface或Ptr，它会感到恐慌。
// 如果v为nil，则返回零值。
func (v Value) Elem() Value { //注：返回接口v包含的值或指针v指向的值
	k := v.kind()
	switch k {
	case Interface:
		var eface interface{}
		if v.typ.NumMethod() == 0 { //注：如果v的类型的方法数为0
			eface = *(*interface{})(v.ptr) //注：v指向的数据转为空接口
		} else {
			eface = (interface{})(*(*interface { //注：#
				M()
			})(v.ptr))
		}
		x := unpackEface(eface) //注：将空接口v转为Value
		if x.flag != 0 {        //注：不知道为什么没有else
			x.flag |= v.flag.ro() //注：设置是否为通过未导出的非嵌入字段获得
		}
		return x
	case Ptr:
		ptr := v.ptr
		if v.flag&flagIndir != 0 { //注：如果指针是间接的
			ptr = *(*unsafe.Pointer)(ptr) //注：获取数据的指针
		}
		// 返回值的地址是v的值。
		if ptr == nil { //注：如果指针是空的
			return Value{}
		}
		tt := (*ptrType)(unsafe.Pointer(v.typ))
		typ := tt.elem
		fl := v.flag&flagRO | flagIndir | flagAddr
		fl |= flag(typ.Kind())
		return Value{typ, ptr, fl}
	}
	panic(&ValueError{"reflect.Value.Elem", v.kind()})
}

// Field 返回结构v的第i个字段。
// 如果v的Kind不是Struct或i不在范围内，则会发生恐慌。
func (v Value) Field(i int) Value { //注：获取结构体v的第i个字段
	if v.kind() != Struct { //注：v的类型只能是结构体
		panic(&ValueError{"reflect.Value.Field", v.kind()})
	}
	tt := (*structType)(unsafe.Pointer(v.typ))
	if uint(i) >= uint(len(tt.fields)) { //注：索引i超出了v拥有方法的数量
		panic("reflect: Field index out of range") //恐慌："字段索引超界"
	}
	field := &tt.fields[i]
	typ := field.typ

	// 从v继承权限位，但清除flagEmbedRO。
	fl := v.flag&(flagStickyRO|flagIndir|flagAddr) | flag(typ.Kind()) //注：继承v的权限位，取出来的字段即为并非通过未导出的嵌入式字段获取
	// 使用未导出的字段会强制flagRO。
	if !field.name.isExported() { //注：字段并非导出字段
		if field.embedded() { //注：如果字段是嵌入式字段
			fl |= flagEmbedRO //注：字段是未导出的嵌入式字段
		} else {
			fl |= flagStickyRO //注：字段是未导出的非嵌入式字段
		}
	}
	// 或者设置了flagIndir并且v.ptr指向结构，或者没有设置flagIndir且v.ptr是实际的结构数据。
	// 在前一种情况下，我们需要v.ptr + 偏移量。
	// 在后一种情况下，我们必须具有field.offset = 0，因此v.ptr + field.offset仍然是正确的地址。
	ptr := add(v.ptr, field.offset(), "same as non-reflect &v.field") //注：v的指针偏移至字段的位置，与非反射&v.field相同
	return Value{typ, ptr, fl}
}

// FieldByIndex 返回与index对应的嵌套字段。
// 如果v的Kind不是struct，它将感到恐慌。
func (v Value) FieldByIndex(index []int) Value { // 注：返回v对应index的嵌套字段（第i次解包获取第index[i]个字段）
	// 例：index = []int{1, 4, 5}
	// 第1次循环：v1 = v.Field(1)
	// 第2次循环：v2 = v1.Field(4)
	// 第3次循环：v3 = v2.Field(5)

	if len(index) == 1 { // 注：#返回唯一的字段
		return v.Field(index[0])
	}
	v.mustBe(Struct)          // 注：必须为结构体类型
	for i, x := range index { // 注：嵌套解包第i次时，选择结构体的第index[i]个字段
		if i > 0 {
			if v.Kind() == Ptr && v.typ.Elem().Kind() == Struct {
				if v.IsNil() { // 注：如果遍历index没有结束，v == nil会引发恐慌
					panic("reflect: indirection through nil pointer to embedded struct") // 恐慌："通过nil指针间接指向嵌入式结构"
				}
				v = v.Elem()
			}
		}
		v = v.Field(x)
	}
	return v
}

// FieldByName 返回具有给定名称的struct字段。
// 如果未找到任何字段，则返回零值。
// 如果v的Kind不是struct，它将感到恐慌。
func (v Value) FieldByName(name string) Value {
	v.mustBe(Struct) // 注：必须为结构体类型
	if f, ok := v.typ.FieldByName(name); ok {
		return v.FieldByIndex(f.Index)
	}
	return Value{}
}

// FieldByNameFunc返回带有满足match函数名称的struct字段。
// 如果v的Kind不是struct，它将感到恐慌。
// 如果未找到任何字段，则返回零值。
func (v Value) FieldByNameFunc(match func(string) bool) Value { // 注：#
	if f, ok := v.typ.FieldByNameFunc(match); ok {
		return v.FieldByIndex(f.Index)
	}
	return Value{}
}

// Float 返回v的基础值，作为float64。
// 如果v的Kind不是Float32或Float64，则会发生恐慌
func (v Value) Float() float64 { // 注：v转为float
	k := v.kind()
	switch k {
	case Float32:
		return float64(*(*float32)(v.ptr))
	case Float64:
		return *(*float64)(v.ptr)
	}
	panic(&ValueError{"reflect.Value.Float", v.kind()})
}

var uint8Type = TypeOf(uint8(0)).(*rtype)

// Index 返回v的第i个元素。
// 如果v的Kind不是Array，Slice或String或i不在范围之内，它会感到恐慌。
func (v Value) Index(i int) Value { //注：获取数组、切片、字符串v的第i个元素（偏移i*v类型的长度所得）
	switch v.kind() {
	case Array:
		tt := (*arrayType)(unsafe.Pointer(v.typ))
		if uint(i) >= uint(tt.len) {
			panic("reflect: array index out of range") //恐慌："数组索引超出范围"
		}
		typ := tt.elem
		offset := uintptr(i) * typ.size

		//或者设置了flagIndir并且v.ptr指向数组，或者未设置flagIndir且v.ptr是实际的数组数据。
		//在前一种情况下，我们需要v.ptr +偏移量。
		//在后一种情况下，我们必须执行Index（0），所以offset = 0，所以v.ptr + offset仍然是正确的地址。
		val := add(v.ptr, offset, "same as &v[i], i < tt.len")
		fl := v.flag&(flagIndir|flagAddr) | v.flag.ro() | flag(typ.Kind()) // 与整体数组相同
		return Value{typ, val, fl}

	case Slice:
		// 元素标志与Ptr的Elem相同。
		// 可寻址，间接，可能是只读的。
		s := (*sliceHeader)(v.ptr)
		if uint(i) >= uint(s.Len) {
			panic("reflect: slice index out of range") //恐慌："切片索引超出范围"
		}
		tt := (*sliceType)(unsafe.Pointer(v.typ))
		typ := tt.elem
		val := arrayAt(s.Data, i, typ.size, "i < s.Len")
		fl := flagAddr | flagIndir | v.flag.ro() | flag(typ.Kind())
		return Value{typ, val, fl}

	case String:
		s := (*stringHeader)(v.ptr)
		if uint(i) >= uint(s.Len) {
			panic("reflect: string index out of range") //恐慌："字符串索引超出范围"
		}
		p := arrayAt(s.Data, i, 1, "i < s.Len")
		fl := v.flag.ro() | flag(Uint8) | flagIndir
		return Value{uint8Type, p, fl}
	}
	panic(&ValueError{"reflect.Value.Index", v.kind()})
}

// Int 返回v的基础值，作为int64。
// 如果v的Kind不是Int，Int8，Int16，Int32或Int64，则会发生恐慌。
func (v Value) Int() int64 { //注：v转为int64
	k := v.kind()
	p := v.ptr
	switch k {
	case Int:
		return int64(*(*int)(p))
	case Int8:
		return int64(*(*int8)(p))
	case Int16:
		return int64(*(*int16)(p))
	case Int32:
		return int64(*(*int32)(p))
	case Int64:
		return *(*int64)(p)
	}
	panic(&ValueError{"reflect.Value.Int", v.kind()})
}

// CanInterface 报告是否可以在不惊慌的情况下使用Interface。
func (v Value) CanInterface() bool { //注：v是否为已导出的字段
	if v.flag == 0 { //注：如果v不合法
		panic(&ValueError{"reflect.Value.CanInterface", Invalid}) //恐慌："不合法的类型"
	}
	return v.flag&flagRO == 0 //注：v不是根据未导出的字段获取的
}

// Interface 返回v的当前值作为Interface{}。
// 等同于：
// var i interface{} =（v的基础值）
// 如果通过访问未导出的struct字段获得了Value，则会感到恐慌。
func (v Value) Interface() (i interface{}) { //注：将v转为空接口并返回，进行字段导出安全检查
	return valueInterface(v, true)
}

func valueInterface(v Value, safe bool) interface{} { //注：#将v转为空接口，safe控制字段v是否进行安全检查（检查是否为未导出的字段）
	if v.flag == 0 { //注：如果字段不合法
		panic(&ValueError{"reflect.Value.Interface", Invalid}) //引发恐慌
	}
	if safe && v.flag&flagRO != 0 { //注：#v是未导出的字段
		//不允许通过接口访问未导出的值，因为它们可能是不应写的指针或不应被调用的方法或函数。
		panic("reflect.Value.Interface: cannot return value obtained from unexported field or method") //恐慌："无法返回从未导出的字段或方法获得的值"
	}
	if v.flag&flagMethod != 0 { //注：v有方法
		v = makeMethodValue("Interface", v) //注：#
	}

	if v.kind() == Interface { //注：如果v的类型是接口
		// 特例：返回接口内的元素。
		// 空接口具有一种布局，所有带方法的接口具有另一种布局。
		if v.NumMethod() == 0 { //注：如果v没有方法，是空接口
			return *(*interface{})(v.ptr) //注：返回空接口类型的v的指针值
		}
		return *(*interface { //注：#
			M()
		})(v.ptr)
	}

	// TODO：将安全传递给packEface，因此，如果safe == true，我们不需要复制吗？
	return packEface(v) //注：将v包装成空接口返回
}

// InterfaceData 以uintptr对的形式返回接口v的值。
// 如果v的Kind不是Interface，它会感到恐慌。
func (v Value) InterfaceData() [2]uintptr { // 注：将v转为[2]uintptr
	// TODO：不推荐使用
	v.mustBe(Interface) // 注：v必须是接口类型
	// 我们将此视为读操作，因此即使未导出的数据也允许它，因为调用者必须导入"unsafe"才能将其转换为可滥用的内容。
	// 接口值总是大于一个单词； 假设flagIndir。
	return *(*[2]uintptr)(v.ptr)
}

// IsNil 报告其参数v是否为nil。 参数必须是chan，func，interface，map，pointer或slice值；
// 如果不是，则IsNil感到恐慌。 请注意，IsNil并不总是等同于Go中与nil的常规比较。
// 例如，如果v是通过使用未初始化的接口变量i调用ValueOf来创建的，则i == nil将为true，但v.IsNil将感到恐慌，因为v将为零值。
func (v Value) IsNil() bool { //注：v是否未初始化（v指向数据的指针为nil）
	k := v.kind()
	switch k {
	case Chan, Func, Map, Ptr, UnsafePointer: //注：v是引用基础类型
		if v.flag&flagMethod != 0 { //注：有方法则返回false
			return false
		}
		ptr := v.ptr
		if v.flag&flagIndir != 0 { //注：如果v指向（指向数据的指针）的指针
			ptr = *(*unsafe.Pointer)(ptr) //注：取出指针
		}
		return ptr == nil //注：判读指针是否为nil
	case Interface, Slice:
		// 如果第一个单词为0，则interface和slice均为零。
		// 两者总是大于一个单词； 假设flagIndir。
		return *(*unsafe.Pointer)(v.ptr) == nil //注：指针为空则为空
	}
	panic(&ValueError{"reflect.Value.IsNil", v.kind()})
}

// IsValid 报告v是否代表值。
// 如果v为零值，则返回false。
// 如果IsValid返回false，则所有其他方法（字符串恐慌除外）。
// 大多数函数和方法从不返回无效的值。
// 如果是，则其文档会明确说明条件。
func (v Value) IsValid() bool { //注：返回v是否合法（判断v.flag是否不为0）
	return v.flag != 0
}

// IsZero 报告v是否为其类型的零值。
// 如果参数无效，则会出现恐慌。
func (v Value) IsZero() bool { // 注：v是否为零值
	switch v.kind() {
	case Bool:
		return !v.Bool()
	case Int, Int8, Int16, Int32, Int64:
		return v.Int() == 0
	case Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
		return v.Uint() == 0
	case Float32, Float64:
		return math.Float64bits(v.Float()) == 0
	case Complex64, Complex128:
		c := v.Complex()
		return math.Float64bits(real(c)) == 0 && math.Float64bits(imag(c)) == 0
	case Array: // 注：遍历每个元素是否为零值
		for i := 0; i < v.Len(); i++ {
			if !v.Index(i).IsZero() {
				return false
			}
		}
		return true
	case Chan, Func, Interface, Map, Ptr, Slice, UnsafePointer:
		return v.IsNil()
	case String:
		return v.Len() == 0
	case Struct: // 注：遍历每个成员是否为零值
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).IsZero() {
				return false
			}
		}
		return true
	default:
		// 这永远都不会发生，但是以后会作为一种保护措施，因为默认值在这里没有意义。
		panic(&ValueError{"reflect.Value.IsZero", v.Kind()})
	}
}

// Kind 返回v的Kind。
// 如果v为零值（IsValid返回false），则Kind返回Invalid。
func (v Value) Kind() Kind { // 注：获取v的类型枚举
	return v.kind()
}

// Len 返回v的长度。
// 如果v的Kind不是Array，Chan，Map，Slice或String，它会感到恐慌。
func (v Value) Len() int { // 注：获取v的擦汗能高度
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
		// 切片大于单词； 假设flagIndir。
		return (*sliceHeader)(v.ptr).Len
	case String:
		// 字符串比一个字大； 假设flagIndir。
		return (*stringHeader)(v.ptr).Len
	}
	panic(&ValueError{"reflect.Value.Len", v.kind()})
}

// MapIndex 返回与map v中的key关联的值。
// 如果v的Kind不是Map，它会感到恐慌。
// 如果在map中未找到key或v为nil，则返回零值。
// 和Go一样，key的值必须可分配给map的key类型。
func (v Value) MapIndex(key Value) Value { // 注：#
	v.mustBe(Map) // 注：必须为map类型
	tt := (*mapType)(unsafe.Pointer(v.typ))

	// 不需要导出密钥，以便DeepEqual和其他程序可以将MapKeys返回的所有密钥用作MapIndex的参数。
	// 但是，如果未导出地图或键，则结果将被视为未导出。
	// 这与结构的行为一致，该结构允许读取但不能写入未导出的字段。
	key = key.assignTo("reflect.Value.MapIndex", tt.key, nil)

	var k unsafe.Pointer
	if key.flag&flagIndir != 0 { // 注：是否为间接指针
		k = key.ptr
	} else {
		k = unsafe.Pointer(&key.ptr)
	}
	e := mapaccess(v.typ, v.pointer(), k) // 注：#
	if e == nil {
		return Value{}
	}
	typ := tt.elem
	fl := (v.flag | key.flag).ro()
	fl |= flag(typ.Kind())
	return copyVal(typ, fl, e)
}

// MapKeys 以未指定的顺序返回包含映射中存在的所有键的切片。
// 如果v的Kind不是Map，它会感到恐慌。
// 如果v代表nil映射，则返回一个空切片。
func (v Value) MapKeys() []Value { // 注：#获取集合v的所有key对象
	v.mustBe(Map) // 注：v必须为map类型
	tt := (*mapType)(unsafe.Pointer(v.typ))
	keyType := tt.key

	fl := v.flag.ro() | flag(keyType.Kind())

	m := v.pointer()
	mlen := int(0)
	if m != nil {
		mlen = maplen(m)
	}
	it := mapiterinit(v.typ, m)
	a := make([]Value, mlen)
	var i int
	for i = 0; i < len(a); i++ {
		key := mapiterkey(it) // 注：#
		if key == nil {
			// 由于我们在上面调用了maplen，因此有人从map中删除了一个条目。 这是一场数据竞争，但是我们对此无能为力。
			break
		}
		a[i] = copyVal(keyType, fl, key) // 注：生成typ = keyType，ptr = key，flag = v.flag
		mapiternext(it)                  // 注：#
	}
	return a[:i]
}

// MapIter 是用于遍历集合的迭代器。
// 参见Value.MapRange。
type MapIter struct { // 注：集合迭代器
	m  Value
	it unsafe.Pointer
}

// Key 返回迭代器当前映射项的key。
func (it *MapIter) Key() Value { // 注：#返回集合迭代器it当前的key
	if it.it == nil { // 注：#
		panic("MapIter.Key called before Next") // 恐慌："在Next之前调用MapIter.Key"
	}
	if mapiterkey(it.it) == nil { // 注：#
		panic("MapIter.Key called on exhausted iterator") // 恐慌："MapIter.Key在疲惫的迭代器上调用"
	}

	t := (*mapType)(unsafe.Pointer(it.m.typ))
	ktype := t.key
	return copyVal(ktype, it.m.flag.ro()|flag(ktype.Kind()), mapiterkey(it.it))
}

// Value 返回迭代器当前映射条目的值。
func (it *MapIter) Value() Value { // 注：#返回集合迭代器it当前的value
	if it.it == nil {
		panic("MapIter.Value called before Next") // 恐慌："在Next之前调用的MapIter.Value"
	}
	if mapiterkey(it.it) == nil {
		panic("MapIter.Value called on exhausted iterator") // 恐慌："在疲惫的迭代器上调用MapIter.Value"
	}

	t := (*mapType)(unsafe.Pointer(it.m.typ))
	vtype := t.elem
	return copyVal(vtype, it.m.flag.ro()|flag(vtype.Kind()), mapiterelem(it.it))
}

// Next 推进集合迭代器并报告是否还有另一个条目。 迭代器用尽时返回false；之后调用Key，Value，Next会引发恐慌
func (it *MapIter) Next() bool { // 注：#将集合迭代器it指向下一个条目，返回此条目是否有效
	if it.it == nil {
		it.it = mapiterinit(it.m.typ, it.m.pointer())
	} else {
		if mapiterkey(it.it) == nil {
			panic("MapIter.Next called on exhausted iterator") // 恐慌："MapIter.Next调用疲惫的迭代器"
		}
		mapiternext(it.it)
	}
	return mapiterkey(it.it) != nil
}

// MapRange 返回集合的范围迭代器。
// 如果v的Kind不是Map，它会感到恐慌。
//
// 调用Next前进迭代器，并调用Key/Value访问每个条目。
// 迭代器用尽时，next返回false。
// MapRange遵循与range语句相同的迭代语义。
//
// 示例：
//
//	iter := reflect.ValueOf(m).MapRange()
// 	for iter.Next() {
//		k := iter.Key()
//		v := iter.Value()
//		...
//	}
//
func (v Value) MapRange() *MapIter { // 工厂函数，生成一个集合迭代器
	v.mustBe(Map) // 注：v必须是集合类型
	return &MapIter{m: v}
}

// copyVal 返回一个包含映射键或ptr值的值，并根据需要分配一个新变量。
func copyVal(typ *rtype, fl flag, ptr unsafe.Pointer) Value { // 注：返回一个新Value，typ为type，指针为ptr，flag为f1
	if ifaceIndir(typ) { // 注：不是间接指针，复制并返回
		// 复制结果，以便将来对地图所做的更改不会更改基础值。
		c := unsafe_New(typ)
		typedmemmove(typ, c, ptr)
		return Value{typ, c, fl | flagIndir}
	}
	return Value{typ, *(*unsafe.Pointer)(ptr), fl}
}

// Method 返回与v的第i个方法相对应的函数值。
// 返回函数上的Call的参数不应包含接收方； 返回的函数将始终使用v作为接收者。
// 如果i超出范围或v是一个nil接口值，则方法将出现紧急情况。
func (v Value) Method(i int) Value { // 注：返回v的第i个方法
	if v.typ == nil {
		panic(&ValueError{"reflect.Value.Method", Invalid}) // 恐慌："类型错误"
	}
	if v.flag&flagMethod != 0 || uint(i) >= uint(v.typ.NumMethod()) {
		panic("reflect: Method index out of range") // 恐慌："方法索引超出范围"
	}
	if v.typ.Kind() == Interface && v.IsNil() {
		panic("reflect: Method on nil interface value") // 恐慌："无接口值的方法"
	}
	fl := v.flag & (flagStickyRO | flagIndir) // 清空 flagEmbedRO
	fl |= flag(Func)
	fl |= flag(i)<<flagMethodShift | flagMethod
	return Value{v.typ, v.ptr, fl}
}

// NumMethod 返回值的方法集中导出的方法的数量。
func (v Value) NumMethod() int { // 注：获取v的已导出方法的数量
	if v.typ == nil {
		panic(&ValueError{"reflect.Value.NumMethod", Invalid}) // 恐慌："类型错误"
	}
	if v.flag&flagMethod != 0 {
		return 0
	}
	return v.typ.NumMethod()
}

// MethodByName 返回与具有给定名称的v方法相对应的函数值。
// 返回函数上的Call的参数不应包含接收方； 返回的函数将始终使用v作为接收者。
// 如果未找到任何方法，则返回零值。
func (v Value) MethodByName(name string) Value { // 注：获取v中名为name的方法
	if v.typ == nil {
		panic(&ValueError{"reflect.Value.MethodByName", Invalid}) // 恐慌："类型错误"
	}
	if v.flag&flagMethod != 0 {
		return Value{}
	}
	m, ok := v.typ.MethodByName(name)
	if !ok {
		return Value{}
	}
	return v.Method(m.Index)
}

// NumField 返回结构v中的字段数。
// 如果v的Kind不是Struct，则会感到恐慌。
func (v Value) NumField() int { //注：获取结构体v的字段数量
	v.mustBe(Struct)
	tt := (*structType)(unsafe.Pointer(v.typ))
	return len(tt.fields)
}

// OverflowComplex 报告complex128 x是否不能用v的类型表示。
// 如果v的Kind不是Complex64或Complex128，则会发生恐慌。
func (v Value) OverflowComplex(x complex128) bool { // 注：获取x是否不能用v的类型表示
	k := v.kind()
	switch k {
	case Complex64:
		return overflowFloat32(real(x)) || overflowFloat32(imag(x)) // 注：x的实部和虚部是否超过float32的界限
	case Complex128:
		return false
	}
	panic(&ValueError{"reflect.Value.OverflowComplex", v.kind()}) // 恐慌："类型错误"
}

// OverflowFloat 报告float64 x是否不能用v的类型表示。
// 如果v的Kind不是Float32或Float64，则会发生恐慌。
func (v Value) OverflowFloat(x float64) bool { // 注：获取x是否不能用v的类型表示
	k := v.kind()
	switch k {
	case Float32:
		return overflowFloat32(x)
	case Float64:
		return false
	}
	panic(&ValueError{"reflect.Value.OverflowFloat", v.kind()})
}

func overflowFloat32(x float64) bool { // 注：浮点类型x是否超过float32的界限
	if x < 0 {
		x = -x
	}
	return math.MaxFloat32 < x && x <= math.MaxFloat64
}

// OverflowInt报告int64 x是否不能用v的类型表示。
// 如果v的Kind不是Int，Int8，Int16，Int32或Int64，则会发生恐慌。
func (v Value) OverflowInt(x int64) bool { // 注：获取x是否不能用v的类型表示
	k := v.kind()
	switch k {
	case Int, Int8, Int16, Int32, Int64:
		bitSize := v.typ.size * 8
		trunc := (x << (64 - bitSize)) >> (64 - bitSize)
		return x != trunc
	}
	panic(&ValueError{"reflect.Value.OverflowInt", v.kind()})
}

// OverflowUint 报告uint64 x是否不能用v的类型表示。
// 如果v的Kind不是Uint，Uintptr，Uint8，Uint16，Uint32或Uint64，它会感到恐慌。
func (v Value) OverflowUint(x uint64) bool { // 注：获取x是否不能用v的类型表示
	k := v.kind()
	switch k {
	case Uint, Uintptr, Uint8, Uint16, Uint32, Uint64:
		bitSize := v.typ.size * 8
		trunc := (x << (64 - bitSize)) >> (64 - bitSize)
		return x != trunc
	}
	panic(&ValueError{"reflect.Value.OverflowUint", v.kind()})
}

// go：不检查ptr
// 这样可以防止在启用-d = checkptr时内联Value.Pointer，从而确保cmd/compile可以识别unsafe.Pointer(v.Pointer())并产生异常。

// Pointer 以uintptr的形式返回v的值。
// 它返回uintptr而不是unsafe.Pointer，因此使用反射的代码无法在不显式导入unsafe包的情况下获取unsafe.Pointers。
// 如果v的Kind不是Chan，Func，Map，Ptr，Slice或UnsafePointer，它会感到恐慌。
//
// 如果v的Kind为Func，则返回的指针是基础代码指针，但不一定足以唯一地标识单个函数。 唯一的保证是，当且仅当v为nil func值时，结果为零。
//
// 如果v的Kind为Slice，则返回的指针指向该切片的第一个元素。 如果分片为nil，则返回值为0。如果分片为空但非nil，则返回值为非零。
func (v Value) Pointer() uintptr { //注：获取v的指针
	//TODO：弃用
	k := v.kind()
	switch k {
	case Chan, Map, Ptr, UnsafePointer: //注：这几个类型直接返回v.ptr
		return uintptr(v.pointer()) //注：返回v指向数据的指针
	case Func:
		if v.flag&flagMethod != 0 { //注：#
			// 正如doc注释所述，返回的指针是基础代码指针，但不一定足以唯一地标识单个函数。
			// 通过反射创建的所有方法表达式都具有相同的基础代码指针，因此它们的指针相等。
			// 这里使用的函数必须与makeMethodValue中使用的函数匹配。
			f := methodValueCall
			return **(**uintptr)(unsafe.Pointer(&f))
		}
		p := v.pointer() //注：否则和上一个case相同
		// 非nil func值指向数据块。
		// 数据块的第一个字是实际代码。
		if p != nil {
			p = *(*unsafe.Pointer)(p)
		}
		return uintptr(p)

	case Slice: //注：v是切片类型
		return (*SliceHeader)(v.ptr).Data //注：返回指向第一个元素的指针
	}
	panic(&ValueError{"reflect.Value.Pointer", v.kind()})
}

// Recv 从通道v接收并返回一个值。
// 如果v的Kind不是Chan，就会感到恐慌。
// 接收阻塞，直到准备好值为止。
// 如果值x对应于通道上的发送，则布尔值ok为true，如果由于通道关闭而接收到零值，则为false。
func (v Value) Recv() (x Value, ok bool) {
	v.mustBe(Chan)     // 注：v必须是通道类型
	v.mustBeExported() // 注：v必须是已导出的对象
	return v.recv(false)
}

// 内部接收，可能是非阻塞（nb）。
// v已知是一个通道。
func (v Value) recv(nb bool) (val Value, ok bool) { // 注：#
	tt := (*chanType)(unsafe.Pointer(v.typ))
	if ChanDir(tt.dir)&RecvDir == 0 {
		panic("reflect: recv on send-only channel") // 恐慌："在只写管道上接收数据"
	}
	t := tt.elem
	val = Value{t, nil, flag(t.Kind())}
	var p unsafe.Pointer
	if ifaceIndir(t) {
		p = unsafe_New(t)
		val.ptr = p
		val.flag |= flagIndir
	} else {
		p = unsafe.Pointer(&val.ptr)
	}
	selected, ok := chanrecv(v.pointer(), nb, p) // 注：#
	if !selected {
		val = Value{}
	}
	return
}

// Send 在管道v上发送x。
// 如果v的种类不是Chan或x的类型与v的元素类型不同，则会感到恐慌。
// 和Go一样，x的值必须可分配给通道的元素类型。
func (v Value) Send(x Value) {
	v.mustBe(Chan)     // 注：要求v是管道类型
	v.mustBeExported() // 注：要求v是已导出变量
	v.send(x, false)
}

// 内部发送，可能是非阻塞的。
// v已知是一个通道。
func (v Value) send(x Value, nb bool) (selected bool) { // 注：#
	tt := (*chanType)(unsafe.Pointer(v.typ))
	if ChanDir(tt.dir)&SendDir == 0 {
		panic("reflect: send on recv-only channel") // 恐慌："在只读管道上发送数据"
	}
	x.mustBeExported()                                 // 注：x必须是已导出数据
	x = x.assignTo("reflect.Value.Send", tt.elem, nil) // 注：#
	var p unsafe.Pointer
	if x.flag&flagIndir != 0 {
		p = x.ptr
	} else {
		p = unsafe.Pointer(&x.ptr)
	}
	return chansend(v.pointer(), p, nb) // 注：#
}

// Set 将x赋给值v。
// 如果CanSet返回false，则会感到恐慌。
// 和Go一样，x的值必须可分配给v的类型。
func (v Value) Set(x Value) { // 注：#
	v.mustBeAssignable() // 注：要求v已分配地址
	x.mustBeExported()   // 不要让未导出的x泄漏
	var target unsafe.Pointer
	if v.kind() == Interface {
		target = v.ptr
	}
	x = x.assignTo("reflect.Set", v.typ, target) // 注：#
	if x.flag&flagIndir != 0 {
		typedmemmove(v.typ, v.ptr, x.ptr)
	} else {
		*(*unsafe.Pointer)(v.ptr) = x.ptr
	}
}

// SetBool 设置v的基础值。
// 如果v的Kind不是Bool或CanSet()为false，则会发生恐慌。
func (v Value) SetBool(x bool) { //注：将bool类型x赋值给Bool类型v
	v.mustBeAssignable() //注：v必须为可分配内存的已导出对象
	v.mustBe(Bool)       //注：v必须为布尔类型
	*(*bool)(v.ptr) = x
}

// SetBytes 设置v的基础值。
// 如果v的基础值不是一个字节片，则会感到恐慌。
func (v Value) SetBytes(x []byte) { // 注：将[]byte类型x赋值给Slice类型v
	v.mustBeAssignable() // 注：v必须为可分配内存的已导出对象
	v.mustBe(Slice)      // 注：v必须是切片类型
	if v.typ.Elem().Kind() != Uint8 {
		panic("reflect.Value.SetBytes of non-byte slice") // 恐慌："非字节切片的reflect.Value.SetBytes"
	}
	*(*[]byte)(v.ptr) = x
}

// setRunes 设置v的基础值。
// 如果v的基础值不是一小段rune（int32s），则会感到恐慌。
func (v Value) setRunes(x []rune) { // 注：将[]rune类型x赋值给Slice类型v
	v.mustBeAssignable() // 注：v必须为可分配内存的已导出对象
	v.mustBe(Slice)      // 注：v必须是切片类型
	if v.typ.Elem().Kind() != Int32 {
		panic("reflect.Value.setRunes of non-rune slice") // 恐慌："非符文切片的reflect.Value.setRunes"
	}
	*(*[]rune)(v.ptr) = x
}

// SetComplex 将v的基础值设置为x。
// 如果v的Kind不是Complex64或Complex128，或者CanSet()为false，则会发生恐慌。
func (v Value) SetComplex(x complex128) { // 注：将complex类型x赋值给complex类型v
	v.mustBeAssignable() // 注：v必须为可分配内存的已导出对象
	switch k := v.kind(); k {
	default:
		panic(&ValueError{"reflect.Value.SetComplex", v.kind()}) // 恐慌："类型错误"
	case Complex64:
		*(*complex64)(v.ptr) = complex64(x)
	case Complex128:
		*(*complex128)(v.ptr) = x
	}
}

// SetFloat 将v的基础值设置为x。
// 如果v的Kind不是Float32或Float64，或者CanSet()为false，则会发生恐慌。
func (v Value) SetFloat(x float64) { // 注：将float类型x赋值给float类型v
	v.mustBeAssignable() // 注：v必须为可分配内存的已导出对象
	switch k := v.kind(); k {
	default:
		panic(&ValueError{"reflect.Value.SetFloat", v.kind()}) // 恐慌："类型错误"
	case Float32:
		*(*float32)(v.ptr) = float32(x)
	case Float64:
		*(*float64)(v.ptr) = x
	}
}

// SetInt 将v的基础值设置为x。
// 如果v的Kind不是Int，Int8，Int16，Int32或Int64，或者CanSet()为false，则会发生恐慌。
func (v Value) SetInt(x int64) { // 注：将int类型x赋值给int类型v
	v.mustBeAssignable() // 注：v必须为可分配内存的已导出对象
	switch k := v.kind(); k {
	default:
		panic(&ValueError{"reflect.Value.SetInt", v.kind()}) // 恐慌："类型错误"
	case Int:
		*(*int)(v.ptr) = int(x)
	case Int8:
		*(*int8)(v.ptr) = int8(x)
	case Int16:
		*(*int16)(v.ptr) = int16(x)
	case Int32:
		*(*int32)(v.ptr) = int32(x)
	case Int64:
		*(*int64)(v.ptr) = x
	}
}

// SetLen 将v的长度设置为n。
// 如果v的Kind不是Slice，或者n为负或大于slice的容量，则会发生恐慌。
func (v Value) SetLen(n int) { // 注：设置Slice类型v的长度len为n，（n <= cap）
	v.mustBeAssignable() // 注：v必须为可分配内存的已导出对象
	v.mustBe(Slice)      // 注：v必须是切片类型
	s := (*sliceHeader)(v.ptr)
	if uint(n) > uint(s.Cap) { // 注：n不能超过cap
		panic("reflect: slice length out of range in SetLen") // 恐慌："切片长度超出SetLen中的范围"
	}
	s.Len = n
}

// SetCap 将v的容量设置为n。
// 如果v的Kind不是Slice，或者n小于length或大于slice的容量，则会发生恐慌。
func (v Value) SetCap(n int) { // 注：设置Slice类型v的容量cap为n，（len <= n <= cap）
	v.mustBeAssignable() // 注：v必须为可分配内存的已导出对象
	v.mustBe(Slice)      // 注：v必须是切片类型
	s := (*sliceHeader)(v.ptr)
	if n < s.Len || n > s.Cap { // 注：n不能超过len与cap的范围
		panic("reflect: slice capacity out of range in SetCap") // 恐慌："切片容量超出SetCap范围"
	}
	s.Cap = n
}

// SetMapIndex 将与集合v中的key关联的元素设置为elem。
// 如果v的Kind不是Map，它会感到恐慌。
// 如果elem为零值，则SetMapIndex会从集合中删除键。
// 否则，如果v持有nil集合，则SetMapIndex会恐慌。
// 和Go一样，key的elem必须可分配给集合的key类型，并且elem的值必须可分配给集合的key类型。
func (v Value) SetMapIndex(key, elem Value) { // 注：#
	v.mustBe(Map)        // 注：v必须是集合类型
	v.mustBeExported()   // 注：v必须是已导出的对象
	key.mustBeExported() // 注：key必须是已导出的对象
	tt := (*mapType)(unsafe.Pointer(v.typ))
	key = key.assignTo("reflect.Value.SetMapIndex", tt.key, nil) // 注：#
	var k unsafe.Pointer
	if key.flag&flagIndir != 0 {
		k = key.ptr
	} else {
		k = unsafe.Pointer(&key.ptr)
	}
	if elem.typ == nil {
		mapdelete(v.typ, v.pointer(), k)
		return
	}
	elem.mustBeExported()
	elem = elem.assignTo("reflect.Value.SetMapIndex", tt.elem, nil)
	var e unsafe.Pointer
	if elem.flag&flagIndir != 0 {
		e = elem.ptr
	} else {
		e = unsafe.Pointer(&elem.ptr)
	}
	mapassign(v.typ, v.pointer(), k, e)
}

// SetUint 将v的基础值设置为x。
// 如果v的Kind不是Uint，Uintptr，Uint8，Uint16，Uint32或Uint64，或者CanSet（）为false，则会发生恐慌。
func (v Value) SetUint(x uint64) { // 注：将uint类型x赋值给uint类型v
	v.mustBeAssignable() // 注：v必须为可分配内存的已导出对象
	switch k := v.kind(); k {
	default:
		panic(&ValueError{"reflect.Value.SetUint", v.kind()})
	case Uint:
		*(*uint)(v.ptr) = uint(x)
	case Uint8:
		*(*uint8)(v.ptr) = uint8(x)
	case Uint16:
		*(*uint16)(v.ptr) = uint16(x)
	case Uint32:
		*(*uint32)(v.ptr) = uint32(x)
	case Uint64:
		*(*uint64)(v.ptr) = x
	case Uintptr:
		*(*uintptr)(v.ptr) = uintptr(x)
	}
}

// SetPointer 将unsafe.Pointer值v设置为x。
// 如果v的Kind不是UnsafePointer，它会感到恐慌。
func (v Value) SetPointer(x unsafe.Pointer) { // 注：将unsafe.Pointer类型x赋值给UnsafePointer类型v
	v.mustBeAssignable()    // 注：v必须为可分配内存的已导出对象
	v.mustBe(UnsafePointer) // 注：v必须是Unsafe.Pointer类型
	*(*unsafe.Pointer)(v.ptr) = x
}

// SetString 将v的基础值设置为x。
// 如果v的Kind不是String或CanSet()为false，则会发生恐慌。
func (v Value) SetString(x string) { // 注：将string类型s赋值给String类型v
	v.mustBeAssignable() // 注：v必须为可分配内存的已导出对象
	v.mustBe(String)     // 注：v必须是字符串类型
	*(*string)(v.ptr) = x
}

// Slice 返回v [i:j]。
// 如果v的Kind不是Array，Slice或String，或者v是不可寻址的数组，或者索引超出范围，则会发生恐慌。
func (v Value) Slice(i, j int) Value { //注：获取数组/切片/字符串类型v[i: j]
	var (
		cap  int
		typ  *sliceType
		base unsafe.Pointer
	)
	switch kind := v.kind(); kind {
	default:
		panic(&ValueError{"reflect.Value.Slice", v.kind()}) // 恐慌："v的类型错误"

	case Array:
		if v.flag&flagAddr == 0 {
			panic("reflect.Value.Slice: slice of unaddressable array") // 恐慌："不可寻址数组切片"
		}
		tt := (*arrayType)(unsafe.Pointer(v.typ)) // 注：转为数组反射类型
		cap = int(tt.len)
		typ = (*sliceType)(unsafe.Pointer(tt.slice)) // 注：转为数组反射类型
		base = v.ptr

	case Slice:
		typ = (*sliceType)(unsafe.Pointer(v.typ)) // 注：转为切片反射类型
		s := (*sliceHeader)(v.ptr)
		base = s.Data
		cap = s.Cap

	case String:
		s := (*stringHeader)(v.ptr)
		if i < 0 || j < i || j > s.Len {
			panic("reflect.Value.Slice: string slice index out of bounds") // 恐慌："字符串切片索引超出范围"
		}
		var t stringHeader
		if i < s.Len {
			t = stringHeader{arrayAt(s.Data, i, 1, "i < s.Len"), j - i} // 注：s.Data偏移i个字节（字节的长度为1）
		}
		return Value{v.typ, unsafe.Pointer(&t), v.flag} // 注：返回字符串的反射类型，数据为字符串v[i: j]，长度为j-i
	}

	if i < 0 || j < i || j > cap {
		panic("reflect.Value.Slice: slice index out of bounds") // 恐慌："切片索引超出范围"
	}

	// 声明slice，以便gc可以在其中看到基本指针。
	var x []unsafe.Pointer

	// 重新解释为*sliceHeader进行编辑。
	s := (*sliceHeader)(unsafe.Pointer(&x))
	s.Len = j - i
	s.Cap = cap - i
	if cap-i > 0 {
		s.Data = arrayAt(base, i, typ.elem.Size(), "i < cap")
	} else {
		// 不要前进指针，以避免指向超出切片末端
		s.Data = base
	}

	fl := v.flag.ro() | flagIndir | flag(Slice)
	return Value{typ.common(), unsafe.Pointer(&x), fl}
}

// Slice3 是slice运算的3索引形式：它返回v[i：j：k]。
// 如果v的Kind不是Array或Slice，或者v是不可寻址的数组，或者索引超出范围，则会发生恐慌。
func (v Value) Slice3(i, j, k int) Value { // 注：获取数组/切片/字符串类型的v[i: j: k]
	var (
		cap  int
		typ  *sliceType
		base unsafe.Pointer
	)
	switch kind := v.kind(); kind {
	default:
		panic(&ValueError{"reflect.Value.Slice3", v.kind()}) // 恐慌："v的类型错误"

	case Array:
		if v.flag&flagAddr == 0 {
			panic("reflect.Value.Slice3: slice of unaddressable array") // 恐慌："不可寻址数组的切片"
		}
		tt := (*arrayType)(unsafe.Pointer(v.typ))
		cap = int(tt.len)
		typ = (*sliceType)(unsafe.Pointer(tt.slice))
		base = v.ptr

	case Slice:
		typ = (*sliceType)(unsafe.Pointer(v.typ))
		s := (*sliceHeader)(v.ptr)
		base = s.Data
		cap = s.Cap
	}

	if i < 0 || j < i || k < j || k > cap {
		panic("reflect.Value.Slice3: slice index out of bounds") // 恐慌："切片索引超出范围"
	}

	// 声明切片，以便垃圾收集器可以看到其中的基本指针。
	var x []unsafe.Pointer

	// 重新解释为*sliceHeader进行编辑。
	s := (*sliceHeader)(unsafe.Pointer(&x))
	s.Len = j - i
	s.Cap = k - i
	if k-i > 0 {
		s.Data = arrayAt(base, i, typ.elem.Size(), "i < k <= cap")
	} else {
		// 不要前进指针，以避免指向超出切片结尾的位置
		s.Data = base
	}

	fl := v.flag.ro() | flagIndir | flag(Slice)
	return Value{typ.common(), unsafe.Pointer(&x), fl}
}

// String 以字符串形式返回字符串v的基础值。
// 由于Go的String方法约定，String是一种特殊情况。
// 与其他获取方法不同的是，如果v的Kind不是String，它不会惊慌。
// 而是返回形式为"<T value>"的字符串，其中T是v的类型。
// fmt包特别对待Values。 它不会隐式调用其String方法，而是打印它们持有的具体值。
func (v Value) String() string { // 注：返回v指向的值的字符串形式，如果v不是字符串，返回v的类型的字符串形式
	switch k := v.kind(); k {
	case Invalid:
		return "<invalid Value>"
	case String:
		return *(*string)(v.ptr)
	}
	// 如果在其他类型的reflect.Value上调用String，则打印某些内容要比恐慌好。 在调试中很有用。
	return "<" + v.Type().String() + " Value>" // 注：返回"<T value>"
}

// TryRecv 尝试从通道v接收值，但不会阻塞。
// 如果v的Kind不是Chan，就会感到恐慌。
// 如果接收方提供了一个值，则x是已传输的值，而ok为true。
// 如果接收无法完成而没有阻塞，则x为零值，ok为假。
// 如果关闭了通道，则x是该通道的元素类型的零值，而ok是false。
func (v Value) TryRecv() (x Value, ok bool) { // 注：#
	v.mustBe(Chan)      // 注：v必须是管道类型
	v.mustBeExported()  // 注：v必须是已导出的对象
	return v.recv(true) // 注：#
}

// TrySend 尝试在管道v上发送x，但不会阻塞。
// 如果v的Kind不是Chan，就会感到恐慌。
// 报告是否发送了值。
// 和Go一样，x的值必须可分配给通道的元素类型。
func (v Value) TrySend(x Value) bool { // 注：#
	v.mustBe(Chan)         // 注：v必须是管道类型
	v.mustBeExported()     // 注：必须是已导出的对象
	return v.send(x, true) // 注：#
}

// Type 返回v的类型。
func (v Value) Type() Type { //注：#
	f := v.flag
	if f == 0 {
		panic(&ValueError{"reflect.Value.Type", Invalid}) //恐慌："非法类型"
	}
	if f&flagMethod == 0 { //注：v没有方法，直接返回类型
		// 简单的情况
		return v.typ
	}

	// 方法值。
	// v.typ描述接收者，而不是方法类型。
	i := int(v.flag) >> flagMethodShift // 注：获取v的已导出方法集合的最大索引
	if v.typ.Kind() == Interface {      // 注：如果v是接口类型
		// 接口上的方法。
		tt := (*interfaceType)(unsafe.Pointer(v.typ))
		if uint(i) >= uint(len(tt.methods)) { //注：最大索引 >= 方法数量，引发恐慌
			panic("reflect: internal error: invalid method index") //恐慌："内部错误：方法索引无效"
		}
		m := &tt.methods[i]         //注：获取最后一个方法
		return v.typ.typeOff(m.typ) //注：#
	}
	//具体类型的方法。
	ms := v.typ.exportedMethods() // 注：获取v的已导出方法
	if uint(i) >= uint(len(ms)) { //注：最大索引 >= 已导出方法数量，引发恐慌
		panic("reflect: internal error: invalid method index") //恐慌："内部错误：方法索引无效"
	}
	m := ms[i]
	return v.typ.typeOff(m.mtyp) //注：#
}

// Uint 返回v的基础值，作为uint64。
// 如果v的Kind不是Uint，Uintptr，Uint8，Uint16，Uint32或Uint64，则会出现恐慌。
func (v Value) Uint() uint64 { //注：将v转为uint64并返回
	k := v.kind()
	p := v.ptr
	switch k {
	case Uint:
		return uint64(*(*uint)(p)) //注：将v指向数据的指针转为*uint，再取值后格式化为uint64
	case Uint8:
		return uint64(*(*uint8)(p))
	case Uint16:
		return uint64(*(*uint16)(p))
	case Uint32:
		return uint64(*(*uint32)(p))
	case Uint64:
		return *(*uint64)(p)
	case Uintptr:
		return uint64(*(*uintptr)(p))
	}
	panic(&ValueError{"reflect.Value.Uint", v.kind()})
}

// go：nocheckptr
// 这样可以防止在启用-d = checkptr时内联Value.UnsafeAddr，从而确保cmd/compile可以识别unsafe.Pointer(v.UnsafeAddr())并生成异常。

// UnsafeAddr 返回指向v数据的指针。
// 适用于高级客户，这些客户也导入了"unsafe"包
// 如果v不可寻址，它会感到恐慌。
func (v Value) UnsafeAddr() uintptr { // 注：返回uintptr类型的v的指针
	// TODO：弃用
	if v.typ == nil {
		panic(&ValueError{"reflect.Value.UnsafeAddr", Invalid}) // 恐慌："错误的类型"
	}
	if v.flag&flagAddr == 0 {
		panic("reflect.Value.UnsafeAddr of unaddressable value") // 恐慌："无法寻址值的reflect.Value.UnsafeAddr"
	}
	return uintptr(v.ptr) // 注：将v的指针转为uintptr
}

// StringHeader 是字符串的运行时表示形式。
// 无法安全或便携地使用它，并且其表示形式可能在以后的版本中更改。
// 此外，"Data"字段不足以保证不会对其进行垃圾回收，因此程序必须保留一个单独的，正确键入的指向基础数据的指针。
type StringHeader struct {
	Data uintptr
	Len  int
}

// stringHeader 是此软件包中使用的StringHeader的安全版本。
type stringHeader struct { // 注：reflect包字符串反射类型
	Data unsafe.Pointer // 注：名称数据的地址
	Len  int            // 注：名称数据的长度
}

// SliceHeader 是切片的运行时表示形式。
// 无法安全或便携地使用它，并且其表示形式可能
// 在更高版本中进行更改。
// 此外，Data字段不足以保证它引用的数据不会被垃圾收集，因此，程序必须保留一个单独的，正确键入的指向基础数据的指针。
type SliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}

// sliceHeader 是此包中使用的SliceHeader的安全版本。
type sliceHeader struct {
	Data unsafe.Pointer // 注：切片的数据
	Len  int            // 注：切片的长度
	Cap  int            // 注：切片的容量
}

func typesMustMatch(what string, t1, t2 Type) { // 注：t1与t2的类型必须一致
	if t1 != t2 {
		panic(what + ": " + t1.String() + " != " + t2.String()) // 恐慌："t1的类型与t2的不一致"
	}
}

// arrayAt 返回p的第i个元素，即元素为eltSize字节宽的数组。
// p指向的数组必须至少包含i + 1个元素：
// 传递i >= len是无效的（但无法在此处检查），因为结果将指向数组之外。
// whySafe必须解释为什么 i < len。 （传递"i < len"是可以的；这样做的好处是可以在呼叫站点显示此假设。）
func arrayAt(p unsafe.Pointer, i int, eltSize uintptr, whySafe string) unsafe.Pointer { //注：p偏移i个eltSize的位置，whysafe描述为什么这个操作是安全的，返回偏移后的指针
	return add(p, uintptr(i)*eltSize, "i < len")
}

// grow 扩容切片s，使其可以容纳更多的值，并在需要时分配更多的容量。 它还返回旧的和新的切片长度。
func grow(s Value, extra int) (Value, int, int) { // 注：扩容切片s至少extra数量，返回扩容后的切片，长度与容量
	i0 := s.Len()    // 注：现有切片长度，len
	i1 := i0 + extra // 注：期望切片长度，newlen
	if i1 < i0 {
		panic("reflect.Append: slice overflow") // 恐慌："切片溢出"
	}
	m := s.Cap() // 注：现有切片容量，cap
	if i1 <= m { // 注：如果newlen没有超过cap，直接修改Len
		return s.Slice(0, i1), i0, i1
	}
	if m == 0 { // 注：如果s是空切片，容量设置为newlen
		m = extra
	} else {
		for m < i1 {
			if i0 < 1024 { // 注：len < 1024时，增长因子 = 2
				m += m
			} else { // 注：len >= 1024时，增长因子 = 1.25
				m += m / 4
			}
		}
	}
	t := MakeSlice(s.Type(), i1, m)
	Copy(t, s)
	return t, i0, i1
}

// Append将值x附加到切片s并返回结果切片。
// 和Go一样，每个x的值必须可分配给slice的元素类型。
func Append(s Value, x ...Value) Value { // 注：切片类型s附加多个元素x
	s.mustBe(Slice)              // 注：s必须是切片类型
	s, i0, i1 := grow(s, len(x)) // 注：s扩容x的长度
	for i, j := i0, 0; i < i1; i, j = i+1, j+1 {
		s.Index(i).Set(x[j]) // 注：s[i] = x[j]
	}
	return s
}

// AppendSlice将切片t附加到切片s上，并返回结果切片。
// 切片s和t必须具有相同的元素类型。
func AppendSlice(s, t Value) Value { // 注：切片类型s附加切片类型t
	s.mustBe(Slice)                                                         // 注：s必须是类型
	t.mustBe(Slice)                                                         // 注：t必须是类型
	typesMustMatch("reflect.AppendSlice", s.Type().Elem(), t.Type().Elem()) // 注：s和t的成员类型必须一致
	s, i0, i1 := grow(s, t.Len())                                           // 注：s扩容t的长度
	Copy(s.Slice(i0, i1), t)
	return s
}

// Copy 将src的内容复制到dst中，直到填满dst或用尽src。
// 返回复制的元素数。
// Dst和src每个都必须具有切片或数组，而dst和src必须具有相同的元素类型。
// 作为一种特殊情况，如果dst的元素类型为Uint8，则src可以具有String类型。
func Copy(dst, src Value) int { // 注：将src复制给dst
	dk := dst.kind()
	if dk != Array && dk != Slice { // 注：dst必须是数组或切片类型
		panic(&ValueError{"reflect.Copy", dk}) // 恐慌："错误的类型"
	}
	if dk == Array { // 注：数组类型的dst必须为可分配内存的已导出对象
		dst.mustBeAssignable()
	}
	dst.mustBeExported() // 注：dst必须是已导出的对象

	sk := src.kind()
	var stringCopy bool
	if sk != Array && sk != Slice { // 注：src如果不是数组或切片类型，检查是否是字符串类型
		stringCopy = sk == String && dst.typ.Elem().Kind() == Uint8
		if !stringCopy { // 注： src不是字符串类型或dst的成员类型不是uint8
			panic(&ValueError{"reflect.Copy", sk}) // 恐慌："错误的类型"
		}
	}
	src.mustBeExported() // 注：src必须是已导出的对象

	de := dst.typ.Elem()
	if !stringCopy { // 注：如果不是字符串拷贝，则dst和src的成员类型必须相同
		se := src.typ.Elem()
		typesMustMatch("reflect.Copy", de, se)
	}

	var ds, ss sliceHeader // 注：dstSlice，srcSlice
	if dk == Array {
		ds.Data = dst.ptr
		ds.Len = dst.Len()
		ds.Cap = ds.Len
	} else {
		ds = *(*sliceHeader)(dst.ptr)
	}
	if sk == Array {
		ss.Data = src.ptr
		ss.Len = src.Len()
		ss.Cap = ss.Len
	} else if sk == Slice {
		ss = *(*sliceHeader)(src.ptr)
	} else {
		sh := *(*stringHeader)(src.ptr)
		ss.Data = sh.Data
		ss.Len = sh.Len
		ss.Cap = sh.Len
	}

	return typedslicecopy(de.common(), ds, ss)
}

// runtimeSelect 是传递给rselect的单个case。
// 必须与../runtime/select.go:/runtimeSelect匹配
type runtimeSelect struct {
	dir SelectDir      // SelectSend，SelectRecv或SelectDefault
	typ *rtype         // 管道类型
	ch  unsafe.Pointer // 管道
	val unsafe.Pointer // ptr到数据（SendDir）或ptr到接收缓冲区（RecvDir）
}

// rselect 运行一个select。 它返回选中的case的索引。
// 如果case是接收的，则用接收的值填充val。
// 常规的OK bool指示接收是否与发送的值相对应。
//go:noescape
func rselect([]runtimeSelect) (chosen int, recvOK bool)

// SelectDir 描述select case的通信方向。
type SelectDir int

// 注意：这些值必须匹配../runtime/select.go:/selectDir。

const (
	_             SelectDir = iota
	SelectSend              // case Chan <- Send
	SelectRecv              // case <-Chan:
	SelectDefault           // default
)

// SelectCase 描述了select操作中的单个case
// 情况的种类取决于Dir，通讯方向。
//
// 如果Dir为SelectDefault，该case代表default
// Chan和Send必须为零值。
//
// 如果Dir为SelectSend，该case代表发送操作。
// 通常，Chan的基础值必须是一个通道，Send的基础值必须可分配给该通道的元素类型。
// 作为一种特殊情况，如果Chan为零值，则忽略case，并且字段Send也将被忽略，并且可以为零或非零。
//
// 如果Dir是SelectRecv，该case代表接收操作。
// 通常，Chan的基础值必须是一个通道，Send必须是零值。
// 如果Chan是零值，则忽略case，但是Send必须仍然是零值。
// 选择接收操作时，Select将返回接收到的值。
//
type SelectCase struct { // 注：select中case的反射类型
	Dir  SelectDir // case的方向
	Chan Value     // 使用的通道（用于发送或接收）
	Send Value     // 要发送的值（用于发送）
}

// Select 执行case列表所描述的选择操作。
// 像Go select语句一样，它阻塞直到至少一种情况可以继续进行，做出统一的伪随机选择，然后执行该情况。
// 它返回所选case的索引，如果该case是接收操作，则返回接收到的值和一个布尔值，
// 该布尔值指示该值是否对应于通道上的发送（而不是因为通道关闭而接收到的零值）。
func Select(cases []SelectCase) (chosen int, recv Value, recvOK bool) { // 注：执行一次select，case为cases，返回执行的cases索引chosen，读取到的数据recv，是否读取到
	// 注意：不要相信调用者没有修改脚下的case数据。
	// 范围是安全的，因为调用者无法修改我们的len副本，并且每次迭代都将自己复制值c。
	runcases := make([]runtimeSelect, len(cases)) // 注：声明与cases相同数量的runtimeSelect
	haveDefault := false                          // 注：是否有default
	for i, c := range cases {                     // 注：遍历cases，将信息赋值给runcases
		rc := &runcases[i] // 注：case
		rc.dir = c.Dir
		// 步骤：
		// 	1. runcases[i].dir = cases[i].Dir
		// 	2. runcases[i].ch  = cases[i].Chan
		// 	3. runcases[i].typ = cases[i].Chan.typ
		// 	4. runcases[i].val = cases[i].Send
		switch c.Dir {
		default:
			panic("reflect.Select: invalid Dir") // 恐慌："无效的通道方向"
		case SelectDefault: // default
			if haveDefault {
				panic("reflect.Select: multiple default cases") // 恐慌："select中有多个default"
			}
			haveDefault = true
			if c.Chan.IsValid() {
				panic("reflect.Select: default case has Chan value") // 恐慌："默认case具有Chan值"
			}
			if c.Send.IsValid() {
				panic("reflect.Select: default case has Send value") // 恐慌："默认案例具有Send值"
			}

		case SelectSend: // 注：Chan <- Send
			ch := c.Chan
			if !ch.IsValid() { // 注：不合法的通道
				break
			}
			ch.mustBe(Chan)     // 注：Chan必须是管道类型
			ch.mustBeExported() // 注：Chan必须是已导出的对象
			tt := (*chanType)(unsafe.Pointer(ch.typ))
			if ChanDir(tt.dir)&SendDir == 0 {
				panic("reflect.Select: SendDir case using recv-only channel") // 恐慌："SendDir使用了只读管道"
			}
			rc.ch = ch.pointer()
			rc.typ = &tt.rtype
			v := c.Send
			if !v.IsValid() {
				panic("reflect.Select: SendDir case missing Send value") // 恐慌："SendDir缺少发送的值"
			}
			v.mustBeExported() // 注：Send必须是已导出的对象
			v = v.assignTo("reflect.Select", tt.elem, nil)
			if v.flag&flagIndir != 0 {
				rc.val = v.ptr
			} else {
				rc.val = unsafe.Pointer(&v.ptr)
			}

		case SelectRecv: // 注：<- Chan
			if c.Send.IsValid() {
				panic("reflect.Select: RecvDir case has Send value") // 恐慌："RecvDir具有发送值"
			}
			ch := c.Chan
			if !ch.IsValid() {
				break
			}
			ch.mustBe(Chan)     // 注：Chan必须是管道类型
			ch.mustBeExported() // 注：Chan必须是已导出的对象
			tt := (*chanType)(unsafe.Pointer(ch.typ))
			if ChanDir(tt.dir)&RecvDir == 0 {
				panic("reflect.Select: RecvDir case using send-only channel") // 恐慌："RecvDir使用了只写管道"
			}
			rc.ch = ch.pointer()
			rc.typ = &tt.rtype
			rc.val = unsafe_New(tt.elem)
		}
	}

	chosen, recvOK = rselect(runcases)      // 注：运行1次select
	if runcases[chosen].dir == SelectRecv { // 注：执行的case是从读取，返回读取到的值
		tt := (*chanType)(unsafe.Pointer(runcases[chosen].typ))
		t := tt.elem
		p := runcases[chosen].val
		fl := flag(t.Kind())
		if ifaceIndir(t) {
			recv = Value{t, p, fl | flagIndir}
		} else {
			recv = Value{t, *(*unsafe.Pointer)(p), fl}
		}
	}
	return chosen, recv, recvOK
}

/*
 * 构造函数
 */

// 在runtime包实现
func unsafe_New(*rtype) unsafe.Pointer           // 注：获取一个新创建的"rtype类型的值"的指针
func unsafe_NewArray(*rtype, int) unsafe.Pointer // 注：获取一个新创建的"rtype类型int长度的数组"的指针

// MakeSlice 为指定的切片类型，长度和容量创建一个新的零初始化切片值。
func MakeSlice(typ Type, len, cap int) Value { //注：创建一个类型为typ，长度为len，空间为cap的切片
	if typ.Kind() != Slice {
		panic("reflect.MakeSlice of non-slice type") //恐慌："非切片类型的reflect.MakeSlice"
	}
	if len < 0 {
		panic("reflect.MakeSlice: negative len") //恐慌："负数长度"
	}
	if cap < 0 {
		panic("reflect.MakeSlice: negative cap") //恐慌："负数空间"
	}
	if len > cap {
		panic("reflect.MakeSlice: len > cap") //恐慌："长度 > 空间"
	}

	s := sliceHeader{unsafe_NewArray(typ.Elem().(*rtype), cap), len, cap}
	return Value{typ.(*rtype), unsafe.Pointer(&s), flagIndir | flag(Slice)}
}

// MakeChan 创建具有指定类型和缓冲区大小的新通道。
func MakeChan(typ Type, buffer int) Value {
	if typ.Kind() != Chan {
		panic("reflect.MakeChan of non-chan type") // 恐慌："非通道类型的reflect.MakeChan"
	}
	if buffer < 0 {
		panic("reflect.MakeChan: negative buffer size") // 恐慌："负数缓冲区大小"
	}
	if typ.ChanDir() != BothDir {
		panic("reflect.MakeChan: unidirectional channel type") // 恐慌："单向通道类型"
	}
	t := typ.(*rtype)
	ch := makechan(t, buffer)
	return Value{t, ch, flag(Chan)}
}

// MakeMap 创建具有指定类型的新集合
func MakeMap(typ Type) Value { // 注：创建类型为typ的集合
	return MakeMapWithSize(typ, 0)
}

// MakeMapWithSize 创建具有指定类型和大约n个元素的初始空间的新地图。
func MakeMapWithSize(typ Type, n int) Value { // 注：创建类型为typ，容量为n的集合
	if typ.Kind() != Map {
		panic("reflect.MakeMapWithSize of non-map type") // 恐慌："非集合类型的reflect.MakeMapWithSize"
	}
	t := typ.(*rtype)
	m := makemap(t, n)
	return Value{t, m, flag(Map)}
}

// Indirect 返回v指向的值。
// 如果v是nil指针，则Indirect返回零值。
// 如果v不是指针，则Indirect返回v。
func Indirect(v Value) Value { // 注：获取指针类型v指向的值
	if v.Kind() != Ptr {
		return v
	}
	return v.Elem()
}

// ValueOf 返回一个新值，该值初始化为接口i中存储的具体值。 ValueOf（nil）返回零值。
func ValueOf(i interface{}) Value { //注：将i转储到堆中，返回Value格式
	if i == nil {
		return Value{}
	}

	// TODO：也许允许Value的内容存在于堆栈中。
	//现在，我们使内容始终转储到堆中。 它使某些地方的生活更加轻松（请参阅下面的chanrecv / mapassign评论）。
	escapes(i) //注：将i转储到堆种

	return unpackEface(i) //注：将i转为Value格式
}

// Zero 返回一个值，该值表示指定类型的零值。
// 结果不同于Value结构的零值，该值根本不代表任何值。
// 例如，Zero(TypeOf(42)) 返回具有Kind Int和值0的值。
// 返回的值既不可寻址也不可设置。
func Zero(typ Type) Value { // 注：返回typ类型的零值
	if typ == nil {
		panic("reflect: Zero(nil)") // 恐慌："Zero(nil)"
	}
	t := typ.(*rtype)
	fl := flag(t.Kind())
	if ifaceIndir(t) {
		return Value{t, unsafe_New(t), fl | flagIndir}
	}
	return Value{t, nil, fl}
}

// New 返回一个值，该值表示指向指定类型的新零值的指针。 也就是说，返回的值的类型为PtrTo(typ).
func New(typ Type) Value { // 注：获取一个新创建的typ类型的指针反射类型
	if typ == nil {
		panic("reflect: New(nil)") // 恐慌："New(nil)"
	}
	t := typ.(*rtype)
	ptr := unsafe_New(t)
	fl := flag(Ptr)
	return Value{t.ptrTo(), ptr, fl}
}

// NewAt 返回一个值，该值表示指向指定类型的值的指针，并使用p作为该指针。
func NewAt(typ Type, p unsafe.Pointer) Value { // 注：#
	fl := flag(Ptr)
	t := typ.(*rtype)
	return Value{t.ptrTo(), p, fl}
}

// assignTo 返回可以直接分配给typ的值v。
// 如果无法将v分配给typ，则会出现紧急情况。
// 要转换为接口类型，target是建议使用的暂存空间。
// 目标必须是初始化的内存（或nil）。
func (v Value) assignTo(context string, dst *rtype, target unsafe.Pointer) Value { // 注：#
	if v.flag&flagMethod != 0 {
		v = makeMethodValue(context, v) // 注：#
	}

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
		x := valueInterface(v, false)
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

// Convert 返回转换为类型t的值v。
// 如果通常的Go转换规则不允许将值v转换为类型t，请转换恐慌。
func (v Value) Convert(t Type) Value { // 注：获取转换为t类型的v
	if v.flag&flagMethod != 0 {
		v = makeMethodValue("Convert", v)
	}
	op := convertOp(t.common(), v.typ) // 注：获取将v的类型转为t的类型的方法
	if op == nil {                     // 注：如果没有这个方法，则恐慌
		panic("reflect.Value.Convert: value of type " + v.typ.String() + " cannot be converted to type " + t.String()) // 恐慌："类型v的值不能转换为类型t"
	}
	return op(v, t)
}

// convertOp 返回将src类型的值转换为dst类型的值的函数。 如果转换非法，则convertOp返回nil。
func convertOp(dst, src *rtype) func(Value, Type) Value { // 注：获取将src类型的值转换为dst类型的值的函数
	switch src.Kind() {
	case Int, Int8, Int16, Int32, Int64: // 注：为int类型提供转为int、float、string类型的方法
		switch dst.Kind() {
		case Int, Int8, Int16, Int32, Int64, Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
			return cvtInt
		case Float32, Float64:
			return cvtIntFloat
		case String:
			return cvtIntString
		}

	case Uint, Uint8, Uint16, Uint32, Uint64, Uintptr: // 注：为uint类型提供转换为int、floag、string类型的方法
		switch dst.Kind() {
		case Int, Int8, Int16, Int32, Int64, Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
			return cvtUint
		case Float32, Float64:
			return cvtUintFloat
		case String:
			return cvtUintString
		}

	case Float32, Float64: // 注：为float类型提供转换为int、float类型的方法
		switch dst.Kind() {
		case Int, Int8, Int16, Int32, Int64:
			return cvtFloatInt
		case Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
			return cvtFloatUint
		case Float32, Float64:
			return cvtFloat
		}

	case Complex64, Complex128: // 注：为复杂类型提供转换为复杂类型的方法
		switch dst.Kind() {
		case Complex64, Complex128:
			return cvtComplex
		}

	case String: // 注：为字符串类型提供转换为int类型切片的方法
		if dst.Kind() == Slice && dst.Elem().PkgPath() == "" {
			switch dst.Elem().Kind() {
			case Uint8:
				return cvtStringBytes
			case Int32:
				return cvtStringRunes
			}
		}

	case Slice: // 注：为切片类型提供转换为字符串类型的方法
		if dst.Kind() == String && src.Elem().PkgPath() == "" {
			switch src.Elem().Kind() {
			case Uint8:
				return cvtBytesString
			case Int32:
				return cvtRunesString
			}
		}

	case Chan: // 注：为管道类型提供转换为管道类型的方法
		if dst.Kind() == Chan && specialChannelAssignability(dst, src) { // 注：#
			return cvtDirect
		}
	}

	// dst和src具有相同的基础类型。
	if haveIdenticalUnderlyingType(dst, src, false) { // 注：#
		return cvtDirect
	}

	// dst和src是具有相同基础基本类型的未定义指针类型。
	if dst.Kind() == Ptr && dst.Name() == "" &&
		src.Kind() == Ptr && src.Name() == "" &&
		haveIdenticalUnderlyingType(dst.Elem().common(), src.Elem().common(), false) {
		return cvtDirect
	}

	if implements(dst, src) { // 注：src可以实现接口类型dst
		if src.Kind() == Interface { // 注：src也是接口，返回接口转接口方法
			return cvtI2I
		}
		return cvtT2I // 注：#
	}

	return nil
}

// makeInt 返回类型t的值，该值等于位（可能被截断），其中t是有符号或无符号int类型。
func makeInt(f flag, bits uint64, t Type) Value { // 注：获取int的反射类型，类型为t，值为bits，属性为f
	typ := t.common()
	ptr := unsafe_New(typ)
	switch typ.size {
	case 1:
		*(*uint8)(ptr) = uint8(bits)
	case 2:
		*(*uint16)(ptr) = uint16(bits)
	case 4:
		*(*uint32)(ptr) = uint32(bits)
	case 8:
		*(*uint64)(ptr) = bits
	}
	return Value{typ, ptr, f | flagIndir | flag(typ.Kind())}
}

// makeFloat 返回类型t等于v的值（可能被截断为float32），其中t是float32或float64类型。
func makeFloat(f flag, v float64, t Type) Value { // 注：获取float的反射类型，类型为t，值为v，属性为f
	typ := t.common()
	ptr := unsafe_New(typ)
	switch typ.size {
	case 4:
		*(*float32)(ptr) = float32(v)
	case 8:
		*(*float64)(ptr) = v
	}
	return Value{typ, ptr, f | flagIndir | flag(typ.Kind())}
}

// makeComplex 返回类型t等于v的值（可能被截断为complex64），其中t是complex64或complex128类型。
func makeComplex(f flag, v complex128, t Type) Value { // 注：获取complex的反射类型，类型为t，值为v，属性为f
	typ := t.common()
	ptr := unsafe_New(typ)
	switch typ.size {
	case 8:
		*(*complex64)(ptr) = complex64(v)
	case 16:
		*(*complex128)(ptr) = v
	}
	return Value{typ, ptr, f | flagIndir | flag(typ.Kind())}
}

func makeString(f flag, v string, t Type) Value { // 注：获取string的反射类型，类型为t，值为v，属性为f
	ret := New(t).Elem()
	ret.SetString(v)
	ret.flag = ret.flag&^flagAddr | f
	return ret
}

func makeBytes(f flag, v []byte, t Type) Value { // 注：获取[]byte的反射类型，类型为t，值为v，属性为f
	ret := New(t).Elem()
	ret.SetBytes(v)
	ret.flag = ret.flag&^flagAddr | f
	return ret
}

func makeRunes(f flag, v []rune, t Type) Value { // 注：获取[]rune的反射类型，类型为t，值为v，属性为f
	ret := New(t).Elem()
	ret.setRunes(v)
	ret.flag = ret.flag&^flagAddr | f
	return ret
}

// 这些转换函数由convertOp返回，用于转换类。
// 例如，第一个函数cvtInt接受带符号int类型的任何值v并返回转换为类型t的值，其中t是任何带符号或无符号int类型。
// convertOp: intXX -> [u]intXX
func cvtInt(v Value, t Type) Value { // 注：int转int
	return makeInt(v.flag.ro(), uint64(v.Int()), t)
}

// convertOp: uintXX -> [u]intXX
func cvtUint(v Value, t Type) Value { // 注：uint转int
	return makeInt(v.flag.ro(), v.Uint(), t)
}

// convertOp: floatXX -> intXX
func cvtFloatInt(v Value, t Type) Value { // 注：float转int
	return makeInt(v.flag.ro(), uint64(int64(v.Float())), t)
}

// convertOp: floatXX -> uintXX
func cvtFloatUint(v Value, t Type) Value { // 注：float转uint
	return makeInt(v.flag.ro(), uint64(v.Float()), t)
}

// convertOp: intXX -> floatXX
func cvtIntFloat(v Value, t Type) Value { // 注：int转float
	return makeFloat(v.flag.ro(), float64(v.Int()), t)
}

// convertOp: uintXX -> floatXX
func cvtUintFloat(v Value, t Type) Value { // 注：uint转float
	return makeFloat(v.flag.ro(), float64(v.Uint()), t)
}

// convertOp: floatXX -> floatXX
func cvtFloat(v Value, t Type) Value { // 注：float转float
	return makeFloat(v.flag.ro(), v.Float(), t)
}

// convertOp: complexXX -> complexXX
func cvtComplex(v Value, t Type) Value { // 注：complex转complex
	return makeComplex(v.flag.ro(), v.Complex(), t)
}

// convertOp: intXX -> string
func cvtIntString(v Value, t Type) Value { // 注：int转string
	return makeString(v.flag.ro(), string(v.Int()), t)
}

// convertOp: uintXX -> string
func cvtUintString(v Value, t Type) Value { // 注：uint转string
	return makeString(v.flag.ro(), string(v.Uint()), t)
}

// convertOp: []byte -> string
func cvtBytesString(v Value, t Type) Value { // 注：[]byte转string
	return makeString(v.flag.ro(), string(v.Bytes()), t)
}

// convertOp: string -> []byte
func cvtStringBytes(v Value, t Type) Value { // 注：string换[]byte
	return makeBytes(v.flag.ro(), []byte(v.String()), t)
}

// convertOp: []rune -> string
func cvtRunesString(v Value, t Type) Value { // 注：[]rune转string
	return makeString(v.flag.ro(), string(v.runes()), t)
}

// convertOp: string -> []rune
func cvtStringRunes(v Value, t Type) Value { // 注：string转[]rune
	return makeRunes(v.flag.ro(), []rune(v.String()), t)
}

// convertOp：直接复制
func cvtDirect(v Value, typ Type) Value { // 注：#
	f := v.flag
	t := typ.common()
	ptr := v.ptr
	if f&flagAddr != 0 { // 注：如果addr为1
		// 间接, mutable word - make a copy
		c := unsafe_New(t)
		typedmemmove(t, c, ptr)
		ptr = c
		f &^= flagAddr // 注：addr设置为0
	}
	return Value{t, ptr, v.flag.ro() | f} // v.flag.ro()|f == f?
}

// convertOp: concrete -> interface
func cvtT2I(v Value, typ Type) Value { // 注：#
	target := unsafe_New(typ.common())
	x := valueInterface(v, false)
	if typ.NumMethod() == 0 {
		*(*interface{})(target) = x
	} else {
		ifaceE2I(typ.(*rtype), x, target)
	}
	return Value{typ.common(), target, v.flag.ro() | flagIndir | flag(Interface)}
}

// convertOp: interface -> interface
func cvtI2I(v Value, typ Type) Value { // 注：#
	if v.IsNil() {
		ret := Zero(typ)
		ret.flag |= v.flag.ro()
		return ret
	}
	return cvtT2I(v.Elem(), typ)
}

// implemented in ../runtime
func chancap(ch unsafe.Pointer) int // 注：获取管道类型ch的容量
func chanclose(ch unsafe.Pointer)   // 注：关闭管道ch
func chanlen(ch unsafe.Pointer) int // 注：获取管道类型ch的长度

// 注意：下面的一些noescape注释从技术上来说是一个谎言，但在此软件包的上下文中是安全的。
// 诸如chansend和mapassign之类的功能不会转义引用对象，但可能会转义引用对象指向的任何内容（它们会做引用对象的浅表副本）。
// 在此包中是安全的，因为引用对象只能指向Value可能指向的内容，并且始终位于堆中（由于ValueOf中的escapes()调用）。

//go:noescape
func chanrecv(ch unsafe.Pointer, nb bool, val unsafe.Pointer) (selected, received bool)

//go:noescape
func chansend(ch unsafe.Pointer, val unsafe.Pointer, nb bool) bool

func makechan(typ *rtype, size int) (ch unsafe.Pointer) // 注：创建类型为type，缓冲区大小为size的双向通道ch
func makemap(t *rtype, cap int) (m unsafe.Pointer)      // 注：创建类型为t，容量为iecap的集合

//go:noescape
func mapaccess(t *rtype, m unsafe.Pointer, key unsafe.Pointer) (val unsafe.Pointer)

//go:noescape
func mapassign(t *rtype, m unsafe.Pointer, key, val unsafe.Pointer)

//go:noescape
func mapdelete(t *rtype, m unsafe.Pointer, key unsafe.Pointer)

// m转义为返回值，但是mapiterinit的调用者不会让返回值转义。
//go:noescape
func mapiterinit(t *rtype, m unsafe.Pointer) unsafe.Pointer // 注：#初始化t类型的集合，数据指针为m

//go:noescape
func mapiterkey(it unsafe.Pointer) (key unsafe.Pointer)

//go:noescape
func mapiterelem(it unsafe.Pointer) (elem unsafe.Pointer)

//go:noescape
func mapiternext(it unsafe.Pointer) // 注：#

//go:noescape
func maplen(m unsafe.Pointer) int // 注：获取集合的长度

// call 调用fn，并复制arg指向的n个参数字节。
// 在返回fn之后，reflectcall将n-retoffset结果字节复制回arg + retoffset，然后再返回。
// 如果将结果字节复制回去，则调用者必须将参数帧类型作为argtype传递，以便调用可以在复制期间执行适当的写障碍。
//
//go:linkname call runtime.reflectcall
func call(argtype *rtype, fn, arg unsafe.Pointer, n uint32, retoffset uint32)

func ifaceE2I(t *rtype, src interface{}, dst unsafe.Pointer)

// memmove 将大小字节从src复制到dst。 不使用写障碍。
//go:noescape
func memmove(dst, src unsafe.Pointer, size uintptr) // 注：将size字节的src复制到dst

// typedmemmove 将类型t的值从src复制到dst。
//go：noescape
func typedmemmove(t *rtype, dst, src unsafe.Pointer) // 注：将t类型的src复制到dst

// typedmemmovepartial类似于typedmemmove，但假设dst和src将字节指向值，并且仅复制大小字节。
//go:noescape
func typedmemmovepartial(t *rtype, dst, src unsafe.Pointer, off, size uintptr) // 注：#

// typedmemclr 将类型t的ptr处的值清零。
//go:noescape
func typedmemclr(t *rtype, ptr unsafe.Pointer) // 注：#

// typedmemclrpartial 与typedmemclr类似，但假设dst将字节指向该值，并且仅清除大小字节。
//go:noescape
func typedmemclrpartial(t *rtype, ptr unsafe.Pointer, off, size uintptr) // 注：#

// typedslicecopy 将elemType值的一部分从src复制到dst，返回复制的元素数。
//go：noescape
func typedslicecopy(elemType *rtype, dst, src sliceHeader) int // 注：将elemType类型的src复制给dst，返回赋值的元素数

//go:noescape
func typehash(t *rtype, p unsafe.Pointer, h uintptr) uintptr // 注：#

// Dummy 注释值x的转义，用于反射代码非常聪明以至于编译器无法遵循的情况。
func escapes(x interface{}) { // 注：将x注释标记为逃逸
	if dummy.b {
		dummy.x = x
	}
}

var dummy struct {
	b bool
	x interface{}
}
