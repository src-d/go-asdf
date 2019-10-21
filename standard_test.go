package asdf

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStandardAscii(t *testing.T) {
	req := require.New(t)
	asdfFile, err := OpenFile("testdata/standard/ascii.asdf", nil)
	req.NoError(err)
	req.NotNil(asdfFile)
}

func TestStandardBasic(t *testing.T) {
	req := require.New(t)
	asdfFile, err := OpenFile("testdata/standard/basic.asdf", nil)
	req.NoError(err)
	req.NotNil(asdfFile)
}

func TestStandardComplex(t *testing.T) {
	req := require.New(t)
	asdfFile, err := OpenFile("testdata/standard/complex.asdf", nil)
	req.NoError(err)
	req.NotNil(asdfFile)
}

func TestStandardCompressed(t *testing.T) {
	req := require.New(t)
	asdfFile, err := OpenFile("testdata/standard/compressed.asdf", nil)
	req.NoError(err)
	req.NotNil(asdfFile)
}

func TestStandardExploded(t *testing.T) {
	req := require.New(t)
	asdfFile, err := OpenFile("testdata/standard/exploded.asdf", nil)
	req.Error(err)
	req.Nil(asdfFile)
}

func TestStandardFloat(t *testing.T) {
	req := require.New(t)
	asdfFile, err := OpenFile("testdata/standard/float.asdf", nil)
	req.NoError(err)
	req.NotNil(asdfFile)
}

func TestStandardInt(t *testing.T) {
	req := require.New(t)
	asdfFile, err := OpenFile("testdata/standard/int.asdf", nil)
	req.NoError(err)
	req.NotNil(asdfFile)
}

func TestStandardShared(t *testing.T) {
	req := require.New(t)
	asdfFile, err := OpenFile("testdata/standard/shared.asdf", nil)
	req.NoError(err)
	req.NotNil(asdfFile)
}

func TestStandardUnicodeBmp(t *testing.T) {
	req := require.New(t)
	asdfFile, err := OpenFile("testdata/standard/unicode_bmp.asdf", nil)
	req.NoError(err)
	req.NotNil(asdfFile)
}

func TestStandardUnicodeSpp(t *testing.T) {
	req := require.New(t)
	asdfFile, err := OpenFile("testdata/standard/unicode_spp.asdf", nil)
	req.NoError(err)
	req.NotNil(asdfFile)
}
