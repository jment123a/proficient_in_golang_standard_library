/*
	结构体与接口
		type Reader interface
		type Writer interface
		type Closer interface
		type Seeker interface
		type ReadWriter interface
		type ReadCloser interface
		type WriteCloser interface
		type ReadWriteCloser interface
		type ReadSeeker interface
		type WriteSeeker interface
		type ReadWriteSeeker interface
		type ReaderFrom interface
		type WriterTo interface
		type ReaderAt interface
		type WriterAt interface
		type ByteReader interface
		type ByteScanner interface
		type ByteWriter interface
		type RuneScanner interface
		type StringWriter interface

		type LimitedReader struct
		type SectionReader struct
		type teeReader struct
		---multi.go
		type eofReader struct
			(eofReader) Read([]byte) (int, error)							总是返回EOF
		type multiReader struct
		type multiWriter struct
		---pipe.go
		type onceError struct
			(a *onceError) Store(err error)			存放错误
			(a *onceError) Load() error				取出错误
		type pipe struct
			(p *pipe) Read(b []byte) (n int, err error) 		从p的写入管道获取数据存入b与p的读取管道中，返回存入b的数据长度n与错误err
			(p *pipe) readCloseError() error					优先返回写入错误，否则返回管道关闭错误
			(p *pipe) CloseRead(err error) error 				设置管道关闭错误，向p.done发送关闭命令
			(p *pipe) Write(b []byte) (n int, err error)		将b写入p.wrCh中，返回写入的数据长度n与错误err
			(p *pipe) writeCloseError() error 					优先返回读取错误，否则返回管道关闭错误
			(p *pipe) CloseWrite(err error) error				设置管道关闭错误，向p.done发送关闭命令
		type PipeReader struct
			(r *PipeReader) Read(data []byte) (n int, err error)	调用pipe.Read
			(r *PipeReader) Close() error							关闭管道
			(r *PipeReader) CloseWithError(err error) error			关闭管道
		type PipeWriter struct
			(w *PipeWriter) Write(data []byte) (n int, err error) 	w写入data
			(w *PipeWriter) Close() error							关闭管道
			(w *PipeWriter) CloseWithError(err error) error 		关闭管道
		---ioutil/ioutil.go
		type nopCloser struct
	函数与方法
		Copy(dst Writer, src Reader) (written int64, err error) 			从src中读取数据写入dst，缓冲区为内部生成，返回拷贝的数据长度written与错误err
		CopyN(dst Writer, src Reader, n int64) (written int64, err error)	从src中读取长度为n的数据写入dst，缓冲区为内部生成，返回拷贝的数据长度written与错误err
		CopyBuffer(...)														从src中读取数据写入dst，缓冲区为buf，返回拷贝的数据长度written与错误err
		copyBuffer(...)														从src中读取数据写入dst，缓冲区为buf，返回拷贝的数据长度written与错误err
		WriteString(w Writer, s string) (n int, err error)					调用w.WriteString或w.Write写入s，返回写入的长度与错误err
		LimitReader(r Reader, n int64) Reader								工厂函数
			(l *LimitedReader) Read(p []byte) (n int, err error)			从l.R读取l.N个数据存到缓冲区p中，返回读取到的数据长度n与错误err
		 NewSectionReader(r ReaderAt, off int64, n int64) *SectionReader	工厂函数
			(s *SectionReader) Read(p []byte) (n int, err error)			#
			(s *SectionReader) Seek(...)									重新定位s.off至s.base与s.limit之间的某个相对位置，可以向前也可以向后
			(s *SectionReader) ReadAt(...)									修剪缓冲区p后从s.r中off位置读取数据到p中，返回读取到的数据长度n与错误err
			(s *SectionReader) Size() int64									返回一共要读取的数据的长度
		TeeReader(r Reader, w Writer) Reader								工厂函数
			(t *teeReader) Read(p []byte) (n int, err error) 				从t.r中读取数据存到缓冲区p中，再写入t.w中，返回写入数据的长度n与错误err
		ReadAtLeast(r Reader, buf []byte, min int) (n int, err error)		调用r.Read至少min字节数据存到buf中，返回读取到的数据长度n与错误err
		ReadFull(r Reader, buf []byte) (n int, err error) 					调用r.Read至少min字节数据存到buf中，返回读取到的数据长度n与错误err
		---multi.go
		MultiReader(readers ...Reader) Reader								工厂函数
			(mr *multiReader) Read(p []byte) (n int, err error)				遍历mr.readers，任何一个Reader读取到数据存放到缓冲区p中就返回读取到的数据长度n与错误err
		MultiWriter(writers ...Writer) Writer								工厂函数，遍历writers，检查是否为multiWriter，追加w.writers或w
			(t *multiWriter) Write(p []byte) (n int, err error)				遍历t.writers，每个Writer都写入p，返回p的长度n与错误err
			(t *multiWriter) WriteString(s string) (n int, err error)		遍历t.writers，执行w.StringWriter或w.Write写入s，返回s的长度n与错误err
		---pipe.go
		Pipe() (*PipeReader, *PipeWriter)									工厂函数
		---ioutil/ioutil.go
		ReadAll(r io.Reader) ([]byte, error) 								从r读取数据，数据长度为512，返回读取到的数据与错误，简化readAll
		readAll(r io.Reader, capacity int64) (b []byte, err error) 			从r读取数据，数据长度为capacity，返回读取到的数据b与错误err
		ReadFile(filename string) ([]byte, error)							打开文件filename，返回所有读取到的数据与错误
		ReadDir(dirname string) ([]os.FileInfo, error)						遍历dirname，返回按文件名称排序后的文件信息与错误
		WriteFile(filename string, data []byte, perm os.FileMode) error 	打开文件filename，模式为perm，写入data，返回错误
		NopCloser(r io.Reader) io.ReadCloser								工厂函数
			(nopCloser) Close() error										总是关闭成功
		type devNull int
			(devNull) Write(p []byte) (int, error) 							总是写入成功
			(devNull) WriteString(s string) (int, error) 					总是写入成功
			(devNull) ReadFrom(r io.Reader) (n int64, err error)			读取r中读取一次数据到缓冲区中，返回读取到的数据长度n与错误err
		---ioutil/tempfile.go
		reseed() uint32 													返回根据当前纳秒级时间戳+进程id组成的随机数种子
		nextRandom() string 												返回一个随机的文件名，长度为10
		TempFile(dir, pattern string) (f *os.File, err error) 				在目录dir下生成临时文件，文件模板为pattern，返回打开的临时文件f与错误err
		prefixAndSuffix(pattern string) (prefix, suffix string)				如果pattern包含"*"，则返回*前的字符串prefix与之后的字符串suffix
		TempDir(dir, pattern string) (name string, err error)				在目录dir下生成临时文件夹，文件模板为pattern，返回临时文件夹name与错误err


*/

package io
