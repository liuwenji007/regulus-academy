package ingest

import (
	"os"
	"strconv"
)

const (
	defaultMaxPDFBytes  = 20 * 1024 * 1024
	defaultMaxPDFPages  = 100
	defaultMaxURLChars  = 80000
	defaultFetchTimeout = 15
)

func maxPDFBytes() int {
	if v := envInt("REGULUS_INGEST_MAX_PDF_BYTES"); v > 0 {
		return v
	}
	return defaultMaxPDFBytes
}

func maxPDFPages() int {
	if v := envInt("REGULUS_INGEST_MAX_PDF_PAGES"); v > 0 {
		return v
	}
	return defaultMaxPDFPages
}

func maxURLChars() int {
	if v := envInt("REGULUS_INGEST_MAX_URL_CHARS"); v > 0 {
		return v
	}
	return defaultMaxURLChars
}

func fetchTimeoutSec() int {
	if v := envInt("REGULUS_INGEST_FETCH_TIMEOUT_SEC"); v > 0 {
		return v
	}
	return defaultFetchTimeout
}

func envInt(key string) int {
	raw := os.Getenv(key)
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}
