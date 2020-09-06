/*
	文件：
		bufio.go	带有缓冲的Reader和Writer
		scan.go		带有缓冲的Scanner

	结构体与接口：
		type Reader struct
		type Writer struct
		type ReadWriter struct

	函数与方法：
		NewReaderSize(rd io.Reader, size int) *Reader						工厂函数，生成一个缓冲区大小为size的io.Reader结构体
		NewReader(rd io.Reader) *Reader										工厂函数，生成一个io.Reader结构体
			(b *Reader) Size() int											获取b的缓冲区大小
			(b *Reader) Reset(r io.Reader)									重置b
			(b *Reader) Peek(n int) ([]byte, error)							b读取n字节数据
			(b *Reader) Discard(n int) (discarded int, err error)			将b的缓冲区读取位置跳过n个字符
			(b *Reader) Buffered() int										获取b的缓冲区可以读取的字节数
			(b *Reader) reset(buf []byte, r io.Reader)						重置b
			(b *Reader) fill()												从b.rd中读取数据写入缓冲区
			(b *Reader) readErr() error										返回b的错误
			--read
			(b *Reader) Read(p []byte) (n int, err error)					将b的缓冲区内的未读数据写入p中
			(b *Reader) ReadByte() (byte, error)							获取b的缓冲区中1字节的未读数据
			(b *Reader) UnreadByte() error									撤回上次ReadByte()
			(b *Reader) ReadRune() (r rune, size int, err error)			从未读数据中获取一个rune
			(b *Reader) UnreadRune() error									撤回上次ReadRune()
			(b *Reader) ReadSlice(delim byte) (line []byte, err error)		获取数据，直到遇到delim
			(b *Reader) ReadLine() (line []byte, isPrefix bool, err error)	获取数据，直到遇到\n或\r\n
			(b *Reader) ReadBytes(delim byte) ([]byte, error)				获取数据，直到遇到delim（保证数据完整）
			(b *Reader) ReadString(delim byte) (string, error)				获取数据，直到遇到delim（保证数据完整）
			--write
			(b *Reader) WriteTo(w io.Writer) (n int64, err error)			将b的所有数据写入w
			(b *Reader) writeBuf(w io.Writer) (int64, error)				将缓冲区中的未读数据写入w

		NewWriterSize(w io.Writer, size int) *Writer						工厂函数，生成一个Writer结构体，缓冲区大小为size
		NewWriter(w io.Writer) *Writer										工厂函数，生成一个Writer结构体
			(b *Writer) Size() int											获取b的缓冲区大小
			(b *Writer) Reset(w io.Writer)									重置b的缓冲区，writer为w
			(b *Writer) Flush() error										将缓冲区中的数据写入writer
			(b *Writer) Available() int										获取缓冲区中未使用的字节数
			(b *Writer) Buffered() int										获取缓冲区中已使用的字节数
			--write
			(b *Writer) Write(p []byte) (nn int, err error)					将p写入writer
			(b *Writer) WriteByte(c byte) error								flush缓冲区，将c写入缓冲区
			(b *Writer) WriteRune(r rune) (size int, err error)				将r写入缓冲区
			(b *Writer) WriteString(s string) (int, error)					将s写入缓冲区
			--read
			(b *Writer) ReadFrom(r io.Reader) (n int64, err error)			从r中读取数据写入writer，直到出现错误

		NewReadWriter(r *Reader, w *Writer) *ReadWriter						工厂函数，生成一个ReadWriter结构体
*/