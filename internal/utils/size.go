package utils

import "fmt"

const (
	kilo int64 = 1024
	mega int64 = kilo * kilo
	giga int64 = mega * kilo
	tera int64 = giga * kilo
	peta int64 = tera * kilo
)

func FormatSizeBytes(b int64) string {
	if b < kilo {
		return fmt.Sprintf("%d B", b)
	}
	if b < mega {
		return fmt.Sprintf("%.2f KB", float64(b)/float64(kilo))
	}
	if b < giga {
		return fmt.Sprintf("%.2f MB", float64(b)/float64(mega))
	}
	if b < tera {
		return fmt.Sprintf("%.2f GB", float64(b)/float64(giga))
	}
	if b < peta {
		return fmt.Sprintf("%.2f TB", float64(b)/float64(tera))
	}
	return fmt.Sprintf("%.2f PB", float64(b)/float64(peta))
}

func BytesToMB(b int64) int64 {
	return b / mega
}

func MegaToBytes(mb int64) int64 {
	return mb * mega
}
