package core

import (
	"encoding/binary"
	"fmt"
	"go/types"
	"log"
	"strconv"
	"strings"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/src-d/go-asdf/schema"
)

// NDArray is defined in https://asdf-standard.readthedocs.io/en/latest/generated/stsci.edu/asdf/core/ndarray-1.0.0.html
// It is similar to `numpy.ndarray` in Python.
type NDArray struct {
	// DataType is the array element type.
	DataType *types.Basic
	// Shape is the array shape: a one-dimensional integer sequence.
	Shape []int
	// ByteOrder is the byte order if the array contains integers.
	ByteOrder binary.ByteOrder
	// Data is the raw array buffer, similar to `numpy.ndarray.data`.
	Data []byte
}

func (arr NDArray) String() string {
	dims := make([]string, 0, len(arr.Shape))
	for _, s := range arr.Shape {
		dims = append(dims, strconv.Itoa(s))
	}
	return fmt.Sprintf("array<%s, %s> of shape [%s]", arr.DataType.String(),
		arr.ByteOrder.String(), strings.Join(dims, ", "))
}

var basicMapping = map[string]*types.Basic{
	"int8": types.Typ[types.Int8],
	"int16": types.Typ[types.Int16],
	"int32": types.Typ[types.Int32],
	"int64": types.Typ[types.Int64],
	"uint8": types.Typ[types.Uint8],
	"uint16": types.Typ[types.Uint16],
	"uint32": types.Typ[types.Uint32],
	"uint64": types.Typ[types.Uint64],
	"float32": types.Typ[types.Float32],
	"float64": types.Typ[types.Float64],
	"complex64": types.Typ[types.Complex64],
	"complex128": types.Typ[types.Complex128],
}

type ndarrayUnmarshaler struct {
}

func (ndaum ndarrayUnmarshaler) Version() semver.Version {
	return semver.MustParse("1.0.0")
}

func (ndaum ndarrayUnmarshaler) UnmarshalYAML(value *yaml.Node) (interface{}, error) {
	if value.Kind != yaml.MappingNode {
		return nil, errors.Errorf("node type must be a mapping for core/ndarray-%s", ndaum.Version())
	}
	arr := &NDArray{Data: make([]byte, 4)}
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
					return nil, errors.Errorf("while parsing core/ndarray-%s: shape[%d] must be " +
						"an integer, got %s", ndaum.Version(), j, sn.Value)
				}
				arr.Shape = append(arr.Shape, dim)
			}
			continue
		}
		if key == "source" {
			src, err := strconv.Atoi(node.Value)
			if err != nil {
				return nil, errors.Wrapf(err, "while parsing core/ndarray-%s/source", ndaum.Version())
			}
			binary.LittleEndian.PutUint32(arr.Data, uint32(src))
			continue
		}
		return nil, errors.Errorf("unknown property of core/ndarray-%s: %s",
			ndaum.Version(), key)
	}
	return arr, nil
}

func init() {
	schema.Definitions["stsci.edu:asdf/core/ndarray"] = []schema.Definition{ndarrayUnmarshaler{}}
}