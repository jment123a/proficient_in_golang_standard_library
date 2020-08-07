// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

// Package unsafe 中包含的操作会绕过Go程序的类型安全。
// 导入unsafe包可能是不可移植的，并且不受Go 1兼容性准则的保护。
package unsafe

// ArbitraryType 此处仅出于文档目的，实际上并不属于unsafe包。 它表示任意Go表达式的类型。
type ArbitraryType int

// Pointer 表示指向任意类型的指针。 有四个特殊操作
// 可用于Pointer类型，而不适用于其他类型：
// -任何类型的指针值都可以转换为Pointer。
// -可以将Pointer转换为任何类型的指针值。
// -可以将uintptr转换为Pointer。
// -可以将Pointer转换为uintptr。
// Pointer因此允许程序击败类型系统并读写任意内存。 使用时应格外小心。
//
// 以下涉及Pointer的模式是有效的。
// 不使用这些模式的代码今天可能无效，或者将来可能无效。
// 即使下面的有效模式也带有重要的警告。
//
// 运行"go vet"可以帮助查找不符合这些模式的Pointer用法，但是对"go vet"的静默并不能保证代码有效。
//
// (1) 将*T1转换为Pointer转换到*T2.
// 假定T2不大于T1，并且两个共享相同的内存布局，则此转换允许将一种类型的数据重新解释为另一种类型的数据。
// 一个示例是math.Float64bits的实现：
//	func Float64bits(f float64) uint64 {
//		return *(*uint64)(unsafe.Pointer(&f))
//	}
//
// (2) 将Pointer转换为uintptr（但不返回给Pointer）。
// 将Pointer转换为uintptr会生成指向整数的指针的值的内存地址。 这种uintptr的通常用法是打印它。
// 将uintptr转换回Pointer通常是无效的。
// uintptr是整数，而不是引用。
// 将Pointer转换为uintptr会创建一个没有指针语义的整数值。
// 即使uintptr拥有某个对象的地址，垃圾回收器也不会在对象移动时更新该uintptr的值，也不会阻止uintptr回收该对象。
// 其余模式列举从uintptr到Pointer的唯一有效转换。
//
// (3) 用算术将Pointer转换为uintptr并返回。
// 如果p指向已分配的对象，则可以通过转换为uintptr，添加偏移量并将其转换回Pointer的方式将其推进对象。
//	p = unsafe.Pointer(uintptr(p) + offset)
// 此模式最常见的用法是访问结构体或数组元素中的字段：
//
//	// 相当于f := unsafe.Pointer(&s.f)
//	f := unsafe.Pointer(uintptr(unsafe.Pointer(&s)) + unsafe.Offsetof(s.f))
//
//	// 相当于e := unsafe.Pointer(&x[i])
//	e := unsafe.Pointer(uintptr(unsafe.Pointer(&x[0])) + i*unsafe.Sizeof(x[0]))
//
// 以这种方式从指针添加和减去偏移量都是有效的。
// 使用＆^舍入指针（通常用于对齐）也是有效的。
// 在所有情况下，结果都必须继续指向原始分配的对象。
// 与C语言不同，将指针移到C的末尾是无效的
// 其原始分配：
//
//	//无效：端点在分配的空间之外。
//	var s thing
//	end = unsafe.Pointer(uintptr(unsafe.Pointer(&s)) + unsafe.Sizeof(s))
//
//	//无效：端点在分配的空间之外。
//	b := make([]byte, n)
//	end = unsafe.Pointer(uintptr(unsafe.Pointer(&b[0])) + uintptr(n))
//
// 请注意，两个转换必须出现在同一个表达式中，并且它们之间只有中间的算术运算：
//
// //无效：uintptr在转换回Pointer之前不能存储在变量中。
//	u := uintptr(p)
//	p = unsafe.Pointer(u + offset)
//
// 请注意，指针必须指向已分配的对象，因此它不能为nil。
//
//	// 无效：nil指针的转换
//	u := unsafe.Pointer(nil)
//	p := unsafe.Pointer(uintptr(u) + offset)
//
// (4) 调用syscall.Syscall时将指针转换为uintptr。
// 包syscall中的Syscall函数将其uintptr参数直接传递给操作系统，然后，操作系统可以根据调用的详细信息将其中一些参数重新解释为指针。
// 也就是说，系统调用实现将某些参数从uintptr隐式转换回指针。
// 如果必须将指针参数转换为uintptr用作参数，则该转换必须出现在调用表达式本身中：
//
//	syscall.Syscall(SYS_READ, uintptr(fd), uintptr(unsafe.Pointer(p)), uintptr(n))
//
// 编译器通过安排引用的已分配对象来处理在汇编中实现的函数的调用的参数列表中转换为uintptr的Pointer，
// 如果有的话，将保留并直到调用完成才移动它，即使仅从类型来看，似乎在调用过程中也不再需要该对象。
// 为了使编译器能够识别这种模式，转换必须出现在参数列表中：
//
// //无效：在系统调用期间隐式转换回Pointer之前，不能将uintptr存储在变量中。
//	u := uintptr(unsafe.Pointer(p))
//	syscall.Syscall(SYS_READ, uintptr(fd), u, uintptr(n))
//
// (5) 将reflect.Value.Pointer或reflect.Value.UnsafeAddr的结果从uintptr转换为Pointer。
// 包reflect的名为Pointer和UnsafeAddr的值方法返回uintptr类型而不是unsafe.Pointer，
// 以防止调用者将结果更改为任意类型，而无需先导入"unsafe"。
// 但是，这意味着结果很脆弱，必须在调用后立即使用相同的表达式将其转换为Pointer：
//
//	p := (*int)(unsafe.Pointer(reflect.ValueOf(new(int)).Pointer()))
//
// 与上述情况一样，在转换之前存储结果是无效的：
//
//	//无效：uintptr无法在转换回Pointer之前存储在变量中。
//	u := reflect.ValueOf(new(int)).Pointer()
//	p := (*int)(unsafe.Pointer(u))
//
// (6)将reflect.SliceHeader或reflect.StringHeader数据字段与指针进行转换。
// 与前面的情况一样，反射数据结构SliceHeader和StringHeader将字段Data声明为uintptr，
// 以防止调用者将结果更改为任意类型，而无需首先导入"unsafe"。
// 但是，这意味着SliceHeader和StringHeader仅在解释实际切片或字符串值的内容时才有效。
//
//	var s string
//	hdr := (*reflect.StringHeader)(unsafe.Pointer(&s)) // case 1
//	hdr.Data = uintptr(unsafe.Pointer(p))              // case 6 (this case)
//	hdr.Len = n
//
// 在这种用法中，hdr.Data实际上是在字符串标题中引用基础指针的另一种方法，而不是uintptr变量本身。
//
// 通常，reflect.SliceHeader和reflect.StringHeader只能用作*reflect.SliceHeader和*reflect.StringHeader指向实际的切片或字符串，而不能用作纯结构。
// 程序不应声明或分配这些结构类型的变量。
//
//	//无效：直接声明的标头将不保存数据作为引用。
//	var hdr reflect.StringHeader
//	hdr.Data = uintptr(unsafe.Pointer(p))
//	hdr.Len = n
//	s := *(*string)(unsafe.Pointer(&hdr)) // p可能已经丢失
//
type Pointer *ArbitraryType

// Sizeof 接受任何类型的表达式x并返回假设变量v的字节大小，就好像v是通过var v = x声明的一样。
// 该大小不包含x可能引用的任何内存。
// 例如，如果x是切片，则Sizeof返回切片描述符的大小，而不是该切片所引用的内存的大小。
// Sizeof的返回值是Go常量。
func Sizeof(x ArbitraryType) uintptr //注：返回x的描述大小，不包含引用的内存，返回的不是引用内存的大小

// Offsetof 返回x表示的字段的结构体内的偏移量，
// 该偏移量的格式必须为structValue.field。
// 换句话说，它返回结构开始与字段开始之间的字节数。
// Offsetof的返回值是Go常数。
func Offsetof(x ArbitraryType) uintptr //注：返回x在所在的结构体内的偏移量

// Alignof 接受任何类型的表达式x并返回假设变量v的所需对齐方式，就好像v是通过var v = x声明的一样。
// 这是最大值m，因此v的地址始终为0 mod m。
// 它与reflect.TypeOf(x).Align()返回的值相同。
// 作为一种特殊情况，如果变量s是结构类型，而f是该结构中的字段，
// 则Alignof(s.f)将返回结构中该类型字段的所需对齐方式。
// 这种情况与reflect.TypeOf(s.f).FieldAlign()返回的值相同。
// Alignof的返回值是Go常量。
func Alignof(x ArbitraryType) uintptr //注：返回x所需的对齐方式
