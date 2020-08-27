/*
	接口与结构体
		type timeout interface
		type PathError struct
			(e *PathError) Error() string	返回格式化的错误
			(e *PathError) Unwrap() error	解包，返回错误Err
			(e *PathError) Timeout() bool	获取错误e是否超时
		type SyscallError struct

	函数与方法
		NewSyscallError(syscall string, err error) error 	工厂函数，生成一个SyscallError
			(e *SyscallError) Error() string				回格式化的错误
			(e *SyscallError) Unwrap() error				解包，返回错误Err
			(e *SyscallError) Timeout() bool				获取错误e是否超时
		IsExist(err error) bool								获取err的底层错误是否为"文件已存在"
		IsNotExist(err error) bool							获取err的底层错误是否为"文件不存在"
		IsPermission(err error) bool 						获取err的底层错误是否为"没有权限"
		IsTimeout(err error) bool 							获取err的底层错误是否超时
		underlyingErrorIs(err, target error) bool 			获取err的底层错误是否为target
		underlyingError(err error) error					获取err的底层错误
		--err
		errInvalid() error   		返回错误："无效的参数"
		errPermission() error		返回错误："没有权限"
		errExist() error      		返回错误："文件已存在"
		errNotExist() error   		返回错误："文件不存在
		errClosed() error    		返回错误："文件已关闭"
		errNoDeadline() error 		返回错误："文件类型不支持截止日期"




	用法


*/