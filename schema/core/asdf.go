package core

import (
	"github.com/Jeffail/gabs/v2"
	"github.com/blang/semver"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/src-d/go-asdf/schema"
)

// Document carries the ASDF object tree with its metadata.
type Document struct {
	// Library is the information about the software library used to create the ASDF file.
	// See https://asdf-standard.readthedocs.io/en/latest/generated/stsci.edu/asdf/core/software-1.0.0.html
	Library *Software
	// History is the list of extensions used to create the ASDF file.
	// See https://asdf-standard.readthedocs.io/en/latest/generated/stsci.edu/asdf/core/asdf-1.1.0.html#history
	History *History
	// Tree is the contents of the ASDF file. It is mapped to a JSON object model.
	// Please refer to https://godoc.org/github.com/Jeffail/gabs
	Tree *gabs.Container
}

type documentUnmarshaler struct {
}

func (du documentUnmarshaler) Version() semver.Version {
	return semver.MustParse("1.1.0")
}

func (du documentUnmarshaler) UnmarshalYAML(value *yaml.Node) (interface{}, error) {
	doc := &Document{Tree: gabs.New()}
	if value.Kind != yaml.MappingNode {
		return nil, errors.Errorf("tag core/asdf-%s requires a mapping node", du.Version())
	}
	for i := 1; i < len(value.Content); i += 2 {
		node := value.Content[i]
		key := value.Content[i-1].Value
		if key == "asdf_library" {
			tag, err := schema.ParseTag(node.Tag)
			if err != nil {
				return nil, err
			}
			def := schema.FindDefinition(tag)
			if def == nil {
				return nil, errors.Errorf("unsupported tag: %s", tag.String())
			}
			obj, err := def.UnmarshalYAML(node)
			if err != nil {
				return nil, errors.Wrapf(err, "while parsing core/asdf-%s/%s", du.Version(), key)
			}
			doc.Library = obj.(*Software)
		} else if key == "history" {
			var err error
			var obj interface{}
			switch node.Kind {
			case yaml.SequenceNode:
				um := schema.FindDefinition(schema.Tag{
					Name: "stsci.edu:asdf/core/history/sequence",
					Version: semver.MustParse("1.1.0")})
				obj, err = um.UnmarshalYAML(node)
			case yaml.MappingNode:
				um := schema.FindDefinition(schema.Tag{
					Name: "stsci.edu:asdf/core/history/mapping",
					Version: semver.MustParse("1.1.0")})
				obj, err = um.UnmarshalYAML(node)
			default:
				err = errors.Errorf("invalid history value type: %d", node.Kind)
			}
			if err != nil {
				return nil, err
			}
			doc.History = obj.(*History)
		} else {
			err := schema.GabsifyYAML(doc.Tree, node, key)
			if err != nil {
				return nil, errors.Wrapf(err, "while transforming core/asdf-%s", du.Version())
			}
			continue
		}
	}
	return doc, nil
}

// IterArrays visits all the contained ndarray-s in the document.
func (doc Document) IterArrays(visitor func(array *NDArray)) {
	queue := []*gabs.Container{doc.Tree}
	for len(queue) > 0 {
		head := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		arr, ok := head.Data().(*NDArray)
		if ok {
			visitor(arr)
		}
		for _, child := range head.Children() {
			queue = append(queue, child)
		}
	}
}

func init() {
	schema.Definitions["stsci.edu:asdf/core/asdf"] = []schema.Definition{documentUnmarshaler{}}
}
