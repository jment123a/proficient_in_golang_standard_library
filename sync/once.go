// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package sync

import (
	"sync/atomic"
)

// Once 是将仅执行一项操作的对象。
type Once struct { // 注：每个Once只会执行一次Do
	// done 指示操作是否已执行。
	// 它在结构中是第一个，因为它在热路径中使用。
	// 在每个呼叫站点内联热路径。
	// 首先完成放置，可以在某些体系结构（amd64/x86）上使用更紧凑的指令，而在其他体系结构上使用更少的指令（用于计算偏移量）。
	done uint32
	m    Mutex
}

// Do 仅当且仅当调用函数f，如果是第一次为Once实例调用Do，换句话说，给定
// 	var once Once
// 如果多次调用了once.Do(f)，即使f在每次调用中具有不同的值，也只有第一次调用会调用f。每个函数要执行都需要一个新的Once实例。
// Do是用于初始化的，必须仅运行一次。 由于f是niladic，因此可能有必要使用函数文字来捕获由Do调用的函数的参数：
// 	config.once.Do(func() { config.init(filename) })
// 因为对Do的调用直到返回对f的一次调用才返回，所以如果f导致调用Do，它将死锁。
// 如果出现紧急情况，Do会认为它已返回； Do的未来调用将不调用而返回f。
func (o *Once) Do(f func()) { // 注：o执行一次f，之后的Do不会再执行
	// 注意：这是Do的错误实现：
	//	if atomic.CompareAndSwapUint32(&o.done, 0, 1) {
	//		f()
	//	}
	// 确保返回时f已完成。
	// 此实现不会实现该保证：
	// 	如果同时进行了两次调用，则cas的获胜者将调用f，第二个将立即返回，而无需等待第一个对f的调用完成。
	// 	这就是为什么慢速路径回退到互斥体的原因，以及为什么atomic.StoreUint32必须延迟到f返回之后的原因。

	if atomic.LoadUint32(&o.done) == 0 { // 注：原子性读取o是否完成
		// 概述了慢速路径，以允许快速路径的内联。
		o.doSlow(f)
	}
}

func (o *Once) doSlow(f func()) { // 注：设置锁，执行f，设置o已完成，释放锁
	o.m.Lock()         // 注：设置锁
	defer o.m.Unlock() // 注：释放锁
	if o.done == 0 {
		defer atomic.StoreUint32(&o.done, 1) // 注：设置o已完成
		f()                                  // 注：执行f
	}
}
