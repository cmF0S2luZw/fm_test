// pkg/version/constraint.go
package version

import (
	"regexp"
	"strings"

	"pm/internal/errors"

	"github.com/Masterminds/semver/v3"
)

func Matches(versionStr, constraintStr string) (bool, error) {
	versionStr = strings.TrimSpace(versionStr)
	constraintStr = strings.TrimSpace(constraintStr)

	if constraintStr == "" {
		return true, nil
	}

	v, err := semver.NewVersion(versionStr)
	if err != nil {
		return false, errors.NewVersionError(versionStr, constraintStr, err)
	}

	constraint, err := ParseConstraint(constraintStr)
	if err != nil {
		return false, errors.NewVersionError(versionStr, constraintStr, err)
	}

	return constraint.Check(v), nil
}

func ParseConstraint(s string) (*semver.Constraints, error) {
	s = normalize(s)

	if s == "" {
		return nil, errors.NewVersionError("", "", nil)
	}

	constraint, err := semver.NewConstraint(s)
	if err != nil {
		return nil, err
	}

	return constraint, nil
}

func normalize(s string) string {
	s = strings.TrimSpace(s)
	s = regexp.MustCompile(`\s*([<>=!~^])\s*`).ReplaceAllString(s, "$1")

	if regexp.MustCompile(`^\d`).MatchString(s) {
		if !strings.HasPrefix(s, "=") &&
			!strings.HasPrefix(s, ">") &&
			!strings.HasPrefix(s, "<") &&
			!strings.HasPrefix(s, "~") &&
			!strings.HasPrefix(s, "^") {
			s = "=" + s
		}
	}

	return s
}
