// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import "unsafe"

//go:cgo_export_static main

// Filled in by runtime/cgo when linked into binary.

//go:linkname _cgo_init _cgo_init
//go:linkname _cgo_thread_start _cgo_thread_start
//go:linkname _cgo_sys_thread_create _cgo_sys_thread_create
//go:linkname _cgo_notify_runtime_init_done _cgo_notify_runtime_init_done
//go:linkname _cgo_callers _cgo_callers
//go:linkname _cgo_set_context_function _cgo_set_context_function
//go:linkname _cgo_yield _cgo_yield

var (
	_cgo_init                     unsafe.Pointer
	_cgo_thread_start             unsafe.Pointer
	_cgo_sys_thread_create        unsafe.Pointer
	_cgo_notify_runtime_init_done unsafe.Pointer
	_cgo_callers                  unsafe.Pointer
	_cgo_set_context_function     unsafe.Pointer
	_cgo_yield                    unsafe.Pointer
)

// iscgo is set to true by the runtime/cgo package
var iscgo bool

// cgoHasExtraM is set on startup when an extra M is created for cgo.
// The extra M must be created before any C/C++ code calls cgocallback.
var cgoHasExtraM bool

// cgoUse is called by cgo-generated code (using go:linkname to get at
// an unexported name). The calls serve two purposes:
// 1) they are opaque to escape analysis, so the argument is considered to
// escape to the heap.
// 2) they keep the argument alive until the call site; the call is emitted after
// the end of the (presumed) use of the argument by C.
// cgoUse should not actually be called (see cgoAlwaysFalse).
func cgoUse(interface{}) { throw("cgoUse should not be called") }

// cgoAlwaysFalse是始终为false的布尔值。
// cgo生成的代码说明是否cgoAlwaysFalse {cgoUse(p)}。
// 编译器无法看到cgoAlwaysFalse始终为false，因此它将发出测试并保留调用，从而提供所需的转义分析结果。 测试比调用低廉
var cgoAlwaysFalse bool

var cgo_yield = &_cgo_yield
