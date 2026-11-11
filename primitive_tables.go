// Constant tables (PrimitiveTables.swift). The low-bit masks live here; the
// pow10/pow5 magnitude tables land in a later increment with the divPow10/
// mulPow10 kernels that consume them.
//
// Naming note: Swift spells these MASK16L/MASK32L/MASK48L (SCREAMING per §6.2).
// Go must lowercase them to keep them unexported (engine-internal); this is the
// usual Go capitalization divergence.

package decimal128

const (
	const_MASK16L = 0xFFFF
	const_MASK32L = 0xFFFF_FFFF
	const_MASK48L = 0xFFFF_FFFF_FFFF
	const_MASK54L = 0x003F_FFFF_FFFF_FFFF
)
