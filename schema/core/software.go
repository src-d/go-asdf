package core

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/src-d/go-asdf/schema"
)

// Software is the information about a library or a tool.
type Software struct {
	// Name and the version.
	schema.Tag
	// Author is the author of the software.
	Author string
	// HomePage is the URL of the software.
	HomePage string
}

// Strings formats the object as a string.
func (s Software) String() string {
	return fmt.Sprintf("%s-%s [%s](%s)", s.Name, s.Version, s.Author, s.HomePage)
}

type softwareUnmarshaler struct {
}

func (lum softwareUnmarshaler) Version() semver.Version {
	return semver.MustParse("1.0.0")
}

func (lum softwareUnmarshaler) UnmarshalYAML(value *yaml.Node) (interface{}, error) {
	if value.Kind != yaml.MappingNode {
		return nil, errors.Errorf("tag core/software-%s requires a mapping node", lum.Version())
	}
	lib := &Software{}
	for i := 1; i < len(value.Content); i += 2 {
		text := value.Content[i].Value
		key := value.Content[i-1].Value
		var err error
		if key == "author" {
			lib.Author = text
		} else if key == "homepage" {
			lib.HomePage = text
		} else if key == "name" {
			lib.Name = text
		} else if key == "version" {
			lib.Version, err = semver.Parse(text)
		} else {
			err = errors.Errorf("invalid key in a core/software-%s element: %s",
				lum.Version(), key)
		}
		if err != nil {
			return nil, err
		}
	}
	return lib, nil
}

func init() {
	schema.Definitions["stsci.edu:asdf/core/software"] = []schema.Definition{softwareUnmarshaler{}}
}
