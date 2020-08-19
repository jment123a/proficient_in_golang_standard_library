//由"go generate"生成的代码； 请勿编辑。

package syscall

import (
	"internal/syscall/windows/sysdll"
	"unsafe"
)

var _ unsafe.Pointer

// Do the interface allocations only once for common
// Errno values.
const (
	errnoERROR_IO_PENDING = 997
)

var (
	errERROR_IO_PENDING error = Errno(errnoERROR_IO_PENDING)
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e Errno) error {
	switch e {
	case 0:
		return nil
	case errnoERROR_IO_PENDING:
		return errERROR_IO_PENDING
	}
	// TODO: add more here, after collecting data on the common
	// error values see on Windows. (perhaps when running
	// all.bat?)
	return e
}

var (
	modkernel32 = NewLazyDLL(sysdll.Add("kernel32.dll"))
	modadvapi32 = NewLazyDLL(sysdll.Add("advapi32.dll"))
	modshell32  = NewLazyDLL(sysdll.Add("shell32.dll"))
	modmswsock  = NewLazyDLL(sysdll.Add("mswsock.dll"))
	modcrypt32  = NewLazyDLL(sysdll.Add("crypt32.dll"))
	modws2_32   = NewLazyDLL(sysdll.Add("ws2_32.dll"))
	moddnsapi   = NewLazyDLL(sysdll.Add("dnsapi.dll"))
	modiphlpapi = NewLazyDLL(sysdll.Add("iphlpapi.dll"))
	modsecur32  = NewLazyDLL(sysdll.Add("secur32.dll"))
	modnetapi32 = NewLazyDLL(sysdll.Add("netapi32.dll"))
	moduserenv  = NewLazyDLL(sysdll.Add("userenv.dll"))

	procGetLastError                       = modkernel32.NewProc("GetLastError")
	procLoadLibraryW                       = modkernel32.NewProc("LoadLibraryW")
	procFreeLibrary                        = modkernel32.NewProc("FreeLibrary")
	procGetProcAddress                     = modkernel32.NewProc("GetProcAddress")
	procGetVersion                         = modkernel32.NewProc("GetVersion")
	procFormatMessageW                     = modkernel32.NewProc("FormatMessageW")
	procExitProcess                        = modkernel32.NewProc("ExitProcess")
	procCreateFileW                        = modkernel32.NewProc("CreateFileW")
	procReadFile                           = modkernel32.NewProc("ReadFile")
	procWriteFile                          = modkernel32.NewProc("WriteFile")
	procSetFilePointer                     = modkernel32.NewProc("SetFilePointer")
	procCloseHandle                        = modkernel32.NewProc("CloseHandle")
	procGetStdHandle                       = modkernel32.NewProc("GetStdHandle")
	procFindFirstFileW                     = modkernel32.NewProc("FindFirstFileW")
	procFindNextFileW                      = modkernel32.NewProc("FindNextFileW")
	procFindClose                          = modkernel32.NewProc("FindClose")
	procGetFileInformationByHandle         = modkernel32.NewProc("GetFileInformationByHandle")
	procGetCurrentDirectoryW               = modkernel32.NewProc("GetCurrentDirectoryW")
	procSetCurrentDirectoryW               = modkernel32.NewProc("SetCurrentDirectoryW")
	procCreateDirectoryW                   = modkernel32.NewProc("CreateDirectoryW")
	procRemoveDirectoryW                   = modkernel32.NewProc("RemoveDirectoryW")
	procDeleteFileW                        = modkernel32.NewProc("DeleteFileW")
	procMoveFileW                          = modkernel32.NewProc("MoveFileW")
	procGetComputerNameW                   = modkernel32.NewProc("GetComputerNameW")
	procSetEndOfFile                       = modkernel32.NewProc("SetEndOfFile")
	procGetSystemTimeAsFileTime            = modkernel32.NewProc("GetSystemTimeAsFileTime")
	procGetTimeZoneInformation             = modkernel32.NewProc("GetTimeZoneInformation")
	procCreateIoCompletionPort             = modkernel32.NewProc("CreateIoCompletionPort")
	procGetQueuedCompletionStatus          = modkernel32.NewProc("GetQueuedCompletionStatus")
	procPostQueuedCompletionStatus         = modkernel32.NewProc("PostQueuedCompletionStatus")
	procCancelIo                           = modkernel32.NewProc("CancelIo")
	procCancelIoEx                         = modkernel32.NewProc("CancelIoEx")
	procCreateProcessW                     = modkernel32.NewProc("CreateProcessW")
	procCreateProcessAsUserW               = modadvapi32.NewProc("CreateProcessAsUserW")
	procOpenProcess                        = modkernel32.NewProc("OpenProcess")
	procTerminateProcess                   = modkernel32.NewProc("TerminateProcess")
	procGetExitCodeProcess                 = modkernel32.NewProc("GetExitCodeProcess")
	procGetStartupInfoW                    = modkernel32.NewProc("GetStartupInfoW")
	procGetCurrentProcess                  = modkernel32.NewProc("GetCurrentProcess")
	procGetProcessTimes                    = modkernel32.NewProc("GetProcessTimes")
	procDuplicateHandle                    = modkernel32.NewProc("DuplicateHandle")
	procWaitForSingleObject                = modkernel32.NewProc("WaitForSingleObject")
	procGetTempPathW                       = modkernel32.NewProc("GetTempPathW")
	procCreatePipe                         = modkernel32.NewProc("CreatePipe")
	procGetFileType                        = modkernel32.NewProc("GetFileType")
	procCryptAcquireContextW               = modadvapi32.NewProc("CryptAcquireContextW")
	procCryptReleaseContext                = modadvapi32.NewProc("CryptReleaseContext")
	procCryptGenRandom                     = modadvapi32.NewProc("CryptGenRandom")
	procGetEnvironmentStringsW             = modkernel32.NewProc("GetEnvironmentStringsW")
	procFreeEnvironmentStringsW            = modkernel32.NewProc("FreeEnvironmentStringsW")
	procGetEnvironmentVariableW            = modkernel32.NewProc("GetEnvironmentVariableW")
	procSetEnvironmentVariableW            = modkernel32.NewProc("SetEnvironmentVariableW")
	procSetFileTime                        = modkernel32.NewProc("SetFileTime")
	procGetFileAttributesW                 = modkernel32.NewProc("GetFileAttributesW")
	procSetFileAttributesW                 = modkernel32.NewProc("SetFileAttributesW")
	procGetFileAttributesExW               = modkernel32.NewProc("GetFileAttributesExW")
	procGetCommandLineW                    = modkernel32.NewProc("GetCommandLineW")
	procCommandLineToArgvW                 = modshell32.NewProc("CommandLineToArgvW")
	procLocalFree                          = modkernel32.NewProc("LocalFree")
	procSetHandleInformation               = modkernel32.NewProc("SetHandleInformation")
	procFlushFileBuffers                   = modkernel32.NewProc("FlushFileBuffers")
	procGetFullPathNameW                   = modkernel32.NewProc("GetFullPathNameW")
	procGetLongPathNameW                   = modkernel32.NewProc("GetLongPathNameW")
	procGetShortPathNameW                  = modkernel32.NewProc("GetShortPathNameW")
	procCreateFileMappingW                 = modkernel32.NewProc("CreateFileMappingW")
	procMapViewOfFile                      = modkernel32.NewProc("MapViewOfFile")
	procUnmapViewOfFile                    = modkernel32.NewProc("UnmapViewOfFile")
	procFlushViewOfFile                    = modkernel32.NewProc("FlushViewOfFile")
	procVirtualLock                        = modkernel32.NewProc("VirtualLock")
	procVirtualUnlock                      = modkernel32.NewProc("VirtualUnlock")
	procTransmitFile                       = modmswsock.NewProc("TransmitFile")
	procReadDirectoryChangesW              = modkernel32.NewProc("ReadDirectoryChangesW")
	procCertOpenSystemStoreW               = modcrypt32.NewProc("CertOpenSystemStoreW")
	procCertOpenStore                      = modcrypt32.NewProc("CertOpenStore")
	procCertEnumCertificatesInStore        = modcrypt32.NewProc("CertEnumCertificatesInStore")
	procCertAddCertificateContextToStore   = modcrypt32.NewProc("CertAddCertificateContextToStore")
	procCertCloseStore                     = modcrypt32.NewProc("CertCloseStore")
	procCertGetCertificateChain            = modcrypt32.NewProc("CertGetCertificateChain")
	procCertFreeCertificateChain           = modcrypt32.NewProc("CertFreeCertificateChain")
	procCertCreateCertificateContext       = modcrypt32.NewProc("CertCreateCertificateContext")
	procCertFreeCertificateContext         = modcrypt32.NewProc("CertFreeCertificateContext")
	procCertVerifyCertificateChainPolicy   = modcrypt32.NewProc("CertVerifyCertificateChainPolicy")
	procRegOpenKeyExW                      = modadvapi32.NewProc("RegOpenKeyExW")
	procRegCloseKey                        = modadvapi32.NewProc("RegCloseKey")
	procRegQueryInfoKeyW                   = modadvapi32.NewProc("RegQueryInfoKeyW")
	procRegEnumKeyExW                      = modadvapi32.NewProc("RegEnumKeyExW")
	procRegQueryValueExW                   = modadvapi32.NewProc("RegQueryValueExW")
	procGetCurrentProcessId                = modkernel32.NewProc("GetCurrentProcessId")
	procGetConsoleMode                     = modkernel32.NewProc("GetConsoleMode")
	procWriteConsoleW                      = modkernel32.NewProc("WriteConsoleW")
	procReadConsoleW                       = modkernel32.NewProc("ReadConsoleW")
	procCreateToolhelp32Snapshot           = modkernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW                    = modkernel32.NewProc("Process32FirstW")
	procProcess32NextW                     = modkernel32.NewProc("Process32NextW")
	procDeviceIoControl                    = modkernel32.NewProc("DeviceIoControl")
	procCreateSymbolicLinkW                = modkernel32.NewProc("CreateSymbolicLinkW")
	procCreateHardLinkW                    = modkernel32.NewProc("CreateHardLinkW")
	procWSAStartup                         = modws2_32.NewProc("WSAStartup")
	procWSACleanup                         = modws2_32.NewProc("WSACleanup")
	procWSAIoctl                           = modws2_32.NewProc("WSAIoctl")
	procsocket                             = modws2_32.NewProc("socket")
	procsetsockopt                         = modws2_32.NewProc("setsockopt")
	procgetsockopt                         = modws2_32.NewProc("getsockopt")
	procbind                               = modws2_32.NewProc("bind")
	procconnect                            = modws2_32.NewProc("connect")
	procgetsockname                        = modws2_32.NewProc("getsockname")
	procgetpeername                        = modws2_32.NewProc("getpeername")
	proclisten                             = modws2_32.NewProc("listen")
	procshutdown                           = modws2_32.NewProc("shutdown")
	procclosesocket                        = modws2_32.NewProc("closesocket")
	procAcceptEx                           = modmswsock.NewProc("AcceptEx")
	procGetAcceptExSockaddrs               = modmswsock.NewProc("GetAcceptExSockaddrs")
	procWSARecv                            = modws2_32.NewProc("WSARecv")
	procWSASend                            = modws2_32.NewProc("WSASend")
	procWSARecvFrom                        = modws2_32.NewProc("WSARecvFrom")
	procWSASendTo                          = modws2_32.NewProc("WSASendTo")
	procgethostbyname                      = modws2_32.NewProc("gethostbyname")
	procgetservbyname                      = modws2_32.NewProc("getservbyname")
	procntohs                              = modws2_32.NewProc("ntohs")
	procgetprotobyname                     = modws2_32.NewProc("getprotobyname")
	procDnsQuery_W                         = moddnsapi.NewProc("DnsQuery_W")
	procDnsRecordListFree                  = moddnsapi.NewProc("DnsRecordListFree")
	procDnsNameCompare_W                   = moddnsapi.NewProc("DnsNameCompare_W")
	procGetAddrInfoW                       = modws2_32.NewProc("GetAddrInfoW")
	procFreeAddrInfoW                      = modws2_32.NewProc("FreeAddrInfoW")
	procGetIfEntry                         = modiphlpapi.NewProc("GetIfEntry")
	procGetAdaptersInfo                    = modiphlpapi.NewProc("GetAdaptersInfo")
	procSetFileCompletionNotificationModes = modkernel32.NewProc("SetFileCompletionNotificationModes")
	procWSAEnumProtocolsW                  = modws2_32.NewProc("WSAEnumProtocolsW")
	procTranslateNameW                     = modsecur32.NewProc("TranslateNameW")
	procGetUserNameExW                     = modsecur32.NewProc("GetUserNameExW")
	procNetUserGetInfo                     = modnetapi32.NewProc("NetUserGetInfo")
	procNetGetJoinInformation              = modnetapi32.NewProc("NetGetJoinInformation")
	procNetApiBufferFree                   = modnetapi32.NewProc("NetApiBufferFree")
	procLookupAccountSidW                  = modadvapi32.NewProc("LookupAccountSidW")
	procLookupAccountNameW                 = modadvapi32.NewProc("LookupAccountNameW")
	procConvertSidToStringSidW             = modadvapi32.NewProc("ConvertSidToStringSidW")
	procConvertStringSidToSidW             = modadvapi32.NewProc("ConvertStringSidToSidW")
	procGetLengthSid                       = modadvapi32.NewProc("GetLengthSid")
	procCopySid                            = modadvapi32.NewProc("CopySid")
	procOpenProcessToken                   = modadvapi32.NewProc("OpenProcessToken")
	procGetTokenInformation                = modadvapi32.NewProc("GetTokenInformation")
	procGetUserProfileDirectoryW           = moduserenv.NewProc("GetUserProfileDirectoryW")
	procGetSystemDirectoryW                = modkernel32.NewProc("GetSystemDirectoryW")
)

func GetLastError() (lasterr error) {
	r0, _, _ := Syscall(procGetLastError.Addr(), 0, 0, 0, 0)
	if r0 != 0 {
		lasterr = Errno(r0)
	}
	return
}

func LoadLibrary(libname string) (handle Handle, err error) {
	var _p0 *uint16
	_p0, err = UTF16PtrFromString(libname)
	if err != nil {
		return
	}
	return _LoadLibrary(_p0)
}

func _LoadLibrary(libname *uint16) (handle Handle, err error) {
	r0, _, e1 := Syscall(procLoadLibraryW.Addr(), 1, uintptr(unsafe.Pointer(libname)), 0, 0)
	handle = Handle(r0)
	if handle == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func FreeLibrary(handle Handle) (err error) {
	r1, _, e1 := Syscall(procFreeLibrary.Addr(), 1, uintptr(handle), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetProcAddress(module Handle, procname string) (proc uintptr, err error) {
	var _p0 *byte
	_p0, err = BytePtrFromString(procname)
	if err != nil {
		return
	}
	return _GetProcAddress(module, _p0)
}

func _GetProcAddress(module Handle, procname *byte) (proc uintptr, err error) {
	r0, _, e1 := Syscall(procGetProcAddress.Addr(), 2, uintptr(module), uintptr(unsafe.Pointer(procname)), 0)
	proc = uintptr(r0)
	if proc == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetVersion() (ver uint32, err error) {
	r0, _, e1 := Syscall(procGetVersion.Addr(), 0, 0, 0, 0)
	ver = uint32(r0)
	if ver == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func formatMessage(flags uint32, msgsrc uintptr, msgid uint32, langid uint32, buf []uint16, args *byte) (n uint32, err error) {
	var _p0 *uint16
	if len(buf) > 0 {
		_p0 = &buf[0]
	}
	r0, _, e1 := Syscall9(procFormatMessageW.Addr(), 7, uintptr(flags), uintptr(msgsrc), uintptr(msgid), uintptr(langid), uintptr(unsafe.Pointer(_p0)), uintptr(len(buf)), uintptr(unsafe.Pointer(args)), 0, 0)
	n = uint32(r0)
	if n == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func ExitProcess(exitcode uint32) {
	Syscall(procExitProcess.Addr(), 1, uintptr(exitcode), 0, 0)
	return
}

func CreateFile(name *uint16, access uint32, mode uint32, sa *SecurityAttributes, createmode uint32, attrs uint32, templatefile int32) (handle Handle, err error) {
	r0, _, e1 := Syscall9(procCreateFileW.Addr(), 7, uintptr(unsafe.Pointer(name)), uintptr(access), uintptr(mode), uintptr(unsafe.Pointer(sa)), uintptr(createmode), uintptr(attrs), uintptr(templatefile), 0, 0)
	handle = Handle(r0)
	if handle == InvalidHandle {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func ReadFile(handle Handle, buf []byte, done *uint32, overlapped *Overlapped) (err error) {
	var _p0 *byte
	if len(buf) > 0 {
		_p0 = &buf[0]
	}
	r1, _, e1 := Syscall6(procReadFile.Addr(), 5, uintptr(handle), uintptr(unsafe.Pointer(_p0)), uintptr(len(buf)), uintptr(unsafe.Pointer(done)), uintptr(unsafe.Pointer(overlapped)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func WriteFile(handle Handle, buf []byte, done *uint32, overlapped *Overlapped) (err error) {
	var _p0 *byte
	if len(buf) > 0 {
		_p0 = &buf[0]
	}
	r1, _, e1 := Syscall6(procWriteFile.Addr(), 5, uintptr(handle), uintptr(unsafe.Pointer(_p0)), uintptr(len(buf)), uintptr(unsafe.Pointer(done)), uintptr(unsafe.Pointer(overlapped)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func SetFilePointer(handle Handle, lowoffset int32, highoffsetptr *int32, whence uint32) (newlowoffset uint32, err error) {
	r0, _, e1 := Syscall6(procSetFilePointer.Addr(), 4, uintptr(handle), uintptr(lowoffset), uintptr(unsafe.Pointer(highoffsetptr)), uintptr(whence), 0, 0)
	newlowoffset = uint32(r0)
	if newlowoffset == 0xffffffff {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CloseHandle(handle Handle) (err error) {
	r1, _, e1 := Syscall(procCloseHandle.Addr(), 1, uintptr(handle), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetStdHandle(stdhandle int) (handle Handle, err error) {
	r0, _, e1 := Syscall(procGetStdHandle.Addr(), 1, uintptr(stdhandle), 0, 0)
	handle = Handle(r0)
	if handle == InvalidHandle {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func findFirstFile1(name *uint16, data *win32finddata1) (handle Handle, err error) {
	r0, _, e1 := Syscall(procFindFirstFileW.Addr(), 2, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(data)), 0)
	handle = Handle(r0)
	if handle == InvalidHandle {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func findNextFile1(handle Handle, data *win32finddata1) (err error) {
	r1, _, e1 := Syscall(procFindNextFileW.Addr(), 2, uintptr(handle), uintptr(unsafe.Pointer(data)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func FindClose(handle Handle) (err error) {
	r1, _, e1 := Syscall(procFindClose.Addr(), 1, uintptr(handle), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetFileInformationByHandle(handle Handle, data *ByHandleFileInformation) (err error) {
	r1, _, e1 := Syscall(procGetFileInformationByHandle.Addr(), 2, uintptr(handle), uintptr(unsafe.Pointer(data)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetCurrentDirectory(buflen uint32, buf *uint16) (n uint32, err error) {
	r0, _, e1 := Syscall(procGetCurrentDirectoryW.Addr(), 2, uintptr(buflen), uintptr(unsafe.Pointer(buf)), 0)
	n = uint32(r0)
	if n == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func SetCurrentDirectory(path *uint16) (err error) {
	r1, _, e1 := Syscall(procSetCurrentDirectoryW.Addr(), 1, uintptr(unsafe.Pointer(path)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CreateDirectory(path *uint16, sa *SecurityAttributes) (err error) {
	r1, _, e1 := Syscall(procCreateDirectoryW.Addr(), 2, uintptr(unsafe.Pointer(path)), uintptr(unsafe.Pointer(sa)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func RemoveDirectory(path *uint16) (err error) {
	r1, _, e1 := Syscall(procRemoveDirectoryW.Addr(), 1, uintptr(unsafe.Pointer(path)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func DeleteFile(path *uint16) (err error) {
	r1, _, e1 := Syscall(procDeleteFileW.Addr(), 1, uintptr(unsafe.Pointer(path)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func MoveFile(from *uint16, to *uint16) (err error) {
	r1, _, e1 := Syscall(procMoveFileW.Addr(), 2, uintptr(unsafe.Pointer(from)), uintptr(unsafe.Pointer(to)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetComputerName(buf *uint16, n *uint32) (err error) {
	r1, _, e1 := Syscall(procGetComputerNameW.Addr(), 2, uintptr(unsafe.Pointer(buf)), uintptr(unsafe.Pointer(n)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func SetEndOfFile(handle Handle) (err error) {
	r1, _, e1 := Syscall(procSetEndOfFile.Addr(), 1, uintptr(handle), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetSystemTimeAsFileTime(time *Filetime) {
	Syscall(procGetSystemTimeAsFileTime.Addr(), 1, uintptr(unsafe.Pointer(time)), 0, 0)
	return
}

func GetTimeZoneInformation(tzi *Timezoneinformation) (rc uint32, err error) {
	r0, _, e1 := Syscall(procGetTimeZoneInformation.Addr(), 1, uintptr(unsafe.Pointer(tzi)), 0, 0)
	rc = uint32(r0)
	if rc == 0xffffffff {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CreateIoCompletionPort(filehandle Handle, cphandle Handle, key uint32, threadcnt uint32) (handle Handle, err error) {
	r0, _, e1 := Syscall6(procCreateIoCompletionPort.Addr(), 4, uintptr(filehandle), uintptr(cphandle), uintptr(key), uintptr(threadcnt), 0, 0)
	handle = Handle(r0)
	if handle == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetQueuedCompletionStatus(cphandle Handle, qty *uint32, key *uint32, overlapped **Overlapped, timeout uint32) (err error) {
	r1, _, e1 := Syscall6(procGetQueuedCompletionStatus.Addr(), 5, uintptr(cphandle), uintptr(unsafe.Pointer(qty)), uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(overlapped)), uintptr(timeout), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func PostQueuedCompletionStatus(cphandle Handle, qty uint32, key uint32, overlapped *Overlapped) (err error) {
	r1, _, e1 := Syscall6(procPostQueuedCompletionStatus.Addr(), 4, uintptr(cphandle), uintptr(qty), uintptr(key), uintptr(unsafe.Pointer(overlapped)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CancelIo(s Handle) (err error) {
	r1, _, e1 := Syscall(procCancelIo.Addr(), 1, uintptr(s), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CancelIoEx(s Handle, o *Overlapped) (err error) {
	r1, _, e1 := Syscall(procCancelIoEx.Addr(), 2, uintptr(s), uintptr(unsafe.Pointer(o)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CreateProcess(appName *uint16, commandLine *uint16, procSecurity *SecurityAttributes, threadSecurity *SecurityAttributes, inheritHandles bool, creationFlags uint32, env *uint16, currentDir *uint16, startupInfo *StartupInfo, outProcInfo *ProcessInformation) (err error) {
	var _p0 uint32
	if inheritHandles {
		_p0 = 1
	} else {
		_p0 = 0
	}
	r1, _, e1 := Syscall12(procCreateProcessW.Addr(), 10, uintptr(unsafe.Pointer(appName)), uintptr(unsafe.Pointer(commandLine)), uintptr(unsafe.Pointer(procSecurity)), uintptr(unsafe.Pointer(threadSecurity)), uintptr(_p0), uintptr(creationFlags), uintptr(unsafe.Pointer(env)), uintptr(unsafe.Pointer(currentDir)), uintptr(unsafe.Pointer(startupInfo)), uintptr(unsafe.Pointer(outProcInfo)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CreateProcessAsUser(token Token, appName *uint16, commandLine *uint16, procSecurity *SecurityAttributes, threadSecurity *SecurityAttributes, inheritHandles bool, creationFlags uint32, env *uint16, currentDir *uint16, startupInfo *StartupInfo, outProcInfo *ProcessInformation) (err error) {
	var _p0 uint32
	if inheritHandles {
		_p0 = 1
	} else {
		_p0 = 0
	}
	r1, _, e1 := Syscall12(procCreateProcessAsUserW.Addr(), 11, uintptr(token), uintptr(unsafe.Pointer(appName)), uintptr(unsafe.Pointer(commandLine)), uintptr(unsafe.Pointer(procSecurity)), uintptr(unsafe.Pointer(threadSecurity)), uintptr(_p0), uintptr(creationFlags), uintptr(unsafe.Pointer(env)), uintptr(unsafe.Pointer(currentDir)), uintptr(unsafe.Pointer(startupInfo)), uintptr(unsafe.Pointer(outProcInfo)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func OpenProcess(da uint32, inheritHandle bool, pid uint32) (handle Handle, err error) {
	var _p0 uint32
	if inheritHandle {
		_p0 = 1
	} else {
		_p0 = 0
	}
	r0, _, e1 := Syscall(procOpenProcess.Addr(), 3, uintptr(da), uintptr(_p0), uintptr(pid))
	handle = Handle(r0)
	if handle == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func TerminateProcess(handle Handle, exitcode uint32) (err error) {
	r1, _, e1 := Syscall(procTerminateProcess.Addr(), 2, uintptr(handle), uintptr(exitcode), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetExitCodeProcess(handle Handle, exitcode *uint32) (err error) {
	r1, _, e1 := Syscall(procGetExitCodeProcess.Addr(), 2, uintptr(handle), uintptr(unsafe.Pointer(exitcode)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetStartupInfo(startupInfo *StartupInfo) (err error) {
	r1, _, e1 := Syscall(procGetStartupInfoW.Addr(), 1, uintptr(unsafe.Pointer(startupInfo)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetCurrentProcess() (pseudoHandle Handle, err error) {
	r0, _, e1 := Syscall(procGetCurrentProcess.Addr(), 0, 0, 0, 0)
	pseudoHandle = Handle(r0)
	if pseudoHandle == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetProcessTimes(handle Handle, creationTime *Filetime, exitTime *Filetime, kernelTime *Filetime, userTime *Filetime) (err error) {
	r1, _, e1 := Syscall6(procGetProcessTimes.Addr(), 5, uintptr(handle), uintptr(unsafe.Pointer(creationTime)), uintptr(unsafe.Pointer(exitTime)), uintptr(unsafe.Pointer(kernelTime)), uintptr(unsafe.Pointer(userTime)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func DuplicateHandle(hSourceProcessHandle Handle, hSourceHandle Handle, hTargetProcessHandle Handle, lpTargetHandle *Handle, dwDesiredAccess uint32, bInheritHandle bool, dwOptions uint32) (err error) {
	var _p0 uint32
	if bInheritHandle {
		_p0 = 1
	} else {
		_p0 = 0
	}
	r1, _, e1 := Syscall9(procDuplicateHandle.Addr(), 7, uintptr(hSourceProcessHandle), uintptr(hSourceHandle), uintptr(hTargetProcessHandle), uintptr(unsafe.Pointer(lpTargetHandle)), uintptr(dwDesiredAccess), uintptr(_p0), uintptr(dwOptions), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func WaitForSingleObject(handle Handle, waitMilliseconds uint32) (event uint32, err error) {
	r0, _, e1 := Syscall(procWaitForSingleObject.Addr(), 2, uintptr(handle), uintptr(waitMilliseconds), 0)
	event = uint32(r0)
	if event == 0xffffffff {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetTempPath(buflen uint32, buf *uint16) (n uint32, err error) {
	r0, _, e1 := Syscall(procGetTempPathW.Addr(), 2, uintptr(buflen), uintptr(unsafe.Pointer(buf)), 0)
	n = uint32(r0)
	if n == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CreatePipe(readhandle *Handle, writehandle *Handle, sa *SecurityAttributes, size uint32) (err error) {
	r1, _, e1 := Syscall6(procCreatePipe.Addr(), 4, uintptr(unsafe.Pointer(readhandle)), uintptr(unsafe.Pointer(writehandle)), uintptr(unsafe.Pointer(sa)), uintptr(size), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetFileType(filehandle Handle) (n uint32, err error) {
	r0, _, e1 := Syscall(procGetFileType.Addr(), 1, uintptr(filehandle), 0, 0)
	n = uint32(r0)
	if n == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CryptAcquireContext(provhandle *Handle, container *uint16, provider *uint16, provtype uint32, flags uint32) (err error) {
	r1, _, e1 := Syscall6(procCryptAcquireContextW.Addr(), 5, uintptr(unsafe.Pointer(provhandle)), uintptr(unsafe.Pointer(container)), uintptr(unsafe.Pointer(provider)), uintptr(provtype), uintptr(flags), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CryptReleaseContext(provhandle Handle, flags uint32) (err error) {
	r1, _, e1 := Syscall(procCryptReleaseContext.Addr(), 2, uintptr(provhandle), uintptr(flags), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CryptGenRandom(provhandle Handle, buflen uint32, buf *byte) (err error) {
	r1, _, e1 := Syscall(procCryptGenRandom.Addr(), 3, uintptr(provhandle), uintptr(buflen), uintptr(unsafe.Pointer(buf)))
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetEnvironmentStrings() (envs *uint16, err error) {
	r0, _, e1 := Syscall(procGetEnvironmentStringsW.Addr(), 0, 0, 0, 0)
	envs = (*uint16)(unsafe.Pointer(r0))
	if envs == nil {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func FreeEnvironmentStrings(envs *uint16) (err error) {
	r1, _, e1 := Syscall(procFreeEnvironmentStringsW.Addr(), 1, uintptr(unsafe.Pointer(envs)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetEnvironmentVariable(name *uint16, buffer *uint16, size uint32) (n uint32, err error) {
	r0, _, e1 := Syscall(procGetEnvironmentVariableW.Addr(), 3, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(buffer)), uintptr(size))
	n = uint32(r0)
	if n == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func SetEnvironmentVariable(name *uint16, value *uint16) (err error) {
	r1, _, e1 := Syscall(procSetEnvironmentVariableW.Addr(), 2, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(value)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func SetFileTime(handle Handle, ctime *Filetime, atime *Filetime, wtime *Filetime) (err error) {
	r1, _, e1 := Syscall6(procSetFileTime.Addr(), 4, uintptr(handle), uintptr(unsafe.Pointer(ctime)), uintptr(unsafe.Pointer(atime)), uintptr(unsafe.Pointer(wtime)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetFileAttributes(name *uint16) (attrs uint32, err error) {
	r0, _, e1 := Syscall(procGetFileAttributesW.Addr(), 1, uintptr(unsafe.Pointer(name)), 0, 0)
	attrs = uint32(r0)
	if attrs == INVALID_FILE_ATTRIBUTES {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func SetFileAttributes(name *uint16, attrs uint32) (err error) {
	r1, _, e1 := Syscall(procSetFileAttributesW.Addr(), 2, uintptr(unsafe.Pointer(name)), uintptr(attrs), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetFileAttributesEx(name *uint16, level uint32, info *byte) (err error) {
	r1, _, e1 := Syscall(procGetFileAttributesExW.Addr(), 3, uintptr(unsafe.Pointer(name)), uintptr(level), uintptr(unsafe.Pointer(info)))
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetCommandLine() (cmd *uint16) {
	r0, _, _ := Syscall(procGetCommandLineW.Addr(), 0, 0, 0, 0)
	cmd = (*uint16)(unsafe.Pointer(r0))
	return
}

func CommandLineToArgv(cmd *uint16, argc *int32) (argv *[8192]*[8192]uint16, err error) {
	r0, _, e1 := Syscall(procCommandLineToArgvW.Addr(), 2, uintptr(unsafe.Pointer(cmd)), uintptr(unsafe.Pointer(argc)), 0)
	argv = (*[8192]*[8192]uint16)(unsafe.Pointer(r0))
	if argv == nil {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func LocalFree(hmem Handle) (handle Handle, err error) {
	r0, _, e1 := Syscall(procLocalFree.Addr(), 1, uintptr(hmem), 0, 0)
	handle = Handle(r0)
	if handle != 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func SetHandleInformation(handle Handle, mask uint32, flags uint32) (err error) {
	r1, _, e1 := Syscall(procSetHandleInformation.Addr(), 3, uintptr(handle), uintptr(mask), uintptr(flags))
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func FlushFileBuffers(handle Handle) (err error) {
	r1, _, e1 := Syscall(procFlushFileBuffers.Addr(), 1, uintptr(handle), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetFullPathName(path *uint16, buflen uint32, buf *uint16, fname **uint16) (n uint32, err error) {
	r0, _, e1 := Syscall6(procGetFullPathNameW.Addr(), 4, uintptr(unsafe.Pointer(path)), uintptr(buflen), uintptr(unsafe.Pointer(buf)), uintptr(unsafe.Pointer(fname)), 0, 0)
	n = uint32(r0)
	if n == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetLongPathName(path *uint16, buf *uint16, buflen uint32) (n uint32, err error) {
	r0, _, e1 := Syscall(procGetLongPathNameW.Addr(), 3, uintptr(unsafe.Pointer(path)), uintptr(unsafe.Pointer(buf)), uintptr(buflen))
	n = uint32(r0)
	if n == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetShortPathName(longpath *uint16, shortpath *uint16, buflen uint32) (n uint32, err error) {
	r0, _, e1 := Syscall(procGetShortPathNameW.Addr(), 3, uintptr(unsafe.Pointer(longpath)), uintptr(unsafe.Pointer(shortpath)), uintptr(buflen))
	n = uint32(r0)
	if n == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CreateFileMapping(fhandle Handle, sa *SecurityAttributes, prot uint32, maxSizeHigh uint32, maxSizeLow uint32, name *uint16) (handle Handle, err error) {
	r0, _, e1 := Syscall6(procCreateFileMappingW.Addr(), 6, uintptr(fhandle), uintptr(unsafe.Pointer(sa)), uintptr(prot), uintptr(maxSizeHigh), uintptr(maxSizeLow), uintptr(unsafe.Pointer(name)))
	handle = Handle(r0)
	if handle == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func MapViewOfFile(handle Handle, access uint32, offsetHigh uint32, offsetLow uint32, length uintptr) (addr uintptr, err error) {
	r0, _, e1 := Syscall6(procMapViewOfFile.Addr(), 5, uintptr(handle), uintptr(access), uintptr(offsetHigh), uintptr(offsetLow), uintptr(length), 0)
	addr = uintptr(r0)
	if addr == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func UnmapViewOfFile(addr uintptr) (err error) {
	r1, _, e1 := Syscall(procUnmapViewOfFile.Addr(), 1, uintptr(addr), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func FlushViewOfFile(addr uintptr, length uintptr) (err error) {
	r1, _, e1 := Syscall(procFlushViewOfFile.Addr(), 2, uintptr(addr), uintptr(length), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func VirtualLock(addr uintptr, length uintptr) (err error) {
	r1, _, e1 := Syscall(procVirtualLock.Addr(), 2, uintptr(addr), uintptr(length), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func VirtualUnlock(addr uintptr, length uintptr) (err error) {
	r1, _, e1 := Syscall(procVirtualUnlock.Addr(), 2, uintptr(addr), uintptr(length), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func TransmitFile(s Handle, handle Handle, bytesToWrite uint32, bytsPerSend uint32, overlapped *Overlapped, transmitFileBuf *TransmitFileBuffers, flags uint32) (err error) {
	r1, _, e1 := Syscall9(procTransmitFile.Addr(), 7, uintptr(s), uintptr(handle), uintptr(bytesToWrite), uintptr(bytsPerSend), uintptr(unsafe.Pointer(overlapped)), uintptr(unsafe.Pointer(transmitFileBuf)), uintptr(flags), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func ReadDirectoryChanges(handle Handle, buf *byte, buflen uint32, watchSubTree bool, mask uint32, retlen *uint32, overlapped *Overlapped, completionRoutine uintptr) (err error) {
	var _p0 uint32
	if watchSubTree {
		_p0 = 1
	} else {
		_p0 = 0
	}
	r1, _, e1 := Syscall9(procReadDirectoryChangesW.Addr(), 8, uintptr(handle), uintptr(unsafe.Pointer(buf)), uintptr(buflen), uintptr(_p0), uintptr(mask), uintptr(unsafe.Pointer(retlen)), uintptr(unsafe.Pointer(overlapped)), uintptr(completionRoutine), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CertOpenSystemStore(hprov Handle, name *uint16) (store Handle, err error) {
	r0, _, e1 := Syscall(procCertOpenSystemStoreW.Addr(), 2, uintptr(hprov), uintptr(unsafe.Pointer(name)), 0)
	store = Handle(r0)
	if store == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CertOpenStore(storeProvider uintptr, msgAndCertEncodingType uint32, cryptProv uintptr, flags uint32, para uintptr) (handle Handle, err error) {
	r0, _, e1 := Syscall6(procCertOpenStore.Addr(), 5, uintptr(storeProvider), uintptr(msgAndCertEncodingType), uintptr(cryptProv), uintptr(flags), uintptr(para), 0)
	handle = Handle(r0)
	if handle == InvalidHandle {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CertEnumCertificatesInStore(store Handle, prevContext *CertContext) (context *CertContext, err error) {
	r0, _, e1 := Syscall(procCertEnumCertificatesInStore.Addr(), 2, uintptr(store), uintptr(unsafe.Pointer(prevContext)), 0)
	context = (*CertContext)(unsafe.Pointer(r0))
	if context == nil {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CertAddCertificateContextToStore(store Handle, certContext *CertContext, addDisposition uint32, storeContext **CertContext) (err error) {
	r1, _, e1 := Syscall6(procCertAddCertificateContextToStore.Addr(), 4, uintptr(store), uintptr(unsafe.Pointer(certContext)), uintptr(addDisposition), uintptr(unsafe.Pointer(storeContext)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CertCloseStore(store Handle, flags uint32) (err error) {
	r1, _, e1 := Syscall(procCertCloseStore.Addr(), 2, uintptr(store), uintptr(flags), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CertGetCertificateChain(engine Handle, leaf *CertContext, time *Filetime, additionalStore Handle, para *CertChainPara, flags uint32, reserved uintptr, chainCtx **CertChainContext) (err error) {
	r1, _, e1 := Syscall9(procCertGetCertificateChain.Addr(), 8, uintptr(engine), uintptr(unsafe.Pointer(leaf)), uintptr(unsafe.Pointer(time)), uintptr(additionalStore), uintptr(unsafe.Pointer(para)), uintptr(flags), uintptr(reserved), uintptr(unsafe.Pointer(chainCtx)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CertFreeCertificateChain(ctx *CertChainContext) {
	Syscall(procCertFreeCertificateChain.Addr(), 1, uintptr(unsafe.Pointer(ctx)), 0, 0)
	return
}

func CertCreateCertificateContext(certEncodingType uint32, certEncoded *byte, encodedLen uint32) (context *CertContext, err error) {
	r0, _, e1 := Syscall(procCertCreateCertificateContext.Addr(), 3, uintptr(certEncodingType), uintptr(unsafe.Pointer(certEncoded)), uintptr(encodedLen))
	context = (*CertContext)(unsafe.Pointer(r0))
	if context == nil {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CertFreeCertificateContext(ctx *CertContext) (err error) {
	r1, _, e1 := Syscall(procCertFreeCertificateContext.Addr(), 1, uintptr(unsafe.Pointer(ctx)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CertVerifyCertificateChainPolicy(policyOID uintptr, chain *CertChainContext, para *CertChainPolicyPara, status *CertChainPolicyStatus) (err error) {
	r1, _, e1 := Syscall6(procCertVerifyCertificateChainPolicy.Addr(), 4, uintptr(policyOID), uintptr(unsafe.Pointer(chain)), uintptr(unsafe.Pointer(para)), uintptr(unsafe.Pointer(status)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func RegOpenKeyEx(key Handle, subkey *uint16, options uint32, desiredAccess uint32, result *Handle) (regerrno error) {
	r0, _, _ := Syscall6(procRegOpenKeyExW.Addr(), 5, uintptr(key), uintptr(unsafe.Pointer(subkey)), uintptr(options), uintptr(desiredAccess), uintptr(unsafe.Pointer(result)), 0)
	if r0 != 0 {
		regerrno = Errno(r0)
	}
	return
}

func RegCloseKey(key Handle) (regerrno error) {
	r0, _, _ := Syscall(procRegCloseKey.Addr(), 1, uintptr(key), 0, 0)
	if r0 != 0 {
		regerrno = Errno(r0)
	}
	return
}

func RegQueryInfoKey(key Handle, class *uint16, classLen *uint32, reserved *uint32, subkeysLen *uint32, maxSubkeyLen *uint32, maxClassLen *uint32, valuesLen *uint32, maxValueNameLen *uint32, maxValueLen *uint32, saLen *uint32, lastWriteTime *Filetime) (regerrno error) {
	r0, _, _ := Syscall12(procRegQueryInfoKeyW.Addr(), 12, uintptr(key), uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(classLen)), uintptr(unsafe.Pointer(reserved)), uintptr(unsafe.Pointer(subkeysLen)), uintptr(unsafe.Pointer(maxSubkeyLen)), uintptr(unsafe.Pointer(maxClassLen)), uintptr(unsafe.Pointer(valuesLen)), uintptr(unsafe.Pointer(maxValueNameLen)), uintptr(unsafe.Pointer(maxValueLen)), uintptr(unsafe.Pointer(saLen)), uintptr(unsafe.Pointer(lastWriteTime)))
	if r0 != 0 {
		regerrno = Errno(r0)
	}
	return
}

func RegEnumKeyEx(key Handle, index uint32, name *uint16, nameLen *uint32, reserved *uint32, class *uint16, classLen *uint32, lastWriteTime *Filetime) (regerrno error) {
	r0, _, _ := Syscall9(procRegEnumKeyExW.Addr(), 8, uintptr(key), uintptr(index), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(nameLen)), uintptr(unsafe.Pointer(reserved)), uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(classLen)), uintptr(unsafe.Pointer(lastWriteTime)), 0)
	if r0 != 0 {
		regerrno = Errno(r0)
	}
	return
}

func RegQueryValueEx(key Handle, name *uint16, reserved *uint32, valtype *uint32, buf *byte, buflen *uint32) (regerrno error) {
	r0, _, _ := Syscall6(procRegQueryValueExW.Addr(), 6, uintptr(key), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(reserved)), uintptr(unsafe.Pointer(valtype)), uintptr(unsafe.Pointer(buf)), uintptr(unsafe.Pointer(buflen)))
	if r0 != 0 {
		regerrno = Errno(r0)
	}
	return
}

func getCurrentProcessId() (pid uint32) {
	r0, _, _ := Syscall(procGetCurrentProcessId.Addr(), 0, 0, 0, 0)
	pid = uint32(r0)
	return
}

func GetConsoleMode(console Handle, mode *uint32) (err error) {
	r1, _, e1 := Syscall(procGetConsoleMode.Addr(), 2, uintptr(console), uintptr(unsafe.Pointer(mode)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func WriteConsole(console Handle, buf *uint16, towrite uint32, written *uint32, reserved *byte) (err error) {
	r1, _, e1 := Syscall6(procWriteConsoleW.Addr(), 5, uintptr(console), uintptr(unsafe.Pointer(buf)), uintptr(towrite), uintptr(unsafe.Pointer(written)), uintptr(unsafe.Pointer(reserved)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func ReadConsole(console Handle, buf *uint16, toread uint32, read *uint32, inputControl *byte) (err error) {
	r1, _, e1 := Syscall6(procReadConsoleW.Addr(), 5, uintptr(console), uintptr(unsafe.Pointer(buf)), uintptr(toread), uintptr(unsafe.Pointer(read)), uintptr(unsafe.Pointer(inputControl)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CreateToolhelp32Snapshot(flags uint32, processId uint32) (handle Handle, err error) {
	r0, _, e1 := Syscall(procCreateToolhelp32Snapshot.Addr(), 2, uintptr(flags), uintptr(processId), 0)
	handle = Handle(r0)
	if handle == InvalidHandle {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func Process32First(snapshot Handle, procEntry *ProcessEntry32) (err error) {
	r1, _, e1 := Syscall(procProcess32FirstW.Addr(), 2, uintptr(snapshot), uintptr(unsafe.Pointer(procEntry)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func Process32Next(snapshot Handle, procEntry *ProcessEntry32) (err error) {
	r1, _, e1 := Syscall(procProcess32NextW.Addr(), 2, uintptr(snapshot), uintptr(unsafe.Pointer(procEntry)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func DeviceIoControl(handle Handle, ioControlCode uint32, inBuffer *byte, inBufferSize uint32, outBuffer *byte, outBufferSize uint32, bytesReturned *uint32, overlapped *Overlapped) (err error) {
	r1, _, e1 := Syscall9(procDeviceIoControl.Addr(), 8, uintptr(handle), uintptr(ioControlCode), uintptr(unsafe.Pointer(inBuffer)), uintptr(inBufferSize), uintptr(unsafe.Pointer(outBuffer)), uintptr(outBufferSize), uintptr(unsafe.Pointer(bytesReturned)), uintptr(unsafe.Pointer(overlapped)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CreateSymbolicLink(symlinkfilename *uint16, targetfilename *uint16, flags uint32) (err error) {
	r1, _, e1 := Syscall(procCreateSymbolicLinkW.Addr(), 3, uintptr(unsafe.Pointer(symlinkfilename)), uintptr(unsafe.Pointer(targetfilename)), uintptr(flags))
	if r1&0xff == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func CreateHardLink(filename *uint16, existingfilename *uint16, reserved uintptr) (err error) {
	r1, _, e1 := Syscall(procCreateHardLinkW.Addr(), 3, uintptr(unsafe.Pointer(filename)), uintptr(unsafe.Pointer(existingfilename)), uintptr(reserved))
	if r1&0xff == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func WSAStartup(verreq uint32, data *WSAData) (sockerr error) {
	r0, _, _ := Syscall(procWSAStartup.Addr(), 2, uintptr(verreq), uintptr(unsafe.Pointer(data)), 0)
	if r0 != 0 {
		sockerr = Errno(r0)
	}
	return
}

func WSACleanup() (err error) {
	r1, _, e1 := Syscall(procWSACleanup.Addr(), 0, 0, 0, 0)
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func WSAIoctl(s Handle, iocc uint32, inbuf *byte, cbif uint32, outbuf *byte, cbob uint32, cbbr *uint32, overlapped *Overlapped, completionRoutine uintptr) (err error) {
	r1, _, e1 := Syscall9(procWSAIoctl.Addr(), 9, uintptr(s), uintptr(iocc), uintptr(unsafe.Pointer(inbuf)), uintptr(cbif), uintptr(unsafe.Pointer(outbuf)), uintptr(cbob), uintptr(unsafe.Pointer(cbbr)), uintptr(unsafe.Pointer(overlapped)), uintptr(completionRoutine))
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func socket(af int32, typ int32, protocol int32) (handle Handle, err error) {
	r0, _, e1 := Syscall(procsocket.Addr(), 3, uintptr(af), uintptr(typ), uintptr(protocol))
	handle = Handle(r0)
	if handle == InvalidHandle {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func Setsockopt(s Handle, level int32, optname int32, optval *byte, optlen int32) (err error) {
	r1, _, e1 := Syscall6(procsetsockopt.Addr(), 5, uintptr(s), uintptr(level), uintptr(optname), uintptr(unsafe.Pointer(optval)), uintptr(optlen), 0)
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func Getsockopt(s Handle, level int32, optname int32, optval *byte, optlen *int32) (err error) {
	r1, _, e1 := Syscall6(procgetsockopt.Addr(), 5, uintptr(s), uintptr(level), uintptr(optname), uintptr(unsafe.Pointer(optval)), uintptr(unsafe.Pointer(optlen)), 0)
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func bind(s Handle, name unsafe.Pointer, namelen int32) (err error) {
	r1, _, e1 := Syscall(procbind.Addr(), 3, uintptr(s), uintptr(name), uintptr(namelen))
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func connect(s Handle, name unsafe.Pointer, namelen int32) (err error) {
	r1, _, e1 := Syscall(procconnect.Addr(), 3, uintptr(s), uintptr(name), uintptr(namelen))
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func getsockname(s Handle, rsa *RawSockaddrAny, addrlen *int32) (err error) {
	r1, _, e1 := Syscall(procgetsockname.Addr(), 3, uintptr(s), uintptr(unsafe.Pointer(rsa)), uintptr(unsafe.Pointer(addrlen)))
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func getpeername(s Handle, rsa *RawSockaddrAny, addrlen *int32) (err error) {
	r1, _, e1 := Syscall(procgetpeername.Addr(), 3, uintptr(s), uintptr(unsafe.Pointer(rsa)), uintptr(unsafe.Pointer(addrlen)))
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func listen(s Handle, backlog int32) (err error) {
	r1, _, e1 := Syscall(proclisten.Addr(), 2, uintptr(s), uintptr(backlog), 0)
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func shutdown(s Handle, how int32) (err error) {
	r1, _, e1 := Syscall(procshutdown.Addr(), 2, uintptr(s), uintptr(how), 0)
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func Closesocket(s Handle) (err error) {
	r1, _, e1 := Syscall(procclosesocket.Addr(), 1, uintptr(s), 0, 0)
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func AcceptEx(ls Handle, as Handle, buf *byte, rxdatalen uint32, laddrlen uint32, raddrlen uint32, recvd *uint32, overlapped *Overlapped) (err error) {
	r1, _, e1 := Syscall9(procAcceptEx.Addr(), 8, uintptr(ls), uintptr(as), uintptr(unsafe.Pointer(buf)), uintptr(rxdatalen), uintptr(laddrlen), uintptr(raddrlen), uintptr(unsafe.Pointer(recvd)), uintptr(unsafe.Pointer(overlapped)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetAcceptExSockaddrs(buf *byte, rxdatalen uint32, laddrlen uint32, raddrlen uint32, lrsa **RawSockaddrAny, lrsalen *int32, rrsa **RawSockaddrAny, rrsalen *int32) {
	Syscall9(procGetAcceptExSockaddrs.Addr(), 8, uintptr(unsafe.Pointer(buf)), uintptr(rxdatalen), uintptr(laddrlen), uintptr(raddrlen), uintptr(unsafe.Pointer(lrsa)), uintptr(unsafe.Pointer(lrsalen)), uintptr(unsafe.Pointer(rrsa)), uintptr(unsafe.Pointer(rrsalen)), 0)
	return
}

func WSARecv(s Handle, bufs *WSABuf, bufcnt uint32, recvd *uint32, flags *uint32, overlapped *Overlapped, croutine *byte) (err error) {
	r1, _, e1 := Syscall9(procWSARecv.Addr(), 7, uintptr(s), uintptr(unsafe.Pointer(bufs)), uintptr(bufcnt), uintptr(unsafe.Pointer(recvd)), uintptr(unsafe.Pointer(flags)), uintptr(unsafe.Pointer(overlapped)), uintptr(unsafe.Pointer(croutine)), 0, 0)
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func WSASend(s Handle, bufs *WSABuf, bufcnt uint32, sent *uint32, flags uint32, overlapped *Overlapped, croutine *byte) (err error) {
	r1, _, e1 := Syscall9(procWSASend.Addr(), 7, uintptr(s), uintptr(unsafe.Pointer(bufs)), uintptr(bufcnt), uintptr(unsafe.Pointer(sent)), uintptr(flags), uintptr(unsafe.Pointer(overlapped)), uintptr(unsafe.Pointer(croutine)), 0, 0)
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func WSARecvFrom(s Handle, bufs *WSABuf, bufcnt uint32, recvd *uint32, flags *uint32, from *RawSockaddrAny, fromlen *int32, overlapped *Overlapped, croutine *byte) (err error) {
	r1, _, e1 := Syscall9(procWSARecvFrom.Addr(), 9, uintptr(s), uintptr(unsafe.Pointer(bufs)), uintptr(bufcnt), uintptr(unsafe.Pointer(recvd)), uintptr(unsafe.Pointer(flags)), uintptr(unsafe.Pointer(from)), uintptr(unsafe.Pointer(fromlen)), uintptr(unsafe.Pointer(overlapped)), uintptr(unsafe.Pointer(croutine)))
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func WSASendTo(s Handle, bufs *WSABuf, bufcnt uint32, sent *uint32, flags uint32, to *RawSockaddrAny, tolen int32, overlapped *Overlapped, croutine *byte) (err error) {
	r1, _, e1 := Syscall9(procWSASendTo.Addr(), 9, uintptr(s), uintptr(unsafe.Pointer(bufs)), uintptr(bufcnt), uintptr(unsafe.Pointer(sent)), uintptr(flags), uintptr(unsafe.Pointer(to)), uintptr(tolen), uintptr(unsafe.Pointer(overlapped)), uintptr(unsafe.Pointer(croutine)))
	if r1 == socket_error {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetHostByName(name string) (h *Hostent, err error) {
	var _p0 *byte
	_p0, err = BytePtrFromString(name)
	if err != nil {
		return
	}
	return _GetHostByName(_p0)
}

func _GetHostByName(name *byte) (h *Hostent, err error) {
	r0, _, e1 := Syscall(procgethostbyname.Addr(), 1, uintptr(unsafe.Pointer(name)), 0, 0)
	h = (*Hostent)(unsafe.Pointer(r0))
	if h == nil {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetServByName(name string, proto string) (s *Servent, err error) {
	var _p0 *byte
	_p0, err = BytePtrFromString(name)
	if err != nil {
		return
	}
	var _p1 *byte
	_p1, err = BytePtrFromString(proto)
	if err != nil {
		return
	}
	return _GetServByName(_p0, _p1)
}

func _GetServByName(name *byte, proto *byte) (s *Servent, err error) {
	r0, _, e1 := Syscall(procgetservbyname.Addr(), 2, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(proto)), 0)
	s = (*Servent)(unsafe.Pointer(r0))
	if s == nil {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func Ntohs(netshort uint16) (u uint16) {
	r0, _, _ := Syscall(procntohs.Addr(), 1, uintptr(netshort), 0, 0)
	u = uint16(r0)
	return
}

func GetProtoByName(name string) (p *Protoent, err error) {
	var _p0 *byte
	_p0, err = BytePtrFromString(name)
	if err != nil {
		return
	}
	return _GetProtoByName(_p0)
}

func _GetProtoByName(name *byte) (p *Protoent, err error) {
	r0, _, e1 := Syscall(procgetprotobyname.Addr(), 1, uintptr(unsafe.Pointer(name)), 0, 0)
	p = (*Protoent)(unsafe.Pointer(r0))
	if p == nil {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func DnsQuery(name string, qtype uint16, options uint32, extra *byte, qrs **DNSRecord, pr *byte) (status error) {
	var _p0 *uint16
	_p0, status = UTF16PtrFromString(name)
	if status != nil {
		return
	}
	return _DnsQuery(_p0, qtype, options, extra, qrs, pr)
}

func _DnsQuery(name *uint16, qtype uint16, options uint32, extra *byte, qrs **DNSRecord, pr *byte) (status error) {
	r0, _, _ := Syscall6(procDnsQuery_W.Addr(), 6, uintptr(unsafe.Pointer(name)), uintptr(qtype), uintptr(options), uintptr(unsafe.Pointer(extra)), uintptr(unsafe.Pointer(qrs)), uintptr(unsafe.Pointer(pr)))
	if r0 != 0 {
		status = Errno(r0)
	}
	return
}

func DnsRecordListFree(rl *DNSRecord, freetype uint32) {
	Syscall(procDnsRecordListFree.Addr(), 2, uintptr(unsafe.Pointer(rl)), uintptr(freetype), 0)
	return
}

func DnsNameCompare(name1 *uint16, name2 *uint16) (same bool) {
	r0, _, _ := Syscall(procDnsNameCompare_W.Addr(), 2, uintptr(unsafe.Pointer(name1)), uintptr(unsafe.Pointer(name2)), 0)
	same = r0 != 0
	return
}

func GetAddrInfoW(nodename *uint16, servicename *uint16, hints *AddrinfoW, result **AddrinfoW) (sockerr error) {
	r0, _, _ := Syscall6(procGetAddrInfoW.Addr(), 4, uintptr(unsafe.Pointer(nodename)), uintptr(unsafe.Pointer(servicename)), uintptr(unsafe.Pointer(hints)), uintptr(unsafe.Pointer(result)), 0, 0)
	if r0 != 0 {
		sockerr = Errno(r0)
	}
	return
}

func FreeAddrInfoW(addrinfo *AddrinfoW) {
	Syscall(procFreeAddrInfoW.Addr(), 1, uintptr(unsafe.Pointer(addrinfo)), 0, 0)
	return
}

func GetIfEntry(pIfRow *MibIfRow) (errcode error) {
	r0, _, _ := Syscall(procGetIfEntry.Addr(), 1, uintptr(unsafe.Pointer(pIfRow)), 0, 0)
	if r0 != 0 {
		errcode = Errno(r0)
	}
	return
}

func GetAdaptersInfo(ai *IpAdapterInfo, ol *uint32) (errcode error) {
	r0, _, _ := Syscall(procGetAdaptersInfo.Addr(), 2, uintptr(unsafe.Pointer(ai)), uintptr(unsafe.Pointer(ol)), 0)
	if r0 != 0 {
		errcode = Errno(r0)
	}
	return
}

func SetFileCompletionNotificationModes(handle Handle, flags uint8) (err error) {
	r1, _, e1 := Syscall(procSetFileCompletionNotificationModes.Addr(), 2, uintptr(handle), uintptr(flags), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func WSAEnumProtocols(protocols *int32, protocolBuffer *WSAProtocolInfo, bufferLength *uint32) (n int32, err error) {
	r0, _, e1 := Syscall(procWSAEnumProtocolsW.Addr(), 3, uintptr(unsafe.Pointer(protocols)), uintptr(unsafe.Pointer(protocolBuffer)), uintptr(unsafe.Pointer(bufferLength)))
	n = int32(r0)
	if n == -1 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func TranslateName(accName *uint16, accNameFormat uint32, desiredNameFormat uint32, translatedName *uint16, nSize *uint32) (err error) {
	r1, _, e1 := Syscall6(procTranslateNameW.Addr(), 5, uintptr(unsafe.Pointer(accName)), uintptr(accNameFormat), uintptr(desiredNameFormat), uintptr(unsafe.Pointer(translatedName)), uintptr(unsafe.Pointer(nSize)), 0)
	if r1&0xff == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetUserNameEx(nameFormat uint32, nameBuffre *uint16, nSize *uint32) (err error) {
	r1, _, e1 := Syscall(procGetUserNameExW.Addr(), 3, uintptr(nameFormat), uintptr(unsafe.Pointer(nameBuffre)), uintptr(unsafe.Pointer(nSize)))
	if r1&0xff == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func NetUserGetInfo(serverName *uint16, userName *uint16, level uint32, buf **byte) (neterr error) {
	r0, _, _ := Syscall6(procNetUserGetInfo.Addr(), 4, uintptr(unsafe.Pointer(serverName)), uintptr(unsafe.Pointer(userName)), uintptr(level), uintptr(unsafe.Pointer(buf)), 0, 0)
	if r0 != 0 {
		neterr = Errno(r0)
	}
	return
}

func NetGetJoinInformation(server *uint16, name **uint16, bufType *uint32) (neterr error) {
	r0, _, _ := Syscall(procNetGetJoinInformation.Addr(), 3, uintptr(unsafe.Pointer(server)), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(bufType)))
	if r0 != 0 {
		neterr = Errno(r0)
	}
	return
}

func NetApiBufferFree(buf *byte) (neterr error) {
	r0, _, _ := Syscall(procNetApiBufferFree.Addr(), 1, uintptr(unsafe.Pointer(buf)), 0, 0)
	if r0 != 0 {
		neterr = Errno(r0)
	}
	return
}

func LookupAccountSid(systemName *uint16, sid *SID, name *uint16, nameLen *uint32, refdDomainName *uint16, refdDomainNameLen *uint32, use *uint32) (err error) {
	r1, _, e1 := Syscall9(procLookupAccountSidW.Addr(), 7, uintptr(unsafe.Pointer(systemName)), uintptr(unsafe.Pointer(sid)), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(nameLen)), uintptr(unsafe.Pointer(refdDomainName)), uintptr(unsafe.Pointer(refdDomainNameLen)), uintptr(unsafe.Pointer(use)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func LookupAccountName(systemName *uint16, accountName *uint16, sid *SID, sidLen *uint32, refdDomainName *uint16, refdDomainNameLen *uint32, use *uint32) (err error) {
	r1, _, e1 := Syscall9(procLookupAccountNameW.Addr(), 7, uintptr(unsafe.Pointer(systemName)), uintptr(unsafe.Pointer(accountName)), uintptr(unsafe.Pointer(sid)), uintptr(unsafe.Pointer(sidLen)), uintptr(unsafe.Pointer(refdDomainName)), uintptr(unsafe.Pointer(refdDomainNameLen)), uintptr(unsafe.Pointer(use)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func ConvertSidToStringSid(sid *SID, stringSid **uint16) (err error) {
	r1, _, e1 := Syscall(procConvertSidToStringSidW.Addr(), 2, uintptr(unsafe.Pointer(sid)), uintptr(unsafe.Pointer(stringSid)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func ConvertStringSidToSid(stringSid *uint16, sid **SID) (err error) {
	r1, _, e1 := Syscall(procConvertStringSidToSidW.Addr(), 2, uintptr(unsafe.Pointer(stringSid)), uintptr(unsafe.Pointer(sid)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetLengthSid(sid *SID) (len uint32) {
	r0, _, _ := Syscall(procGetLengthSid.Addr(), 1, uintptr(unsafe.Pointer(sid)), 0, 0)
	len = uint32(r0)
	return
}

func CopySid(destSidLen uint32, destSid *SID, srcSid *SID) (err error) {
	r1, _, e1 := Syscall(procCopySid.Addr(), 3, uintptr(destSidLen), uintptr(unsafe.Pointer(destSid)), uintptr(unsafe.Pointer(srcSid)))
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func OpenProcessToken(h Handle, access uint32, token *Token) (err error) {
	r1, _, e1 := Syscall(procOpenProcessToken.Addr(), 3, uintptr(h), uintptr(access), uintptr(unsafe.Pointer(token)))
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetTokenInformation(t Token, infoClass uint32, info *byte, infoLen uint32, returnedLen *uint32) (err error) {
	r1, _, e1 := Syscall6(procGetTokenInformation.Addr(), 5, uintptr(t), uintptr(infoClass), uintptr(unsafe.Pointer(info)), uintptr(infoLen), uintptr(unsafe.Pointer(returnedLen)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func GetUserProfileDirectory(t Token, dir *uint16, dirLen *uint32) (err error) {
	r1, _, e1 := Syscall(procGetUserProfileDirectoryW.Addr(), 3, uintptr(t), uintptr(unsafe.Pointer(dir)), uintptr(unsafe.Pointer(dirLen)))
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}

func getSystemDirectory(dir *uint16, dirLen uint32) (len uint32, err error) {
	r0, _, e1 := Syscall(procGetSystemDirectoryW.Addr(), 2, uintptr(unsafe.Pointer(dir)), uintptr(dirLen), 0)
	len = uint32(r0)
	if len == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = EINVAL
		}
	}
	return
}
