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

func errInvalid() error    { return oserror.ErrInvalid }
func errPermission() error { return oserror.ErrPermission }
func errExist() error      { return oserror.ErrExist }
func errNotExist() error   { return oserror.ErrNotExist }
func errClosed() error     { return oserror.ErrClosed }
func errNoDeadline() error { return poll.ErrNoDeadline }

type timeout interface {
	Timeout() bool
}

// PathError records an error and the operation and file path that caused it.
type PathError struct {
	Op   string
	Path string
	Err  error
}

func (e *PathError) Error() string { return e.Op + " " + e.Path + ": " + e.Err.Error() }

func (e *PathError) Unwrap() error { return e.Err }

// Timeout reports whether this error represents a timeout.
func (e *PathError) Timeout() bool {
	t, ok := e.Err.(timeout)
	return ok && t.Timeout()
}

// SyscallError records an error from a specific system call.
type SyscallError struct {
	Syscall string
	Err     error
}

func (e *SyscallError) Error() string { return e.Syscall + ": " + e.Err.Error() }

func (e *SyscallError) Unwrap() error { return e.Err }

// Timeout reports whether this error represents a timeout.
func (e *SyscallError) Timeout() bool {
	t, ok := e.Err.(timeout)
	return ok && t.Timeout()
}

// NewSyscallError returns, as an error, a new SyscallError
// with the given system call name and error details.
// As a convenience, if err is nil, NewSyscallError returns nil.
func NewSyscallError(syscall string, err error) error {
	if err == nil {
		return nil
	}
	return &SyscallError{syscall, err}
}

// IsExist 返回一个布尔值，指示是否已知该错误以报告文件或目录已存在。
// ErrExist以及一些系统调用错误都可以满足要求。
func IsExist(err error) bool {
	return underlyingErrorIs(err, ErrExist)
}

// IsNotExist 返回一个布尔值，该布尔值指示是否已知该错误以报告文件或目录不存在。
// ErrNotExist以及一些系统调用错误都可以满足要求。
func IsNotExist(err error) bool {
	return underlyingErrorIs(err, ErrNotExist)
}

// IsPermission returns a boolean indicating whether the error is known to
// report that permission is denied. It is satisfied by ErrPermission as well
// as some syscall errors.
func IsPermission(err error) bool {
	return underlyingErrorIs(err, ErrPermission)
}

// IsTimeout returns a boolean indicating whether the error is known
// to report that a timeout occurred.
func IsTimeout(err error) bool {
	terr, ok := underlyingError(err).(timeout)
	return ok && terr.Timeout()
}

func underlyingErrorIs(err, target error) bool {
	// Note that this function is not errors.Is:
	// underlyingError only unwraps the specific error-wrapping types
	// that it historically did, not all errors implementing Unwrap().
	err = underlyingError(err)
	if err == target {
		return true
	}
	// To preserve prior behavior, only examine syscall errors.
	e, ok := err.(syscallErrorType)
	return ok && e.Is(target)
}

// underlyingError returns the underlying error for known os error types.
func underlyingError(err error) error {
	switch err := err.(type) {
	case *PathError:
		return err.Err
	case *LinkError:
		return err.Err
	case *SyscallError:
		return err.Err
	}
	return err
}
