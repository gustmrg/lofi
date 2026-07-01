package version

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"
)

//go:embed VERSION
var embeddedVersion string

// BuiltVersion may be set by release builds with:
// -ldflags "-X github.com/gustmrg/lofi.BuiltVersion=X.Y.Z"
var BuiltVersion string

func Current() string {
	if BuiltVersion != "" {
		return Normalize(BuiltVersion)
	}
	return Normalize(embeddedVersion)
}

func Display(v string) string {
	return "LoFi " + Tag(v)
}

func Tag(v string) string {
	return "v" + Normalize(v)
}

func Normalize(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	return v
}

func Compare(a, b string) (int, error) {
	va, err := parse(a)
	if err != nil {
		return 0, err
	}
	vb, err := parse(b)
	if err != nil {
		return 0, err
	}
	for i := range va.nums {
		if va.nums[i] > vb.nums[i] {
			return 1, nil
		}
		if va.nums[i] < vb.nums[i] {
			return -1, nil
		}
	}
	return comparePrerelease(va.pre, vb.pre), nil
}

type semver struct {
	nums [3]int
	pre  string
}

func parse(v string) (semver, error) {
	v = Normalize(v)
	pre := ""
	if i := strings.IndexByte(v, '-'); i >= 0 {
		pre = v[i+1:]
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return semver{}, fmt.Errorf("invalid semver %q", v)
	}
	var nums [3]int
	for i, part := range parts {
		if part == "" {
			return semver{}, fmt.Errorf("invalid semver %q", v)
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return semver{}, fmt.Errorf("invalid semver %q", v)
		}
		nums[i] = n
	}
	return semver{nums: nums, pre: pre}, nil
}

func comparePrerelease(a, b string) int {
	if a == "" && b == "" {
		return 0
	}
	if a == "" {
		return 1
	}
	if b == "" {
		return -1
	}
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	for i := 0; i < len(ap) && i < len(bp); i++ {
		ai, aerr := strconv.Atoi(ap[i])
		bi, berr := strconv.Atoi(bp[i])
		switch {
		case aerr == nil && berr == nil:
			if ai > bi {
				return 1
			}
			if ai < bi {
				return -1
			}
		case aerr == nil:
			return -1
		case berr == nil:
			return 1
		default:
			if ap[i] > bp[i] {
				return 1
			}
			if ap[i] < bp[i] {
				return -1
			}
		}
	}
	if len(ap) > len(bp) {
		return 1
	}
	if len(ap) < len(bp) {
		return -1
	}
	return 0
}
