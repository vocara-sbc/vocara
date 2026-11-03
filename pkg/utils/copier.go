package utils

import "github.com/jinzhu/copier"

// Copy performs a shallow struct copy from src to dst.
func Copy(dst, src interface{}) error {
	return copier.Copy(dst, src)
}

// CopyWithOption copies a struct with options.
func CopyWithOption(dst, src interface{}, opt copier.Option) error {
	return copier.CopyWithOption(dst, src, opt)
}

// MustCopy copies a struct and panics on failure.
func MustCopy(dst, src interface{}) {
	if err := copier.Copy(dst, src); err != nil {
		panic(err)
	}
}

// CopySlice copies a struct slice.
func CopySlice(dst, src interface{}) error {
	return copier.Copy(dst, src)
}
