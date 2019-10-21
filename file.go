package asdf

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"strings"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"golang.org/x/exp/mmap"
	"gopkg.in/yaml.v3"

	"github.com/src-d/go-asdf/schema"
	"github.com/src-d/go-asdf/schema/core"
)

// File is the top level, ASDF file type.
type File struct {
	core.Document
	// FormatVersion corresponds to the contents of #ASDF header comment.
	FormatVersion semver.Version
	// FormatVersion corresponds to the contents of #ASDF_STANDARD header comment.
	StandardVersion semver.Version
}

// ProgressCallback allows tracking the file loading progress. Both done *and* total will grow dynamically.
type ProgressCallback func(done, total int)

var borderMarks = [][]byte{
	append([]byte{'.', '.', '.', '\n'}, blockMagic[:]...),
	append([]byte{'.', '.', '.', '\r', '\n'}, blockMagic[:]...),
}

// OpenFile reads ASDF from the file system.
func OpenFile(fileName string, progress ProgressCallback) (*File, error) {
	reader, err := mmap.Open(fileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open %s", fileName)
	}
	defer reader.Close()
	return Open(io.NewSectionReader(reader, 0, int64(reader.Len())), progress)
}

// Open reads ASDF from a seekable reader.
func Open(reader io.ReadSeeker, progress ProgressCallback) (*File, error) {
	file := &File{}
	if progress == nil {
		progress = func(_, _ int) {}
	}
	progress(0, 2)
	var err error
	file.FormatVersion, file.StandardVersion, err = parseHeader(reader)
	if err != nil {
		return nil, err
	}
	tree, blockOffset, err := parseTree(reader)
	progress(1, 2)
	if err != nil {
		return nil, err
	}
	tag, err := schema.ParseTag(tree.Tag)
	if err != nil {
		return nil, errors.Errorf("invalid top level tag: %v", err)
	}
	def := schema.FindDefinition(tag)
	if def == nil {
		return nil, errors.Errorf("unknown top level tag: %s", tree.Tag)
	}
	doc, err := def.UnmarshalYAML(tree)
	if err != nil {
		return nil, err
	}
	file.Document = *doc.(*core.Document)
	progress(2, 2)
	if _, err = reader.Seek(int64(blockOffset), io.SeekStart); err != nil {
		return nil, err
	}
	err = readAndResolveBlocks(&file.Document, reader, progress)
	return file, err
}

func readAndResolveBlocks(doc *core.Document, reader io.Reader, progress ProgressCallback) error {
	arrays := map[int][]*core.NDArray{}
	maxIndex := -1
	doc.IterArrays(func(arr *core.NDArray) {
		index := int(binary.LittleEndian.Uint32(arr.Data))
		if index > maxIndex {
			maxIndex = index
		}
		arrays[index] = append(arrays[index], arr)
	})
	progress(2, maxIndex+2)
	steps := 2
	for i := 0; i < maxIndex; i++ {
		block, err := ReadBlock(reader)
		if err != nil {
			return errors.Wrapf(err, "reading block #%d", i)
		}
		blockArrays, exist := arrays[i]
		if !exist {
			// Orphaned block which is not used by any array
			continue
		}
		err = block.Uncompress()
		if err != nil {
			return errors.Wrapf(err, "uncompressing block #%d", i)
		}
		for _, arr := range blockArrays {
			if len(arr.Data) == 4 {
				arr.Data = block.Data
				continue
			}
			// There is a position
			offset := int(binary.LittleEndian.Uint32(arr.Data[4:8]))
			strides := make([]int, (len(arr.Data)-8)/4)
			if len(strides) == 0 {
				arr.Data = block.Data[offset : offset+arr.CountBytes()]
				continue
			}
			// Construct the new contiguous array from scratch
			size := 1
			for j := range strides {
				stride := int(binary.LittleEndian.Uint32(arr.Data[8+j*4 : 8+(j+1)*4]))
				strides[j] = stride
				size *= stride
			}
			arr.Data = make([]byte, arr.CountBytes())
			chunkSize := arr.Shape[len(arr.Shape)-1] * arr.ElementSize()
			if len(arr.Shape) == 1 {
				copy(arr.Data, block.Data[offset:offset+chunkSize])
				continue
			}
			lastStride := strides[len(strides)-1] * arr.ElementSize()
			balance := make([]int, len(arr.Shape)-1)
			offsets := make([]int, len(arr.Shape)-1)
			for j := 0; j < len(arr.Shape)-1; j++ {
				balance[j] = arr.Shape[j]
				offsets[j] = offset
			}
			arrOffset := 0
			for focus := 0; focus >= 0; {
				copy(arr.Data[arrOffset:arrOffset+chunkSize], block.Data[offset:offset+chunkSize])
				arrOffset += chunkSize
				offset += lastStride
				for focus = len(arr.Shape) - 2; focus >= 0; {
					balance[focus]--
					if balance[focus] == 0 {
						balance[focus] = arr.Shape[focus]
						offset = offsets[focus] + strides[focus]
						for dim := focus; dim < len(arr.Shape)-1; dim++ {
							offsets[dim] = offset
						}
						focus--
					} else {
						break
					}
				}
			}
		}
		progress(steps+i+1, len(arrays)+2)
	}
	return nil
}

func parseTree(reader io.ReadSeeker) (*yaml.Node, int, error) {
	border, borderLen, err := findBorder(reader)
	if err != nil {
		return nil, 0, err
	}
	var yamlReader io.Reader = reader
	if border >= 0 {
		buffer := make([]byte, border+3)
		_, err = io.ReadFull(reader, buffer)
		if err != nil {
			return nil, 0, err
		}
		yamlReader = bytes.NewBuffer(buffer)
		// Position the reader at the beginning of the first block
		_, err = reader.Seek(int64(borderLen-3-len(blockMagic)), io.SeekCurrent)
		if err != nil {
			return nil, 0, err
		}
	}
	decoder := yaml.NewDecoder(yamlReader)
	doc := yaml.Node{}
	err = decoder.Decode(&doc)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to decode YAML")
	}
	if border < 0 {
		// This will indicate that there are no blocks
		_, err = reader.Seek(0, io.SeekEnd)
		if err != nil {
			return nil, 0, err
		}
	}
	if len(doc.Content) != 1 {
		return nil, 0, errors.New(
			"invalid format: the document must contain exactly one root element")
	}
	tree := doc.Content[0]
	return tree, border + borderLen - len(blockMagic), nil
}

func findBorder(reader io.ReadSeeker) (int, int, error) {
	for _, mark := range borderMarks {
		borderPos, err := Grep(reader, mark)
		if err != nil {
			return -1, 0, errors.Wrap(err, "while searching for the first binary block")
		}
		_, err = reader.Seek(0, io.SeekStart)
		if err != nil {
			return -1, 0, errors.Wrap(err, "while searching for the first binary block")
		}
		if borderPos >= 0 {
			return borderPos, len(mark), nil
		}
	}
	return -1, 0, nil
}

func parseHeader(reader io.ReadSeeker) (semver.Version, semver.Version, error) {
	dummy := semver.Version{}
	scanner := bufio.NewScanner(reader)
	scanner.Scan()
	err := scanner.Err()
	if err != nil {
		return dummy, dummy, errors.Wrap(err, "failed to read the file header")
	}
	header := scanner.Text()
	if !strings.HasPrefix(header, "#ASDF ") {
		return dummy, dummy, errors.Errorf("invalid ASDF file header, the first line must start "+
			"with \"#ASDF \": %s", header)
	}
	formatVersion, err := semver.Make(header[6:])
	if err != nil {
		return dummy, dummy, errors.Errorf("invalid ASDF file header, cannot parse semver from "+
			"\"%s\"", header[6:])
	}
	scanner.Scan()
	err = scanner.Err()
	if err != nil {
		return dummy, dummy, errors.Wrap(err, "failed to read the file header")
	}
	header = scanner.Text()
	if !strings.HasPrefix(header, "#ASDF_STANDARD ") {
		return dummy, dummy, errors.Errorf("invalid ASDF file header, the second line must start "+
			"with \"#ASDF_STANDARD \": %s", header)
	}
	standardVersion, err := semver.Make(header[15:])
	if err != nil {
		return dummy, dummy, errors.Errorf("invalid ASDF file header, cannot parse semver from "+
			"\"%s\"", header[15:])
	}
	_, err = reader.Seek(0, io.SeekStart)
	return formatVersion, standardVersion, err
}
