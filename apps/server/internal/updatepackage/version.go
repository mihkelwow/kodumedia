package updatepackage

import (
	"fmt"
	"strconv"
	"strings"
)

type semanticVersion struct {
	major, minor, patch int
	pre                 []string
}

func CompareVersions(left, right string) (int, error) {
	leftVersion, err := parseSemanticVersion(left)
	if err != nil {
		return 0, err
	}
	rightVersion, err := parseSemanticVersion(right)
	if err != nil {
		return 0, err
	}
	for _, pair := range [][2]int{{leftVersion.major, rightVersion.major}, {leftVersion.minor, rightVersion.minor}, {leftVersion.patch, rightVersion.patch}} {
		if pair[0] < pair[1] {
			return -1, nil
		}
		if pair[0] > pair[1] {
			return 1, nil
		}
	}
	return comparePrerelease(leftVersion.pre, rightVersion.pre), nil
}

func parseSemanticVersion(value string) (semanticVersion, error) {
	value = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "v"))
	if value == "" {
		return semanticVersion{}, fmt.Errorf("empty semantic version")
	}
	value = strings.SplitN(value, "+", 2)[0]
	parts := strings.SplitN(value, "-", 2)
	core := strings.Split(parts[0], ".")
	if len(core) != 3 {
		return semanticVersion{}, fmt.Errorf("%q is not semantic version major.minor.patch", value)
	}
	numbers := make([]int, 3)
	for index, part := range core {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return semanticVersion{}, fmt.Errorf("invalid semantic version component %q", part)
		}
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return semanticVersion{}, fmt.Errorf("invalid semantic version component %q", part)
		}
		numbers[index] = number
	}
	version := semanticVersion{major: numbers[0], minor: numbers[1], patch: numbers[2]}
	if len(parts) == 2 {
		if parts[1] == "" {
			return semanticVersion{}, fmt.Errorf("empty prerelease")
		}
		version.pre = strings.Split(parts[1], ".")
		for _, identifier := range version.pre {
			if identifier == "" {
				return semanticVersion{}, fmt.Errorf("empty prerelease identifier")
			}
			for _, character := range identifier {
				if !(character >= '0' && character <= '9') && !(character >= 'A' && character <= 'Z') && !(character >= 'a' && character <= 'z') && character != '-' {
					return semanticVersion{}, fmt.Errorf("invalid prerelease identifier %q", identifier)
				}
			}
		}
	}
	return version, nil
}

func comparePrerelease(left, right []string) int {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}
	if len(left) == 0 {
		return 1
	}
	if len(right) == 0 {
		return -1
	}
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for index := 0; index < limit; index++ {
		leftNumber, leftErr := strconv.Atoi(left[index])
		rightNumber, rightErr := strconv.Atoi(right[index])
		switch {
		case leftErr == nil && rightErr == nil:
			if leftNumber < rightNumber {
				return -1
			}
			if leftNumber > rightNumber {
				return 1
			}
		case leftErr == nil:
			return -1
		case rightErr == nil:
			return 1
		default:
			if left[index] < right[index] {
				return -1
			}
			if left[index] > right[index] {
				return 1
			}
		}
	}
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	return 0
}
