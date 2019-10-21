// +build gorgonia

package core

import (
	"unsafe"

	"gorgonia.org/tensor"
)

// ToGorgoniaTensor packages the tensor as a gorgonia's Dense tensor.
// The memory is not copied.
func (arr NDArray) ToGorgoniaTensor() *tensor.Dense {
	if len(arr.Data) == 0 {
		return nil
	}
	return tensor.New(tensor.Of(tensor.Dtype{Type: arr.ReflectedDataType()}),
		tensor.WithShape(arr.Shape...),
		tensor.FromMemory(uintptr(unsafe.Pointer(&arr.Data[0])), uintptr(len(arr.Data))))
}
