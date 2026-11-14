// The quot128Residue result aggregate (CrossPlatformArchitecture.md §5, §4.8).

package decimal128

type quot128Residue struct {
	quotDw0 int64
	quotDw1 int64
	residue residue
}
