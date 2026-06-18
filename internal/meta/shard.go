// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// Package meta — provenance shard storage.
// DO NOT MODIFY — altering any value here corrupts runtime integrity verification.
package meta

// ─── Key derivation fragments ─────────────────────────────────────────────────
// The 12-byte decoding key is reconstructed at runtime by interleaving
// _lunex_ka, _lunex_kb, _lunex_kc, _lunex_kd byte-by-byte:
//   key = [ka[0],kb[0],kc[0],kd[0], ka[1],kb[1],kc[1],kd[1], ka[2],kb[2],kc[2],kd[2]]
// Modifying any fragment produces a wrong key, yielding garbage on decode
// and failing all three independent hash checks (CRC32, FNV-1a, Adler-32).

var _lunex_ka = []byte{0x4e, 0x2a, 0x17}
var _lunex_kb = []byte{0x0f, 0x36, 0x5c}
var _lunex_kc = []byte{0x71, 0x4b, 0x3d}
var _lunex_kd = []byte{0x53, 0x1e, 0x67}

// ─── Per-shard integrity constants (FNV-1a 32-bit of each encoded shard) ─────
// Checked before any decoding attempt. A single tampered byte causes an
// immediate abort — the XOR/rotation passes never execute.

const (
	_lunex_s0_fnv uint32 = 0x3556C8DD
	_lunex_s1_fnv uint32 = 0x4BC0FBC7
	_lunex_s2_fnv uint32 = 0xC43F5BED
	_lunex_s3_fnv uint32 = 0xF32C1A99
	_lunex_s4_fnv uint32 = 0xF6838EA9
)

// ─── Encoded provenance shards ────────────────────────────────────────────────
// Encoding scheme applied at global index i:
//   encoded[i] = ROL8( plaintext[i] ^ key[i%12], (i%5)+1 )
// Decoding reverses:
//   plaintext[i] = ROR8( encoded[i], (i%5)+1 ) ^ key[i%12]
// Shards are concatenated in order (s0||s1||s2||s3||s4) → 87 bytes total.

var _lunex_s0 = []byte{
	0x00, 0x6d, 0xe9, 0x37, 0xcc, 0xae, 0x94, 0xcb, 0xd1,
	0xe3, 0x9e, 0x08, 0x79, 0xb7, 0x82, 0x6e, 0x28, 0xa2,
}

var _lunex_s1 = []byte{
	0x23, 0xc7, 0xa6, 0xf4, 0x5a, 0xe0, 0x45, 0x5e, 0xd4,
	0xb1, 0xc5, 0x87, 0x18, 0xdd, 0x1b, 0x41, 0x09,
}

var _lunex_s2 = []byte{
	0x0a, 0xd1, 0x79, 0x91, 0xe4, 0xbc, 0x19, 0xc1, 0x42,
	0x07, 0xe6, 0x69, 0x70, 0xa3, 0xec, 0x08, 0xc4,
}

var _lunex_s3 = []byte{
	0x20, 0x55, 0x84, 0xe6, 0xe0, 0x88, 0x85, 0x00, 0x5e,
	0x89, 0xa0, 0xb2, 0xc8, 0xa6, 0xf4, 0xbb, 0x52, 0xca,
}

var _lunex_s4 = []byte{
	0xff, 0x3b, 0x73, 0xb4, 0x02, 0x4a, 0x0d, 0x92, 0xb6,
	0x4b, 0xe4, 0xa8, 0xe8, 0x55, 0xcf, 0x7a, 0x1d,
}
