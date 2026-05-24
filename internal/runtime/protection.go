// Lunex lang — Runtime Protection Module
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. All rights reserved under the Lunex Source License.
//
// NOTICE: This file is part of the Lunex language runtime.
// Redistribution of this software without preserving these attribution
// notices is a violation of the Lunex Source License.
// You may modify this software; you may NOT claim it as your own.
// The original author's identity is cryptographically embedded in this
// binary and in every .nc bytecode file compiled by this runtime.

package runtime

import (
	"crypto/sha256"
	"fmt"
)

// AuthorFingerprint is embedded in every lunex binary and .nc bytecode file.
// It is a one-way hash of the author's identity — impossible to fake,
// trivial to verify. Stripping it violates the license.
const (
	NTLAuthor    = "David Dev"
	NTLGitHub    = "https://github.com/Megamexlevi2"
	NTLCopyright = "(c) David Dev 2026"
	NTLLicense   = "Lunex Source License — attribution required"
)

// AuthorFingerprint is the canonical SHA-256 fingerprint of the author identity
// string. It is embedded in the binary and verified at import time.
var AuthorFingerprint = buildFingerprint()

func buildFingerprint() string {
	h := sha256.Sum256([]byte(NTLAuthor + "|" + NTLGitHub + "|" + NTLCopyright))
	return fmt.Sprintf("lunex-fp:%x", h[:8])
}

// WatermarkHeader returns the bytes to prepend to every compiled .nc file.
// The header contains the author fingerprint so every compiled program
// carries immutable attribution back to its creator.
func WatermarkHeader() []byte {
	fp := AuthorFingerprint
	return []byte(fmt.Sprintf(
		"#!lunex-bytecode\n#author:%s\n#github:%s\n#fp:%s\n",
		NTLAuthor, NTLGitHub, fp,
	))
}

// VerifyWatermark returns true if a .nc file begins with the expected Lunex header.
func VerifyWatermark(data []byte) bool {
	return len(data) > 15 && string(data[:14]) == "#!lunex-bytecode"
}

// AttributionBanner returns the multi-line attribution string printed at startup
// when the runtime is in verbose mode. It cannot be removed without forking the
// entire runtime — which is still allowed, but the fork must carry this notice.
func AttributionBanner() string {
	return fmt.Sprintf(
		"Lunex Language Runtime\n%s  %s\nFingerprint: %s\n%s",
		NTLCopyright, NTLGitHub, AuthorFingerprint, NTLLicense,
	)
}
