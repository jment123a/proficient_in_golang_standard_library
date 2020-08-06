// 版权所有2009 The Go Authors。 版权所有。
//  此源代码的使用受BSD样式的约束
//可以在LICENSE文件中找到的许可证。

// Package ioutil 实现了一些I/O实用程序功能。
package ioutil

import (
	"bytes"
	"io"
	"os"
	"sort"
	"sync"
)

// readAll 从r读取直到出现错误或EOF，并从分配给指定容量的内部缓冲区中返回读取的数据。
func readAll(r io.Reader, capacity int64) (b []byte, err error) { //注：从r读取数据，数据长度为capacity，返回读取到的数据b与错误err
	var buf bytes.Buffer
	//如果缓冲区溢出，我们将获得bytes.ErrTooLarge。
	//将其返回为错误。 还有其他恐慌情绪。
	defer func() {
		e := recover() //注：捕获恐慌
		if e == nil {
			return
		}
		if panicErr, ok := e.(error); ok && panicErr == bytes.ErrTooLarge { //注：捕获错误"缓冲区太大"，转换为错误
			err = panicErr
		} else {
			panic(e)
		}
	}()
	if int64(int(capacity)) == capacity { //注：扩展缓冲区至当前系统对应长度
		buf.Grow(int(capacity))
	}
	_, err = buf.ReadFrom(r) //注：从r中读取数据存到buf中
	return buf.Bytes(), err
}

// ReadAll 从r读取直到出现错误或EOF，然后返回读取的数据。
// 成功的调用返回err == nil，而不是err == EOF。
// 因为ReadAll被定义为从src读取直到EOF，所以它不会将读取的EOF视为要报告的错误。
func ReadAll(r io.Reader) ([]byte, error) { //注：从r读取数据，数据长度为512，返回读取到的数据与错误
	return readAll(r, bytes.MinRead)
}

// ReadFile 读取filename命名的文件并返回内容。
// 成功的调用返回err == nil，而不是err == EOF。
// 由于ReadFile读取整个文件，因此不会将Read中的EOF视为要报告的错误。
func ReadFile(filename string) ([]byte, error) { //注：打开文件filename，返回所有读取到的数据与错误
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// FileInfo会准确地告诉我们要读取多少，这是一个很好的选择，但不能肯定，因此让我们尝试一下，但要为错误的答案做好准备。
	var n int64 = bytes.MinRead

	if fi, err := f.Stat(); err == nil {
		//作为readAll的初始容量，在Size为零的情况下，请使用Size +一些额外的值，并在Read填充缓冲区后避免再次分配。
		//readAll调用将廉价地读取其分配的内部缓冲区。
		//如果大小不正确，我们要么浪费一些空间，要么根据需要重新分配，但是在绝大多数情况下，我们会使其正确。
		if size := fi.Size() + bytes.MinRead; size > n {
			n = size
		}
	}
	return readAll(f, n)
}

// WriteFile 将数据写入文件名命名的文件。
// 如果文件不存在，WriteFile将使用权限perm（在umask之前）创建该文件； 否则WriteFile会在写入之前将其截断。
func WriteFile(filename string, data []byte, perm os.FileMode) error { //注：打开文件filename，模式为perm，写入data，返回错误
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm) //注：打开文件filename，只写、如果没有则创建、从位置0开始写入
	if err != nil {
		return err
	}
	_, err = f.Write(data)             //注：写入
	if err1 := f.Close(); err == nil { //注：关闭
		err = err1
	}
	return err
}

// ReadDir 读取以目录名命名的目录，并返回按文件名排序的目录条目列表。
func ReadDir(dirname string) ([]os.FileInfo, error) { //注：遍历dirname，返回按文件名称排序后的文件信息与错误
	f, err := os.Open(dirname) //注：打开文件夹
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(-1) //注：读取所有文件
	f.Close()                  //注：关闭
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() }) //注：按文件名称排序
	return list, nil
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil } //注：总是关闭成功

// NopCloser 返回一个带有无操作Close方法的ReadCloser，该方法包装提供的Reader r。
func NopCloser(r io.Reader) io.ReadCloser { //工厂函数
	return nopCloser{r}
}

type devNull int

// devNull实现了ReaderFrom的优化，因此io.Copy到ioutil.Discard可以避免做不必要的工作。
var _ io.ReaderFrom = devNull(0)

func (devNull) Write(p []byte) (int, error) { //注：总是写入成功
	return len(p), nil
}

func (devNull) WriteString(s string) (int, error) { //注：总是写入成功
	return len(s), nil
}

var blackHolePool = sync.Pool{ //工厂函数
	New: func() interface{} {
		b := make([]byte, 8192)
		return &b
	},
}

func (devNull) ReadFrom(r io.Reader) (n int64, err error) { //注：读取r中读取一次数据到缓冲区中，返回读取道德数据长度n与错误n
	bufp := blackHolePool.Get().(*[]byte) //注：从对象缓冲池中获取缓冲区对象
	readSize := 0
	for {
		readSize, err = r.Read(*bufp) //注：从r中读取数据到bufp
		n += int64(readSize)
		if err != nil {
			blackHolePool.Put(bufp) //注：将缓冲区对象放入对象缓冲池
			if err == io.EOF {
				return n, nil
			}
			return
		}
	}
}

// Discard 是一个io.Writer，所有Write调用均不执行任何操作而成功。
var Discard io.Writer = devNull(0)
