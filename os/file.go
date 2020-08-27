// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

// Package os 为操作系统功能提供了平台无关的接口。
// 设计类似于Unix，尽管错误处理类似于Go。 失败的调用返回错误类型而不是错误编号的值。
// 通常，错误中会提供更多信息。
// 例如，如果采用文件名的调用失败，例如Open或Stat，则错误将在打印时包含失败的文件名，并且类型为*PathError，可以将其解压缩以获取更多信息。
//
// os接口旨在在所有操作系统上保持统一。
// 通常不可用的功能出现在特定于系统的软件包syscall中。
//
// 这是一个简单的示例，打开一个文件并读取其中的一些内容。
//
//	file, err := os.Open("file.go") // 用于读取访问。
//	if err != nil {
//		log.Fatal(err)
//	}
//
// 如果打开失败，错误字符串将不言自明，例如
//
//	open file.go: no such file or directory
//
// 然后可以将文件的数据读取为一个字节片。 读取和写入从参数切片的长度中获取字节数。
//
//	data := make([]byte, 100)
//	count, err := file.Read(data)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("read %d bytes: %q\n", count, data[:count])
//
// 注意：对文件的最大并行操作数可能受操作系统或系统的限制。 该数目应该很高，但是超过该数目可能会降低性能或引起其他问题。
//
package os

import (
	"errors"
	"internal/poll"
	"internal/testlog"
	"io"
	"runtime"
	"syscall"
	"time"
)

// Name 返回显示给Open的文件名。
func (f *File) Name() string { return f.name } // 注：获取f的名称

// Stdin、Stdout和Stderr是打开的文件，它们指向标准输入，标准输出和标准错误文件描述符。
//
// 请注意，Go运行时会为恐慌和崩溃写入标准错误；
// 关闭Stderr可能会使这些消息转到其他地方，甚至可能到达以后打开的文件中。
var (
	Stdin  = NewFile(uintptr(syscall.Stdin), "/dev/stdin")   // 注：标准输入文件描述符，例：fmt.Scanf
	Stdout = NewFile(uintptr(syscall.Stdout), "/dev/stdout") // 注：标准输出文件描述符，例：fmt.Printf
	Stderr = NewFile(uintptr(syscall.Stderr), "/dev/stderr") // 注：标准错误文件描述符，例：fmt.Errorf
)

// OpenFile的标志包装基础系统的标志。 并非所有标志都可以在给定的系统上实现。
const (
	// 必须指定O_RDONLY，O_WRONLY或O_RDWR之一。
	O_RDONLY int = syscall.O_RDONLY // 以只读方式打开文件。
	O_WRONLY int = syscall.O_WRONLY // 以只写方式打开文件。
	O_RDWR   int = syscall.O_RDWR   // 以读写方式打开文件。
	// 剩余的值可以用来控制行为。
	O_APPEND int = syscall.O_APPEND // 写入时将数据追加到文件中
	O_CREATE int = syscall.O_CREAT  // 创建一个新文件（如果不存在）。
	O_EXCL   int = syscall.O_EXCL   // 与O_CREATE一起使用，文件必须不存在。
	O_SYNC   int = syscall.O_SYNC   // 为同步I/O打开。
	O_TRUNC  int = syscall.O_TRUNC  // 在打开时截断常规可写文件。
)

// 求值。
//
// 不推荐使用：使用io.SeekStart，io.SeekCurrent和io.SeekEnd。
const (
	SEEK_SET int = 0 // 相对于文件的原点查找
	SEEK_CUR int = 1 // 相对于当前偏移量的搜索
	SEEK_END int = 2 // 相对于末尾查找
)

// LinkError 会在链接或符号链接或重命名系统调用及其引起路径的过程中记录错误。
type LinkError struct {
	Op  string
	Old string
	New string
	Err error
}

func (e *LinkError) Error() string { //注：格式化错误并返回
	return e.Op + " " + e.Old + " " + e.New + ": " + e.Err.Error()
}

func (e *LinkError) Unwrap() error { //注：返回error
	return e.Err
}

// Read 从文件中读取多达len(b)个字节。
// 返回读取的字节数和遇到的任何错误。
// 在文件末尾，Read返回0，即io.EOF。
func (f *File) Read(b []byte) (n int, err error) { // 注：#
	if err := f.checkValid("read"); err != nil { // 注：#f是否有效
		return 0, err
	}
	n, e := f.read(b)
	return n, f.wrapErr("read", e)
}

// ReadAt 从字节偏移量关闭的文件中读取len(b)个字节。
// 返回读取的字节数和错误（如果有）。
// 当n < len(b)时，ReadAt始终返回非nil错误。
// 在文件末尾，该错误是io.EOF。
func (f *File) ReadAt(b []byte, off int64) (n int, err error) { // 注：#
	if err := f.checkValid("read"); err != nil { // 注：#
		return 0, err
	}

	if off < 0 {
		return 0, &PathError{"readat", f.name, errors.New("negative offset")}
	}

	for len(b) > 0 {
		m, e := f.pread(b, off)
		if e != nil {
			err = f.wrapErr("read", e)
			break
		}
		n += m
		b = b[m:]
		off += int64(m)
	}
	return
}

// Write 将len(b)个字节写入文件。
// 返回写入的字节数和错误（如果有）。
// 当n != len(b)时，Write返回一个非nil错误。
func (f *File) Write(b []byte) (n int, err error) { // 注：#
	if err := f.checkValid("write"); err != nil { // 注：#
		return 0, err
	}
	n, e := f.write(b)
	if n < 0 {
		n = 0
	}
	if n != len(b) {
		err = io.ErrShortWrite
	}

	epipecheck(f, e)

	if e != nil {
		err = f.wrapErr("write", e)
	}

	return n, err
}

var errWriteAtInAppendMode = errors.New("os: invalid use of WriteAt on file opened with O_APPEND") // 错误："在以O_APPEND打开的文件上无效使用WriteAt"

// WriteAt 从字节偏移量开始将len(b)字节写入文件。
// 返回写入的字节数和错误（如果有）。
// 当n != len(b)时，WriteAt返回非nil错误。
//
// 如果使用O_APPEND标志打开了文件，则WriteAt返回错误。
func (f *File) WriteAt(b []byte, off int64) (n int, err error) { // 注：#
	if err := f.checkValid("write"); err != nil { // 注：#
		return 0, err
	}
	if f.appendMode {
		return 0, errWriteAtInAppendMode
	}

	if off < 0 {
		return 0, &PathError{"writeat", f.name, errors.New("negative offset")}
	}

	for len(b) > 0 {
		m, e := f.pwrite(b, off)
		if e != nil {
			err = f.wrapErr("write", e)
			break
		}
		n += m
		b = b[m:]
		off += int64(m)
	}
	return
}

// Seek 将下一次在文件上读取或写入的偏移量设置为偏移量，根据whence进行解释：0表示相对于文件原点，1表示相对于当前偏移量，2表示相对于末尾。
// 返回新的偏移量和错误（如果有）。
// 未指定使用O_APPEND打开的文件的Seek行为。
//
// 如果f是目录，则Seek的行为因操作系统而异； 您可以在类似Unix的操作系统上找到目录的开头，但不能在Windows上找到。
func (f *File) Seek(offset int64, whence int) (ret int64, err error) { // 注：#
	if err := f.checkValid("seek"); err != nil { // 注：#
		return 0, err
	}
	r, e := f.seek(offset, whence)
	if e == nil && f.dirinfo != nil && r != 0 {
		e = syscall.EISDIR
	}
	if e != nil {
		return 0, f.wrapErr("seek", e)
	}
	return r, nil
}

// WriteString 类似于Write，但是写入字符串s的内容，而不是字节的片段。
func (f *File) WriteString(s string) (n int, err error) { // 注：向f写入s，返回写入的字符长度n与错误err
	return f.Write([]byte(s))
}

// Mkdir 使用指定的名称和权限位（在umask之前）创建一个新目录。
// 如果有错误，它将是*PathError类型。
func Mkdir(name string, perm FileMode) error {
	if runtime.GOOS == "windows" && isWindowsNulName(name) { // 注：如果操作系统是windows并且name == "NUL"
		return &PathError{"mkdir", name, syscall.ENOTDIR} // 注：返回路径错误，ERROR_PATH_NOT_FOUND
	}
	e := syscall.Mkdir(fixLongPath(name), syscallMode(perm)) // 注：创建文件夹name

	if e != nil {
		return &PathError{"mkdir", name, e} // 错误："路径错误"
	}

	// mkdir(2)本身不会处理*BSD和Solaris上的粘滞位
	if !supportsCreateWithStickyBit && perm&ModeSticky != 0 { // 注：#
		e = setStickyBit(name)

		if e != nil {
			Remove(name)
			return e
		}
	}

	return nil
}

// setStickyBit adds ModeSticky to the permission bits of path, non atomic.
func setStickyBit(name string) error {
	fi, err := Stat(name)
	if err != nil {
		return err
	}
	return Chmod(name, fi.Mode()|ModeSticky)
}

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func Chdir(dir string) error {
	if e := syscall.Chdir(dir); e != nil {
		testlog.Open(dir) // observe likely non-existent directory
		return &PathError{"chdir", dir, e}
	}
	if log := testlog.Logger(); log != nil {
		wd, err := Getwd()
		if err == nil {
			log.Chdir(wd)
		}
	}
	return nil
}

// Open 打开命名文件以供读取。 如果成功，则可以使用返回文件上的方法进行读取；
// 关联的文件描述符的模式为O_RDONLY。
// 如果有错误，它将是*PathError类型。
func Open(name string) (*File, error) {
	return OpenFile(name, O_RDONLY, 0)
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned File can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func Create(name string) (*File, error) {
	return OpenFile(name, O_RDWR|O_CREATE|O_TRUNC, 0666)
}

// OpenFile 是广义的open调用； 大多数用户将改为使用“打开”或“创建”。
// 它打开带有指定标志（O_RDONLY等）的命名文件。
// 如果文件不存在，并且传递了O_CREATE标志，则使用模式perm（在umask之前）创建文件。
// 如果成功，则可以将返回的File上的方法用于I / O。
// 如果有错误，它将是*PathError类型。
func OpenFile(name string, flag int, perm FileMode) (*File, error) {
	testlog.Open(name)
	f, err := openFileNolog(name, flag, perm)
	if err != nil {
		return nil, err
	}
	f.appendMode = flag&O_APPEND != 0

	return f, nil
}

// lstat is overridden in tests.
var lstat = Lstat

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func Rename(oldpath, newpath string) error {
	return rename(oldpath, newpath)
}

// Many functions in package syscall return a count of -1 instead of 0.
// Using fixCount(call()) instead of call() corrects the count.
func fixCount(n int, err error) (int, error) {
	if n < 0 {
		n = 0
	}
	return n, err
}

// wrapErr wraps an error that occurred during an operation on an open file.
// It passes io.EOF through unchanged, otherwise converts
// poll.ErrFileClosing to ErrClosed and wraps the error in a PathError.
func (f *File) wrapErr(op string, err error) error {
	if err == nil || err == io.EOF {
		return err
	}
	if err == poll.ErrFileClosing {
		err = ErrClosed
	}
	return &PathError{op, f.name, err}
}

// TempDir 返回用于临时文件的默认目录。
//
// 在Unix系统上，如果非空，则返回$TMPDIR，否则返回/tmp。
// 在Windows上，它使用GetTempPath，从%TMP%，%TEMP%，%USERPROFILE%或Windows目录返回第一个非空值。
// 在方案9中，它返回/tmp。
//
// 该目录既不能保证存在也不具有访问权限。
func TempDir() string {
	return tempDir()
}

// UserCacheDir returns the default root directory to use for user-specific
// cached data. Users should create their own application-specific subdirectory
// within this one and use that.
//
// On Unix systems, it returns $XDG_CACHE_HOME as specified by
// https://standards.freedesktop.org/basedir-spec/basedir-spec-latest.html if
// non-empty, else $HOME/.cache.
// On Darwin, it returns $HOME/Library/Caches.
// On Windows, it returns %LocalAppData%.
// On Plan 9, it returns $home/lib/cache.
//
// If the location cannot be determined (for example, $HOME is not defined),
// then it will return an error.
func UserCacheDir() (string, error) {
	var dir string

	switch runtime.GOOS {
	case "windows":
		dir = Getenv("LocalAppData")
		if dir == "" {
			return "", errors.New("%LocalAppData% is not defined")
		}

	case "darwin":
		dir = Getenv("HOME")
		if dir == "" {
			return "", errors.New("$HOME is not defined")
		}
		dir += "/Library/Caches"

	case "plan9":
		dir = Getenv("home")
		if dir == "" {
			return "", errors.New("$home is not defined")
		}
		dir += "/lib/cache"

	default: // Unix
		dir = Getenv("XDG_CACHE_HOME")
		if dir == "" {
			dir = Getenv("HOME")
			if dir == "" {
				return "", errors.New("neither $XDG_CACHE_HOME nor $HOME are defined")
			}
			dir += "/.cache"
		}
	}

	return dir, nil
}

// UserConfigDir returns the default root directory to use for user-specific
// configuration data. Users should create their own application-specific
// subdirectory within this one and use that.
//
// On Unix systems, it returns $XDG_CONFIG_HOME as specified by
// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html if
// non-empty, else $HOME/.config.
// On Darwin, it returns $HOME/Library/Application Support.
// On Windows, it returns %AppData%.
// On Plan 9, it returns $home/lib.
//
// If the location cannot be determined (for example, $HOME is not defined),
// then it will return an error.
func UserConfigDir() (string, error) {
	var dir string

	switch runtime.GOOS {
	case "windows":
		dir = Getenv("AppData")
		if dir == "" {
			return "", errors.New("%AppData% is not defined")
		}

	case "darwin":
		dir = Getenv("HOME")
		if dir == "" {
			return "", errors.New("$HOME is not defined")
		}
		dir += "/Library/Application Support"

	case "plan9":
		dir = Getenv("home")
		if dir == "" {
			return "", errors.New("$home is not defined")
		}
		dir += "/lib"

	default: // Unix
		dir = Getenv("XDG_CONFIG_HOME")
		if dir == "" {
			dir = Getenv("HOME")
			if dir == "" {
				return "", errors.New("neither $XDG_CONFIG_HOME nor $HOME are defined")
			}
			dir += "/.config"
		}
	}

	return dir, nil
}

// UserHomeDir returns the current user's home directory.
//
// On Unix, including macOS, it returns the $HOME environment variable.
// On Windows, it returns %USERPROFILE%.
// On Plan 9, it returns the $home environment variable.
func UserHomeDir() (string, error) {
	env, enverr := "HOME", "$HOME"
	switch runtime.GOOS {
	case "windows":
		env, enverr = "USERPROFILE", "%userprofile%"
	case "plan9":
		env, enverr = "home", "$home"
	}
	if v := Getenv(env); v != "" {
		return v, nil
	}
	// On some geese the home directory is not always defined.
	switch runtime.GOOS {
	case "android":
		return "/sdcard", nil
	case "darwin":
		if runtime.GOARCH == "arm" || runtime.GOARCH == "arm64" {
			return "/", nil
		}
	}
	return "", errors.New(enverr + " is not defined")
}

// Chmod changes the mode of the named file to mode.
// If the file is a symbolic link, it changes the mode of the link's target.
// If there is an error, it will be of type *PathError.
//
// A different subset of the mode bits are used, depending on the
// operating system.
//
// On Unix, the mode's permission bits, ModeSetuid, ModeSetgid, and
// ModeSticky are used.
//
// On Windows, only the 0200 bit (owner writable) of mode is used; it
// controls whether the file's read-only attribute is set or cleared.
// The other bits are currently unused. For compatibility with Go 1.12
// and earlier, use a non-zero mode. Use mode 0400 for a read-only
// file and 0600 for a readable+writable file.
//
// On Plan 9, the mode's permission bits, ModeAppend, ModeExclusive,
// and ModeTemporary are used.
func Chmod(name string, mode FileMode) error { return chmod(name, mode) }

// Chmod changes the mode of the file to mode.
// If there is an error, it will be of type *PathError.
func (f *File) Chmod(mode FileMode) error { return f.chmod(mode) }

// SetDeadline sets the read and write deadlines for a File.
// It is equivalent to calling both SetReadDeadline and SetWriteDeadline.
//
// Only some kinds of files support setting a deadline. Calls to SetDeadline
// for files that do not support deadlines will return ErrNoDeadline.
// On most systems ordinary files do not support deadlines, but pipes do.
//
// A deadline is an absolute time after which I/O operations fail with an
// error instead of blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to Read or Write.
// After a deadline has been exceeded, the connection can be refreshed
// by setting a deadline in the future.
//
// An error returned after a timeout fails will implement the
// Timeout method, and calling the Timeout method will return true.
// The PathError and SyscallError types implement the Timeout method.
// In general, call IsTimeout to test whether an error indicates a timeout.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (f *File) SetDeadline(t time.Time) error {
	return f.setDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls and any
// currently-blocked Read call.
// A zero value for t means Read will not time out.
// Not all files support setting deadlines; see SetDeadline.
func (f *File) SetReadDeadline(t time.Time) error {
	return f.setReadDeadline(t)
}

// SetWriteDeadline sets the deadline for any future Write calls and any
// currently-blocked Write call.
// Even if Write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
// Not all files support setting deadlines; see SetDeadline.
func (f *File) SetWriteDeadline(t time.Time) error {
	return f.setWriteDeadline(t)
}

// SyscallConn returns a raw file.
// This implements the syscall.Conn interface.
func (f *File) SyscallConn() (syscall.RawConn, error) {
	if err := f.checkValid("SyscallConn"); err != nil {
		return nil, err
	}
	return newRawConn(f)
}

// isWindowsNulName 报告在Windows上名称是否为os.DevNull('NUL')。
// 在任何情况下，如果名称为“"NUL"，则返回True。
func isWindowsNulName(name string) bool { // 注：获取name是否为"NUL"
	if len(name) != 3 {
		return false
	}
	if name[0] != 'n' && name[0] != 'N' {
		return false
	}
	if name[1] != 'u' && name[1] != 'U' {
		return false
	}
	if name[2] != 'l' && name[2] != 'L' {
		return false
	}
	return true
}
