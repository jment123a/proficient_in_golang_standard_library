// 版权所有2016 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package sync

import (
	"sync/atomic"
	"unsafe"
)

// Map 类似于Go map[interface{}]interface{}，但是可以安全地被多个goroutine并发使用，而无需额外的锁定或协调。
// 加载，存储和删除以固定的固定时间运行。
//
// Map类型是特殊的。 大多数代码应改用带有单独锁定或协调功能的普通Go映射，以提高类型安全性，并使其更易于维护其他不变式以及映射内容。
//
// Map类型针对两种常见用例进行了优化：
// （1）给定key的条目仅写入一次但读取多次，例如在仅增长的高速缓存中；
// （2）当多个goroutine读取，写入时 ，并覆盖不相交的key集的条目。
// 在这两种情况下，与与单独的Mutex或RWMutex配对的Go映射相比，使用Map可以显着减少锁争用。
//
// 零Map为空，可以使用。 首次使用后不得复制Map。
type Map struct {
	mu Mutex // 注：互斥锁

	// read 包含Map内容中可安全进行并发访问的部分（带有或不带有mu）。
	// read 字段本身始终可以安全加载，但必须仅在mu保持状态下存储。
	// 存储在read中的条目可以在没有mu的情况下同时更新，但是更新以前删除的条目要求将该条目复制到脏映射，并且在保留mu的情况下不删除它。
	read atomic.Value // 只读

	// dirty 包含Map内容中需要保留mu的部分。 为了确保可以将脏映射快速提升到读取映射，它还包括读取映射中的所有未删除条目。
	// 删除的条目不会存储在脏映射中。 必须先清除干净映射中的已删除条目，然后将其添加到脏映射中，然后才能向其存储新值。
	// 如果脏映射为nil，则对映射的下一次写入将通过创建干净映射的浅拷贝（省略陈旧的条目）来初始化它。
	dirty map[interface{}]*entry

	// misses 计算自上次更新读取映射以来锁定mu以确定密钥是否存在所需的负载数。
	// 一旦发生足够的未命中以支付复制脏映射的成本，该脏映射将被提升为已读映射（处于未修改状态），并且该映射的下一个存储区将创建新的脏副本。
	misses int
}

// readOnly 是原子结构存储在Map.read字段中的不变结构。
type readOnly struct {
	m       map[interface{}]*entry
	amended bool // 如果脏映射包含不在m中的某个键，则为true。注：表示dirty中是否存在m中没有的key
}

// expunged 是一个任意指针，用于标记已从脏映射中删除的条目。
var expunged = unsafe.Pointer(new(interface{})) // 注：如果entry.p为expunged，表示该条目已删除

// entry 是映射中对应于特定键的插槽。
type entry struct {
	// p指向为条目存储的interface{}值。
	// 如果p == nil，则该条目已被删除，而m.dirty == nil。
	// 如果p == expunged，则该条目已被删除，m.dirty != nil，并且m.dirty中缺少该条目。
	// 否则，该条目有效并记录在m.read.m[key]中；如果m.dirty != nil，则记录在m.dirty[key]中。
	// 可以通过用nil进行原子替换来删除条目：下次创建m.dirty时，它将自动用expunged替换nil并使m.dirty[key]保持不变。
	// 如果p == expunged，则可以通过原子替换来更新条目的关联值。
	// 如果p == expunged，则只有在首先设置m.dirty[key] = e 之后才能更新条目的关联值，以便使用脏映射的查找找到该条目。
	p unsafe.Pointer // *interface{}
}

func newEntry(i interface{}) *entry { // 工厂函数，生成一个entry结构体
	return &entry{p: unsafe.Pointer(&i)}
}

// Load 返回键中存储在映射中的值；如果不存在任何值，则返回nil。
// 确定结果表明是否在地图中找到了值。
func (m *Map) Load(key interface{}) (value interface{}, ok bool) {
	read, _ := m.read.Load().(readOnly) // 注：原子性地读取
	e, ok := read.m[key]                // 注：获取key的value
	if !ok && read.amended {
		m.mu.Lock()
		// 如果在m.mu被阻止时提升了m.dirty，请避免报告虚假的遗漏。 （如果不会再丢失相同密钥的负载，则不值得复制该密钥的脏映射。）
		read, _ = m.read.Load().(readOnly)
		e, ok = read.m[key]
		if !ok && read.amended {
			e, ok = m.dirty[key]
			// 不管是否存在该条目，都记录一个未命中：此键将采用慢速路径，直到将脏映射提升为已读映射为止。
			m.missLocked()
		}
		m.mu.Unlock()
	}
	if !ok {
		return nil, false
	}
	return e.load()
}

func (e *entry) load() (value interface{}, ok bool) { // 注：原子性的获取e指向的值
	p := atomic.LoadPointer(&e.p)  // 注：原子性读取条目的指针
	if p == nil || p == expunged { // 注：如果条目已删除，返回nil
		return nil, false
	}
	return *(*interface{})(p), true // 注：返回条目的值
}

// Store 设置键的值。
func (m *Map) Store(key, value interface{}) { // 注：（原子操作）#
	read, _ := m.read.Load().(readOnly)                 // 注：读取read
	if e, ok := read.m[key]; ok && e.tryStore(&value) { // 注：获取read[key]的entry，尝试将value写入这个entry中
		return
	}

	m.mu.Lock()                        // 注：尝试赋值失败，上锁
	read, _ = m.read.Load().(readOnly) // 注：再次读取read，将转为readOnly类型
	if e, ok := read.m[key]; ok {      // 注：读取read成功
		if e.unexpungeLocked() { // 注：判断如果e已删除，设置为nil
			// 该条目先前已删除，这意味着存在一个非零的脏映射，并且该条目不在其中。
			m.dirty[key] = e
		}
		e.storeLocked(&value) // 注：将read[key]赋值为value
	} else if e, ok := m.dirty[key]; ok { // 注：读取read失败，读取脏映射成功
		e.storeLocked(&value) // 注：将dirty[key]赋值为value
	} else { // 注：read与dirty中都没有这个key
		if !read.amended { // 注：dirty不存在read中没有的key
			// 我们将第一个新键添加到脏映射。
			// 确保已分配它，并将只读映射标记为不完整。
			m.dirtyLocked() // 注：将m.read中未删除的数据拷贝到空m.dirty中
			m.read.Store(readOnly{m: read.m, amended: true})
		}
		m.dirty[key] = newEntry(value)
	}
	m.mu.Unlock()
}

// tryStore 如果条目未删除，则存储一个值。
//
// 如果删除了该条目，则tryStore返回false并使该条目保持不变。
func (e *entry) tryStore(i *interface{}) bool { // 注：原子性的尝试将i赋值给e
	// 读取不到
	for { // 注：自旋读取
		p := atomic.LoadPointer(&e.p) // 注：获取条目的数据
		if p == expunged {            // 注：如果e已经被删除，返回false
			return false
		}
		if atomic.CompareAndSwapPointer(&e.p, p, unsafe.Pointer(i)) { // 注：原子性的将i赋值给e
			return true
		}
	}
}

// unexpungeLocked 确保该条目未标记为清除。
//
// 如果该条目先前已删除，则必须在解锁m.mu之前将其添加到脏映射中。
func (e *entry) unexpungeLocked() (wasExpunged bool) { // 注：原子性的判断如果e已删除，设置为nil
	// 注：
	// 如果e未删除，返回false
	// 如果e已删除，返回true
	// 如果e为空，返回false
	return atomic.CompareAndSwapPointer(&e.p, expunged, nil)
}

// storeLocked 无条件地将值存储到条目。
//
// 必须知道该条目不会被清除。
func (e *entry) storeLocked(i *interface{}) { // 注：原子性的将e赋值为i
	atomic.StorePointer(&e.p, unsafe.Pointer(i))
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (m *Map) LoadOrStore(key, value interface{}) (actual interface{}, loaded bool) {
	// Avoid locking if it's a clean hit.
	read, _ := m.read.Load().(readOnly)
	if e, ok := read.m[key]; ok {
		actual, loaded, ok := e.tryLoadOrStore(value)
		if ok {
			return actual, loaded
		}
	}

	m.mu.Lock()
	read, _ = m.read.Load().(readOnly)
	if e, ok := read.m[key]; ok {
		if e.unexpungeLocked() {
			m.dirty[key] = e
		}
		actual, loaded, _ = e.tryLoadOrStore(value)
	} else if e, ok := m.dirty[key]; ok {
		actual, loaded, _ = e.tryLoadOrStore(value)
		m.missLocked()
	} else {
		if !read.amended {
			// We're adding the first new key to the dirty map.
			// Make sure it is allocated and mark the read-only map as incomplete.
			m.dirtyLocked()
			m.read.Store(readOnly{m: read.m, amended: true})
		}
		m.dirty[key] = newEntry(value)
		actual, loaded = value, false
	}
	m.mu.Unlock()

	return actual, loaded
}

// tryLoadOrStore atomically loads or stores a value if the entry is not
// expunged.
//
// If the entry is expunged, tryLoadOrStore leaves the entry unchanged and
// returns with ok==false.
func (e *entry) tryLoadOrStore(i interface{}) (actual interface{}, loaded, ok bool) {
	p := atomic.LoadPointer(&e.p)
	if p == expunged {
		return nil, false, false
	}
	if p != nil {
		return *(*interface{})(p), true, true
	}

	// Copy the interface after the first load to make this method more amenable
	// to escape analysis: if we hit the "load" path or the entry is expunged, we
	// shouldn't bother heap-allocating.
	ic := i
	for {
		if atomic.CompareAndSwapPointer(&e.p, nil, unsafe.Pointer(&ic)) {
			return i, false, true
		}
		p = atomic.LoadPointer(&e.p)
		if p == expunged {
			return nil, false, false
		}
		if p != nil {
			return *(*interface{})(p), true, true
		}
	}
}

// Delete deletes the value for a key.
func (m *Map) Delete(key interface{}) {
	read, _ := m.read.Load().(readOnly)
	e, ok := read.m[key]
	if !ok && read.amended {
		m.mu.Lock()
		read, _ = m.read.Load().(readOnly)
		e, ok = read.m[key]
		if !ok && read.amended {
			delete(m.dirty, key)
		}
		m.mu.Unlock()
	}
	if ok {
		e.delete()
	}
}

func (e *entry) delete() (hadValue bool) {
	for {
		p := atomic.LoadPointer(&e.p)
		if p == nil || p == expunged {
			return false
		}
		if atomic.CompareAndSwapPointer(&e.p, p, nil) {
			return true
		}
	}
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
//
// Range does not necessarily correspond to any consistent snapshot of the Map's
// contents: no key will be visited more than once, but if the value for any key
// is stored or deleted concurrently, Range may reflect any mapping for that key
// from any point during the Range call.
//
// Range may be O(N) with the number of elements in the map even if f returns
// false after a constant number of calls.
func (m *Map) Range(f func(key, value interface{}) bool) {
	// We need to be able to iterate over all of the keys that were already
	// present at the start of the call to Range.
	// If read.amended is false, then read.m satisfies that property without
	// requiring us to hold m.mu for a long time.
	read, _ := m.read.Load().(readOnly)
	if read.amended {
		// m.dirty contains keys not in read.m. Fortunately, Range is already O(N)
		// (assuming the caller does not break out early), so a call to Range
		// amortizes an entire copy of the map: we can promote the dirty copy
		// immediately!
		m.mu.Lock()
		read, _ = m.read.Load().(readOnly)
		if read.amended {
			read = readOnly{m: m.dirty}
			m.read.Store(read)
			m.dirty = nil
			m.misses = 0
		}
		m.mu.Unlock()
	}

	for k, e := range read.m {
		v, ok := e.load()
		if !ok {
			continue
		}
		if !f(k, v) {
			break
		}
	}
}

func (m *Map) missLocked() {
	m.misses++
	if m.misses < len(m.dirty) {
		return
	}
	m.read.Store(readOnly{m: m.dirty})
	m.dirty = nil
	m.misses = 0
}

func (m *Map) dirtyLocked() { // 注：（原子操作）将m.read中未删除的数据拷贝到空m.dirty中
	if m.dirty != nil { // 注：如果dirty不为空，结束
		return
	}

	read, _ := m.read.Load().(readOnly)                 // 注：读取read
	m.dirty = make(map[interface{}]*entry, len(read.m)) // 注：创建一个dirty
	for k, e := range read.m {                          // 注：遍历read，将不为已删除的entry赋值给dirty
		if !e.tryExpungeLocked() { // 注：e不为已删除
			m.dirty[k] = e // 注：将entry赋值给ditry
		}
	}
}

func (e *entry) tryExpungeLocked() (isExpunged bool) { // 注：（原子性操作）获取e是否已删除
	p := atomic.LoadPointer(&e.p) // 注：读取e的数据
	for p == nil {                // 注：#如果没有获取到p，自旋
		if atomic.CompareAndSwapPointer(&e.p, nil, expunged) { // 注：如果e的指针为空，赋值为已删除，返回true
			return true
		}
		p = atomic.LoadPointer(&e.p) // 注：再次读取e的数据，如果读取到了，跳出循环
	}
	return p == expunged // 注：返回e是否已删除
}
