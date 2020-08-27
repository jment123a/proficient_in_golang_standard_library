// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。
package os

import (
	"internal/oserror"
	"internal/poll"
)

// 一些常见系统调用错误的可移植类比。
//
// 从此包返回的错误可能会针对带有error.Is的这些错误进行测试。
var (
	// ErrInvalid表示参数无效。
	// 当接收器为nil时，File方法将返回此错误。
	ErrInvalid = errInvalid() // "无效的参数"

	ErrPermission = errPermission() // "没有权限"
	ErrExist      = errExist()      // "文件已存在"
	ErrNotExist   = errNotExist()   // "文件不存在
	ErrClosed     = errClosed()     // "文件已关闭"
	ErrNoDeadline = errNoDeadline() // "文件类型不支持截止日期"
)

func errInvalid() error    { return oserror.ErrInvalid }    // 注："无效的参数"
func errPermission() error { return oserror.ErrPermission } // 注："没有权限"
func errExist() error      { return oserror.ErrExist }      // 注："文件已存在"
func errNotExist() error   { return oserror.ErrNotExist }   // 注："文件不存在
func errClosed() error     { return oserror.ErrClosed }     // 注："文件已关闭"
func errNoDeadline() error { return poll.ErrNoDeadline }    // 注："文件类型不支持截止日期"

type timeout interface {
	Timeout() bool
}

// PathError 记录错误以及导致该错误的操作和文件路径。
type PathError struct { // 注：#记录带有错误文件路径的错误
	Op   string // 注：出现错误的操作
	Path string // 注：出现错误的文件路径
	Err  error  // 注：出现的错误
}

func (e *PathError) Error() string { return e.Op + " " + e.Path + ": " + e.Err.Error() } // 注：返回格式化的错误

func (e *PathError) Unwrap() error { return e.Err } // 注：解包，返回错误Err

// Timeout 报告此错误是否表示超时。
func (e *PathError) Timeout() bool { // 注：获取错误e是否超时
	t, ok := e.Err.(timeout) // 注：如果e实现timeout接口，执行Timeout方法
	return ok && t.Timeout()
}

// SyscallError 记录来自特定系统调用的错误。
type SyscallError struct {
	Syscall string // 注：出现错误的系统调用
	Err     error  // 注：出现的错误
}

func (e *SyscallError) Error() string { return e.Syscall + ": " + e.Err.Error() } // 注：返回格式化的错误

func (e *SyscallError) Unwrap() error { return e.Err } // 注：解包，返回错误Err

// Timeout 报告此错误是否表示超时。
func (e *SyscallError) Timeout() bool { // 注：获取错误e是否超时
	t, ok := e.Err.(timeout) // 注：如果e实现timeout接口，执行Timeout方法
	return ok && t.Timeout()
}

// NewSyscallError 作为错误返回具有给定系统调用名称和错误详细信息的新SyscallError。
// 为方便起见，如果err为nil，则NewSyscallError返回nil。
func NewSyscallError(syscall string, err error) error { // 注：工厂函数，生成一个SyscallError
	if err == nil {
		return nil
	}
	return &SyscallError{syscall, err}
}

// IsExist 返回一个布尔值，指示是否已知该错误以报告文件或目录已存在。
// ErrExist以及一些系统调用错误都可以满足要求。
func IsExist(err error) bool { // 注：获取err的底层错误是否为"文件已存在"
	return underlyingErrorIs(err, ErrExist)
}

// IsNotExist 返回一个布尔值，该布尔值指示是否已知该错误以报告文件或目录不存在。
// ErrNotExist以及一些系统调用错误都可以满足要求。
func IsNotExist(err error) bool { // 注：获取err的底层错误是否为"文件不存在"
	return underlyingErrorIs(err, ErrNotExist)
}

// IsPermission 返回一个布尔值，指示是否已知该错误，以报告权限被拒绝。 ErrPermission以及一些系统调用错误都可以满足要求。
func IsPermission(err error) bool { // 注：获取err的底层错误是否为"没有权限"
	return underlyingErrorIs(err, ErrPermission)
}

// IsTimeout 返回一个布尔值，指示是否已知该错误以报告发生了超时。
func IsTimeout(err error) bool { // 注：获取err的底层错误是否超时
	terr, ok := underlyingError(err).(timeout)
	return ok && terr.Timeout()
}

func underlyingErrorIs(err, target error) bool { // 注：获取err的底层错误是否为target
	// 请注意，此函数不是错误。
	// underlyingError仅解包其过去所做的特定错误包装类型，而不是实现Unwrap()的所有错误。
	err = underlyingError(err)
	if err == target { // 注：如果err与target的底层错误相同，返回true
		return true
	}
	// 要保留以前的行为，请仅检查syscall错误。
	e, ok := err.(syscallErrorType) // 注：#否则检查syscall错误
	return ok && e.Is(target)
}

// underlyingError 返回已知操作系统错误类型的基础错误。
func underlyingError(err error) error { // 注：获取err的底层错误
	switch err := err.(type) { // 注：如果err的类型为PathError/LinkError/SyscallError，返回底层错误
	case *PathError:
		return err.Err
	case *LinkError:
		return err.Err
	case *SyscallError:
		return err.Err
	}
	return err // 注：否则返回本身
}
