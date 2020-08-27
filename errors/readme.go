/*
	结构体
		type errorString struct { //错误信息
			s string //错误内容
		}
	函数与方法
		Unwrap(err error) error						如果err包含Unwrap方法，则执行，否则返回nil
		Is(err, target error) bool					err链使用Unwrap()解包，直接判断或调用Is()与target进行比较，如果有与target相等的err则直接返回true，否则直至err解包至nil
		As(err error, target interface{}) bool		遍历err链，将err赋值给target，或err.As(target)，返回是否成功
		New(text string) error
			(e *errorString) Error() string			返回错误字符串
*/