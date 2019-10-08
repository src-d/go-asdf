package schema

import (
	"sort"

	"github.com/blang/semver"
	"gopkg.in/yaml.v3"
)

type Definition interface {
	Version() semver.Version
	UnmarshalYAML(value *yaml.Node) (interface{}, error)
}

var Definitions = map[string][]Definition{}

// FindDefinition returns the schema definition for the given tag, or nil if no such definition exist.
func FindDefinition(tag Tag) Definition {
	defs, exist := Definitions[tag.Name]
	if !exist {
		return nil
	}
	x := sort.Search(len(defs), func(i int) bool {
		return defs[i].Version().GTE(tag.Version)
	})
	if x == len(defs) {
		return nil
	}
	return defs[x]
}
