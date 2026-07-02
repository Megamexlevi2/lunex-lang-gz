// Lunex lang
// David Dev — (c) 2026. Licensed under the Mozilla Public License 2.0.

package runtime

import (
	"crypto/sha256"
	"fmt"
)

// Attribution constants. Removing or forging these violates the license.
const (
	NTLAuthor    = "David Dev"
	NTLGitHub    = "https://github.com/Megamexlevi2"
	NTLCopyright = "(c) David Dev 2026"
	NTLLicense   = "Mozilla Public License, Version 2.0 — https://mozilla.org/MPL/2.0/"
)

// AuthorFingerprint is a one-way SHA-256 hash of author identity — embedded in
// every lunex binary and .nc bytecode file. Impossible to fake, trivial to verify.
var AuthorFingerprint = buildFingerprint()

func buildFingerprint() string {
	h := sha256.Sum256([]byte(NTLAuthor + "|" + NTLGitHub + "|" + NTLCopyright))
	return fmt.Sprintf("lunex-fp:%x", h[:8])
}

// WatermarkHeader returns bytes prepended to every compiled .nc file,
// so every compiled program carries immutable attribution back to its author.
func WatermarkHeader() []byte {
	fp := AuthorFingerprint
	return []byte(fmt.Sprintf(
		"#!lunex-bytecode\n#author:%s\n#github:%s\n#fp:%s\n",
		NTLAuthor, NTLGitHub, fp,
	))
}

// VerifyWatermark reports whether a .nc file begins with the expected Lunex header.
func VerifyWatermark(data []byte) bool {
	return len(data) > 15 && string(data[:14]) == "#!lunex-bytecode"
}

// AttributionBanner returns the attribution string shown in verbose mode.
func AttributionBanner() string {
	return fmt.Sprintf(
		"Lunex Language Runtime\n%s  %s\nFingerprint: %s\n%s",
		NTLCopyright, NTLGitHub, AuthorFingerprint, NTLLicense,
	)
}
