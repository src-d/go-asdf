package core

import (
	"github.com/Jeffail/gabs/v2"
	"github.com/blang/semver"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/src-d/go-asdf/schema"
)

type Document struct {
	Library *Software
	History *History
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
