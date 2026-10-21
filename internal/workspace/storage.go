package workspace

import (
	"crypto/sha256"
	"fmt"
	"path"
	"strings"
)

// storageLabel escapes ambiguous separators, case and Windows-unsafe bytes.
// Ordinary lowercase names keep their familiar spelling on every platform.
func storageLabel(value string) string {
	parts := strings.Split(value, "/")
	for i, part := range parts {
		var b strings.Builder
		for j := 0; j < len(part); j++ {
			c := part[j]
			safe := c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '.' || c == '-'
			ambiguous := c == '-' && (j == 0 || j == len(part)-1 || part[j-1] == '-' || part[j+1] == '-')
			if !safe || ambiguous || c == '.' && j == len(part)-1 {
				fmt.Fprintf(&b, "~%02x", c)
			} else {
				b.WriteByte(c)
			}
		}
		label := b.String()
		base := strings.SplitN(part, ".", 2)[0]
		reserved := base == "con" || base == "prn" || base == "aux" || base == "nul" || len(base) == 4 && (strings.HasPrefix(base, "com") || strings.HasPrefix(base, "lpt")) && base[3] >= '1' && base[3] <= '9'
		if reserved || label == "_root" {
			label = fmt.Sprintf("~%02x%s", part[0], label[1:])
		}
		parts[i] = label
	}
	label := strings.Join(parts, "--")
	// Bound component length for long scope paths; keep a readable prefix.
	if len(label) > 120 {
		digest := sha256.Sum256([]byte(value))
		label = label[:96] + "~h" + fmt.Sprintf("%x", digest[:8])
	}
	return label
}

func scopeStorageLabel(scope string) string {
	if scope == "." {
		return "_root"
	}
	return storageLabel(scope)
}

func patternStorageRoot(scope, id string) string {
	return path.Join("patterns", scopeStorageLabel(scope), storageLabel(strings.Replace(id, "@", "/", 1)))
}
