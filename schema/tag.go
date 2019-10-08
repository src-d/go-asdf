package schema

import (
	"strings"

	"github.com/blang/semver"
	"github.com/pkg/errors"
)

// Tag represents a versioned entity such as an ASDF tag.
type Tag struct {
	Name    string
	Version semver.Version
}

// ParseTag parses the ASDF tag from a string.
func ParseTag(str string) (Tag, error) {
	dashPos := strings.IndexRune(str, '-')
	if dashPos < 0 {
		return Tag{}, errors.Errorf("cannot parse tag: \"%s\": no version separator (dash)",
			str)
	}
	name := str[:dashPos]
	if strings.HasPrefix(name, "tag:") {
		name = name[4:]
	}
	version, err := semver.Make(str[dashPos+1:])
	if err != nil {
		return Tag{}, err
	}
	return Tag{name, version}, nil
}

func (tag Tag) String() string {
	return tag.Name + "-" + tag.Version.String()
}
