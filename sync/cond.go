// 版权所有2011 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package sync

import (
	"sync/atomic"
	"unsafe"
)

// Cond 实现一个条件变量，它是goroutines等待或宣布事件发生的集合点。
// 每个Cond都有一个关联的Locker L（通常是*Mutex或*RWMutex），在更改条件和调用Wait方法时必须将其保留。
// 第一次使用后不得复制条件。
type Cond struct {
	noCopy noCopy

	// L 是在观察或改变条件时持有
	L Locker

	notify  notifyList
	checker copyChecker
}

// NewCond 返回带有锁l的新Cond。
func NewCond(l Locker) *Cond { // 工厂函数，生成一个Cond结构体
	return &Cond{L: l}
}

// Wait 原子地解锁c.L并中止调用goroutine的执行。
// 稍后恢复执行后，等待锁定c.L，然后再返回。
// 与其他系统不同，等待不会返回，除非被广播或信号唤醒。
//
// 因为在等待第一次恢复时c.L未被锁定，所以调用者通常无法假定等待返回时条件为真。
// 而是，调用者应在循环中等待：
//
//    c.L.Lock()
//    for !condition() {
//        c.Wait()
//    }
//    ... 利用条件 ...
//    c.L.Unlock()
//
func (c *Cond) Wait() { // 注：#
	c.checker.check()                     // 注：检查c是否发生拷贝
	t := runtime_notifyListAdd(&c.notify) // 注：
	c.L.Unlock()
	runtime_notifyListWait(&c.notify, t)
	c.L.Lock()
}

// Signal 唤醒一个等待在c上的goroutine，如果有的话。
//
// 在通话过程中，允许但不要求呼叫者保持c.L。
func (c *Cond) Signal() { // 注：#
	c.checker.check()
	runtime_notifyListNotifyOne(&c.notify)
}

// Broadcast wakes all goroutines waiting on c.
//
// It is allowed but not required for the caller to hold c.L
// during the call.
func (c *Cond) Broadcast() {
	c.checker.check()
	runtime_notifyListNotifyAll(&c.notify)
}

// copyChecker 保持指针指向自身，以检测对象复制。
type copyChecker uintptr // 注：

func (c *copyChecker) check() { // 注：检查c是否发生拷贝
	if uintptr(*c) != uintptr(unsafe.Pointer(c)) && // 注：检查c是否被拷贝
		!atomic.CompareAndSwapUintptr((*uintptr)(c), 0, uintptr(unsafe.Pointer(c))) && // 注：#将c的指针赋值给c的指针
		uintptr(*c) != uintptr(unsafe.Pointer(c)) { // 注：再次检查c是否被拷贝
		panic("sync.Cond is copied") // 恐慌："发生拷贝"
	}
}

// noCopy 可以嵌入到第一次使用后不得复制的结构中。
// 有关详细信息，请参见https://golang.org/issues/8005#issuecomment-190753527。
type noCopy struct{}

// Lock 是`go vet`中的-copylocks检查器使用的无操作项。
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
