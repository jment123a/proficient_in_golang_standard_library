// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

// 通过反射进行深度平等测试

package reflect

import "unsafe"

// 在deepValueEqual期间，必须跟踪正在进行的检查。
// 比较算法假定重新遇到它们时，所有进行中的检查都是真实的。
// 访问的比较存储在按访问索引的地图中。
type visit struct {
	a1  unsafe.Pointer
	a2  unsafe.Pointer
	typ Type
}

// 使用反射类型测试深度相等性。 map参数跟踪已经看到的比较，这允许对递归类型进行短路。

/*
	注：
	数组：比较每个元素
	切片：比较是否分配内存、长度、地址、每个元素
	接口：比较是否均为nil、数据
	指针：比较地址、数据
	结构：比较每个成员
	集合：比较是否分配内存、长度、地址、每个索引
	方法：比较是否均为nil
	其他：==比较
*/
func deepValueEqual(v1, v2 Value, visited map[visit]bool, depth int) bool { // 注：v1与v2是否深度相等，visited为已经比较过的地址，depth为深度（调试用）
	if !v1.IsValid() || !v2.IsValid() {
		return v1.IsValid() == v2.IsValid() // 注：是否为不合法数字
	}
	if v1.Type() != v2.Type() {
		return false
	}

	// if depth > 10 { panic("deepValueEqual") }	// for debugging

	// 我们希望避免在访问的visited中放置过多的内容。
	// 对于可能遇到的任何可能的参考循环，hard(v1，v2)需要为循环中的至少一种类型返回true，并且获取Value的内部指针是安全有效的。
	hard := func(v1, v2 Value) bool { // 注：v1和v2是（Map, Slice, Ptr, Interface）类型且都分配了地址返回true
		switch v1.Kind() {
		case Map, Slice, Ptr, Interface:
			// Nil指针不能是循环的。 避免将它们放在访问过的map中。
			return !v1.IsNil() && !v2.IsNil()
		}
		return false
	}

	if hard(v1, v2) { // 注：（Map, Slice, Ptr, Interface）类型且都分配了地址
		// 对于Ptr或Map值，我们需要检查flagIndir，方法是调用指针方法。
		// 对于Slice或Interface，始终设置flagIndir，并且使用v.ptr就足够了。
		ptrval := func(v Value) unsafe.Pointer { // 注：ptr与map要检查是否间接引用
			switch v.Kind() {
			case Ptr, Map:
				return v.pointer()
			default:
				return v.ptr
			}
		}
		addr1 := ptrval(v1)                  // 注：取出v1指向数据的地址
		addr2 := ptrval(v2)                  // 注：取出v2指向数据的地址
		if uintptr(addr1) > uintptr(addr2) { // 注：保证v1的地址比v2小
			// 规范顺序以减少访问的条目数。
			// 假定移动垃圾回收器。
			addr1, addr2 = addr2, addr1
		}

		// 学习：如果v1和v2不深度相等，那么整个函数会返回false，这里记住v为true就没用了
		// 如果为true，下次再遇到v1和v2将会直接返回true

		// 如果已经看到参考，则短路。
		typ := v1.Type()
		v := visit{addr1, addr2, typ}
		if visited[v] { // 注：visited已存在则返回true
			return true
		}

		// 记住以备后用。
		visited[v] = true

	}

	switch v1.Kind() {
	case Array: // 注：数组中的每个元素都执行一次比较，深度+1
		for i := 0; i < v1.Len(); i++ {
			if !deepValueEqual(v1.Index(i), v2.Index(i), visited, depth+1) {
				return false
			}
		}
		return true
	case Slice:
		if v1.IsNil() != v2.IsNil() { // 注：判断内存分配
			return false
		}
		if v1.Len() != v2.Len() { // 注：判断长度
			return false
		}
		if v1.Pointer() == v2.Pointer() { // 注：判断地址
			return true
		}
		for i := 0; i < v1.Len(); i++ { // 注：地址不同、长度相同的数据，每个元素都进行比较，深度+1
			if !deepValueEqual(v1.Index(i), v2.Index(i), visited, depth+1) {
				return false
			}
		}
		return true
	case Interface:
		if v1.IsNil() || v2.IsNil() { // 注：判断内存分配
			return v1.IsNil() == v2.IsNil()
		}
		return deepValueEqual(v1.Elem(), v2.Elem(), visited, depth+1)
	case Ptr:
		if v1.Pointer() == v2.Pointer() { // 注：判断地址
			return true
		}
		return deepValueEqual(v1.Elem(), v2.Elem(), visited, depth+1)
	case Struct:
		for i, n := 0, v1.NumField(); i < n; i++ { // 注：每个成员都进行比较，深度+1
			if !deepValueEqual(v1.Field(i), v2.Field(i), visited, depth+1) {
				return false
			}
		}
		return true
	case Map:
		if v1.IsNil() != v2.IsNil() { // 注：判断内存分配
			return false
		}
		if v1.Len() != v2.Len() { // 注：判断长度
			return false
		}
		if v1.Pointer() == v2.Pointer() { // 注：判断地址
			return true
		}
		// 注：地址不同、长度相同的数据，每个成员都进行比较，深度+1
		for _, k := range v1.MapKeys() { // 注：遍历所有key
			val1 := v1.MapIndex(k)
			val2 := v2.MapIndex(k)
			if !val1.IsValid() || !val2.IsValid() || !deepValueEqual(val1, val2, visited, depth+1) {
				return false
			}
		}
		return true
	case Func:
		if v1.IsNil() && v2.IsNil() { // 注：func均为nil才为true
			return true
		}
		// 没有比这更好的了:
		return false
	default:
		// 正常平等就足够了
		return valueInterface(v1, false) == valueInterface(v2, false)
	}
}

// DeepEqual 报告x和y是否``深度相等'' ，定义如下。
// 如果满足以下情况之一，则两个相同类型的值将完全相等。
// 不同类型的值永远不会完全相等。
// 当数组值对应的元素高度相等时，数组的值深度相等。
// 如果导出和未导出的对应字段深度相等，则结构值深度相等。
// 如果两者均为nil，则func值非常相等； 否则，它们就不会完全平等。
// 如果接口值具有完全相等的具体值，则它们是高度相等的。
//
// Map 当满足以下所有条件时，集合值就非常相等：
// 它们都是nil或都是非nil，它们具有相同的长度，并且它们是相同的映射对象或它们的对应键（使用Go equals匹配）映射为深度相等的值。
// 如果指针值使用Go的==运算符相等，或者它们指向深度相等的值，则它们的深度相等。
//
// Slice 当满足以下所有条件时，切片值将完全相等：
// 它们均为nil或均为非nil，它们具有相同的长度，并且它们指向同一基础数组的
// 相同初始条目(即, &x[0] == &y[0]) 或它们对应的数组元素（最大长度）非常相等。
// 请注意，非零空片和零片(例如[]byte{} 喝 []byte(nil)) 并不完全相等。
//
// 如果其他值 - 数字，布尔值，字符串和通道 - 如果使用Go的==运算符相等，则它们将非常相等。
//
// 通常，DeepEqual是Go的==运算符的递归松弛。
// 但是，如果没有一些不一致，就不可能实现这个想法。
// 特别是，值可能与自身不相等，可能是因为它是func类型（通常无法比较），或者是浮点NaN值（在浮点比较中不等于其自身），
// 或因为它是包含此类值的数组，结构或接口。
//
// 另一方面，即使指针值指向或包含此类有问题的值，它们也始终等于其自身，
// 因为它们使用Go的==运算符进行比较，并且无论内容如何，都足以使其深度相等 。
// 已定义DeepEqual，以便对切片和贴图应用相同的快捷方式：
// 如果x和y是相同的切片或相同的贴图，则无论内容如何，它们的深度都相等。
//
// 当DeepEqual遍历数据值时，可能会发现一个循环。
// DeepEqual在第二次及以后比较两个之前比较过的指针值时，会将这些值视为相等，而不是检查它们所指向的值。
// 这样可确保DeepEqual终止。
func DeepEqual(x, y interface{}) bool { // 注：获取x与y是否深度相等
	if x == nil || y == nil {
		return x == y
	}
	v1 := ValueOf(x) // 注：将x与y转储到堆中，转为Value
	v2 := ValueOf(y)
	if v1.Type() != v2.Type() { // 注：类型不同返回false
		return false
	}
	return deepValueEqual(v1, v2, make(map[visit]bool), 0)
}
