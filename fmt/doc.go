// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

/*
	Package fmt 使用与C的printf和scanf类似的功能实现格式化的I/O。 格式"verbs"是从C派生的，但更简单。
	Printing

	The verbs:
	常规:
		%v	在打印结构时使用默认格式的值，加号（%+v）添加字段名称
		%#v	该值的Go语法表示形式
		%T	值类型的Go语法表示形式
		%%	文字百分号； 不消耗任何价值
	布尔类型:
		%t	单词true或false

		注：
		%t：true或false
	整数:
		%b	二进制
		%c	相应的Unicode代码点表示的字符
		%d	十进制
		%o	八进制
		%O	带有0o前缀的八进制
		%q	使用Go语法安全地转义的单引号字符文字
		%x	使用小写字母表示的十六进制
		%X	使用大写字母表示的十六进制
		%U	Unicode格式: U+1234; 与"U+%04X"相同
	浮点和复杂成分：
		%b	指数为2的无小数科学计数法，采用strconv.FormatFloat的形式为'b'格式，例如 -123456p-78
		%e	科学记数法, e.g. -1.234456e+78
		%E	科学记数法, e.g. -1.234456E+78
		%f	小数点但无指数, e.g. 123.456
		%F	%f的同义词
		%g	%e适用于大指数，否则为%f。 精度将在下面讨论。
		%G	%E用于大指数，否则为%F
		%x	十六进制表示法（具有两个指数的十进制幂）, e.g. -0x1.23abcp+20
		%X	大写十六进制符号， e.g. -0X1.23ABCP+20
	字符串与字节切片（与这些verbs等效地对待）
		%s	字符串或片的未解释字节
		%q	使用Go语法安全地转义的双引号字符串
		%x	十六进制，小写，每字节两个字符
		%X	十六进制，大写，每字节两个字符
	切片:
		%p	以十六进制表示的第0个元素的地址，开头为0x
	指针:
		%p	以十六进制表示，以0x开头。%b，%d，%o，%x和%X verb也可以与指针一起使用，将值格式化为整数。

	%v的默认格式是:
		bool:                    %t
		int, int8 etc.:          %d
		uint, uint8 etc.:        %d, 如果用%#v打印，则为%#x
		float32, complex64, etc: %g
		string:                  %s
		chan:                    %p
		pointer:                 %p
	对于复合对象，将使用以下规则递归打印元素，其布局如下：
		struct:             {field0 field1 ...}
		array, slice:       [elem0 elem1 ...]
		maps:               map[key1:value1 key2:value2 ...]
		pointer to above:   &{}, &[], &map[]

	宽度由动词前的可选十进制数字指定。
	如果不存在，则宽度是表示值所必需的。
	精度在（可选）宽度之后指定一个句点，后跟一个十进制数字。 如果不存在句点，则使用默认精度。
	没有以下数字的句点指定精度为零。
	案例:
		%f     默认宽度，默认精度
		%9f    宽度9，默认精度
		%.2f   默认宽度，精度2
		%9.2f  宽度9，精度2
		%9.f   宽度9，精度0

	宽度和精度以Unicode代码点（即rune）为单位进行度量。
	（这与C的printf不同，后者的单位始终以字节为单位。）两个标志中的一个或两个都可以用字符'*'替换，
	从而使它们的值从下一个操作数获得（在格式化之前）。 必须是int类型。

	对于大多数值，width是要输出的最小符文数，必要时用空格填充格式化的表单。

	但是，对于字符串，字节片和字节数组，精度会限制要格式化的输入的长度（而不是输出的大小），并在必要时将其截断。
	通常，它以verb为单位进行度量，但是对于这些类型，使用%x或%X格式进行格式化时，将以字节为单位进行度量。

	对于浮点值，width设置字段的最小宽度，而precision设置小数点后的位数（如果适用），除了%g/%G precision设置最大有效位数（删除零位）。
	例如，给定12.345，格式%6.3f可打印12.345，而%.3g可打印12.3。 %e，%f和%#g的默认精度为6； 对于%g，它是唯一标识该值所需的最少位数。

	对于复数，宽度和精度分别应用于这两个分量，并在结果中加上括号，因此将%f应用于1.2 + 3.4i生成（1.200000 + 3.400000i）。

	其他标志:
		+	始终为数字值打印符号；保证%q(%+q)的纯ASCII输出
		-	在右侧而不是左侧填充空格（左对齐字段）
		#	备用格式：将前导0b表示为二进制（%#b），0表示八进制（%#o），0x或0X表示十六进制（%#x或%#X）； 为#p（%#p）抑制0x;
			对于%q，如果strconv.CanBackquote返回true，则输出原始（加反引号）字符串；
			始终为%e，%E，%f，%F，%g和%G打印小数点；
			不要删除%g和%G的尾随零；
			写例如 U+0078'x'（如果字符可用于%U（%#U）打印）。
		' '	（空格）在数字（%d）处留一个省略号；
			在打印字符串或切片的字节之间放置十六进制空格（%x，%X）
		0	用前导零而不是空格填充；对于数字，这会在符号后移动填充

	标志被不期望使用它们的verb忽略。
	例如，没有替代的十进制格式，因此%#d和%d的行为相同。

	对于每个类似于Printf的函数，还有一个Print函数，该函数不采用任何格式，等效于为每个操作数说%v。
	Println的另一个变体在操作数之间插入空格，并添加换行符。

	无论verb如何，如果操作数是接口值，那么将使用内部具体值，而不是接口本身。
	从而:
		var i interface{} = 23
		fmt.Printf("%v\n", i)
	会输出 23.

	除非使用%T和%p进行打印，否则特殊的格式注意事项适用于实现某些接口的操作数。 按申请顺序：

	1. 如果操作数是reflect.Value，则将操作数替换为其所持有的具体值，然后使用下一个规则继续打印。

	2. 如果操作数实现Formatter接口，则将调用它。 Formatter提供了对格式的精细控制。

	3. 如果%v与#标志（%#v）一起使用，并且操作数实现GoStringer接口，则将调用该接口。

	如果格式（对于Println等，隐式为%v）对字符串（%s %q %v %x %X）有效，则以下两个规则适用：

	4. 如果操作数实现error接口，则将调用Error方法将对象转换为字符串，然后根据verb的要求对其进行格式化（如果有）。

	5. 如果操作数实现方法String() string，则将调用该方法将对象转换为字符串，然后将根据verb的要求对其进行格式化（如果有）。

	对于切片和结构之类的复合操作数，格式递归地应用于每个操作数的元素，而不是整个操作数。
	因此%q将引用字符串切片的每个元素，而%6.2f将控制浮点数组的每个元素的格式。

	但是，在打印带有字符串状动词（%s %q %x %X）的字节片时，将其与字符串等同地视为一个项。

	在诸如以下情况下避免递归
		type X string
		func (x X) String() string { return Sprintf("<%s>", x) }
	在重复之前转换值：
		func (x X) String() string { return Sprintf("<%s>", string(x)) }
	无限递归也可以由自引用数据结构触发，例如包含自身作为元素的切片（如果该类型具有String方法）。
	但是，这种病理情况很少见，而且该软件包也无法防止它们。

	打印结构体时，fmt无法并且因此不会在未导出的字段上调用诸如Error或String之类的格式化方法。
	显式参数索引:
	在Printf，Sprintf和Fprintf中，默认行为是为每个格式化动词格式化在调用中传递的连续参数。
	但是，verb前的符号[n]表示将改为格式化第n个单索引参数。 宽度或精度的'*'之前的相同符号选择保存该值的参数索引。
	在处理了带括号的表达式[n]后，除非另外指出，否则后续动词将使用自变量n + 1，n + 2等。

	例如，
		fmt.Sprintf("%[2]d %[1]d\n", 11, 22)
	将产生"22 11"，而
		fmt.Sprintf("%[3]*.[2]*[1]f", 12.0, 2, 6)
	相当于
		fmt.Sprintf("%6.2f", 12.0)
	将产生"12.00"。 由于显式索引会影响后续动词，因此可以通过为第一个要重复的参数重置索引来多次使用相同的符号来打印相同的值：
	fmt.Sprintf("%d %d %#[1]x %#x", 16, 17)
	将产生"16 17 0x10 0x11"

	格式错误:

	如果为verb给出了无效的参数，例如为%d提供了一个字符串，则生成的字符串将包含问题的描述，如以下示例所示：
		类型错误或动词未知: %!verb(type=value)
			Printf("%d", "hi"):        %!d(string=hi)
		参数太多: %!(EXTRA type=value)
			Printf("hi", "guys"):      hi%!(EXTRA string=guys)
		参数太少: %!verb(MISSING)
			Printf("hi%d"):            hi%!d(MISSING)
		宽度或精度为非整数: %!(BADWIDTH) or %!(BADPREC)
			Printf("%*s", 4.5, "hi"):  %!(BADWIDTH)hi
			Printf("%.*s", 4.5, "hi"): %!(BADPREC)hi
		无效或无效使用参数索引: %!(BADINDEX)
			Printf("%*[2]d", 7):       %!d(BADINDEX)
			Printf("%.[2]d", 7):       %!d(BADINDEX)

	所有错误均以字符串"%!"开头。 有时后面跟一个字符（verb），并以括号括起来。

	如果Error或String方法在由打印例程调用时触发了紧急情况，则fmt软件包会重新格式化来自紧急情况的错误消息，
	并用表明它来自fmt软件包的指示来装饰它。
	例如，如果String方法调用panic("bad")，则格式化后的消息看起来像
		%!s(PANIC=bad)

	%!s仅显示发生故障时正在使用的verb。 但是，如果恐慌是由nil接收者导致Error或String方法引起的，则输出为未经修饰的字符串"<nil>"。

	Scanning

	一组类似的功能会扫描格式化的文本以产生值。
	从os.Stdin读取的Scan，Scanf和Scanln; Fscan，Fscanf和Fscanln从指定的io.Reader读取；
	从参数字符串读取Sscan，Sscanf和Sscanln。

	Scan, Fscan, Sscan将输入中的换行符视为空格。

	Scanln, Fscanln and Sscanln停止在换行符处进行扫描，并要求在项目后跟换行符或EOF。

	Scanf, Fscanf, and Sscanf 根据类似于Printf的格式字符串解析参数。
	在下面的文本中，"空格"表示除换行符外的任何Unicode空格字符。

	在格式字符串中，由%字符引入的verb消耗并解析输入； 这些verb将在下面更详细地描述。
	格式中除%，空格或换行符以外的其他字符将完全消耗必须存在的输入字符。
	格式字符串中包含零个或多个空格的换行符会在输入中消耗零个或多个空格，后跟一个换行符或输入结尾。
	格式字符串中换行符后的空格在输入中消耗零个或多个空格。 否则，格式字符串中任何一个或多个空格的运行都会在输入中占用尽可能多的空格。
	除非格式字符串中的空格行与换行符相邻，否则该行必须占用输入中至少一个空格或找到输入的结尾。

	空格和换行的处理与C的scanf系列不同：在C中，换行被视为任何其他空格，并且当格式字符串中的空格运行在输入中找不到要使用的空格时，这绝不是错误。

	这些verb的行为类似于Printf的行为。
	例如，%x将扫描一个整数作为十六进制数，%v将扫描默认表示格式的值。
	未实现Printf verb %p和%T以及标志#和+。
	对于浮点和复数值，所有有效的格式化verb（%b %e %E %E %f %F %g %G %x %X和%v）都是等效的，
	并且接受十进制和十六进制表示法（例如："2.3e+7"，"0x4.5p-8"）和数字分隔的下划线（例如："3.14159_26535_89793"）。

	verb处理的输入是隐式用空格分隔的：
	除%c外，每个verb的实现都从丢弃其余输入中的前导空格开始，%s（和%v读入字符串）停止使用第一个空格或换行符的输入。

	当扫描不带格式或带有%v的整数时，可以接受熟悉的基本设置前缀0b（二进制），0o和0（八进制）和0x（十六进制），以数字分隔的下划线也是如此。

	宽度在输入文本中解释，但是没有用于精确扫描的语法（没有%5.2f，只有%5f）。
	如果提供了width，它将在修剪前导空格后应用，并指定要满足verb阅读的最大符文数。 例如，
	   Sscanf(" 1234567 ", "%5s%d", &s, &i)
	会将s设置为"12345"，而将i设置为67
	   Sscanf(" 12 34 567 ", "%5s%d", &s, &i)
	将s设置为"12"，将i设置为34。

	在所有扫描功能中，回车符后紧跟换行符被视为普通换行符（\r\n表示与\n相同）。

	在所有扫描功能中，如果操作数实现了Scan方法（即，它实现了Scanner接口），则该方法将用于扫描该操作数的文本。
	另外，如果扫描的参数数量小于提供的参数数量，则会返回错误。

	所有要扫描的参数都必须是指向Scanner接口的基本类型或实现的指针。

	像Scanf和Fscanf一样，Sscanf不需要消耗其整个输入。
	无法恢复Sscanf使用了多少输入字符串。

	注意：Fscan等可以读取返回的输入后的一个字符（rune），这意味着调用扫描例程的循环可能会跳过某些输入。
	仅当输入值之间没有空格时，这通常是一个问题。
	如果提供给Fscan的阅读器实现了ReadRune，则该方法将用于读取字符。
	如果阅读器还实现了UnreadRune，则将使用该方法保存字符，并且后续调用不会丢失数据。
	要将ReadRune和UnreadRune方法附加到没有该功能的阅读器，请使用bufio.NewReader。
*/
package fmt
