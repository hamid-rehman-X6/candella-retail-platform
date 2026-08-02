package config

import (
	"bufio"
	"os"
	"strings"
)

// utf8BOM is the UTF-8 byte-order mark (EF BB BF) that some Windows editors
// prepend to text files. It is built from raw bytes so this source file contains
// no literal BOM — which the Go compiler rejects anywhere but the first byte.
var utf8BOM = string([]byte{0xEF, 0xBB, 0xBF})

// LoadDotEnv reads simple KEY=VALUE lines from the given file (if present) and
// sets them in the process environment, WITHOUT overriding variables that are
// already set (a real exported env var always wins). It is a tiny, dependency-free
// convenience for local development — production injects real environment variables
// directly, so a missing file is not an error.
//
// Blank lines and lines starting with '#' are ignored; surrounding quotes on the
// value are stripped; CRLF endings and a leading UTF-8 BOM are tolerated.
func LoadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // no .env file — perfectly fine
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), utf8BOM))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}
