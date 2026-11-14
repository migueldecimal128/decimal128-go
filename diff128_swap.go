// The diff128Swap result aggregate (CrossPlatformArchitecture.md §5, §4.8).

package decimal128

type diff128Swap struct {
	diffDw0 int64
	diffDw1 int64
	swap    bool
}
