// 版权所有2014 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package atomic

import (
	"unsafe"
)

// Value 提供原子加载并存储一致类型的值。
// Value的零值从Load返回nil。
// 调用Store后，不得复制Value。
// 首次使用后，不得复制Value。
type Value struct {
	v interface{}
}

// ifaceWords 是interface{}内部表示形式。
type ifaceWords struct { // 注：表现为空接口的反射类型
	typ  unsafe.Pointer
	data unsafe.Pointer
}

// Load 返回由最新Store设置的值。
// 如果没有针对此值的存储调用，则返回nil。
func (v *Value) Load() (x interface{}) { // 注：原子性的将v转为interface{}类型
	// 步骤：
	// 1. 加载v的类型，读取不到表示为v还没有存入值
	// 2. 加载v的数据，将数据赋值给x
	vp := (*ifaceWords)(unsafe.Pointer(v))         // 注：将v转为空接口
	typ := LoadPointer(&vp.typ)                    // 注：原子性的获取v的类型
	if typ == nil || uintptr(typ) == ^uintptr(0) { // 注：#未取出v的类型，返回nil
		// 第一次Store尚未完成。
		return nil
	}
	data := LoadPointer(&vp.data)           // 注：原子性的获取v的数据
	xp := (*ifaceWords)(unsafe.Pointer(&x)) // 注：赋值给空接口
	xp.typ = typ
	xp.data = data
	return
}

// Store 将Value的值设置为x。
// 对给定值的所有Store调用都必须使用相同具体类型的值。
// 存储不一致类型的panic，Store(nil)也是如此。
func (v *Value) Store(x interface{}) { // 注：原子性的将v赋值给v
	// 步骤：
	// 1. 开始自旋锁
	// 		2. 加载v的类型，直到

	if x == nil {
		panic("sync/atomic: store of nil value into Value") // 恐慌："将零值存储到Value中"
	}
	vp := (*ifaceWords)(unsafe.Pointer(v))
	xp := (*ifaceWords)(unsafe.Pointer(&x))
	for { // 注：无限循环，直到Store完成
		typ := LoadPointer(&vp.typ) // 注：#原子性的获取v的类型
		if typ == nil {             // 注：如果没有获取到v的类型，进行抢占
			// 尝试启动第一次Store。
			// 禁用抢占，以便其他goroutine可以使用活动的spin wait等待完成； 这样GC不会偶然看到伪造的类型。
			runtime_procPin()                                                      // 注：禁用抢占
			if !CompareAndSwapPointer(&vp.typ, nil, unsafe.Pointer(^uintptr(0))) { // 注：原子性的比较与交换失败，对v的抢占失败
				runtime_procUnpin() // 注：启用抢占
				continue
			}
			// 完成第一次Store
			StorePointer(&vp.data, xp.data) // 注：原子性的存储x的类型给v
			StorePointer(&vp.typ, xp.typ)   // 注：原子性的存储x的数据给v
			runtime_procUnpin()             // 注：启用抢占
			return
		}
		if uintptr(typ) == ^uintptr(0) { // 注：#v的类型获取失败
			// 进行中的第一次Stroe。 等待。
			// 由于我们在第一次Store附近禁用了抢占，因此我们可以等待主动旋转。
			continue
		}
		// 第一次Store完成。检查类型并覆盖数据。
		if typ != xp.typ {
			panic("sync/atomic: store of inconsistently typed value into Value") // 恐慌："将不一致类型的值存储到Value中"
		}
		StorePointer(&vp.data, xp.data) // 注：如果直接抢占到了v，直接赋值
		return
	}
}

// 禁用/启用抢占，在运行时实现。
func runtime_procPin()   // 注：禁用抢占
func runtime_procUnpin() // 注：启用抢占
