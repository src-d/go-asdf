// +build gorgonia

package core

import (
	"errors"
	"unsafe"

	"gorgonia.org/tensor"
)

// ToGorgoniaTensor packages the tensor as a gorgonia's Dense tensor.
// The memory is not copied.
func (arr NDArray) ToGorgoniaTensor() (*tensor.Dense, error) {
	if len(arr.Data) == 0 {
		return nil, nil
	}
	if arr.ByteOrder.String() != hbo.String() {
		return nil, errors.New("endianness mismatch, you need to call EnsureHostEndianness()")
	}
	return tensor.New(tensor.Of(tensor.Dtype{Type: arr.ReflectedDataType()}),
		tensor.WithShape(arr.Shape...),
		tensor.FromMemory(uintptr(unsafe.Pointer(&arr.Data[0])), uintptr(len(arr.Data)))), nil
}
