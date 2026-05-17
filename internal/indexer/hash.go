package indexer

import (
	"crypto/sha256"
	"encoding/hex"
)

// SymbolHash returns a 16-char (64-bit) hex digest of the symbol body. Used
// to invalidate memory observations linked to a symbol when its body changes.
// 64 bits is enough — collisions across a single project are vanishingly
// unlikely and the failure mode (stale linkage) is benign.
func SymbolHash(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:8])
}
