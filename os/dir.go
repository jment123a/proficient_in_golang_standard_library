// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

// Readdir 读取与file关联的目录的内容，并按目录顺序返回最多n个FileInfo值的切片，这将由Lstat返回。
// 对同一文件的后续调用将产生进一步的FileInfo。
// 如果n> 0，则Readdir最多返回n个FileInfo结构。
// 在这种情况下，如果Readdir返回一个空片，它将返回一个非null错误，说明原因。 在目录末尾，错误是io.EOF。
// 如果n <= 0，则Readdir在单个片中返回目录中的所有FileInfo。
// 在这种情况下，如果Readdir成功（一直读取到目录的末尾），它将返回切片和nil错误。
// 如果它在目录末尾之前遇到错误，则Readdir返回读取的FileInfo直到该点为止，并且返回非nil错误。
func (f *File) Readdir(n int) ([]FileInfo, error) {
	if f == nil {
		return nil, ErrInvalid
	}
	return f.readdir(n)
}

// Readdirnames reads the contents of the directory associated with file
// and returns a slice of up to n names of files in the directory,
// in directory order. Subsequent calls on the same file will yield
// further names.
//
// If n > 0, Readdirnames returns at most n names. In this case, if
// Readdirnames returns an empty slice, it will return a non-nil error
// explaining why. At the end of a directory, the error is io.EOF.
//
// If n <= 0, Readdirnames returns all the names from the directory in
// a single slice. In this case, if Readdirnames succeeds (reads all
// the way to the end of the directory), it returns the slice and a
// nil error. If it encounters an error before the end of the
// directory, Readdirnames returns the names read until that point and
// a non-nil error.
func (f *File) Readdirnames(n int) (names []string, err error) {
	if f == nil {
		return nil, ErrInvalid
	}
	return f.readdirnames(n)
}
