// 版权所有2016 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。
package reflect

import "unsafe"

// Swapper 返回一个函数，该函数交换提供的slice中的元素。
//
// 如果提供的接口不是切片，则交换器会出现混乱。
func Swapper(slice interface{}) func(i, j int) { // 注：获取使slice的第i个元素与第j个元素交换的方法
	v := ValueOf(slice)
	if v.Kind() != Slice { // 注：slice不是切片，引发恐慌
		panic(&ValueError{Method: "Swapper", Kind: v.Kind()}) // 恐慌："类型错误"
	}
	// 大小为0和1的切片的快速路径。无须交换。
	switch v.Len() { // 注：slice的长度至少为2
	case 0:
		return func(i, j int) { panic("reflect: slice index out of range") } // 恐慌："切片索引超出范围"
	case 1:
		return func(i, j int) {
			if i != 0 || j != 0 {
				panic("reflect: slice index out of range") // 恐慌："切片索引超出范围"
			}
		}
	}

	typ := v.Type().Elem().(*rtype)
	size := typ.Size()
	hasPtr := typ.ptrdata != 0

	// 一些常见的案例，不使用memmove:
	if hasPtr {
		if size == ptrSize { // 注：slice = *[]unsafe.Pointer
			ps := *(*[]unsafe.Pointer)(v.ptr)
			return func(i, j int) { ps[i], ps[j] = ps[j], ps[i] }
		}
		if typ.Kind() == String { // 注：slice = *[]string
			ss := *(*[]string)(v.ptr)
			return func(i, j int) { ss[i], ss[j] = ss[j], ss[i] }
		}
	} else {
		switch size {
		case 8: // 注：slice = *[]int64
			is := *(*[]int64)(v.ptr)
			return func(i, j int) { is[i], is[j] = is[j], is[i] }
		case 4: // 注：slice = *[]int32
			is := *(*[]int32)(v.ptr)
			return func(i, j int) { is[i], is[j] = is[j], is[i] }
		case 2: // 注：slice = *[]int16
			is := *(*[]int16)(v.ptr)
			return func(i, j int) { is[i], is[j] = is[j], is[i] }
		case 1: // 注：slice = *[]int8
			is := *(*[]int8)(v.ptr)
			return func(i, j int) { is[i], is[j] = is[j], is[i] }
		}
	}

	s := (*sliceHeader)(v.ptr)
	tmp := unsafe_New(typ) // 交换暂存空间

	return func(i, j int) {
		if uint(i) >= uint(s.Len) || uint(j) >= uint(s.Len) {
			panic("reflect: slice index out of range") // 恐慌："切片索引超出范围"
		}
		val1 := arrayAt(s.Data, i, size, "i < s.Len") // 注：指针移动到第i个元素的位置
		val2 := arrayAt(s.Data, j, size, "j < s.Len") // 注：指针移动到第j个元素的位置
		typedmemmove(typ, tmp, val1)                  // 注： t = a
		typedmemmove(typ, val1, val2)                 // 注： a = b
		typedmemmove(typ, val2, tmp)                  // 注： b = t
	}
}
