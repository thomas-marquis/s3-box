package values

const (
	Byte uint64 = 1
	KiB         = Byte << (10 * iota)
	MiB
	GiB
	TiB
	PiB
)
