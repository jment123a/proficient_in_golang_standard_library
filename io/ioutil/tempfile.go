// 版权所有2010 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。
package ioutil

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// rand 数字状态。
// 我们会生成随机的临时文件名，以便很有可能该文件尚不存在-将TempFile中的尝试次数减至最少。
var rand uint32       //注：如果为0则没有生成过随机数种子
var randmu sync.Mutex //注：生成随机数的锁

func reseed() uint32 { //注：返回根据当前纳秒级时间戳+进程id组成的随机数种子
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

func nextRandom() string { //注：返回一个随机的文件名，长度为10
	randmu.Lock() //注：上锁
	r := rand
	if r == 0 { //注：生成随机数种子
		r = reseed()
	}
	r = r*1664525 + 1013904223 // 数值算法中的常量
	rand = r
	randmu.Unlock()                           //注：解锁
	return strconv.Itoa(int(1e9 + r%1e9))[1:] //注：返回为字符串,长度为10
}

// TempFile 在目录dir中创建一个新的临时文件，打开该文件进行读取和写入，并返回结果*os.File。
// 通过采用模式并在结尾添加随机字符串来生成文件名。 如果pattern包含"*"，则随机字符串将替换最后的"*"。
// 如果dir是空字符串，则TempFile使用默认目录存储临时文件(请参见os.TempDir)。
// 多个同时调用TempFile的程序将不会选择同一文件。
// 调用者可以使用f.Name()查找文件的路径名。 不再需要该文件时，调用方有责任删除它。
func TempFile(dir, pattern string) (f *os.File, err error) { //注：在目录dir下生成临时文件，文件模板为pattern，返回打开的临时文件f与错误err
	if dir == "" { //注：如果没设置临时目录路径，则使用默认临时目录
		dir = os.TempDir()
	}

	prefix, suffix := prefixAndSuffix(pattern) //注：拆分文件名

	nconflict := 0
	for i := 0; i < 10000; i++ { //注：遍历一万次
		name := filepath.Join(dir, prefix+nextRandom()+suffix)            //注：组合文件路径
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600) //注：打开文件，读写、如果文件不存在则创建新文件、对UNIX系统兼容
		if os.IsExist(err) {                                              //注：判断文件是否已存在
			if nconflict++; nconflict > 10 { //注：如果连续10次重复，则更换随机种子
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		break
	}
	return
}

// prefixAndSuffix 按最后一个通配符"*"拆分模式，如果适用，返回前缀作为"*"之前的部分，并返回后缀作为"*"之后的部分。
func prefixAndSuffix(pattern string) (prefix, suffix string) { //注：如果pattern包含"*"，则返回*前的字符串prefix与之后的字符串suffix
	if pos := strings.LastIndex(pattern, "*"); pos != -1 {
		prefix, suffix = pattern[:pos], pattern[pos+1:]
	} else {
		prefix = pattern
	}
	return
}

// TempDir 在目录dir中创建一个新的临时目录。
// 目录名称是通过采用模式并在末尾应用随机字符串来生成的。
// 如果pattern包含"*"，则随机字符串将替换最后的"*"。 TempDir返回新目录的名称。
// 如果dir是空字符串，则TempDir使用默认目录存储临时文件（请参见os.TempDir）。
// 多个同时调用TempDir的程序将不会选择同一目录。 不再需要该目录时，调用方有责任删除它。
func TempDir(dir, pattern string) (name string, err error) { //注：在目录dir下生成临时文件夹，文件模板为pattern，返回临时文件夹name与错误err
	if dir == "" { //注：如果没设置临时目录路径，则使用默认临时目录
		dir = os.TempDir()
	}

	prefix, suffix := prefixAndSuffix(pattern) //注：拆分文件名

	nconflict := 0
	for i := 0; i < 10000; i++ { //注：遍历一万次
		try := filepath.Join(dir, prefix+nextRandom()+suffix) //注：组合文件夹路径
		err = os.Mkdir(try, 0700)                             //注：创建文件夹
		if os.IsExist(err) {                                  //注：判断文件夹是否已存在
			if nconflict++; nconflict > 10 { //注：如果连续10次重复，则更换随机种子
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		if os.IsNotExist(err) { //注：如果目录不存在，也查不到目录的信息，则返回错误
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return "", err
			}
		}
		if err == nil {
			name = try
		}
		break
	}
	return
}
