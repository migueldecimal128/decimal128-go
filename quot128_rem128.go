// The quot128Rem128 result aggregate (CrossPlatformArchitecture.md §5, §4.8).

package decimal128

type quot128Rem128 struct {
	quotDw0 int64
	quotDw1 int64
	remDw0  int64
	remDw1  int64
}
