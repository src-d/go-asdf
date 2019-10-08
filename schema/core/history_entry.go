package core

import (
	"github.com/blang/semver"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/src-d/go-asdf/schema"
)

// HistoryEntry is defined in https://asdf-standard.readthedocs.io/en/latest/generated/stsci.edu/asdf/core/history_entry-1.0.0.html
type HistoryEntry struct {
	// Description is the description of the transformation performed.
	Description string
	// Time is the timestamp of the transformation, in UTC.
	Time string
	// Software is the list of https://asdf-standard.readthedocs.io/en/latest/generated/stsci.edu/asdf/core/software-1.0.0.html
	// It may contain one or more elements.
	Software []*Software
}

type historyEntryUnmarshaler struct {
}

func (heum historyEntryUnmarshaler) Version() semver.Version {
	return semver.MustParse("1.0.0")
}

func (heum historyEntryUnmarshaler) UnmarshalYAML(value *yaml.Node) (interface{}, error) {
	if value.Kind != yaml.MappingNode {
		return nil, errors.Errorf("node type must be a mpping for core/history_entry-%s",
			heum.Version())
	}
	he := &HistoryEntry{}
	for i := 1; i < len(value.Content); i += 2 {
		node := value.Content[i]
		key := value.Content[i-1].Value
		if key == "description" {
			he.Description = node.Value
		} else if key == "time" {
			he.Time = node.Value
		} else if key == "software" {
			children := node.Content
			if node.Kind != yaml.SequenceNode {
				children = []*yaml.Node{node}
			}
			for _, child := range children {
				tag, err := schema.ParseTag(child.Tag)
				if err != nil {
					return nil, err
				}
				def := schema.FindDefinition(tag)
				if def == nil {
					return nil, errors.Errorf("unsupported tag: %s", tag.String())
				}
				sw, err := def.UnmarshalYAML(child)
				if err != nil {
					return nil, errors.Wrapf(err, "while parsing core/history_entry-%s/software",
						heum.Version())
				}
				he.Software = append(he.Software, sw.(*Software))
			}
		} else {
			return nil, errors.Errorf("invalid key in a core/history_entry-%s: %s",
				heum.Version(), key)
		}
	}
	return he, nil
}

func init() {
	schema.Definitions["stsci.edu:asdf/core/history_entry"] = []schema.Definition{historyEntryUnmarshaler{}}
}