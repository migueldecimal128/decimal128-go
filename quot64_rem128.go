// The quot64Rem128 result aggregate (CrossPlatformArchitecture.md §5, §4.8).

package decimal128

type quot64Rem128 struct {
	quotDw0 int64
	remDw0  int64
	remDw1  int64
}
