package utils

import "github.com/jinzhu/copier"

// Copy 结构体拷贝（浅拷贝）
// 将 src 中的字段拷贝到 dst
func Copy(dst, src interface{}) error {
	return copier.Copy(dst, src)
}

// CopyWithOption 带选项的结构体拷贝
func CopyWithOption(dst, src interface{}, opt copier.Option) error {
	return copier.CopyWithOption(dst, src, opt)
}

// MustCopy 结构体拷贝（失败时 panic）
func MustCopy(dst, src interface{}) {
	if err := copier.Copy(dst, src); err != nil {
		panic(err)
	}
}

// CopySlice 结构体切片拷贝
func CopySlice(dst, src interface{}) error {
	return copier.Copy(dst, src)
}
