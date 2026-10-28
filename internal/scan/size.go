package scan

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ParseSize converts text such as "10MiB", "500kib" or "1024" into bytes.
//
// Only binary units are accepted (B, KiB, MiB, GiB). Decimal units such as
// "MB" are rejected on purpose: they are ambiguous, and guessing wrong would
// silently change what the tool reports. Only whole numbers are accepted.
func ParseSize(s string) (int64, error) {
	text := strings.TrimSpace(s)

	// Split into the leading digits and whatever follows.
	i := 0
	for i < len(text) && text[i] >= '0' && text[i] <= '9' {
		i++
	}
	digits, unit := text[:i], strings.ToLower(strings.TrimSpace(text[i:]))

	if digits == "" {
		return 0, fmt.Errorf("invalid size %q: expected a whole number such as 10MiB", s)
	}
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q: number is too large", s)
	}

	var multiplier int64
	switch unit {
	case "", "b":
		multiplier = 1
	case "kib":
		multiplier = 1 << 10
	case "mib":
		multiplier = 1 << 20
	case "gib":
		multiplier = 1 << 30
	default:
		return 0, fmt.Errorf("invalid size %q: unknown unit %q (use B, KiB, MiB or GiB)", s, unit)
	}

	if n == 0 {
		return 0, fmt.Errorf("invalid size %q: must be greater than zero", s)
	}
	// Integer overflow in Go wraps around silently, so check before multiplying.
	if n > math.MaxInt64/multiplier {
		return 0, fmt.Errorf("invalid size %q: number is too large", s)
	}
	return n * multiplier, nil
}
