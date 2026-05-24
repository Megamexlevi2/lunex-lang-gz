// Lunex lang — NC/NTZ bytecode container format.
// The NC (Lunex Compiled) container stores a 48-byte NTLI header followed by an
// XOR-scrambled payload containing the original source text.  When a Zig NTZ
// compiled opcode section is present, its byte length is stored in header
// bytes 40-43 (little-endian u32) and the raw opcodes are appended at the
// end of the file so the Zig runtime can detect and execute them directly.
//
// File layout:
//   [0:4]   magic "lxi"
//   [4:6]   version 0x0500
//   [6]     flags (0x01 = has source)
//   [7]     reserved
//   [8:24]  SHA-256 partial hash (first 16 bytes of file content hash)
//   [24:28] source text length
//   [28:32] sub-chunk count
//   [32:36] name length
//   [36:40] sentinel bytes 0xA7 0x3E 0xC1 0x5B
//   [40:44] NTZ opcode section length in bytes (0 = no NTZ section)
//   [44:48] reserved / zero
//   [48:]   XOR-scrambled payload (name, source file, source text, sub-chunks)
//   [end-ntz_len:end]  raw Zig VM opcodes (only when hdr[40:44] > 0)

package bytecode

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

var ntliMagic = [4]byte{'n', 't', 'l', 'i'}

const ntliVersion uint16 = 0x0500
const ntliHeaderSize = 48

var internalMagic = [4]byte{0x1c, 0x9a, 0x4e, 0x03}

const internalVersion uint16 = 0x0603

// buildLunexIHeader constructs the 48-byte NTLI file header.
// ntzLen is the byte length of the NTZ opcode section (0 = absent).
func buildLunexIHeader(chunk *Chunk, ntzLen uint32) [ntliHeaderSize]byte {
	var hdr [ntliHeaderSize]byte
	copy(hdr[0:4], ntliMagic[:])
	binary.LittleEndian.PutUint16(hdr[4:6], ntliVersion)
	hdr[6] = 0x01
	hdr[7] = 0x00

	h := sha256.New()
	h.Write([]byte(chunk.SourceFile))
	h.Write([]byte(chunk.SourceText))
	digest := h.Sum(nil)
	copy(hdr[8:24], digest[:16])

	binary.LittleEndian.PutUint32(hdr[24:28], uint32(len(chunk.SourceText)))
	binary.LittleEndian.PutUint32(hdr[28:32], uint32(len(chunk.SubChunks)))
	binary.LittleEndian.PutUint32(hdr[32:36], uint32(len(chunk.Name)))

	// Sentinel bytes — unchanged from original format.
	hdr[36] = 0xA7
	hdr[37] = 0x3E
	hdr[38] = 0xC1
	hdr[39] = 0x5B

	// NTZ section length — read by the Zig runtime to locate opcodes.
	// Bytes 44-47 remain zero (reserved).
	binary.LittleEndian.PutUint32(hdr[40:44], ntzLen)
	return hdr
}

// EncodeNC encodes a Chunk into the NC binary format without an NTZ section.
func EncodeNC(chunk *Chunk) ([]byte, error) {
	return encodeNCWithNTZ(chunk, nil)
}

// EncodeNCWithNTZ encodes a Chunk into the NC binary format and appends
// ntzOpcodes as the NTZ section when the slice is non-empty.
func EncodeNCWithNTZ(chunk *Chunk, ntzOpcodes []byte) ([]byte, error) {
	return encodeNCWithNTZ(chunk, ntzOpcodes)
}

func encodeNCWithNTZ(chunk *Chunk, ntzOpcodes []byte) ([]byte, error) {
	var buf bytes.Buffer

	ntzLen := uint32(len(ntzOpcodes))
	hdr := buildLunexIHeader(chunk, ntzLen)
	buf.Write(hdr[:])

	var payload bytes.Buffer
	payload.Write(internalMagic[:])
	writeU16(&payload, internalVersion)
	writeU16(&payload, 0)
	writeString(&payload, chunk.Name)
	writeString(&payload, chunk.SourceFile)
	writeString(&payload, chunk.SourceText)
	writeU32(&payload, uint32(len(chunk.SubChunks)))
	for _, sub := range chunk.SubChunks {
		writeString(&payload, sub.Name)
		writeString(&payload, sub.SourceFile)
		writeString(&payload, sub.SourceText)
	}

	scrambled := xorScramble(payload.Bytes(), ncKey)
	buf.Write(scrambled)

	// Append NTZ opcodes at the very end of the file.
	if ntzLen > 0 {
		buf.Write(ntzOpcodes)
	}

	return buf.Bytes(), nil
}

// DecodeNC decodes an NC file back into a Chunk.
// The NTZ opcode section (if any) is not included in the returned Chunk —
// use NTZSection to retrieve those bytes separately.
func DecodeNC(data []byte) (*Chunk, error) {
	if len(data) < ntliHeaderSize+8 {
		return nil, fmt.Errorf("invalid object: file too short")
	}

	if data[0] != 'n' || data[1] != 't' || data[2] != 'l' || data[3] != 'i' {
		return nil, fmt.Errorf("invalid object: not a recognized Lunex format")
	}

	ver := binary.LittleEndian.Uint16(data[4:6])
	if ver != ntliVersion {
		return nil, fmt.Errorf("invalid object: version mismatch (got 0x%04x)", ver)
	}

	// The NTZ section is the last ntzLen bytes of the file; exclude them from
	// the scrambled payload so DecodeNC stays backward-compatible.
	ntzLen := binary.LittleEndian.Uint32(data[40:44])
	payloadEnd := len(data)
	if ntzLen > 0 && int(ntzLen) <= payloadEnd-ntliHeaderSize {
		payloadEnd -= int(ntzLen)
	}

	payload := xorUnscramble(data[ntliHeaderSize:payloadEnd], ncKey)

	if len(payload) < 8 {
		return nil, fmt.Errorf("invalid object: corrupt payload")
	}

	var magic [4]byte
	copy(magic[:], payload[:4])
	if magic != internalMagic {
		return nil, fmt.Errorf("invalid object: bad inner signature")
	}

	innerVer := binary.LittleEndian.Uint16(payload[4:6])
	if innerVer != internalVersion {
		return nil, fmt.Errorf("invalid object: inner version mismatch (got 0x%04x)", innerVer)
	}

	r := bytes.NewReader(payload[8:])

	name, err := readString(r)
	if err != nil {
		return nil, err
	}
	srcFile, err := readString(r)
	if err != nil {
		return nil, err
	}
	srcText, err := readString(r)
	if err != nil {
		return nil, err
	}

	chunk := &Chunk{
		Name:       name,
		SourceFile: srcFile,
		SourceText: srcText,
	}

	subCount, err := readU32(r)
	if err != nil {
		return nil, err
	}
	for i := uint32(0); i < subCount; i++ {
		sName, _ := readString(r)
		sSrc, _ := readString(r)
		sText, _ := readString(r)
		chunk.SubChunks = append(chunk.SubChunks, &Chunk{
			Name:       sName,
			SourceFile: sSrc,
			SourceText: sText,
		})
	}

	return chunk, nil
}

// NTZSection returns the raw NTZ opcode bytes appended to an NC file.
// Returns nil if the file has no NTZ section or is not a valid NC file.
func NTZSection(data []byte) []byte {
	if len(data) < ntliHeaderSize {
		return nil
	}
	if data[0] != 'n' || data[1] != 't' || data[2] != 'l' || data[3] != 'i' {
		return nil
	}
	ntzLen := binary.LittleEndian.Uint32(data[40:44])
	if ntzLen == 0 || int(ntzLen) > len(data)-ntliHeaderSize {
		return nil
	}
	return data[len(data)-int(ntzLen):]
}

// 64-byte XOR key used to scramble the NC payload.
var ncKey = []byte{
	0xc7, 0x3b, 0xa2, 0x58, 0xf1, 0x0d, 0x6e, 0x94,
	0x27, 0xbb, 0x43, 0xe0, 0x7c, 0x15, 0xd8, 0xa9,
	0x52, 0xfe, 0x30, 0x8d, 0x61, 0x1a, 0xc4, 0x77,
	0xeb, 0x09, 0x56, 0xf3, 0xae, 0x2b, 0x84, 0x5d,
	0x13, 0x98, 0xe7, 0x4c, 0xb0, 0x2f, 0x71, 0xda,
	0x3e, 0x95, 0xc0, 0x68, 0xf7, 0x19, 0x82, 0x4b,
	0xa4, 0x5e, 0x07, 0xcd, 0x36, 0xb9, 0x7f, 0xe2,
	0x0c, 0xd5, 0x88, 0x41, 0xfa, 0x23, 0x6a, 0x9e,
}

func xorScramble(data, key []byte) []byte {
	out := make([]byte, len(data))
	kl := len(key)
	for i, b := range data {
		ki := i % kl
		prev := byte(0)
		if i > 0 {
			prev = out[i-1]
		}
		out[i] = (b ^ key[ki] ^ byte(i*3+ki*7)) + prev&0x1f
	}
	return out
}

func xorUnscramble(data, key []byte) []byte {
	out := make([]byte, len(data))
	kl := len(key)
	for i, b := range data {
		ki := i % kl
		prev := byte(0)
		if i > 0 {
			prev = data[i-1]
		}
		out[i] = (b - prev&0x1f) ^ key[ki] ^ byte(i*3+ki*7)
	}
	return out
}

func writeU16(w *bytes.Buffer, v uint16) {
	b := [2]byte{}
	binary.LittleEndian.PutUint16(b[:], v)
	w.Write(b[:])
}

func writeU32(w *bytes.Buffer, v uint32) {
	b := [4]byte{}
	binary.LittleEndian.PutUint32(b[:], v)
	w.Write(b[:])
}

func writeString(w *bytes.Buffer, s string) {
	writeU32(w, uint32(len(s)))
	w.WriteString(s)
}

func readU16(r *bytes.Reader) (uint16, error) {
	b := [2]byte{}
	if _, err := r.Read(b[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b[:]), nil
}

func readU32(r *bytes.Reader) (uint32, error) {
	b := [4]byte{}
	if _, err := r.Read(b[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b[:]), nil
}

func writeI64(w *bytes.Buffer, v int64) {
	b := [8]byte{}
	binary.LittleEndian.PutUint64(b[:], uint64(v))
	w.Write(b[:])
}

func readI64(r *bytes.Reader) (int64, error) {
	b := [8]byte{}
	if _, err := r.Read(b[:]); err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(b[:])), nil
}

func readString(r *bytes.Reader) (string, error) {
	ln, err := readU32(r)
	if err != nil {
		return "", err
	}
	if ln == 0 {
		return "", nil
	}
	buf := make([]byte, ln)
	if _, err := r.Read(buf); err != nil {
		return "", err
	}
	return string(buf), nil
}
