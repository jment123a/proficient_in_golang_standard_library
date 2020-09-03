package strconv

// ParseBool 返回由字符串表示的布尔值。
// 接受1，t，T，TRUE，true，True，0，f，F，FALSE，false，False。
// 其他任何值都会返回错误。
func ParseBool(str string) (bool, error) { // 注：将字符串str转为布尔类型，返回布尔值与错误
	switch str {
	case "1", "t", "T", "true", "TRUE", "True":
		return true, nil
	case "0", "f", "F", "false", "FALSE", "False":
		return false, nil
	}
	return false, syntaxError("ParseBool", str)
}

// FormatBool 根据b的值返回"true"或"false"。
func FormatBool(b bool) string { // 注：将布尔类型b转为字符串并返回
	if b {
		return "true"
	}
	return "false"
}

// AppendBool 根据b的值将“true"或"false"附加到dst并返回扩展缓冲区。
func AppendBool(dst []byte, b bool) []byte { // 注：将布尔类型b转为字符串追加值dst中
	if b {
		return append(dst, "true"...)
	}
	return append(dst, "false"...)
}
