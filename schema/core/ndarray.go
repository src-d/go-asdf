package core

import (
	"encoding/binary"
	"fmt"
	"go/types"
	"log"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/blang/semver"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/src-d/go-asdf/schema"
)

// NDArray is defined in https://asdf-standard.readthedocs.io/en/latest/generated/stsci.edu/asdf/core/ndarray-1.0.0.html
// It is similar to `numpy.ndarray` in Python.
type NDArray struct {
	// DataType is the tensor element type.
	DataType *types.Basic
	// Shape is the tensor shape: a one-dimensional integer sequence.
	Shape []int
	// ByteOrder is the byte order if the tensor contains integers.
	ByteOrder binary.ByteOrder
	// Data is the raw tensor buffer, similar to `numpy.ndarray.data`.
	Data []byte
}

type ndarrayPosition struct {
	// Strides is the numbers of bytes to step in each dimension when traversing the tensor.
	Strides []int
	// Offset is the number of bytes to initially skip in the block.
	Offset int
}

// String formats the tensor as a string. The actual contents are not included.
func (arr NDArray) String() string {
	dims := make([]string, 0, len(arr.Shape))
	for _, s := range arr.Shape {
		dims = append(dims, strconv.Itoa(s))
	}
	return fmt.Sprintf("array<%s, %s> of shape [%s]", arr.DataType.String(),
		arr.ByteOrder.String(), strings.Join(dims, ", "))
}

// ReflectedDataType returns the data type as a reflect.Type.
func (arr NDArray) ReflectedDataType() reflect.Type {
	return reflectMapping[arr.DataType.Name()]
}

// ElementSize returns the data type size in bytes.
func (arr NDArray) ElementSize() int {
	return int((&types.StdSizes{WordSize: 8, MaxAlign: 8}).Sizeof(arr.DataType.Underlying()))
}

// CountElements returns the total number of elements in the tensor.
func (arr NDArray) CountElements() int {
	size := 1
	for _, dim := range arr.Shape {
		size *= dim
	}
	return size
}

// CountBytes returns the size of the tensor in bytes.
func (arr NDArray) CountBytes() int {
	return arr.CountElements() * arr.ElementSize()
}

// EnsureHostEndianness changes the endianness to host as needed.
func (arr *NDArray) EnsureHostEndianness() {
	if arr.ByteOrder.String() == hbo.String() {
		return
	}
	dts := arr.ElementSize()
	if dts == 1 {
		return
	}
	// we cannot run in-place because several arrays can reference the same byte slice
	fixed := make([]byte, len(arr.Data))
	for offset := 0; offset < len(arr.Data); offset += dts {
		switch dts {
		case 2:
			hbo.PutUint16(fixed[offset:offset+2], arr.ByteOrder.Uint16(arr.Data[offset:offset+2]))
		case 4:
			hbo.PutUint32(fixed[offset:offset+4], arr.ByteOrder.Uint32(arr.Data[offset:offset+4]))
		case 8:
			hbo.PutUint64(fixed[offset:offset+8], arr.ByteOrder.Uint64(arr.Data[offset:offset+8]))
		}
	}
	arr.Data = fixed
	arr.ByteOrder = hbo
}

var basicMapping = map[string]*types.Basic{
	"int8":       types.Typ[types.Int8],
	"int16":      types.Typ[types.Int16],
	"int32":      types.Typ[types.Int32],
	"int64":      types.Typ[types.Int64],
	"uint8":      types.Typ[types.Uint8],
	"uint16":     types.Typ[types.Uint16],
	"uint32":     types.Typ[types.Uint32],
	"uint64":     types.Typ[types.Uint64],
	"float32":    types.Typ[types.Float32],
	"float64":    types.Typ[types.Float64],
	"complex64":  types.Typ[types.Complex64],
	"complex128": types.Typ[types.Complex128],
}

var reflectMapping = map[string]reflect.Type{
	"int8":       reflect.TypeOf(int8(0)),
	"int16":      reflect.TypeOf(int16(0)),
	"int32":      reflect.TypeOf(int32(0)),
	"int64":      reflect.TypeOf(int64(0)),
	"uint8":      reflect.TypeOf(uint8(0)),
	"uint16":     reflect.TypeOf(uint16(0)),
	"uint32":     reflect.TypeOf(uint32(0)),
	"uint64":     reflect.TypeOf(uint64(0)),
	"float32":    reflect.TypeOf(float32(0)),
	"float64":    reflect.TypeOf(float64(0)),
	"complex64":  reflect.TypeOf(complex64(0)),
	"complex128": reflect.TypeOf(complex128(0)),
}

type ndarrayUnmarshaler struct {
}

func (ndaum ndarrayUnmarshaler) Version() semver.Version {
	return semver.MustParse("1.0.0")
}

func (ndaum ndarrayUnmarshaler) UnmarshalYAML(value *yaml.Node) (interface{}, error) {
	pos := ndarrayPosition{}
	var inlineData *gabs.Container
	arr := &NDArray{Data: make([]byte, 4), ByteOrder: hbo}

	gabsifyInlineData := func(node *yaml.Node) error {
		root := gabs.New()
		node.Tag = ""
		err := schema.GabsifyYAML(root, node, "data")
		if err != nil {
			return errors.Wrapf(err, "while parsing core/ndarray-%s/source: failed "+
				"to process the inline data", ndaum.Version())
		}
		inlineData = root.Path("data")
		return nil
	}

	if value.Kind == yaml.SequenceNode {
		err := gabsifyInlineData(value)
		if err != nil {
			return nil, err
		}
	} else if value.Kind != yaml.MappingNode {
		return nil, errors.Errorf("node type must be a sequence or a mapping for core/ndarray-%s", ndaum.Version())
	} else {
		for i := 1; i < len(value.Content); i += 2 {
			node := value.Content[i]
			key := value.Content[i-1].Value
			if key == "datatype" {
				var exists bool
				arr.DataType, exists = basicMapping[node.Value]
				if !exists {
					log.Printf("unsupported dtype %s - falling back to uint8", node.Value)
					arr.DataType = basicMapping["uint8"]
				}
				continue
			}
			if key == "byteorder" {
				if node.Value == "little" {
					arr.ByteOrder = binary.LittleEndian
				} else if node.Value == "big" {
					arr.ByteOrder = binary.BigEndian
				} else {
					return nil, errors.Errorf("while parsing core/ndarray-%s: unknown byte order: %s",
						ndaum.Version(), node.Value)
				}
				continue
			}
			if key == "shape" {
				if node.Kind != yaml.SequenceNode {
					return nil, errors.Errorf("while parsing core/ndarray-%s: shape must be a sequence",
						ndaum.Version())
				}
				for j, sn := range node.Content {
					dim, err := strconv.Atoi(sn.Value)
					if err != nil {
						return nil, errors.Errorf("while parsing core/ndarray-%s: shape[%d] must be "+
							"an integer, got %s", ndaum.Version(), j, sn.Value)
					}
					arr.Shape = append(arr.Shape, dim)
				}
				continue
			}
			if key == "source" {
				src, err := strconv.Atoi(node.Value)
				if err != nil {
					return nil, errors.Errorf("while parsing core/ndarray-%s/source: external blocks "+
						"are not supported: %s", ndaum.Version(), node.Value)
				}
				binary.LittleEndian.PutUint32(arr.Data, uint32(src))
				continue
			}
			if key == "strides" {
				if node.Kind != yaml.SequenceNode {
					return nil, errors.Errorf("while parsing core/ndarray-%s: strides must be a sequence",
						ndaum.Version())
				}
				for j, sn := range node.Content {
					stride, err := strconv.Atoi(sn.Value)
					if err != nil {
						return nil, errors.Errorf("while parsing core/ndarray-%s: strides[%d] must be "+
							"an integer, got %s", ndaum.Version(), j, sn.Value)
					}
					if stride < 1 {
						return nil, errors.Errorf("while parsing core/ndarray-%s: strides[%d] must be "+
							"greater than 0, got %s", ndaum.Version(), j, sn.Value)
					}
					pos.Strides = append(pos.Strides, stride)
				}
				continue
			}
			if key == "offset" {
				var err error
				pos.Offset, err = strconv.Atoi(node.Value)
				if err != nil {
					return nil, errors.Wrapf(err, "while parsing core/ndarray-%s/source: offset "+
						"must be an integer", ndaum.Version())
				}
				if pos.Offset < 0 {
					return nil, errors.Errorf("while parsing core/ndarray-%s: offset may not be "+
						"negative (%d)", ndaum.Version(), pos.Offset)
				}
				continue
			}
			if key == "data" {
				err := gabsifyInlineData(node)
				if err != nil {
					return nil, err
				}
				continue
			}
			return nil, errors.Errorf("unknown property of core/ndarray-%s: %s",
				ndaum.Version(), key)
		}
		if pos.Strides != nil || pos.Offset != 0 {
			buffer := make([]byte, 4+4*len(pos.Strides))
			binary.LittleEndian.PutUint32(buffer, uint32(pos.Offset))
			for i, stride := range pos.Strides {
				binary.LittleEndian.PutUint32(buffer[4+i*4:], uint32(stride))
			}
		}
	}
	if inlineData != nil {
		err := applyInlineData(arr, inlineData)
		if err != nil {
			return nil, errors.Wrapf(err, "while parsing core/ndarray-%s/source: failed "+
				"to process the inline data", ndaum.Version())
		}
	}
	return arr, nil
}

func applyInlineData(arr *NDArray, data *gabs.Container) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("%v", r)
		}
	}()
	if len(data.Children()) == 0 {
		if arr.Shape != nil && (len(arr.Shape) != 1 || arr.Shape[0] != 1) {
			return errors.Errorf("overridden shape is incompatible with the inline data: %v", arr.Shape)
		}
		arr.Shape = []int{1}
		if arr.DataType == nil {
			arr.DataType = inferDataType(data.Data())
		}
		arr.Data = make([]byte, arr.ElementSize())
		elementToBytes(data.Data(), arr.DataType, arr.Data)
		return nil
	}
	elem := data
	for len(elem.Children()) > 0 {
		arr.Shape = append(arr.Shape, len(elem.Children()))
		elem, err = elem.ArrayElement(0)
		if err != nil {
			panic(err)
		}
	}
	if arr.DataType == nil {
		isFloat := false
		seq := []*gabs.Container{data}
		for len(seq) > 0 && !isFloat {
			slice := seq[len(seq)-1]
			seq = seq[:len(seq)-1]
			for _, subseq := range slice.Children() {
				if len(subseq.Children()) > 0 {
					seq = append(seq, subseq)
				} else if (inferDataType(subseq.Data()).Info() & types.IsFloat) != 0 {
					isFloat = true
					break
				}
			}
		}
		if isFloat {
			arr.DataType = types.Typ[types.Float64]
		} else {
			arr.DataType = types.Typ[types.Int]
		}
	}
	arr.Data = make([]byte, arr.CountBytes())
	offset := 0
	ds := arr.ElementSize()
	seq := []*gabs.Container{data}
	for len(seq) > 0 {
		slice := seq[len(seq)-1]
		seq = seq[:len(seq)-1]
		for i := range slice.Children() {
			subseq := slice.Children()[len(slice.Children())-i-1]
			if len(subseq.Children()) > 0 {
				seq = append(seq, subseq)
			} else {
				elementToBytes(subseq.Data(), arr.DataType, arr.Data[offset:offset+ds])
				offset += ds
				break
			}
		}
	}
	return nil
}

func inferDataType(elem interface{}) *types.Basic {
	switch elem.(type) {
	case int:
		return types.Typ[types.Int]
	case float64:
		return types.Typ[types.Float64]
	default:
		log.Panicf("unexpected array element type: %s: %v", reflect.TypeOf(elem), elem)
	}
	return nil
}

func elementToBytes(elem interface{}, dtype *types.Basic, out []byte) {
	inferDataType(elem)
	switch dtype.Kind() {
	case types.Int8:
	case types.Uint8:
		out[0] = byte(elem.(int))
	case types.Int16:
	case types.Uint16:
		hbo.PutUint16(out, uint16(elem.(int)))
	case types.Int32:
	case types.Uint32:
		hbo.PutUint32(out, uint32(elem.(int)))
	case types.Int64:
	case types.Uint64:
		hbo.PutUint64(out, uint64(elem.(int)))
	case types.Float32:
		if floatVal, ok := elem.(float64); ok {
			hbo.PutUint32(out, math.Float32bits(float32(floatVal)))
		} else {
			hbo.PutUint32(out, math.Float32bits(float32(elem.(int))))
		}
	case types.Float64:
		if floatVal, ok := elem.(float64); ok {
			hbo.PutUint64(out, math.Float64bits(floatVal))
		} else {
			hbo.PutUint64(out, math.Float64bits(float64(elem.(int))))
		}
	}
}

func init() {
	schema.Definitions["stsci.edu:asdf/core/ndarray"] = []schema.Definition{ndarrayUnmarshaler{}}
}
