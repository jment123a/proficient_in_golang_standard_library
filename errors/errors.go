// 版权所有2011 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

// Package errors 实现了处理错误的功能。
//
// New函数创建错误，其唯一的内容是文本消息。
//
// Unwrap，Is和As函数处理可能会包装其他错误的错误。
// 如果错误的类型具有方法，则该错误会包装另一个错误
// Unwrap() error
// 如果e.Unwrap()返回一个非nil的错误w，那么我们说e包装了w。
// Unwrap解包已包装的错误。 如果其参数的类型具有Unwrap方法，则将调用该方法一次。 否则，它返回nil。
// 创建包装错误的一种简单方法是调用fmt.Errorf并应用％w动词
// 错误参数：
// errors.Unwrap(fmt.Errorf("... %w ...", ..., err, ...))
// returns err.
//
// Is 依次展开第一个参数，以查找与第二个参数匹配的错误。 它报告是否找到匹配项。 它应该优先于简单的相等性检查使用：
// if errors.Is(err, os.ErrExist) 优于 if err == os.ErrExist
//
// 因为如果err包装os.ErrExist，前者将成功。
//
// As 依次解开第一个参数，以寻找可以分配给第二个参数的错误，该错误必须是指针。 如果成功，它将执行分配并返回true。 否则，它返回false。 表格
//
//	var perr *os.PathError
//	if errors.As(err, &perr) {
//		fmt.Println(perr.Path)
//	}
//
// 优于
//
//	if perr, ok := err.(*os.PathError); ok {
//		fmt.Println(perr.Path)
//	}
//
// 因为如果err包装*os.PathError，前者将成功。
package errors

// New 返回一个错误，该错误的格式为给定的文本。
//每次调用New都会返回一个不同的错误值，即使文本相同。
func New(text string) error { //工厂函数，包装错误
	return &errorString{text}
}

// errorString是错误的简单实现。
type errorString struct {
	s string
}

func (e *errorString) Error() string { //注：返回错误字符串
	return e.s
}
