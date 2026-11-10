// The small result-aggregate types (CrossPlatformArchitecture.md §5, §4.8;
// PrimitiveLayer.md §4.2). Several core operations must return more than one
// value; under the Go value-return memory model (D-G4) these are returned by
// value, like Swift/C — no Java/Kotlin fill-in pool, so no destination-first
// rewrite. Fields are role-major, ascending limb order (D23). The residue field
// trails the quotient limbs (the JVM header-gap reorder of §4.8 applies only to
// the JVM ports, not to a Go value type).

package decimal128

type quot64Rem64 struct {
	quotDw0 int64
	remDw0  int64
}

type quot64Rem128 struct {
	quotDw0 int64
	remDw0  int64
	remDw1  int64
}

type quot128Rem64 struct {
	quotDw0 int64
	quotDw1 int64
	remDw0  int64
}

type quot128Rem128 struct {
	quotDw0 int64
	quotDw1 int64
	remDw0  int64
	remDw1  int64
}

type quot128Residue struct {
	quotDw0 int64
	quotDw1 int64
	residue residue
}

type quot256Rem64 struct {
	quotDw0 int64
	quotDw1 int64
	quotDw2 int64
	quotDw3 int64
	remDw0  int64
}

type quot256Rem128 struct {
	quotDw0 int64
	quotDw1 int64
	quotDw2 int64
	quotDw3 int64
	remDw0  int64
	remDw1  int64
}

type quot256Residue struct {
	quotDw0 int64
	quotDw1 int64
	quotDw2 int64
	quotDw3 int64
	residue residue
}

type diff128Swap struct {
	diffDw0 int64
	diffDw1 int64
	swap    bool
}

type diff256Swap struct {
	diffDw0 int64
	diffDw1 int64
	diffDw2 int64
	diffDw3 int64
	swap    bool
}

type stripped128StripCount struct {
	strippedDw0 int64
	strippedDw1 int64
	stripCount  int64
}

// quot320Residue is the D48 divPow10 result: a 320-bit quotient with the
// residue of the discarded digits (divPow10_q320res_*; Swift Quot320.swift).
type quot320Residue struct {
	quotDw0 int64
	quotDw1 int64
	quotDw2 int64
	quotDw3 int64
	quotDw4 int64
	residue residue
}

func (r quot320Residue) quot() u320 {
	return u320{r.quotDw0, r.quotDw1, r.quotDw2, r.quotDw3, r.quotDw4}
}

// quot320Rem160 is the divRem_320_160_mg result: a 320-bit quotient and a
// remainder below the (<= 160-bit) divisor.
type quot320Rem160 struct {
	quotDw0 int64
	quotDw1 int64
	quotDw2 int64
	quotDw3 int64
	quotDw4 int64
	remDw0  int64
	remDw1  int64
	remDw2  int64
}

func (r quot320Rem160) quot() u320 {
	return u320{r.quotDw0, r.quotDw1, r.quotDw2, r.quotDw3, r.quotDw4}
}

func (r quot320Rem160) rem() u192 { return u192{r.remDw0, r.remDw1, r.remDw2} }
