# go-asdf [![GoDoc](https://godoc.org/github.com/src-d/go-asdf?status.svg)](http://godoc.org/github.com/src-d/go-asdf) [![Build Status](https://travis-ci.com/src-d/go-asdf.svg?branch=master)](https://travis-ci.com/src-d/go-asdf) [![codecov](https://codecov.io/github/src-d/go-asdf/coverage.svg)](https://codecov.io/gh/src-d/go-asdf) [![Go Report Card](https://goreportcard.com/badge/github.com/src-d/go-asdf)](https://goreportcard.com/report/github.com/src-d/go-asdf) [![Apache 2.0 license](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

[Advanced Scientific Data Format](https://github.com/spacetelescope/asdf-standard) reader library in pure Go.

The blocks are eagerly read and uncompressed. The tree is mapped with [gabs](https://github.com/Jeffail/gabs).

### Usage

```go
import "github.com/src-d/go-asdf"

fmt.Println(asdf.OpenFile("path/to/file.asdf", nil).Tree)
```

### Contributions

...are welcome, see [CONTRIBUTING](CONTRIBUTING.md) and [code of conduct](CODE_OF_CONDUCT.md).

### License

Apache 2.0, see [LICENSE](LICENSE).