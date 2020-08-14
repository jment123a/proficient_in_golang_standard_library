// 版权所有2018 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package fmt

import "errors"

// Errorf 根据格式说明符进行格式化，然后将字符串作为满足错误的值返回。
// 如果格式说明符包含带有错误操作数的%w，则返回的错误将实现Unwrap方法，返回操作数。
// 包含多个%w动词或向其提供未实现错误接口的操作数是无效的。 另外，%w是%v的同义词。
func Errorf(format string, a ...interface{}) error { //注：获取使用format格式化a时发生的错误
	p := newPrinter()
	p.wrapErrs = true
	p.doPrintf(format, a)
	s := string(p.buf)
	var err error
	if p.wrappedErr == nil { //注：是否获取到error
		err = errors.New(s) //注:返回error，内容为格式化后的a
	} else {
		err = &wrapError{s, p.wrappedErr} //注：返回wrapError，错误信息为格式化后的a，错误为获取到的错误
	}
	p.free()
	return err
}

type wrapError struct {
	msg string
	err error
}

func (e *wrapError) Error() string {
	return e.msg
}

func (e *wrapError) Unwrap() error {
	return e.err
}
