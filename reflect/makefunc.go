// 版权所有2012 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

// MakeFunc实现。
package reflect

import (
	"unsafe"
)

// makeFuncImpl 是实现MakeFunc返回的函数的闭包值。
// 此类型的前三个字必须与methodValue和runtime.reflectMethodValue保持同步。
// 任何更改都应反映在这三个方面。
type makeFuncImpl struct {
	code   uintptr
	stack  *bitVector // 参数和结果的ptrmap
	argLen uintptr    // 只是参数
	ftyp   *funcType
	fn     func([]Value) []Value
}

// MakeFunc 返回给定类型的新函数，该函数包装function fn。 调用时，该新函数将执行以下操作：
// 	-将其参数转换为值的一部分。
// 	-运行 results := fn(args)。
// 	-以值的切片形式返回结果，每个形式结果一个。
// 实现fn可以假设参数Value slice具有typ指定的参数的数量和类型。
// 如果typ描述了可变参数函数，则最终值本身就是代表可变参数的切片，就像可变函数的主体一样。
// fn返回的结果Value slice必须具有typ指定的结果的数量和类型。
// Value.Call方法允许调用者根据Values调用类型化的函数； 相反，MakeFunc允许调用者根据值来实现类型化的函数。
// 文档的"示例"部分包含如何使用MakeFunc为不同类型构建交换函数的说明。
func MakeFunc(typ Type, fn func(args []Value) (results []Value)) Value { // 注：#
	if typ.Kind() != Func {
		panic("reflect: call of MakeFunc with non-Func type") // 恐慌："用非Func类型调用MakeFunc"
	}

	t := typ.common()
	ftyp := (*funcType)(unsafe.Pointer(t))

	// 间接转到func值（虚拟）以获得实际的代码地址。 （Go func值是指向C函数指针的指针。https://golang.org/s/go11func。）
	dummy := makeFuncStub
	code := **(**uintptr)(unsafe.Pointer(&dummy))

	// makeFuncImpl包含供运行时使用的堆栈映射
	_, argLen, _, stack, _ := funcLayout(ftyp, nil)

	impl := &makeFuncImpl{code: code, stack: stack, argLen: argLen, ftyp: ftyp, fn: fn}

	return Value{t, unsafe.Pointer(impl), flag(Func)}
}

// makeFuncStub 是一个汇编函数，是从MakeFunc返回的函数的代码一半。
// 它期望*callReflectFunc作为其上下文寄存器，其工作是调用callReflect(ctxt, frame)，
// 其中ctxt是上下文寄存器，而frame是指向传入参数帧中第一个单词的指针。
func makeFuncStub()

// 此类型的前3个字必须与makeFuncImpl和runtime.reflectMethodValue保持同步。
// 任何更改都应反映在这三个方面。
type methodValue struct {
	fn     uintptr
	stack  *bitVector // 参数和结果的ptrmap
	argLen uintptr    // 只是参数
	method int
	rcvr   Value
}

// makeMethodValue 将v从方法值的rcvr + 方法索引表示形式转换为实际的方法功能值，
// 该值实际上是带有特殊位设置的接收器值，成为真正的功能值-包含实际功能的值。
// 就包Reflect的用户而言，输出在语义上等同于输入，但是真正的func表示可以由Convert和Interface和Assign之类的代码处理。
func makeMethodValue(op string, v Value) Value { // 注：#
	if v.flag&flagMethod == 0 { //注：v没有已导出方法
		panic("reflect: internal error: invalid use of makeMethodValue") //恐慌："内部错误：无效使用makeMethodValue"
	}

	// 忽略flagMethod位，v描述接收方，而不是方法类型。
	fl := v.flag & (flagRO | flagAddr | flagIndir) //注：获取v是否为导出字段、是否分配内存地址、是否为指针
	fl |= flag(v.typ.Kind())                       //注：获取v的类型
	rcvr := Value{v.typ, v.ptr, fl}                //注：Value类型为v的类型，数据为v的指针，v的为是否导出字段、是否分配内存地址、是否为指针与v的类型

	//v.Type返回方法值的实际类型。
	ftyp := (*funcType)(unsafe.Pointer(v.Type().(*rtype)))

	//间接转到func值（虚拟）以获得实际的代码地址。 （Go func值是指向C函数指针的指针。https://golang.org/s/go11func。）
	dummy := methodValueCall
	code := **(**uintptr)(unsafe.Pointer(&dummy))

	// methodValue包含供运行时使用的堆栈映射
	_, argLen, _, stack, _ := funcLayout(ftyp, nil)

	fv := &methodValue{
		fn:     code,
		stack:  stack,
		argLen: argLen,
		method: int(v.flag) >> flagMethodShift,
		rcvr:   rcvr,
	}

	// 如果方法不合适，则会引起恐慌。
	// 如果忽略此错误，则在调用期间仍会发生恐慌，但我们希望Interface（）和其他操作尽早失败。
	methodReceiver(op, fv.rcvr, fv.method) // 注：#

	return Value{&ftyp.rtype, unsafe.Pointer(fv), v.flag&flagRO | flag(Func)}
}

// methodValueCall 是一个汇编函数，是makeMethodValue返回的函数的代码一半。
// 它期望*methodValue作为其上下文寄存器，并且其工作是调用callMethod(ctxt，frame)，
// 其中ctxt是上下文寄存器，而frame是指向传入参数帧中第一个字的指针。
func methodValueCall() // 注：#
