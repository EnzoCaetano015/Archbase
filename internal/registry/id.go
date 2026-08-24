package registry

import (
	"fmt"
	"regexp"
)

var patternIDExpression = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._-]*@[a-z0-9][a-z0-9._-]*$`)

type PatternID string

func ParsePatternID(value string) (PatternID, error) {
	if !patternIDExpression.MatchString(value) {
		return "", fmt.Errorf("invalid pattern ID %q: expected stack/type@id using lowercase characters", value)
	}
	return PatternID(value), nil
}

func (id PatternID) String() string { return string(id) }
