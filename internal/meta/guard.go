// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// Package meta — runtime integrity guards.
// DO NOT MODIFY — this file is part of the binary integrity chain.
package meta

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/adler32"
	"hash/crc32"
	"hash/fnv"
	"os"
	"time"
)

// ─── Hash constants (verified against decoded provenance) ─────────────────────

// _lunex_crc is the expected CRC32-IEEE of the decoded provenance record.
const _lunex_crc uint32 = 0x1850C8D0

// _lunex_fnv is the expected FNV-1a 32-bit hash of the decoded provenance record.
const _lunex_fnv uint32 = 0x55D77B0A

// _lunex_adler is the expected Adler-32 of the decoded provenance record.
const _lunex_adler uint32 = 0x0B7B1D72

// _lunex_anchor is the first 16 hex chars of the SHA-256 of the decoded provenance.
// Verified against crypto/sha256 at startup and again by the integrity watcher.
const _lunex_anchor = "79cbf28fe7c6eba0"

// _lunex_shown prevents duplicate display on re-entrant calls.
var _lunex_shown bool

// ─── Key derivation ───────────────────────────────────────────────────────────

// deriveKey interleaves the four 3-byte fragments into a 12-byte key:
//   [ka[0], kb[0], kc[0], kd[0], ka[1], kb[1], kc[1], kd[1], ...]
func deriveKey() []byte {
	key := make([]byte, 12)
	for i := 0; i < 3; i++ {
		key[i*4+0] = _lunex_ka[i]
		key[i*4+1] = _lunex_kb[i]
		key[i*4+2] = _lunex_kc[i]
		key[i*4+3] = _lunex_kd[i]
	}
	return key
}

// ─── Shard validation ─────────────────────────────────────────────────────────

// fnv1a32 computes the FNV-1a 32-bit hash of data.
func fnv1a32(data []byte) uint32 {
	h := fnv.New32a()
	h.Write(data)
	return h.Sum32()
}

// validateShards checks the FNV-1a fingerprint of each encoded shard before
// decoding. This detects any single-byte tamper before the XOR pass runs.
func validateShards() bool {
	checks := [5]struct {
		shard []byte
		want  uint32
	}{
		{_lunex_s0, _lunex_s0_fnv},
		{_lunex_s1, _lunex_s1_fnv},
		{_lunex_s2, _lunex_s2_fnv},
		{_lunex_s3, _lunex_s3_fnv},
		{_lunex_s4, _lunex_s4_fnv},
	}
	for _, c := range checks {
		if fnv1a32(c.shard) != c.want {
			return false
		}
	}
	return true
}

// ─── Decode ───────────────────────────────────────────────────────────────────

// ror8 rotates byte b right by n bits.
func ror8(b byte, n uint) byte {
	n &= 7
	return (b >> n) | (b << (8 - n))
}

// decodeProvenance reassembles the five shards and applies the inverse encoding:
//   plaintext[i] = ROR8( encoded[i], (i%5)+1 ) ^ key[i%12]
func decodeProvenance() []byte {
	key := deriveKey()
	combined := make([]byte, 0, 87)
	combined = append(combined, _lunex_s0...)
	combined = append(combined, _lunex_s1...)
	combined = append(combined, _lunex_s2...)
	combined = append(combined, _lunex_s3...)
	combined = append(combined, _lunex_s4...)
	out := make([]byte, len(combined))
	for i, b := range combined {
		out[i] = ror8(b, uint(i%5)+1) ^ key[i%12]
	}
	return out
}

// ─── Integrity verification ───────────────────────────────────────────────────

// verifyIntegrity runs all four checks against the decoded provenance:
//   1. Per-shard FNV-1a (pre-decode)
//   2. CRC32-IEEE of decoded bytes
//   3. FNV-1a 32-bit of decoded bytes
//   4. Adler-32 of decoded bytes
//   5. SHA-256 prefix vs _lunex_anchor
// Returns true only when every check passes.
func verifyIntegrity() bool {
	if !validateShards() {
		return false
	}
	decoded := decodeProvenance()

	if crc32.ChecksumIEEE(decoded) != _lunex_crc {
		return false
	}

	if fnv1a32(decoded) != _lunex_fnv {
		return false
	}

	if adler32.Checksum(decoded) != _lunex_adler {
		return false
	}

	sum := sha256.Sum256(decoded)
	anchor := hex.EncodeToString(sum[:])[:16]
	if anchor != _lunex_anchor {
		return false
	}

	return true
}

// ─── Startup check ────────────────────────────────────────────────────────────

// verifyAndDisplay runs all integrity checks and, when the CLI is invoked with
// no arguments, prints the authorship record. Executed via init() — before
// main() and before any CLI flag is parsed.
func verifyAndDisplay() {
	if _lunex_shown {
		return
	}
	if !verifyIntegrity() {
		fmt.Fprintln(os.Stderr,
			"lunex: fatal: build integrity check failed\n"+
				"The binary has been modified or is corrupted.")
		os.Exit(2)
	}
	if len(os.Args) == 1 {
		fmt.Println(string(decodeProvenance()))
		fmt.Println()
		_lunex_shown = true
	}
}

// ─── Background watcher ───────────────────────────────────────────────────────

// startIntegrityWatcher launches a goroutine that re-runs all integrity checks
// every 60 seconds. If any check fails after startup, the process is terminated.
// This prevents runtime patching of the shard or key arrays in memory.
func startIntegrityWatcher() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if !verifyIntegrity() {
				fmt.Fprintln(os.Stderr,
					"lunex: fatal: runtime integrity violation detected")
				os.Exit(2)
			}
		}
	}()
}

// ─── Exported surface ─────────────────────────────────────────────────────────

// Provenance returns the decoded, integrity-verified authorship record.
// Terminates the process if any integrity check fails.
func Provenance() string {
	if !verifyIntegrity() {
		fmt.Fprintln(os.Stderr, "lunex: fatal: provenance integrity check failed")
		os.Exit(2)
	}
	return string(decodeProvenance())
}

// Seal is called from main after all subsystems are initialised.
// It starts the background integrity watcher. Must be called exactly once.
func Seal() {
	startIntegrityWatcher()
}

// ─── Automatic init ───────────────────────────────────────────────────────────

// init runs before main(). Credits are bound here — not in any help printer —
// so editing or removing the help text has zero effect on credit display.
func init() {
	verifyAndDisplay()
}
