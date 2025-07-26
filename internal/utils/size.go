package utils

import "fmt"

const (
	kilo int = 1024
	mega int = kilo * kilo
	giga int = mega * kilo
	tera int = giga * kilo
	peta int = tera * kilo
)

func FormatSizeBytes(b int) string {
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

func BytesToMB(b int) int {
	return b / mega
}

func MegaToBytes(mb int) int {
	return mb * mega
}
