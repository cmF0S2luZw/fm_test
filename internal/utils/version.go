package utils

import (
	"regexp"
	"strings"
)

func ExtractVersion(filename string) string {
	name := strings.TrimSuffix(filename, ".zip")
	name = strings.TrimSuffix(name, ".tar.gz")
	name = strings.TrimSuffix(name, ".tgz")

	re := regexp.MustCompile(`[-_v]?(\d+\.\d+(?:\.\d+)?(?:-[a-zA-Z0-9]+)?)$`)
	matches := re.FindStringSubmatch(name)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}
