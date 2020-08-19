/*
flag包用于解析命令行标志
命令解析格式：
	"xxx -a 123"
	"xxx -a=123"
	"xxx --a 123"
	"xxx --a=123"

支持的参数类型：
	bool
	int
	int64
	uint
	uint64
	string
	float64
	time.Duration

数据结构：
	type FlagSet struct {
		Usage func()
		name          string           //注：应用的绝对路径
		parsed        bool             //注：f是否执行过Parse转换参数
		actual        map[string]*Flag //注：实参列表
		formal        map[string]*Flag //注：形参列表
		args          []string         // 标志后的参数
		errorHandling ErrorHandling    //注：发生错误时的行为
		output        io.Writer        // nil表示stderr； 使用out()访问器
	}
	type Flag struct {
		Name     string // 在命令行上显示的名称
		Usage    string // 帮助信息
		Value    Value  // 设定值
		DefValue string // 默认值（以文本形式）； 使用信息，注：Value的字符串表现形式，用于提示
	}
	FlagSet：标志集合
		formal：形参集合（接受什么标志与对应的参数类型）
		actual：实参集合（命令行参数中提供了哪些标志与参数）
		args：参数集合（输入的字符串根据，空格分隔）
	Flag：标志
		Name：标志
		Usage：帮助信息，用法
		Value：参数，默认值
		DefValue：参数的字符串，为输出值类型的提示

名词：（之后用名词表示）
	xxx：命令
	-a：标志
	123：参数

当前应用程序的标志集合：
	CommandLine：args为os.Args

函数与方法列表：
	1. xxxValue：参数接受的8种数据类型（用xxx代替）
		1) newxxxValue(val xxx, p *xxx) *xxxValue		工厂函数，将val赋值给p，返回boolValue类型的p
		2) (b *xxxValue) Set(s string) error			获取xxx类型的b
		3) (b *xxxValue) String() string				将b格式化为字符串并返回
		4) func (b *boolValue) IsBoolFlag() bool		只有boolValue类型才有的方法，返回true
	2. FlagSet：标志集合的方法
		1) (f *FlagSet) Output() io.Writer				获取f.output，如果为nil返回os.Stderr
		2) (f *FlagSet) SetOutput(output io.Writer)		将f.output设置为output

		3) (f *FlagSet) Name() string 					获取f.name
		4) (f *FlagSet) ErrorHandling() ErrorHandling	获取f.errorHandling，解析发生错误时的行为

		5) (f *FlagSet) Lookup(name string) *Flag		返回形参集合中key为name的标志
		6) (f *FlagSet) Set(name, value string) error	将f中key为name的形参赋值为value，拷贝给实参

		7) (f *FlagSet) Parse(arguments []string) error	将arguments解析赋值给f.formal与f.actual
		8) (f *FlagSet) parseOne() (bool, error)		将最多2个f.args元素解析赋值给f.formal与f.actual
		9) (f *FlagSet) Parsed() bool					是否执行过Parse
		10) (f *FlagSet) Arg(i int) string				返回第i个参数的值
		11) (f *FlagSet) NArg() int						返回参数的数量
		12) (f *FlagSet) Args() []string				返回所有参数

		13) (f *FlagSet) VisitAll(fn func(*Flag))		遍历排序后的形参集合，将其作为参数调用fn
		14) (f *FlagSet) Visit(fn func(*Flag))			遍历排序后的实参集合，将其作为参数调用fn

		15) (f *FlagSet) usage()  						打印标志集合f的用法
		16) (f *FlagSet) defaultUsage()					打印标志集合f的用法（显示标题）
		17) (f *FlagSet) PrintDefaults()				打印标志集合f的用法

		18) (f *FlagSet) NFlag() int					返回f的实参数量
		19) NewFlagSet(name string, errorHandling ErrorHandling) *FlagSet		工厂函数
		20) (f *FlagSet) failf(format string, a ...interface{}) error 			输出错误与f的用法

	3. Flag：标志的方法
		1) sortFlags(flags map[string]*Flag) []*Flag							将flags转为[]*Flag格式，根据名称排序并返回
		2) isZeroValue(flag *Flag, value string) bool							#
		3) UnquoteUsage(flag *Flag) (name string, usage string)					获取flag的标志name与用法usage
	4. CommandLine的方法
		同2
	5. XXXVar：参数接受的8种数据类型（用XXX代替）
		1) (f *FlagSet) XXX(name string, value XXX, usage string) *XXX			将val赋值给p，将默认值p、名称name、用法usage作为形参添加至f中
		2) (f *FlagSet) XXXVar(p *XXX, name string, value XXX, usage string)	将默认值p、名称name、用法usage作为形参添加至f中
		3) (f *FlagSet) Var(value Value, name string, usage string)				为f添加一个形参，参数名称为name，参数值与默认值位value，参数说明为usage
*/