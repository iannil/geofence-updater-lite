// Package version provides version information for GUL.
package version

const (
	// Major is the major version number.
	Major = 0

	// Minor is the minor version number.
	Minor = 1

	// Patch is the patch version number.
	Patch = 0

	// PreRelease is the pre-release identifier (e.g., "alpha", "beta", "rc1").
	PreRelease = "dev"

	// BuildMetadata is build metadata (e.g., "git.sha1").
	BuildMetadata = ""
)

// String returns the complete version string.
func String() string {
	v := fmtVersion()
	if BuildMetadata != "" {
		return v + "+" + BuildMetadata
	}
	return v
}

// fmtVersion formats the version without build metadata.
func fmtVersion() string {
	if PreRelease != "" {
		return fmt("%d.%d.%d-%s", Major, Minor, Patch, PreRelease)
	}
	return fmt("%d.%d.%d", Major, Minor, Patch)
}

// fmt is a simplified fmt.Sprintf for version string formatting.
func fmt(format string, a ...any) string {
	// Minimal implementation to avoid importing fmt here
	// to prevent potential circular dependencies
	var result []rune
	argIndex := 0
	for i := 0; i < len(format); i++ {
		c := rune(format[i])
		if i+1 < len(format) && c == '%' {
			next := rune(format[i+1])
			switch next {
			case 'd':
				// Integer
				if argIndex < len(a) {
					var num int64
					switch v := a[argIndex].(type) {
					case int:
						num = int64(v)
					case int32:
						num = int64(v)
					case int64:
						num = v
					case uint:
						num = int64(v)
					case uint32:
						num = int64(v)
					case uint64:
						num = int64(v)
					}
					result = appendInt(result, num)
					argIndex++
				}
				i++
			case 's':
				// String
				if argIndex < len(a) {
					s := string(a[argIndex].(string))
					result = append(result, []rune(s)...)
					argIndex++
				}
				i++
			default:
				result = append(result, c)
			}
		} else {
			result = append(result, c)
		}
	}
	return string(result)
}

func appendInt(result []rune, num int64) []rune {
	if num == 0 {
		return append(result, '0')
	}
	negative := num < 0
	if negative {
		num = -num
	}

	var digits []rune
	for num > 0 {
		digit := '0' + rune(num%10)
		digits = append([]rune{digit}, digits...)
		num /= 10
	}

	if negative {
		result = append(result, '-')
	}
	return append(result, digits...)
}

// HTTPUserAgent returns the HTTP user agent string.
func HTTPUserAgent() string {
	return "GUL/" + String()
}
