package types

import "fmt"

// ByteSize represents the various terms for storage space.
type ByteSize float64

// Storage space terms.
const (
	_           = iota // ignore first value by assigning to blank identifier
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	EB
	ZB
	YB
)

func (b ByteSize) String() string {

	switch {
	case b >= YB:
		return fmt.Sprintf("%.2f YB", b/YB)
	case b >= ZB:
		return fmt.Sprintf("%.2f ZB", b/ZB)
	case b >= EB:
		return fmt.Sprintf("%.2f EB", b/EB)
	case b >= PB:
		return fmt.Sprintf("%.2f PB", b/PB)
	case b >= TB:
		return fmt.Sprintf("%.2f TB", b/TB)
	case b >= GB:
		return fmt.Sprintf("%.2f GB", b/GB)
	case b >= MB:
		return fmt.Sprintf("%.1f MB", b/MB)
	case b >= KB:
		return fmt.Sprintf("%.0f KB", b/KB)
	}

	return fmt.Sprintf("%f Bytes", b)
}
