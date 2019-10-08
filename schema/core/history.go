package core

import (
	"github.com/blang/semver"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/src-d/go-asdf/schema"
)

// History is defined in https://asdf-standard.readthedocs.io/en/latest/generated/stsci.edu/asdf/core/asdf-1.1.0.html#history
type History struct {
	// Extensions is the list of transformations applied to the document.
	Extensions []*ExtensionMetadata
	// File change history.
	Entries []*HistoryEntry
}

type historySequenceUnmarshaler struct {
}

func (hsum historySequenceUnmarshaler) Version() semver.Version {
	return semver.MustParse("1.1.0")
}

func (hsum historySequenceUnmarshaler) UnmarshalYAML(value *yaml.Node) (interface{}, error) {
	history := &History{}
	if value.Kind != yaml.SequenceNode {
		return nil, errors.Errorf("tag core/history-%s requires a sequence node", hsum.Version())
	}
	tag, err := schema.ParseTag(value.Content[0].Tag)
	if err != nil {
		return nil, err
	}
	def := schema.FindDefinition(tag)
	if def == nil {
		return nil, errors.Errorf("unsupported tag: %s", tag.String())
	}
	for i, node := range value.Content {
		obj, err := def.UnmarshalYAML(node)
		if err != nil {
			return nil, errors.Wrapf(err, "while parsing %s/%s[%d]", value.Tag, tag.String(), i)
		}
		history.Entries = append(history.Entries, obj.(*HistoryEntry))
	}
	return history, nil
}

type historyMappingUnmarshaler struct {
}

func (hmum historyMappingUnmarshaler) Version() semver.Version {
	return semver.MustParse("1.1.0")
}

func (hmum historyMappingUnmarshaler) UnmarshalYAML(value *yaml.Node) (interface{}, error) {
	history := &History{}
	if value.Kind != yaml.MappingNode {
		return nil, errors.Errorf("tag core/history-%s requires a mapping node", hmum.Version())
	}
	for i := 1; i < len(value.Content); i += 2 {
		node := value.Content[i]
		key := value.Content[i-1].Value
		if (key != "extensions" && key != "entries") || node.Kind != yaml.SequenceNode ||
			len(node.Content) == 0 || node.Content[0].Tag == "" {
			return nil, errors.Errorf("invalid key in a core/history-%s element: %s",
				hmum.Version(), key)
		}
		if node.Kind != yaml.SequenceNode {
			return nil, errors.Errorf("invalid node type at core/history-%s/%s",
				hmum.Version(), key)
		}
		tag, err := schema.ParseTag(node.Content[0].Tag)
		if err != nil {
			return nil, err
		}
		def := schema.FindDefinition(tag)
		if def == nil {
			return nil, errors.Errorf("unsupported tag: %s", tag.String())
		}
		for j, sub := range node.Content {
			obj, err := def.UnmarshalYAML(sub)
			if err != nil {
				return nil, errors.Wrapf(err, "while parsing core/history-%s/%s[%d]",
					hmum.Version(), key, j)
			}
			if key == "extensions" {
				history.Extensions = append(history.Extensions, obj.(*ExtensionMetadata))
			} else {
				history.Entries = append(history.Entries, obj.(*HistoryEntry))
			}
		}
	}
	return history, nil
}

func init() {
	schema.Definitions["stsci.edu:asdf/core/history/sequence"] =
		[]schema.Definition{historySequenceUnmarshaler{}}
	schema.Definitions["stsci.edu:asdf/core/history/mapping"] =
		[]schema.Definition{historyMappingUnmarshaler{}}
}
