// 版权所有2018 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package errors

import (
	"internal/reflectlite"
)

//Unwrap 如果err的类型包含返回错误的Unwrap方法，则返回对err调用Unwrap方法的结果。
//否则，Unwrap返回nil。
func Unwrap(err error) error { //注：如果err包含Unwrap方法，则执行，否则返回nil
	u, ok := err.(interface {
		Unwrap() error
	})
	if !ok {
		return nil
	}
	return u.Unwrap()
}

// Is 报告err链中的任何错误是否与目标匹配。
// 链由err本身组成，后面是通过重复调用Unwrap获得的错误序列。
// 如果错误等于目标，或者它实现了Is(error) bool使得Is(target)返回true，则认为该错误与目标匹配。
// 错误类型可能提供Is方法，因此可以将其视为等效于现有错误。
// 例如，如果MyError定义
// func(m MyError) Is(target error) bool {return target == os.ErrExist}
// 然后Is(MyError {}，os.ErrExist)返回true。 有关标准库中的示例，请参见syscall.Errno.Is。
func Is(err, target error) bool { //注：err链使用Unwrap()解包，直接判断或调用Is()与target进行比较，如果有与target相等的err则直接返回true，否则直至err解包至nil
	if target == nil { //注：nil可以进行比较
		return err == target
	}

	isComparable := reflectlite.TypeOf(target).Comparable() //注：target的类型是否可以进行比较
	for {
		if isComparable && err == target { //注：如果可以直接比较，进行比较
			return true
		}
		if x, ok := err.(interface{ Is(error) bool }); ok && x.Is(target) { //注：如果err实现了Is方法，则执行Is比较
			return true
		}
		// TODO：考虑支持target.Is（err）。
		// 这将允许用户定义谓词，但也可能允许处理草率的API，从而更容易摆脱它们。
		if err = Unwrap(err); err == nil { //注：err解包
			return false
		}
	}
}

// As 查找err链中与目标匹配的第一个错误，如果匹配，则将target设置为该错误值并返回true。 否则，它返回false。
// 链由err本身组成，后面是通过重复调用Unwrap获得的错误序列。
// 如果错误的具体值可分配给目标指向的值，或者错误具有方法As(interface{})bool使得As(target)返回true，则错误与目标匹配。
// 在后一种情况下，As方法负责设置目标。
// 错误类型可能提供As方法，因此可以将其视为其他错误类型。
// 如果target不是实现错误的类型或任何接口类型的非nil指针，则作为恐慌。
func As(err error, target interface{}) bool { //注：遍历err链，将err赋值给target，或err.As(target)，返回是否成功
	if target == nil {
		panic("errors: target cannot be nil") //注：恐慌"target不能为nil"
	}
	val := reflectlite.ValueOf(target) //注：获取target的值反射
	typ := val.Type()
	if typ.Kind() != reflectlite.Ptr || val.IsNil() { //注：必须为非空指针
		panic("errors: target must be a non-nil pointer") //注：恐慌"目标必须是非空指针"
	}
	if e := typ.Elem(); e.Kind() != reflectlite.Interface && !e.Implements(errorType) { //注：如果e的类型不是接口并且不包含error类型，发生恐慌
		panic("errors: *target must be interface or implement error") //注：恐慌"必须是接口或实现错误"
	}
	targetType := typ.Elem()
	for err != nil {
		if reflectlite.TypeOf(err).AssignableTo(targetType) { //注：如果err的值可以分配给target
			val.Elem().Set(reflectlite.ValueOf(err)) //注：将err赋值给target，返回true
			return true
		}
		if x, ok := err.(interface{ As(interface{}) bool }); ok && x.As(target) { //注：如果err实现As方法，执行后返回true
			return true
		}
		err = Unwrap(err) //注：解包错误
	}
	return false
}

//注：error的元素类型
var errorType = reflectlite.TypeOf((*error)(nil)).Elem()
