//版权所有2009 The Go Authors。 版权所有。
//此源代码的使用受BSD样式的约束
//可以在LICENSE文件中找到的许可证。

// Pipe 适配器以连接期望io.Reader的代码和期望io.Writer的代码。

package io

import (
	"errors"
	"sync"
)

// OnceError 是一个只会存储一次错误的对象。
type onceError struct {
	sync.Mutex //守卫
	err        error
}

func (a *onceError) Store(err error) { //注：存放错误
	a.Lock()
	defer a.Unlock()
	if a.err != nil {
		return
	}
	a.err = err
}
func (a *onceError) Load() error { //注：取出错误
	a.Lock()
	defer a.Unlock()
	return a.err
}

// ErrClosedPipe 是用于封闭管道上的读取或写入操作的错误。
var ErrClosedPipe = errors.New("io: read/write on closed pipe") //注：因"在已经关闭的管道进行读取/写入"返回的错误

//pipe 是PipeReader和PipeWriter底层的共享管道结构。
type pipe struct {
	wrMu sync.Mutex  //序列化写操作 注：写锁（WriteMutex）
	wrCh chan []byte //注：写入管道
	rdCh chan int    //注：读取管道

	once sync.Once     //保护Close完成
	done chan struct{} //注：是否Close完成
	rerr onceError     //注：读取错误
	werr onceError     //注：写入错误
}

func (p *pipe) Read(b []byte) (n int, err error) { //注：从p的写入管道获取数据存入b与p的读取管道中，返回存入b的数据长度n与错误err
	select { //注：如果pipe已经关闭，返回错误
	case <-p.done:
		return 0, p.readCloseError()
	default:
	}

	select { //注：等待写入管道获取到数据或pipe关闭
	case bw := <-p.wrCh: //注：将写入管道里的数据拷贝给b，然后发送到读取管道中，返回拷贝给b的数据长度n与错误err
		nr := copy(b, bw)
		p.rdCh <- nr
		return nr, nil
	case <-p.done: //注：如果pipe已经关闭，返回错误
		return 0, p.readCloseError()
	}
}

func (p *pipe) readCloseError() error { //注：优先返回写入错误，否则返回管道关闭错误
	rerr := p.rerr.Load()
	if werr := p.werr.Load(); rerr == nil && werr != nil {
		return werr
	}
	return ErrClosedPipe
}

func (p *pipe) CloseRead(err error) error { //注：设置管道关闭错误，向p.done发送关闭命令
	if err == nil {
		err = ErrClosedPipe
	}
	p.rerr.Store(err)                   //注：写入管道关闭错误
	p.once.Do(func() { close(p.done) }) //注：执行一次关闭p.done操作，p.wrCh与p.rdCh没有被关闭
	return nil
}

func (p *pipe) Write(b []byte) (n int, err error) { //注：将b写入p.wrCh中，返回写入的数据长度n与错误err
	select { //注：如果pipe已经关闭，返回错误
	case <-p.done:
		return 0, p.writeCloseError()
	default: //注：否则，上写锁
		p.wrMu.Lock()
		defer p.wrMu.Unlock()
	}

	for once := true; once || len(b) > 0; once = false { //注：至少执行一次，直到b被全部写入p.wrCh中，或管道关闭
		select {
		case p.wrCh <- b:
			nw := <-p.rdCh
			b = b[nw:]
			n += nw
		case <-p.done: //注：如果pipe已经关闭，返回错误
			return n, p.writeCloseError()
		}
	}
	return n, nil
}

func (p *pipe) writeCloseError() error { //注：优先返回读取错误，否则返回管道关闭错误
	werr := p.werr.Load()
	if rerr := p.rerr.Load(); werr == nil && rerr != nil { //注：优先返回读取错误，否则返回管道关闭错误
		return rerr
	}
	return ErrClosedPipe
}

func (p *pipe) CloseWrite(err error) error { //注：设置管道关闭错误，向p.done发送关闭命令
	if err == nil {
		err = EOF
	}
	p.werr.Store(err)                   //注：写入管道关闭错误
	p.once.Do(func() { close(p.done) }) //注：执行一次关闭p.done操作，p.wrCh与p.rdCh没有被关闭
	return nil
}

// PipeReader 是管道的读取部分。
type PipeReader struct {
	p *pipe
}

// Read 实现标准的Read接口：
// 它从管道读取数据，阻塞直到Writer到达或写入端关闭。
// 如果写入端由于错误而关闭，则该错误将以err的形式返回； 否则err为EOF。
func (r *PipeReader) Read(data []byte) (n int, err error) { //注：调用pipe.Read
	return r.p.Read(data)
}

// Close 关闭Reader； 随后写入管道进行一半的操作将返回错误ErrClosedPipe。
func (r *PipeReader) Close() error { // 注：#关闭管道
	return r.CloseWithError(nil)
}

// CloseWithError 关闭阅读器； 随后写入管道进行一半的操作将返回错误err。
// CloseWithError永远不会覆盖以前的错误（如果存在），并且始终返回nil。
func (r *PipeReader) CloseWithError(err error) error { // 注：#关闭管道
	return r.p.CloseRead(err)
}

// PipeWriter 是管道的写入部分。
type PipeWriter struct {
	p *pipe
}

// Write 实现标准的Write接口：
// 它将数据写入管道，直到一个或多个读取器消耗完所有数据或关闭读取端为止，它一直阻塞。
// 如果读取端因错误而关闭，则该err返回为err; 否则err是ErrClosedPipe。
func (w *PipeWriter) Write(data []byte) (n int, err error) { // 注：w写入data
	return w.p.Write(data)
}

// Close 关闭Writer； 从管道读取的一半进行的后续读取将不返回任何字节和EOF。
func (w *PipeWriter) Close() error { // 注：关闭管道
	return w.CloseWithError(nil)
}

// CloseWithError 关闭编写器； 从管道读取的一半进行的后续读取将不返回任何字节，并且错误err；如果err为nil，则返回EOF。
// CloseWithError永远不会覆盖以前的错误（如果存在），并且始终返回nil。
func (w *PipeWriter) CloseWithError(err error) error { // 注：关闭管道
	return w.p.CloseWrite(err)
}

// Pipe 创建一个同步的内存管道。
// 它可以用来连接期望io.Reader的代码和期望io.Writer的代码。
//
// 管道上的读取和写入是一对一匹配的，除非需要多个读取来消耗单个写入。
// 也就是说，每次对PipeWriter的写入都将阻塞，直到它满足从PipeReader读取的一个或多个读取，这些读取会完全消耗已写入的数据。
// 将数据直接从Write复制到相应的Read（或多个Read）； 没有内部缓冲。
//
// 可以并行或通过Close并行调用Read和Write是安全的。
// 并行调用Read和并行调用Write也是安全的：
// 各个呼叫将按顺序进行门控。
func Pipe() (*PipeReader, *PipeWriter) { //工厂函数
	p := &pipe{
		wrCh: make(chan []byte),
		rdCh: make(chan int),
		done: make(chan struct{}),
	}
	return &PipeReader{p}, &PipeWriter{p}
}
