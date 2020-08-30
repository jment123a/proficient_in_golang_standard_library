// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

// TODO：此文件仅包含土耳其语和Azeri的特殊大小写规则。
// 它应该包含具有特殊大小写规则的所有语言，并且可以自动生成，但是首先需要开发一些API。

package unicode

var TurkishCase SpecialCase = _TurkishCase
var _TurkishCase = SpecialCase{
	CaseRange{0x0049, 0x0049, d{0, 0x131 - 0x49, 0}},
	CaseRange{0x0069, 0x0069, d{0x130 - 0x69, 0, 0x130 - 0x69}},
	CaseRange{0x0130, 0x0130, d{0, 0x69 - 0x130, 0}},
	CaseRange{0x0131, 0x0131, d{0x49 - 0x131, 0, 0x49 - 0x131}},
}

var AzeriCase SpecialCase = _TurkishCase
