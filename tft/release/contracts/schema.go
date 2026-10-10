package contracts

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// schemaRangePattern 支持 "^MAJOR.MINOR[.PATCH]"（锁定 major）
// 与 ">=MAJOR.MINOR[.PATCH],<MAJOR.MINOR[.PATCH]"（显式范围）。
var (
	caretRangePattern  = regexp.MustCompile(`^\^(\d+)(?:\.(\d+))?(?:\.(\d+))?$`)
	explicitRangeParts = 2
)

// SupportsSchema 判断 reader 范围声明（reader_schema_range）是否覆盖
// schemaVersion。reader 声明非法本身也是错误（返回的 error 用于
// ErrCategorySchema 归因）。
//
// 语义：
//   - "^1.2"：接受 1.2.0 <= v < 2.0.0（major 锁定，MINOR/PATCH 向上兼容）；
//   - ">=1.0,<1.5"：闭开区间语义，接受 1.0.0 <= v < 1.5.0；
//   - 版本号解析失败、语法非法 → 报错（不静默放行）。
func SupportsSchema(readerRange, schemaVersion string) error {
	if readerRange == "" {
		return fmt.Errorf("reader_schema_range 为空")
	}
	if schemaVersion == "" {
		return fmt.Errorf("schema_version 为空")
	}
	sv, err := parseSemver(schemaVersion)
	if err != nil {
		return fmt.Errorf("schema_version 非法 %q: %w", schemaVersion, err)
	}

	if m := caretRangePattern.FindStringSubmatch(readerRange); m != nil {
		lo := semver{major: atoi(m[1]), minor: atoi(m[2]), patch: atoi(m[3])}
		hi := semver{major: lo.major + 1}
		if !sv.inRange(lo, hi) {
			return fmt.Errorf("schema %s 不在 %s（要求 [%s, %s)）", schemaVersion, readerRange, lo, hi)
		}
		return nil
	}

	// 显式范围：>=a.b[.c],<d.e[.f]
	parts := strings.SplitN(readerRange, ",", -1)
	if len(parts) != explicitRangeParts ||
		!strings.HasPrefix(parts[0], ">=") || !strings.HasPrefix(parts[1], "<") {
		return fmt.Errorf("reader_schema_range 语法非法：%q（支持 ^x.y 或 >=x.y,<x.y）", readerRange)
	}
	lo, err := parseSemver(strings.TrimPrefix(parts[0], ">="))
	if err != nil {
		return fmt.Errorf("下界非法：%w", err)
	}
	hi, err := parseSemver(strings.TrimPrefix(parts[1], "<"))
	if err != nil {
		return fmt.Errorf("上界非法：%w", err)
	}
	if !sv.inRange(lo, hi) {
		return fmt.Errorf("schema %s 不在 %s（要求 [%s, %s)）", schemaVersion, readerRange, lo, hi)
	}
	return nil
}

type semver struct{ major, minor, patch int }

func (s semver) String() string { return fmt.Sprintf("%d.%d.%d", s.major, s.minor, s.patch) }

func (s semver) less(o semver) bool {
	if s.major != o.major {
		return s.major < o.major
	}
	if s.minor != o.minor {
		return s.minor < o.minor
	}
	return s.patch < o.patch
}

// inRange 半开区间 [lo, hi)。
func (s semver) inRange(lo, hi semver) bool {
	return !s.less(lo) && s.less(hi)
}

func parseSemver(v string) (semver, error) {
	var s semver
	if v == "" {
		return s, fmt.Errorf("空版本号")
	}
	// 兼容 "v1.2.3" 前缀写法（与 Python 端一致）
	if strings.HasPrefix(v, "v") {
		v = v[1:]
	}
	parts := strings.Split(v, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return s, fmt.Errorf("版本号 %q 必须为 MAJOR.MINOR[.PATCH]", v)
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return s, fmt.Errorf("版本号段 %q 非法", p)
		}
		nums[i] = n
	}
	return semver{nums[0], nums[1], nums[2]}, nil
}

func atoi(s string) int {
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}
