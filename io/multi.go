// 版权所有2010 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

package io

type eofReader struct{}

func (eofReader) Read([]byte) (int, error) { //注：总是返回EOF
	return 0, EOF
}

type multiReader struct {
	readers []Reader
}

func (mr *multiReader) Read(p []byte) (n int, err error) { //注：遍历mr.readers，任何一个Reader读取到数据存放到缓冲区p中就返回读取到的数据长度n与错误err
	for len(mr.readers) > 0 {
		//优化以使嵌套的multiReader扁平化（问题13558）。
		if len(mr.readers) == 1 { //注：如果mr只有1个Reader，则检查这个Reader是不是multiReader
			if r, ok := mr.readers[0].(*multiReader); ok { //注：如果是，遍历这个Reader
				mr.readers = r.readers
				continue
			}
		}
		n, err = mr.readers[0].Read(p)
		if err == EOF { //注：Reader遇到EOF错误，则丢掉这个Reader
			//使用eofReader而不是nil来避免扁平化后出现nil恐慌（问题18232）。
			mr.readers[0] = eofReader{} //允许更早的GC
			mr.readers = mr.readers[1:]
		}
		if n > 0 || err != EOF { //注：从任意Reader读取到数据，直接返回
			if err == EOF && len(mr.readers) > 0 {
				//暂时不要返回EOF。 仍然有更多的Reader。
				err = nil
			}
			return
		}
	}
	return 0, EOF
}

// MultiReader 返回一个Reader，该Reader是提供的输入readers的逻辑串联。 它们被顺序读取。
// 一旦所有输入均返回EOF，读取将返回EOF。
// 如果任何读取器返回非零，非EOF错误，则Read将返回该错误。
func MultiReader(readers ...Reader) Reader { //工厂函数
	r := make([]Reader, len(readers))
	copy(r, readers)
	return &multiReader{r}
}

type multiWriter struct {
	writers []Writer
}

func (t *multiWriter) Write(p []byte) (n int, err error) { //注：遍历t.writers，每个Writer都写入p，返回p的长度n与错误err
	for _, w := range t.writers { //注：遍历t.writers
		n, err = w.Write(p) //注：每个Writer都写入p
		if err != nil {
			return
		}
		if n != len(p) {
			err = ErrShortWrite
			return
		}
	}
	return len(p), nil
}

var _ StringWriter = (*multiWriter)(nil) //注：#

func (t *multiWriter) WriteString(s string) (n int, err error) { //注：遍历t.writers，执行w.StringWriter或w.Write写入s，返回s的长度n与错误err
	var p []byte                  //如果需要的话可以延迟初始化
	for _, w := range t.writers { //注：遍历t.writers
		if sw, ok := w.(StringWriter); ok { //注：如果w有StringWriter方法则执行
			n, err = sw.WriteString(s)
		} else { //注：否则Write写入s
			if p == nil {
				p = []byte(s)
			}
			n, err = w.Write(p)
		}
		if err != nil {
			return
		}
		if n != len(s) {
			err = ErrShortWrite
			return
		}
	}
	return len(s), nil
}

// MultiWriter 创建一个Writer，该Writer将其写入复制到所有提供的Writer中，类似于Unix tee(1)命令。
// 遍历写入每个Writer，每次写一个
// 如果列出的写程序返回错误，则整个写操作将停止并返回错误； 它不会在列表中继续下去。
func MultiWriter(writers ...Writer) Writer { //工厂函数，注：遍历writers，检查是否为multiWriter，追加w.writers或w
	allWriters := make([]Writer, 0, len(writers))
	for _, w := range writers {
		if mw, ok := w.(*multiWriter); ok {
			allWriters = append(allWriters, mw.writers...)
		} else {
			allWriters = append(allWriters, w)
		}
	}
	return &multiWriter{allWriters}
}
