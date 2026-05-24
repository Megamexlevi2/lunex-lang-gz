// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

// Package bridge defines the NCP (Lunex Communication Protocol), the binary
// pipe between the Go compiler and the Zig runtime. Don't rely on this wire
// format from external tools — it's internal and can change between versions.
package bridge

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
)

var ncpMagic = [4]byte{0x4E, 0x54, 0x4C, 0x01} // "Lunex\x01"

const ncpVersion uint8 = 0x01

// Messages sent from Go → Zig
const (
	MsgExecPipe uint8 = 0x01 // raw .nc bytecode
	MsgExecFile uint8 = 0x02 // null-terminated file path
	MsgRtInfo   uint8 = 0x03 // ask Zig for runtime info
	MsgRtManage uint8 = 0x04 // runtime management command
	MsgRtClean  uint8 = 0x05 // clear the JIT cache
	MsgKill     uint8 = 0x06 // graceful shutdown
)

// Messages sent from Zig → Go
const (
	MsgRespOK         uint8 = 0x80
	MsgRespErr        uint8 = 0x81
	MsgRespExit       uint8 = 0x82 // payload = uint8 exit code
	MsgStdout         uint8 = 0x83 // chunk of stdout bytes
	MsgStderr         uint8 = 0x84 // chunk of stderr bytes
	MsgRuntimeDirInfo uint8 = 0x85
	MsgRtInfoResp     uint8 = 0x86
	MsgEnd            uint8 = 0xFF
)

// Frame flags
const (
	FlagNone       uint8 = 0x00
	FlagCompressed uint8 = 0x01 // reserved, not used yet
	FlagDebug      uint8 = 0x02
	FlagNoCache    uint8 = 0x04 // skip all caches — always set in pipe mode
)

// Frame wire layout (little-endian, 20-byte header):
//
//	[0:4]   magic       — 0x4E 0x54 0x4C 0x01
//	[4]     version
//	[5]     msg_type
//	[6]     flags
//	[7]     reserved    — must be 0x00
//	[8:10]  seq         — wraps at 65535
//	[10:12] reserved
//	[12:16] payload_len
//	[16:20] payload_crc — CRC-32/IEEE; 0 if empty
//	[20:]   payload
const frameHeaderSize = 20

// Frame is the basic unit on the wire.
type Frame struct {
	MsgType uint8
	Flags   uint8
	Seq     uint16
	Payload []byte
}

// Encode serialises the frame to bytes.
func (f *Frame) Encode() []byte {
	payloadLen := uint32(len(f.Payload))
	payloadCRC := uint32(0)
	if payloadLen > 0 {
		payloadCRC = crc32.ChecksumIEEE(f.Payload)
	}

	buf := make([]byte, frameHeaderSize+int(payloadLen))
	copy(buf[0:4], ncpMagic[:])
	buf[4] = ncpVersion
	buf[5] = f.MsgType
	buf[6] = f.Flags
	buf[7] = 0x00
	binary.LittleEndian.PutUint16(buf[8:10], f.Seq)
	buf[10] = 0x00
	buf[11] = 0x00
	binary.LittleEndian.PutUint32(buf[12:16], payloadLen)
	binary.LittleEndian.PutUint32(buf[16:20], payloadCRC)
	if payloadLen > 0 {
		copy(buf[20:], f.Payload)
	}
	return buf
}

// DecodeHeader parses the first 20 bytes of a frame.
func DecodeHeader(hdr []byte) (msgType, flags uint8, seq uint16, payloadLen uint32, payloadCRC uint32, err error) {
	if len(hdr) < frameHeaderSize {
		err = fmt.Errorf("ncp: header too short (%d bytes)", len(hdr))
		return
	}
	if hdr[0] != ncpMagic[0] || hdr[1] != ncpMagic[1] || hdr[2] != ncpMagic[2] || hdr[3] != ncpMagic[3] {
		err = fmt.Errorf("ncp: invalid magic — expected Lunex\\x01, got %X %X %X %X", hdr[0], hdr[1], hdr[2], hdr[3])
		return
	}
	if hdr[4] != ncpVersion {
		err = fmt.Errorf("ncp: unsupported version %d (want %d) — rebuild lunex", hdr[4], ncpVersion)
		return
	}
	msgType = hdr[5]
	flags = hdr[6]
	seq = binary.LittleEndian.Uint16(hdr[8:10])
	payloadLen = binary.LittleEndian.Uint32(hdr[12:16])
	payloadCRC = binary.LittleEndian.Uint32(hdr[16:20])
	return
}

// ValidatePayload checks the CRC of received bytes.
func ValidatePayload(payload []byte, expectedCRC uint32) error {
	if len(payload) == 0 {
		return nil
	}
	if got := crc32.ChecksumIEEE(payload); got != expectedCRC {
		return fmt.Errorf("ncp: CRC mismatch (got 0x%08X, want 0x%08X) — data corrupted", got, expectedCRC)
	}
	return nil
}

// ErrorFrame is the payload for MsgRespErr.
// Wire: code(2) + line(4) + col(2) + msg_len(2) + msg + hint_len(2) + hint
type ErrorFrame struct {
	Code   uint16
	Line   uint32
	Column uint16
	Msg    string
	Hint   string
}

func EncodeErrorFrame(ef ErrorFrame) []byte {
	msgB := []byte(ef.Msg)
	hintB := []byte(ef.Hint)
	buf := make([]byte, 2+4+2+2+len(msgB)+2+len(hintB))
	off := 0
	binary.LittleEndian.PutUint16(buf[off:], ef.Code)
	off += 2
	binary.LittleEndian.PutUint32(buf[off:], ef.Line)
	off += 4
	binary.LittleEndian.PutUint16(buf[off:], ef.Column)
	off += 2
	binary.LittleEndian.PutUint16(buf[off:], uint16(len(msgB)))
	off += 2
	copy(buf[off:], msgB)
	off += len(msgB)
	binary.LittleEndian.PutUint16(buf[off:], uint16(len(hintB)))
	off += 2
	copy(buf[off:], hintB)
	return buf
}

func DecodeErrorFrame(payload []byte) (ErrorFrame, error) {
	if len(payload) < 10 {
		return ErrorFrame{}, fmt.Errorf("ncp: error frame too short")
	}
	var ef ErrorFrame
	off := 0
	ef.Code = binary.LittleEndian.Uint16(payload[off:])
	off += 2
	ef.Line = binary.LittleEndian.Uint32(payload[off:])
	off += 4
	ef.Column = binary.LittleEndian.Uint16(payload[off:])
	off += 2
	msgLen := int(binary.LittleEndian.Uint16(payload[off:]))
	off += 2
	if off+msgLen > len(payload) {
		return ErrorFrame{}, fmt.Errorf("ncp: error frame truncated at msg")
	}
	ef.Msg = string(payload[off : off+msgLen])
	off += msgLen
	if off+2 > len(payload) {
		return ef, nil
	}
	hintLen := int(binary.LittleEndian.Uint16(payload[off:]))
	off += 2
	if off+hintLen <= len(payload) {
		ef.Hint = string(payload[off : off+hintLen])
	}
	return ef, nil
}

// RuntimeDirInfo tells Go where Zig is keeping its runtime files.
// Wire: dir_len(4) + dir (absolute path, UTF-8)
type RuntimeDirInfo struct {
	Dir string
}

func EncodeRuntimeDirInfo(r RuntimeDirInfo) []byte {
	b := []byte(r.Dir)
	buf := make([]byte, 4+len(b))
	binary.LittleEndian.PutUint32(buf[:4], uint32(len(b)))
	copy(buf[4:], b)
	return buf
}

func DecodeRuntimeDirInfo(payload []byte) (RuntimeDirInfo, error) {
	if len(payload) < 4 {
		return RuntimeDirInfo{}, fmt.Errorf("ncp: runtime-dir-info too short")
	}
	dirLen := int(binary.LittleEndian.Uint32(payload[:4]))
	if 4+dirLen > len(payload) {
		return RuntimeDirInfo{}, fmt.Errorf("ncp: runtime-dir-info truncated")
	}
	return RuntimeDirInfo{Dir: string(payload[4 : 4+dirLen])}, nil
}
