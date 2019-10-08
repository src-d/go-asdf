package asdf

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenFile(t *testing.T) {
	req := require.New(t)
	asdfFile, err := OpenFile("testdata/default.asdf",
		func(done, total int) {})
	req.NoError(err)
	req.NotNil(asdfFile)
}

func TestFindBorderPositive(t *testing.T) {
	req := require.New(t)
	for _, mark := range borderMarks {
		for s := 0; s < 3; s++ {
			for d := 0; d < 12; d++ {
				for e := 0; e < 3; e++ {
					if s == 0 && d > 0 {
						continue
					}
					buf := &bytes.Buffer{}
					buf.Write(bytes.Repeat([]byte{0}, s*bufferSize-d))
					buf.Write(mark)
					buf.Write(bytes.Repeat([]byte{0}, bufferSize*e))
					pos, size, err := findBorder(bytes.NewReader(buf.Bytes()))
					req.NoErrorf(err, "s %d d %d e %d", s, d, e)
					req.Equalf(s*bufferSize-d, pos, "s %d d %d e %d", s, d, e)
					req.Equalf(len(mark), size, "s %d d %d e %d", s, d, e)
				}
			}
		}
	}
}

func TestFindBorderNegative(t *testing.T) {
	req := require.New(t)
	for x := 0; x < bufferSize*3; x += 1024 {
		buf := bytes.NewBuffer(bytes.Repeat([]byte{'.'}, x))
		pos, size, err := findBorder(bytes.NewReader(buf.Bytes()))
		req.NoErrorf(err, "x %d", x)
		req.Equal(-1, pos, "x %d", x)
		req.Equal(0, size, "x %d", x)
	}
}
