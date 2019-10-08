package core

import (
	"github.com/blang/semver"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/src-d/go-asdf/schema"
)

// ExtensionMetadata is defined in https://asdf-standard.readthedocs.io/en/latest/generated/stsci.edu/asdf/core/extension_metadata-1.0.0.html
type ExtensionMetadata struct {
	// Class is the value of `extension_class` property.
	Class string
	// Package indicates the name and the version of the extension.
	Package schema.Tag
}

type extensionMetadataUnmarshaler struct {
}

func (emum extensionMetadataUnmarshaler) Version() semver.Version {
	return semver.MustParse("1.0.0")
}

func (emum extensionMetadataUnmarshaler) UnmarshalYAML(value *yaml.Node) (interface{}, error) {
	if value.Kind != yaml.MappingNode {
		return nil, errors.Errorf("node type must be a mpping for core/extension_metadata-%s",
			emum.Version())
	}
	em := &ExtensionMetadata{}
	for i := 1; i < len(value.Content); i += 2 {
		node := value.Content[i]
		key := value.Content[i-1].Value
		if key == "extension_class" {
			em.Class = node.Value
		} else if key == "software" {
			// TODO(vmarkovtsev): https://github.com/spacetelescope/asdf/issues/709
			for j := 1; j < len(node.Content); j += 2 {
				text := node.Content[j].Value
				prop := node.Content[j-1].Value
				var err error
				if prop == "name" {
					em.Package.Name = text
				} else if prop == "version" {
					em.Package.Version, err = semver.Parse(text)
				}
				if err != nil {
					return nil, errors.Wrapf(err, "while parsing %s", value.Tag)
				}
			}
		} else {
			return nil, errors.Errorf("invalid key in a core/extension_metadata-%s element: %s",
				emum.Version(), key)
		}
	}
	return em, nil
}

func init() {
	schema.Definitions["stsci.edu:asdf/core/extension_metadata"] = []schema.Definition{extensionMetadataUnmarshaler{}}
}
