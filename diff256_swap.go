// The diff256Swap result aggregate (CrossPlatformArchitecture.md §5, §4.8).

package decimal128

type diff256Swap struct {
	diffDw0 int64
	diffDw1 int64
	diffDw2 int64
	diffDw3 int64
	swap    bool
}
