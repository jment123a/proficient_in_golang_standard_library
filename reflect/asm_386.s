// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"
#include "funcdata.h"

// makeFuncStub 是MakeFunc返回的函数的代码一半。
// 有关更多详细信息，请参见makefunc.go中有关makeFuncStub声明的注释。
// 这里没有argsize，gc在调用站点生成argsize信息。
TEXT ·makeFuncStub(SB),(NOSPLIT|WRAPPER),$16
	NO_LOCAL_POINTERS
	MOVL	DX, 0(SP)
	LEAL	argframe+0(FP), CX
	MOVL	CX, 4(SP)
	MOVB	$0, 12(SP)
	LEAL	12(SP), AX
	MOVL	AX, 8(SP)
	CALL	·callReflect(SB)
	RET

// methodValueCall 是makeMethodValue返回的函数的代码一半。
// 有关更多详细信息，请参见makefunc.go中有关methodValueCall声明的注释。
// 这里没有argsize，gc在调用站点生成argsize信息。
TEXT ·methodValueCall(SB),(NOSPLIT|WRAPPER),$16
	NO_LOCAL_POINTERS
	MOVL	DX, 0(SP)
	LEAL	argframe+0(FP), CX
	MOVL	CX, 4(SP)
	MOVB	$0, 12(SP)
	LEAL	12(SP), AX
	MOVL	AX, 8(SP)
	CALL	·callMethod(SB)
	RET
