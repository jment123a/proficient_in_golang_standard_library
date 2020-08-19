// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

/*
	Package flag 实现命令行标志解析。
	用法
	使用flag.String()，Bool()，Int()等定义标志。
	这将声明一个整数标志-flagname，它存储在指针ip中，类型为*int。
	import "flag"
	var ip = flag.Int("flagname", 1234, "help message for flagname")
	如果愿意，可以使用Var()函数将标志绑定到变量。
		var flagvar int
		func init() {
			flag.IntVar(&flagvar, "flagname", 1234, "help message for flagname")
		}
	或者，您可以创建满足Value接口（带有指针接收器）的自定义标志，并将它们耦合到标志解析，方法是：
		flag.Var(&flagVal, "name", "help message for flagname")
	对于此类标志，默认值只是变量的初始值。
	定义所有标志后，调用
		flag.Parse()
	将命令行解析为定义的标志。
	然后可以直接使用标志。 如果您自己使用这些标志，它们都是指针。 如果您绑定到变量，它们就是值。
		fmt.Println("ip has value ", *ip)
		fmt.Println("flagvar has value ", flagvar)
	解析之后，标记后面的参数可以用作切片flag.Args()或单独用作flag.Arg(i)。
	参数从0到flag.NArg()-1进行索引。

	命令行标志语法
	允许使用以下形式：
		-flag
		-flag=x
		-flag x  // 仅非布尔标志
	可以使用一个或两个减号； 它们是等效的。
	布尔标志不允许使用最后一种形式，因为命令的含义
		cmd -x *
	其中*是Unix shell通配符，如果存在名为0，false等的文件，则将更改。您必须使用-flag = false形式来关闭布尔标志。

	标志解析在第一个非标志参数("-"是非标志参数)之前或终止符"-"之后停止。

	整数标志接受1234、0664、0x1234，并且可以为负。
	布尔标志可以是：
		1, 0, t, f, T, F, true, false, TRUE, FALSE, True, False
	持续时间标志接受任何对time.ParseDuration有效的输入。

	命令行标志的默认集合由顶级功能控制。
	FlagSet类型允许人们定义独立的标志集，例如在命令行界面中实现子命令。
	FlagSet的方法类似于命令行标志集的顶级功能。
*/
package flag

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ErrHelp 如果调用-help或-h标志但未定义此类标志，则返回错误。
var ErrHelp = errors.New("flag: help requested") //注：标志：请求帮助

// errParse 如果标志的值解析失败（例如Int的整数无效），则Set返回。
// 然后通过failf进行包装以提供更多信息。
var errParse = errors.New("parse error") //注：解析错误

// errRange 如果标志的值超出范围，则由Set返回。
// 然后通过failf进行包装以提供更多信息。
var errRange = errors.New("value out of range") //注：值超出范围

func numError(err error) error { //注：对语法错误与超界错误进行格式化
	ne, ok := err.(*strconv.NumError)
	if !ok {
		return err
	}
	if ne.Err == strconv.ErrSyntax {
		return errParse
	}
	if ne.Err == strconv.ErrRange {
		return errRange
	}
	return err
}

// -- bool
type boolValue bool

func newBoolValue(val bool, p *bool) *boolValue { //工厂函数，将val赋值给p，返回boolValue类型的p
	*p = val
	return (*boolValue)(p)
}

func (b *boolValue) Set(s string) error { //注：将s转为bool赋值给b
	v, err := strconv.ParseBool(s) //注：将s转为布尔类型
	if err != nil {
		err = errParse
	}
	*b = boolValue(v) //注：将v赋值给b
	return err
}

func (b *boolValue) Get() interface{} { return bool(*b) } //注：获取bool类型的b

func (b *boolValue) String() string { return strconv.FormatBool(bool(*b)) } 将b格式化为字符串并返回

func (b *boolValue) IsBoolFlag() bool { return true } //注：只有boolValue类型才有的方法，返回true

// 可选接口，用于指示可以提供不带"=value"文本的布尔标志
type boolFlag interface {
	Value
	IsBoolFlag() bool
}

// -- int
type intValue int

func newIntValue(val int, p *int) *intValue { //工厂函数，将val赋值给p，返回intValue类型的p
	*p = val
	return (*intValue)(p)
}

func (i *intValue) Set(s string) error { //注：将s转为数字后赋值给i
	v, err := strconv.ParseInt(s, 0, strconv.IntSize)
	if err != nil {
		err = numError(err)
	}
	*i = intValue(v)
	return err
}

func (i *intValue) Get() interface{} { return int(*i) } //注：获取int类型i

func (i *intValue) String() string { return strconv.Itoa(int(*i)) } //注：将i转为字符串并返回

// -- int64
type int64Value int64

func newInt64Value(val int64, p *int64) *int64Value { //工厂函数，将val赋值给p，返回int64Value类型的p
	*p = val
	return (*int64Value)(p)
}

func (i *int64Value) Set(s string) error { //注：将s转为int64后赋值给i
	v, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		err = numError(err)
	}
	*i = int64Value(v)
	return err
}

func (i *int64Value) Get() interface{} { return int64(*i) } //注：获取int64类型i

func (i *int64Value) String() string { return strconv.FormatInt(int64(*i), 10) } //注：将i转为字符串并返回

// -- uint
type uintValue uint

func newUintValue(val uint, p *uint) *uintValue { //工厂函数，将val赋值给p，返回uint类型的p
	*p = val
	return (*uintValue)(p)
}

func (i *uintValue) Set(s string) error { //注：将s转为uint后赋值给i
	v, err := strconv.ParseUint(s, 0, strconv.IntSize)
	if err != nil {
		err = numError(err)
	}
	*i = uintValue(v)
	return err
}

func (i *uintValue) Get() interface{} { return uint(*i) } //注：获取uint类型i

func (i *uintValue) String() string { return strconv.FormatUint(uint64(*i), 10) } //注：将i转为字符串并返回

// -- uint64
type uint64Value uint64

func newUint64Value(val uint64, p *uint64) *uint64Value { //工厂函数，将val赋值给p，返回uint64Value类型的p
	*p = val
	return (*uint64Value)(p)
}

func (i *uint64Value) Set(s string) error { //注：将s转为uint赋值给i
	v, err := strconv.ParseUint(s, 0, 64)
	if err != nil {
		err = numError(err)
	}
	*i = uint64Value(v)
	return err
}

func (i *uint64Value) Get() interface{} { return uint64(*i) } //注：获取uint64类型i

func (i *uint64Value) String() string { return strconv.FormatUint(uint64(*i), 10) } //注：将i转为字符串并返回

// -- string
type stringValue string

func newStringValue(val string, p *string) *stringValue { //工厂函数，将val赋值给p，返回strinValue类型的p
	*p = val
	return (*stringValue)(p)
}

func (s *stringValue) Set(val string) error { //注：将val赋值给s
	*s = stringValue(val)
	return nil
}

func (s *stringValue) Get() interface{} { return string(*s) } //注：获取string类型的s

func (s *stringValue) String() string { return string(*s) } //注：获取string类型的s，同上

// -- float64
type float64Value float64

func newFloat64Value(val float64, p *float64) *float64Value { //工厂函数，将val赋值给p，返回float64Value类型的p
	*p = val
	return (*float64Value)(p)
}

func (f *float64Value) Set(s string) error { //注：将s转为float64类型赋值给f
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		err = numError(err)
	}
	*f = float64Value(v)
	return err
}

func (f *float64Value) Get() interface{} { return float64(*f) } //注：获取float64类型的f

func (f *float64Value) String() string { return strconv.FormatFloat(float64(*f), 'g', -1, 64) } //注：将f转为字符串并返回

// -- time.Duration
type durationValue time.Duration

func newDurationValue(val time.Duration, p *time.Duration) *durationValue { //工厂函数将val赋值给p，返回durationValue类型的p
	*p = val
	return (*durationValue)(p)
}

func (d *durationValue) Set(s string) error { //注：将s转为时间赋值给d
	v, err := time.ParseDuration(s)
	if err != nil {
		err = errParse
	}
	*d = durationValue(v)
	return err
}

func (d *durationValue) Get() interface{} { return time.Duration(*d) } //注：返回time.Duration类型的d

func (d *durationValue) String() string { return (*time.Duration)(d).String() } //注：将d转为字符串并返回

// Value 是存储在标志中的动态值的接口。
// （默认值表示为字符串。）
// 如果某个Value的IsBoolFlag() bool方法返回true，则命令行解析器将-name等效于-name = true，而不使用下一个命令行参数。
// 对于每个存在的标志，按命令行顺序调用一次Set。
// 标志包可以使用零值接收器（例如nil指针）调用String方法。
type Value interface {
	String() string
	Set(string) error
}

// Getter 是一个接口，允许检索值的内容。
// 它包装Value接口，而不是它的一部分，因为它出现在Go 1及其兼容性规则之后。 此包提供的所有值类型都满足Getter接口。
type Getter interface {
	Value
	Get() interface{}
}

// ErrorHandling 定义在解析失败时FlagSet.Parse的行为。
type ErrorHandling int //注：发生错误时的行为

// 如果解析失败，这些常量将使FlagSet.Parse表现为所描述的。
const (
	ContinueOnError ErrorHandling = iota // 返回一个描述性错误。
	ExitOnError                          // 调用 os.Exit(2).
	PanicOnError                         // 产生恐慌并带有描述性错误。
)

// FlagSet 代表一组已定义的标志。
// FlagSet 的零值没有名称，并且具有ContinueOnError错误处理。
// 标志名称在FlagSet中必须唯一。
// 尝试定义名称已被使用的标志将引起恐慌。
//
// 例：执行 go run main.go
type FlagSet struct {
	// Usage是在解析标志时发生错误时调用的函数。
	// 该字段是一个函数（不是方法），可以更改为指向自定义错误处理程序。
	// 调用Usage后会发生什么情况取决于ErrorHandling设置。
	// 对于命令行，默认为ExitOnError，在调用Usage后退出程序。
	Usage func()

	name          string           //注：应用的绝对路径
	parsed        bool             //注：f是否执行过Parse转换参数
	actual        map[string]*Flag //注：实参列表
	formal        map[string]*Flag //注：形参列表
	args          []string         // 标志后的参数
	errorHandling ErrorHandling    //注：发生错误时的行为
	output        io.Writer        // nil表示stderr； 使用out()访问器
}

// Flag 表示标志的状态。
// 例：fs.String("A", "1", "`int`参数A")
// Name：A
// Usage：`int`参数A
// Value：1
// DefValue：1
type Flag struct {
	Name     string // 在命令行上显示的名称
	Usage    string // 帮助信息
	Value    Value  // 设定值
	DefValue string // 默认值（以文本形式）； 使用信息，注：Value的字符串表现形式，用于提示
}

// sortFlags 将这些标志作为按字典顺序的切片返回。
func sortFlags(flags map[string]*Flag) []*Flag { //注：将flags转为[]*Flag格式，根据名称排序并返回
	result := make([]*Flag, len(flags))
	i := 0
	for _, f := range flags {
		result[i] = f
		i++
	}
	sort.Slice(result, func(i, j int) bool { //注：按名称排序
		return result[i].Name < result[j].Name
	})
	return result
}

// Output 返回用法和错误消息的目的地。 如果未设置输出或将其设置为nil，则返回os.Stderr。
func (f *FlagSet) Output() io.Writer { //注：获取f.output，如果为nil返回os.Stderr
	if f.output == nil {
		return os.Stderr
	}
	return f.output
}

// Name 返回标志集的名称。
func (f *FlagSet) Name() string { //注：获取f.name
	return f.name
}

// ErrorHandling 返回标志集的错误处理行为。
func (f *FlagSet) ErrorHandling() ErrorHandling { //注：获取f.errorHandling，解析发生错误时的行为
	return f.errorHandling
}

// SetOutput 设置用法和错误消息的目的地。
// 如果输出为nil，则使用os.Stderr。
func (f *FlagSet) SetOutput(output io.Writer) { //注：将f.output设置为output
	f.output = output
}

// VisitAll 按字典顺序访问标志，分别对fn调用。
// 它访问所有标志，即使未设置也是如此。
func (f *FlagSet) VisitAll(fn func(*Flag)) { //注：遍历排序后的形参集合，将其作为参数调用fn
	for _, flag := range sortFlags(f.formal) {
		fn(flag)
	}
}

// VisitAll 按字典顺序访问命令行标志，为每个标志调用fn。 它访问所有标志，甚至没有设置的标志。
func VisitAll(fn func(*Flag)) { //注：遍历CommandLine的排序后的f.formal，将其作为参数调用fn
	CommandLine.VisitAll(fn)
}

// Visit 按字典顺序访问标志，分别对fn调用。
// 它仅访问已设置的那些标志。
func (f *FlagSet) Visit(fn func(*Flag)) { //注：遍历排序后的实参集合，将其作为参数调用fn
	for _, flag := range sortFlags(f.actual) {
		fn(flag)
	}
}

// Visit 按字典顺序访问命令行标志，为每个标志调用fn。 它仅访问已设置的那些标志。
func Visit(fn func(*Flag)) { //注：遍历CommandLine的排序后的f.actual，将其作为参数调用fn
	CommandLine.Visit(fn)
}

// Lookup 返回命名标志的Flag结构，如果不存在则返回nil。
func (f *FlagSet) Lookup(name string) *Flag { //注：返回形参集合中key为name的标志
	return f.formal[name]
}

// Lookup 返回命名命令行标志的Flag结构，如果不存在则返回nil。
func Lookup(name string) *Flag { //注：返回CommandLine的formal中key位name的Flag
	return CommandLine.formal[name]
}

// Set 设置命名标志的值。
func (f *FlagSet) Set(name, value string) error { //注：将f中key为name的形参赋值为value，拷贝给实参
	flag, ok := f.formal[name]
	if !ok {
		return fmt.Errorf("no such flag -%v", name) //注：没有这个标志
	}
	err := flag.Value.Set(value) //注：f.formal[name]的Value赋值为value
	if err != nil {
		return err
	}
	if f.actual == nil {
		f.actual = make(map[string]*Flag)
	}
	f.actual[name] = flag //注：将f.formal[name]赋值给f.actual[name]
	return nil
}

// Set 设置命名命令行标志的值。
func Set(name, value string) error { //注：将CommandLineshe中key为name的形参赋值为value，拷贝给实参
	return CommandLine.Set(name, value)
}

// isZeroValue 确定字符串是否代表标志的零值。
func isZeroValue(flag *Flag, value string) bool { //注：#
	// 构建标志的Value类型的零值，并查看调用其String方法的结果是否等于传入的值。
	// 除非Value类型本身是接口类型，否则此方法有效。
	typ := reflect.TypeOf(flag.Value) //注：获取参数值的类型
	var z reflect.Value
	if typ.Kind() == reflect.Ptr { //注：
		z = reflect.New(typ.Elem())
	} else {
		z = reflect.Zero(typ)
	}
	return value == z.Interface().(Value).String()
}

// UnquoteUsage 从usage字符串中提取一个带反引号的名称作为标志，并返回它和未引用的usage。
// 给定"要显示的name"，它会返回("名称", "要显示的名称")。
// 如果没有反引号，则该名称是对标记值类型的有根据的猜测，如果标记为布尔值，则为空字符串。
func UnquoteUsage(flag *Flag) (name string, usage string) { //注：获取flag的标志name与帮助信息usage
	// 寻找一个用引号引起来的名称，但不要使用字符串包。
	usage = flag.Usage
	for i := 0; i < len(usage); i++ {
		if usage[i] == '`' {
			for j := i + 1; j < len(usage); j++ {
				if usage[j] == '`' {
					name = usage[i+1 : j]                  //注：帮助信息中``包含的内容
					usage = usage[:i] + name + usage[j+1:] //注：去掉``字符的u帮助信息
					return name, usage
				}
			}
			break // 只有一个反引号； 使用类型名称。
		}
	}
	// 没有明确的名称，因此如果可以找到，请使用type。
	name = "value"
	switch flag.Value.(type) { //注：遍历类型，赋值name
	case boolFlag:
		name = ""
	case *durationValue:
		name = "duration"
	case *float64Value:
		name = "float"
	case *intValue, *int64Value:
		name = "int"
	case *stringValue:
		name = "string"
	case *uintValue, *uint64Value:
		name = "uint"
	}
	return
}

// PrintDefaults 除非另有配置，否则将标准定义错误打印到集合中所有定义的命令行标志的默认值。
// 有关更多信息，请参见全局函数PrintDefaults的文档。
// 例1：fs.String("A", "1", "`int`参数A")
//   -A int
//    	int参数A (default "1")
// 例2：fs.String("B", "0", "参数B")
//   -B string
//    	参数B (default "0")
func (f *FlagSet) PrintDefaults() { //注：打印标志集合f的用法
	f.VisitAll(func(flag *Flag) { //注：遍历f的形参
		s := fmt.Sprintf("  -%s", flag.Name) // -的前两个空格； 请参阅接下来的两个评论。注：打印'  -' + 标志
		name, usage := UnquoteUsage(flag)    //注：获取标志与帮助信息（帮助信息应该是空的）
		if len(name) > 0 {
			s += " " + name //注：' ' + 标志
		}
		// 一个ASCII字母的布尔标志非常普遍，我们特别对待它们，将它们的用法放在同一行上。
		if len(s) <= 4 { // space, space, '-', 'x'.
			s += "\t"
		} else {
			// 制表符触发4和8个制表符停止位之前的四个空格。
			s += "\n    \t"
		}
		s += strings.ReplaceAll(usage, "\n", "\n    \t") //注：字符串\n替换为"\n    \t"

		if !isZeroValue(flag, flag.DefValue) {
			if _, ok := flag.Value.(*stringValue); ok { //注：如果参数是string类型，使用%q格式化（go语法输出字符串）
				// 在值上加上引号
				s += fmt.Sprintf(" (default %q)", flag.DefValue)
			} else {
				s += fmt.Sprintf(" (default %v)", flag.DefValue) //注：否则使用%v格式化（默认类型）
			}
		}
		fmt.Fprint(f.Output(), s, "\n") //注：将s写入os.Stderr
	})
}

// PrintDefaults 除非另行配置，否则PrintDefaults会显示一条使用情况消息，
// 显示所有已定义的命令行标志的默认设置，除非出现其他配置，否则将显示标准错误。
// 对于整数标志x，默认输出的形式为
//	-x int
//		usage-message-for-x (default 7)
// 用法消息将显示在单独的行上，除了带有一字节名称的bool标志外，其他任何内容都不会显示。
// 对于布尔标志，将省略类型，如果标志名称为一个字节，则用法消息将显示在同一行上。
// 如果类型的默认值为零，则省略括号的默认值。
// 可以通过在标志的用法字符串中加上反引号来更改列出的类型（此处为int）。
// 消息中的第一个此类项目被视为要在消息中显示的参数名称，并且在显示时从消息中删除反引号。
// 例如，给定
//	flag.String("I", "", "search `directory` for include files")
// 将会输出
//	-I directory
//		search directory for include files.
//
// 若要更改标志消息的目标，请调用CommandLine.SetOutput。
func PrintDefaults() { //注：输出CommandLine的形参的介绍
	CommandLine.PrintDefaults()
}

// defaultUsage 是打印使用情况消息的默认功能。
func (f *FlagSet) defaultUsage() { //注：打印标志集合f的用法（显示标题）
	if f.name == "" {
		fmt.Fprintf(f.Output(), "Usage:\n") //注：打印Usage:\n
	} else {
		fmt.Fprintf(f.Output(), "Usage of %s:\n", f.name) //注：打印Usage of + f.name:\n
	}
	f.PrintDefaults()
}

// 注意：用法不仅是defaultUsage(CommandLine)，因为它（通过godoc标志Usage）用作如何编写自己的用法函数的示例。

// Usage 将用法消息打印出来，记录所有已定义的命令行标志到CommandLine的输出，默认情况下为os.Stderr。
// 解析标志时发生错误时调用。
// 该函数是一个变量，可以更改为指向自定义函数。
// 默认情况下，它会打印一个简单的标题并调用PrintDefaults; 有关输出格式及其控制方法的详细信息，请参见PrintDefaults的文档。
// 自定义使用功能可以选择退出程序； 默认情况下，无论如何退出都会发生，因为命令行的错误处理策略设置为ExitOnError。
var Usage = func() { //注：输出CommandLine的形参介绍
	fmt.Fprintf(CommandLine.Output(), "Usage of %s:\n", os.Args[0])
	PrintDefaults()
}

// NFlag 返回已设置的标志数。
func (f *FlagSet) NFlag() int { return len(f.actual) } //注：返回f的实参数量

// NFlag 返回已设置的命令行标志的数量。
func NFlag() int { return len(CommandLine.actual) } //注：返回CommandLine的实参数量

// Arg 返回第i个参数。 Arg(0)是标志已处理后的第一个剩余参数。
// 如果请求的元素不存在，则Arg返回一个空字符串。
func (f *FlagSet) Arg(i int) string { //注：返回f.args[i]
	if i < 0 || i >= len(f.args) {
		return ""
	}
	return f.args[i]
}

// Arg 返回第i个命令行参数。Arg(0)是标志已处理后的第一个剩余参数。
// 如果请求的元素不存在，则Arg返回一个空字符串。
func Arg(i int) string { //注：返回CommandLine.Arg(i)
	return CommandLine.Arg(i)
}

// NArg 是在处理标志后剩余的参数个数。
func (f *FlagSet) NArg() int { return len(f.args) } //注：返回参数的数量

// NArg 是在处理标志后剩余的参数个数。
func NArg() int { return len(CommandLine.args) } //注：返回CommandLine参数的数量

// Args 返回非标志参数。
func (f *FlagSet) Args() []string { return f.args } //注：获取f.args

// Args 返回非标志命令行参数。
func Args() []string { return CommandLine.args } //注：获取CommandLine.args

// BoolVar 用指定的名称，默认值和用法字符串定义一个bool标志。
// 参数p指向一个bool变量，用于存储该标志的值。
func (f *FlagSet) BoolVar(p *bool, name string, value bool, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至f中
	f.Var(newBoolValue(value, p), name, usage)
}

// BoolVar 用指定的名称，默认值和用法字符串定义一个bool标志。
// 参数p指向一个bool变量，用于存储该标志的值。
func BoolVar(p *bool, name string, value bool, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至CommandLine中
	CommandLine.Var(newBoolValue(value, p), name, usage)
}

// Bool 用指定的名称，默认值和用法字符串定义一个bool标志。
// 返回值是存储标记值的bool变量的地址。
// 调用链：Bool（创建一个返回值）——BoolVar（为FlagSet创建一个形参）——newBoolValue（返回BoolValue类型的value）——Var（为FlagSet创建一个形参）
func (f *FlagSet) Bool(name string, value bool, usage string) *bool { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至f中
	p := new(bool)
	f.BoolVar(p, name, value, usage)
	return p
}

// Bool 用指定的名称，默认值和用法字符串定义一个bool标志。
// 返回值是存储标记值的bool变量的地址。
func Bool(name string, value bool, usage string) *bool { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至CommandLine中
	return CommandLine.Bool(name, value, usage)
}

// IntVar 用指定的名称，默认值和用法字符串定义一个int标志。
// 参数p指向一个int变量，用于存储该标志的值。
func (f *FlagSet) IntVar(p *int, name string, value int, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至f中
	f.Var(newIntValue(value, p), name, usage)
}

// IntVar 用指定的名称，默认值和用法字符串定义一个int标志。
// 参数p指向一个int变量，用于存储该标志的值。
func IntVar(p *int, name string, value int, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至CommandLine中
	CommandLine.Var(newIntValue(value, p), name, usage)
}

// Int 用指定的名称，默认值和用法字符串定义一个int标志。
// 返回值是存储该标志值的int变量的地址。
func (f *FlagSet) Int(name string, value int, usage string) *int { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至f中
	p := new(int)
	f.IntVar(p, name, value, usage)
	return p
}

// Int 用指定的名称，默认值和用法字符串定义一个int标志。
// 返回值是存储该标志值的int变量的地址。
func Int(name string, value int, usage string) *int { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至CommandLine中
	return CommandLine.Int(name, value, usage)
}

// Int64Var 定义一个具有指定名称，默认值和用法字符串的int64标志。
// 参数p指向一个int64变量，该变量用于存储标志的值。
func (f *FlagSet) Int64Var(p *int64, name string, value int64, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至f中
	f.Var(newInt64Value(value, p), name, usage)
}

// Int64Var 定义一个具有指定名称，默认值和用法字符串的int64标志。
// 参数p指向一个int64变量，该变量用于存储标志的值。
func Int64Var(p *int64, name string, value int64, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至CommandLine中
	CommandLine.Var(newInt64Value(value, p), name, usage)
}

// Int64定义一个具有指定名称，默认值和用法字符串的int64标志。
// 返回值是存储该标志值的int64变量的地址。
func (f *FlagSet) Int64(name string, value int64, usage string) *int64 { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至f中
	p := new(int64)
	f.Int64Var(p, name, value, usage)
	return p
}

// Int64 定义一个具有指定名称，默认值和用法字符串的int64标志。
// 返回值是存储该标志值的int64变量的地址。
func Int64(name string, value int64, usage string) *int64 { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至CommandLine中
	return CommandLine.Int64(name, value, usage)
}

// UintVar 定义具有指定名称，默认值和用法字符串的uint标志。
// 参数p指向用于存储标志值的uint变量。
func (f *FlagSet) UintVar(p *uint, name string, value uint, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至f中
	f.Var(newUintValue(value, p), name, usage)
}

// UintVar 定义具有指定名称，默认值和用法字符串的uint标志。
// 参数p指向用于存储标志值的uint变量。
func UintVar(p *uint, name string, value uint, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至CommandLine中
	CommandLine.Var(newUintValue(value, p), name, usage)
}

// Uint 定义具有指定名称，默认值和用法字符串的uint标志。
// 返回值是存储标志值的uint变量的地址。
func (f *FlagSet) Uint(name string, value uint, usage string) *uint { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至f中
	p := new(uint)
	f.UintVar(p, name, value, usage)
	return p
}

// Uint 定义具有指定名称，默认值和用法字符串的uint标志。
// 返回值是存储标志值的uint变量的地址。
func Uint(name string, value uint, usage string) *uint { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至CommandLine中
	return CommandLine.Uint(name, value, usage)
}

// Uint64Var 定义具有指定名称，默认值和用法字符串的uint64标志。
// 参数p指向uint64变量，该变量用于存储标志的值.
func (f *FlagSet) Uint64Var(p *uint64, name string, value uint64, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至f中
	f.Var(newUint64Value(value, p), name, usage)
}

// Uint64Var 定义具有指定名称，默认值和用法字符串的uint64标志。
// 参数p指向uint64变量，该变量用于存储标志的值
func Uint64Var(p *uint64, name string, value uint64, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至CommandLine中
	CommandLine.Var(newUint64Value(value, p), name, usage)
}

// Uint64 定义具有指定名称，默认值和用法字符串的uint64标志。
// 返回值是存储标志值的uint64变量的地址。
func (f *FlagSet) Uint64(name string, value uint64, usage string) *uint64 { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至f中
	p := new(uint64)
	f.Uint64Var(p, name, value, usage)
	return p
}

// Uint64 定义具有指定名称，默认值和用法字符串的uint64标志。
// 返回值是存储标志值的uint64变量的地址。
func Uint64(name string, value uint64, usage string) *uint64 { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至CommandLine中
	return CommandLine.Uint64(name, value, usage)
}

// StringVar 定义具有指定名称，默认值和用法字符串的字符串标志。
// 参数p指向一个字符串变量，用于在其中存储标志的值。
func (f *FlagSet) StringVar(p *string, name string, value string, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至f中
	f.Var(newStringValue(value, p), name, usage)
}

// StringVar 定义具有指定名称，默认值和用法字符串的字符串标志。
// 参数p指向一个字符串变量，用于在其中存储标志的值。
func StringVar(p *string, name string, value string, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至CommandLine中
	CommandLine.Var(newStringValue(value, p), name, usage)
}

// String 定义具有指定名称，默认值和用法字符串的字符串标志。
// 返回值是存储标志值的字符串变量的地址。
func (f *FlagSet) String(name string, value string, usage string) *string { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至f中
	p := new(string)
	f.StringVar(p, name, value, usage)
	return p
}

// String 定义具有指定名称，默认值和用法字符串的字符串标志。
// 返回值是存储标志值的字符串变量的地址。
func String(name string, value string, usage string) *string { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至CommandLine中
	return CommandLine.String(name, value, usage)
}

// Float64Var 用指定的名称，默认值和用法字符串定义一个float64标志。
// 参数p指向一个float64变量，用于存储该标志的值。
func (f *FlagSet) Float64Var(p *float64, name string, value float64, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至f中
	f.Var(newFloat64Value(value, p), name, usage)
}

// Float64Var 用指定的名称，默认值和用法字符串定义一个float64标志。
// 参数p指向一个float64变量，用于存储该标志的值。
func Float64Var(p *float64, name string, value float64, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至CommandLine中
	CommandLine.Var(newFloat64Value(value, p), name, usage)
}

// Float64 用指定的名称，默认值和用法字符串定义一个float64标志。
// 返回值是一个float64变量的地址，该变量存储标志的值。
func (f *FlagSet) Float64(name string, value float64, usage string) *float64 { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至f中
	p := new(float64)
	f.Float64Var(p, name, value, usage)
	return p
}

// Float64 用指定的名称，默认值和用法字符串定义一个float64标志。
// 返回值是一个float64变量的地址，该变量存储标志的值。
func Float64(name string, value float64, usage string) *float64 { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至CommandLine中
	return CommandLine.Float64(name, value, usage)
}

// DurationVar 使用指定的名称，默认值和用法字符串定义一个time.Duration标志。
// 参数p指向time.Duration变量，用于存储标志的值。
// 该标志接受time.ParseDuration可接受的值。
func (f *FlagSet) DurationVar(p *time.Duration, name string, value time.Duration, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至f中
	f.Var(newDurationValue(value, p), name, usage)
}

// DurationVar 使用指定的名称，默认值和用法字符串定义一个time.Duration标志。
// 参数p指向time.Duration变量，用于存储标志的值。
// 该标志接受time.ParseDuration可接受的值。
func DurationVar(p *time.Duration, name string, value time.Duration, usage string) { //注：将默认值p、名称name、用法usage作为形参添加至CommandLine中
	CommandLine.Var(newDurationValue(value, p), name, usage)
}

// Duration 使用指定的名称，默认值和用法字符串定义一个time.Duration标志。
// 返回值是time.Duration变量的地址，该变量存储标志的值。
// 该标志接受time.ParseDuration可接受的值。
func (f *FlagSet) Duration(name string, value time.Duration, usage string) *time.Duration { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至f中
	p := new(time.Duration)
	f.DurationVar(p, name, value, usage)
	return p
}

// Duration 使用指定的名称，默认值和用法字符串定义一个time.Duration标志。
// 返回值是time.Duration变量的地址，该变量存储标志的值。
// 该标志接受time.ParseDuration可接受的值。
func Duration(name string, value time.Duration, usage string) *time.Duration { //注：将val赋值给p，将默认值p、名称name、用法usage作为形参添加至CommandLine中
	return CommandLine.Duration(name, value, usage)
}

// Var 用指定的名称和用法字符串定义一个标志。
// 标志的类型和值由类型为Value的第一个参数表示，该参数通常保存用户定义的Value实现。
// 例如，调用者可以创建一个标志，该标志可以通过给分片使用Value方法来将逗号分隔的字符串转换为分片。
// 特别是，Set会将逗号分隔的字符串分解为切片。
func (f *FlagSet) Var(value Value, name string, usage string) { //注：为f添加一个形参，参数名称为name，参数值与默认值位value，参数说明为usage
	// 记住默认值是一个字符串； 它不会改变。
	flag := &Flag{name, usage, value, value.String()} //注：创建一个flag结构体
	_, alreadythere := f.formal[name]
	if alreadythere { //注：如果已经存在名称相同的形参
		var msg string
		if f.name == "" {
			msg = fmt.Sprintf("flag redefined: %s", name) //注：flag redefined: 形参名称
		} else {
			msg = fmt.Sprintf("%s flag redefined: %s", f.name, name) //注：标志集合名称 flag redefined: 形参名称
		}
		fmt.Fprintln(f.Output(), msg)
		panic(msg) // 仅当标志声明为相同名称时发生
	}
	if f.formal == nil { //注：赋值形参给实参
		f.formal = make(map[string]*Flag)
	}
	f.formal[name] = flag
}

// Var 用指定的名称和用法字符串定义一个标志。
// 标志的类型和值由类型为Value的第一个参数表示，该参数通常保存用户定义的Value实现。
// 例如，调用者可以创建一个标志，该标志可以通过给分片使用Value方法来将逗号分隔的字符串转换为分片。
// 特别是，Set会将逗号分隔的字符串分解为切片。
func Var(value Value, name string, usage string) { //注：为CommandLine添加一个形参，参数名称为name，参数值与默认值位value，参数说明为usage
	CommandLine.Var(value, name, usage)
}

// failf 将格式化的错误和用法消息打印到标准错误，并返回错误。
func (f *FlagSet) failf(format string, a ...interface{}) error { //注：输出错误与f的用法
	err := fmt.Errorf(format, a...) //注：根据format格式化a为error
	fmt.Fprintln(f.Output(), err)   //注：将error写入os.Stderr
	f.usage()                       //注：打印f的用法
	return err
}

// usage 如果指定了标志，则为标志集调用Usage方法，否则，调用适当的默认用法函数。
func (f *FlagSet) usage() { //注：打印标志集合f的用法
	if f.Usage == nil {
		f.defaultUsage()
	} else {
		f.Usage()
	}
}

// parseOne 解析一个标志。 它报告是否看到一个标志。
// 注：解析后的arg会被丢弃
func (f *FlagSet) parseOne() (bool, error) { //注：将最多2个f.args元素解析赋值给f.formal与f.actual（赋值给f.formal[x].Value，再将f.formal[x]拷贝一份赋值给f.actual[x]）
	// 例1：f.args[0] = "-a"，f.args[1] = "123"
	// 例2：f.args[0] = "--c=123"
	if len(f.args) == 0 { //注：f.args不能为空
		return false, nil
	}
	s := f.args[0]
	if len(s) < 2 || s[0] != '-' { //注：第1个字符必须为-
		return false, nil
	}

	// 例1：numMinuses = 1
	// 例2：numMinuses = 2
	numMinuses := 1
	if s[1] == '-' { //注：如果第2个字符也是-
		numMinuses++
		if len(s) == 2 { // "--" 终止标志，注：f.arg == "--"，去掉这个arg
			f.args = f.args[1:]
			return false, nil
		}
	}

	// 例1：name = a
	// 例2：name = c=123
	name := s[numMinuses:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		return false, f.failf("bad flag syntax: %s", s) //注：错误标志语法，例：--、---、-=等
	}

	// 例1：value = ""，name = "a"，hasValue = false
	// 例3：value = "123"，name = "c"，hasValue = true
	// 这是一个标志。 有参数吗？
	f.args = f.args[1:] //注：丢弃正在解析的arg
	hasValue := false
	value := ""
	for i := 1; i < len(name); i++ {
		if name[i] == '=' { //注：如果参数含有=
			value = name[i+1:]
			hasValue = true
			name = name[0:i]
			break
		}
	}
	m := f.formal
	flag, alreadythere := m[name] // BUG，注：检查name是否存在，如果参数为-help或-h，返回用法，例：xxx -help
	if !alreadythere {
		if name == "help" || name == "h" { // 很好的帮助消息的特殊情况。
			f.usage()
			return false, ErrHelp
		}
		return false, f.failf("flag provided but not defined: -%s", name) //注：错误"提供但未定义的标志"
	}

	if fv, ok := flag.Value.(boolFlag); ok && fv.IsBoolFlag() { // 特殊情况：不需要arg，注：如果标志要求布尔类型
		if hasValue { //注：如果参数后面后有值
			if err := fv.Set(value); err != nil { //注：赋值失败
				return false, f.failf("invalid boolean value %q for -%s: %v", value, name, err) //注：错误"name的无效布尔值value：err"
			}
		} else { //注：参数后面没有值，默认为true
			if err := fv.Set("true"); err != nil { //注：赋值失败
				return false, f.failf("invalid boolean flag %s: %v", name, err) //注：错误"无效的布尔值标志name：err"
			}
		}
	} else {
		// 它必须具有一个值，该值可能是下一个参数。
		if !hasValue && len(f.args) > 0 {
			// 值是下一个参数
			hasValue = true
			value, f.args = f.args[0], f.args[1:]
		}
		if !hasValue { //注：要求连个参数，但没有参数了
			return false, f.failf("flag needs an argument: -%s", name) //注：错误"标志需要一个参数"
		}
		if err := flag.Value.Set(value); err != nil { //注：赋值发生错误
			return false, f.failf("invalid value %q for flag -%s: %v", value, name, err) //注：错误"标志的值％q无效"
		}
	}
	if f.actual == nil { //注：将形参赋值给实参
		f.actual = make(map[string]*Flag)
	}
	f.actual[name] = flag
	return true, nil
}

// Parse 解析参数列表中的标志定义，其中不包括命令名称。
// 必须在定义了FlagSet中的所有标志之后并且在程序访问标志之前必须调用它。
// 如果设置了-help或-h，但未定义，则返回值为ErrHelp。
func (f *FlagSet) Parse(arguments []string) error { //注：将arguments解析赋值给f.formal与f.actual
	f.parsed = true
	f.args = arguments
	for {
		seen, err := f.parseOne() //注：将最多2个f.args元素解析赋值给f.formal与f.actual
		if seen {                 //注：如果赋值成功则继续
			continue
		}
		if err == nil { //注：
			break
		}
		switch f.errorHandling { //注：发生错误时的行为
		case ContinueOnError:
			return err
		case ExitOnError:
			os.Exit(2)
		case PanicOnError:
			panic(err)
		}
	}
	return nil
}

// Parsed 报告是否已调用f.Parse。
func (f *FlagSet) Parsed() bool { //注：返回f.parsed，f是否执行过Parse
	return f.parsed
}

// Parse 从os.Args[1:]解析命令行标志。 必须在定义所有标志之后并且在程序访问标志之前调用。
func Parse() { //注：将os.Args[1:]解析赋值给CommandLine.formal与f.actual
	// 忽略错误； 命令行设置为ExitOnError。
	CommandLine.Parse(os.Args[1:])
}

// Parsed 报告是否已解析命令行标志。
func Parsed() bool { //注：返回CommandLine.parsed，CommandLine是否执行过Parse
	return CommandLine.Parsed()
}

// CommandLine 是从os.Args解析的默认命令行标志集。
// 顶级功能（例如BoolVar，Arg等）是CommandLine方法的包装。
var CommandLine = NewFlagSet(os.Args[0], ExitOnError) //注：生成一个本地FlagSet标志集合

func init() { //注：初始化CommandLine的用法
	// 通过调用全局用法覆盖通用FlagSet默认用法。
	// 注意：这不是CommandLine.Usage = Usage，因为我们希望任何最终调用都可以使用Usage的任何更新值，而不是运行此行时的值。
	CommandLine.Usage = commandLineUsage
}

func commandLineUsage() { //注：CommandLine的用法
	Usage()
}

// NewFlagSet 返回带有指定名称和错误处理属性的新的空标志集。 如果名称不为空，则会在默认用法消息和错误消息中打印该名称。
func NewFlagSet(name string, errorHandling ErrorHandling) *FlagSet { //工厂函数，将name与errorHandling作为FlagSet的成员并返回
	f := &FlagSet{
		name:          name,
		errorHandling: errorHandling,
	}
	f.Usage = f.defaultUsage
	return f
}

// Init 设置标志集的名称和错误处理属性。
// 默认情况下，零FlagSet使用一个空名称和ContinueOnError错误处理策略。
func (f *FlagSet) Init(name string, errorHandling ErrorHandling) { //注：初始化f
	f.name = name
	f.errorHandling = errorHandling
}
