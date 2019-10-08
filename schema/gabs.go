package schema

import (
	"log"
	"strconv"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type hnode struct {
	Path []string
	Node *yaml.Node
}

// GabsifyYAML inserts the contents of a YAML node as another property of the JSON object, `key`.
func GabsifyYAML(container *gabs.Container, root *yaml.Node, key string) error {
	queue := []hnode{{[]string{key}, root}}
	for len(queue) > 0 {
		head := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if head.Node.Tag != "" && !strings.HasPrefix(head.Node.Tag, "!!") {
			tag, err := ParseTag(head.Node.Tag)
			if err != nil {
				return errors.Wrapf(err, "while converting %s", strings.Join(head.Path, "."))
			}
			def := FindDefinition(tag)
			if def == nil {
				log.Printf("unsupported tag at %s: %s", strings.Join(head.Path, "."),
					tag.String())
			} else {
				obj, err := def.UnmarshalYAML(head.Node)
				if err != nil {
					return errors.Wrapf(err, "while converting %s", strings.Join(head.Path, "."))
				}
				_, err = container.Set(obj, head.Path...)
				if err != nil {
					return errors.Wrapf(err, "while converting %s", strings.Join(head.Path, "."))
				}
				continue
			}
		}
		switch head.Node.Kind {
		case yaml.ScalarNode:
			narrowType := false
			if head.Node.Style == 0 {
				intval, err := strconv.Atoi(head.Node.Value)
				if err == nil {
					narrowType = true
					_, err = container.Set(intval, head.Path...)
					if err != nil {
						return errors.Wrapf(err, "while converting %s", strings.Join(head.Path, "."))
					}
				} else {
					boolval, err := strconv.ParseBool(head.Node.Value)
					if err == nil {
						narrowType = true
						_, err = container.Set(boolval, head.Path...)
						if err != nil {
							return errors.Wrapf(err, "while converting %s", strings.Join(head.Path, "."))
						}
					} else {
						floatval, err := strconv.ParseFloat(head.Node.Value, 64)
						if err == nil {
							narrowType = true
							_, err = container.Set(floatval, head.Path...)
							if err != nil {
								return errors.Wrapf(err, "while converting %s", strings.Join(head.Path, "."))
							}
						}
					}
				}
			}
			if !narrowType {
				_, err := container.Set(head.Node.Value, head.Path...)
				if err != nil {
					return errors.Wrapf(err, "while converting %s", strings.Join(head.Path, "."))
				}
			}
		case yaml.MappingNode:
			_, err := container.Object(head.Path...)
			if err != nil {
				return errors.Wrapf(err, "while converting %s", strings.Join(head.Path, "."))
			}
			for i := 1; i < len(head.Node.Content); i += 2 {
				node := head.Node.Content[i]
				key := head.Node.Content[i-1].Value
				path := make([]string, len(head.Path)+1)
				copy(path, head.Path)
				path[len(head.Path)] = key
				queue = append(queue, hnode{Path: path, Node: node})
			}
		case yaml.SequenceNode:
			_, err := container.ArrayOfSize(len(head.Node.Content), head.Path...)
			if err != nil {
				return errors.Wrapf(err, "while converting %s", strings.Join(head.Path, "."))
			}
			for i, node := range head.Node.Content {
				path := make([]string, len(head.Path)+1)
				copy(path, head.Path)
				path[len(head.Path)] = strconv.Itoa(i)
				queue = append(queue, hnode{Path: path, Node: node})
			}
		case yaml.AliasNode:
			queue = append(queue, hnode{Path: head.Path, Node: head.Node.Alias})
		}
	}
	return nil
}
