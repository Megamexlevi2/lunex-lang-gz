// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package bytecode

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ntlxMagic = [4]byte{'n', 't', 'l', 'x'}

const ntlxVersion uint16 = 0x0500
const ntlxHeaderSize = 64

var naxInternalMagic = [4]byte{0x3d, 0x7e, 0xa1, 0x02}

const naxInternalVersion uint16 = 0x0603

type NAXEntry struct {
	Name string
	Data []byte
}

type NAXArchive struct {
	Version   uint16
	BuildTime int64
	Entries   []NAXEntry
	MainEntry string
}

func buildNTLXHeader(arch *NAXArchive) [ntlxHeaderSize]byte {
	var hdr [ntlxHeaderSize]byte
	copy(hdr[0:4], ntlxMagic[:])
	binary.LittleEndian.PutUint16(hdr[4:6], ntlxVersion)
	hdr[6] = 0x01
	hdr[7] = 0x00
	binary.LittleEndian.PutUint64(hdr[8:16], uint64(arch.BuildTime))
	binary.LittleEndian.PutUint32(hdr[16:20], uint32(len(arch.Entries)))
	binary.LittleEndian.PutUint32(hdr[20:24], uint32(len(arch.MainEntry)))
	hdr[24] = 0xF3
	hdr[25] = 0x8A
	hdr[26] = 0x4D
	hdr[27] = 0x2C
	hdr[28] = 0x00
	hdr[29] = 0x00
	hdr[30] = 0x00
	hdr[31] = 0x00
	return hdr
}

func PackDirectory(dir string, outputFile string) error {
	arch := &NAXArchive{
		Version:   naxInternalVersion,
		BuildTime: time.Now().Unix(),
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("cannot read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".lx") {
			continue
		}
		ncName := strings.TrimSuffix(name, ".lx") + ".nc"
		ntlPath := filepath.Join(dir, name)
		ncPath := filepath.Join(dir, ncName)
		if err := BuildNCFile(ntlPath, ncPath); err != nil {
			return fmt.Errorf("compile error for %s: %w", name, err)
		}
		fmt.Printf("  compiled %s → %s\n", name, ncName)
	}

	mainFound := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".nc") {
			name = strings.TrimSuffix(name, ".lx") + ".nc"
		}
		if !strings.HasSuffix(name, ".nc") {
			continue
		}
		fullPath := filepath.Join(dir, name)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		arch.Entries = append(arch.Entries, NAXEntry{Name: name, Data: data})
		if name == "main.nc" {
			arch.MainEntry = name
			mainFound = true
		}
	}

	if !mainFound {
		return fmt.Errorf(
			"no main.lx or main.nc found in %s\n  a .nax archive requires main.lx as its entry point\n  create main.lx with your program's entry point first",
			dir,
		)
	}

	if len(arch.Entries) == 0 {
		return fmt.Errorf("no source files found in %s", dir)
	}

	raw, err := encodeNAX(arch)
	if err != nil {
		return err
	}

	return os.WriteFile(outputFile, raw, 0644)
}

func LoadNAX(path string) (*NAXArchive, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeNAX(data)
}

func NAXGetEntry(arch *NAXArchive, name string) ([]byte, bool) {
	for _, e := range arch.Entries {
		if e.Name == name {
			return e.Data, true
		}
	}
	return nil, false
}

func encodeNAX(arch *NAXArchive) ([]byte, error) {
	var buf bytes.Buffer

	hdr := buildNTLXHeader(arch)
	buf.Write(hdr[:])

	var payload bytes.Buffer
	payload.Write(naxInternalMagic[:])
	writeU16(&payload, naxInternalVersion)
	writeU16(&payload, 0)
	writeI64(&payload, arch.BuildTime)
	writeString(&payload, arch.MainEntry)
	writeU32(&payload, uint32(len(arch.Entries)))
	for _, e := range arch.Entries {
		writeString(&payload, e.Name)
		writeU32(&payload, uint32(len(e.Data)))
		payload.Write(e.Data)
	}

	scrambled := naxScramble(payload.Bytes())
	buf.Write(scrambled)
	return buf.Bytes(), nil
}

func decodeNAX(data []byte) (*NAXArchive, error) {
	if len(data) < ntlxHeaderSize+8 {
		return nil, fmt.Errorf("invalid archive: file too short")
	}
	if data[0] != 'n' || data[1] != 't' || data[2] != 'l' || data[3] != 'x' {
		return nil, fmt.Errorf("invalid archive: not a recognized Lunex archive format")
	}

	ver := binary.LittleEndian.Uint16(data[4:6])
	if ver != ntlxVersion {
		return nil, fmt.Errorf("invalid archive: version mismatch (got 0x%04x)", ver)
	}

	payload := naxUnscramble(data[ntlxHeaderSize:])
	if len(payload) < 8 {
		return nil, fmt.Errorf("invalid archive: corrupt payload")
	}

	var magic [4]byte
	copy(magic[:], payload[:4])
	if magic != naxInternalMagic {
		return nil, fmt.Errorf("invalid archive: bad inner signature")
	}

	innerVer := binary.LittleEndian.Uint16(payload[4:6])
	if innerVer != naxInternalVersion {
		return nil, fmt.Errorf("invalid archive: inner version mismatch")
	}

	r := bytes.NewReader(payload[8:])

	buildTime, err := readI64(r)
	if err != nil {
		return nil, err
	}
	mainEntry, err := readString(r)
	if err != nil {
		return nil, err
	}
	count, err := readU32(r)
	if err != nil {
		return nil, err
	}

	arch := &NAXArchive{
		Version:   innerVer,
		BuildTime: buildTime,
		MainEntry: mainEntry,
	}

	for i := uint32(0); i < count; i++ {
		name, err := readString(r)
		if err != nil {
			return nil, err
		}
		size, err := readU32(r)
		if err != nil {
			return nil, err
		}
		entryData := make([]byte, size)
		if _, err := r.Read(entryData); err != nil {
			return nil, err
		}
		arch.Entries = append(arch.Entries, NAXEntry{Name: name, Data: entryData})
	}
	return arch, nil
}

var naxKey = []byte{
	0x9b, 0xe4, 0x2c, 0x71, 0xd3, 0x5a, 0x08, 0xf6,
	0x47, 0xbc, 0x1e, 0x83, 0x60, 0xad, 0x35, 0x7f,
	0xc2, 0x58, 0x0d, 0xea, 0x91, 0x3c, 0x76, 0xb4,
	0x2a, 0x69, 0xf0, 0x14, 0xa7, 0x5e, 0xcd, 0x42,
	0x88, 0x1b, 0x64, 0xd9, 0x37, 0xfc, 0x55, 0xa0,
	0x0c, 0x73, 0xbe, 0x29, 0x8d, 0xe1, 0x4f, 0x96,
	0x23, 0xda, 0x61, 0xb7, 0x3e, 0x80, 0xc5, 0x1d,
	0x52, 0xaf, 0x04, 0x99, 0x6b, 0xd8, 0x20, 0x7c,
}

func naxScramble(data []byte) []byte {
	out := make([]byte, len(data))
	kl := len(naxKey)
	for i, b := range data {
		ki := i % kl
		prev := byte(0)
		if i > 0 {
			prev = out[i-1]
		}
		out[i] = ((b ^ naxKey[ki]) + byte(i*7+ki*3)) ^ prev
	}
	return out
}

func naxUnscramble(data []byte) []byte {
	out := make([]byte, len(data))
	kl := len(naxKey)
	for i, b := range data {
		ki := i % kl
		prev := byte(0)
		if i > 0 {
			prev = data[i-1]
		}
		out[i] = ((b ^ prev) - byte(i*7+ki*3)) ^ naxKey[ki]
	}
	return out
}

// ExtractNAXEntry unpacks a .nax archive and returns the main entry's .nc bytes.
func ExtractNAXEntry(data []byte) ([]byte, error) {
	arch, err := decodeNAX(data)
	if err != nil {
		return nil, fmt.Errorf("cannot decode .nax: %w", err)
	}
	entryName := arch.MainEntry
	if entryName == "" && len(arch.Entries) > 0 {
		entryName = arch.Entries[0].Name
	}
	for _, e := range arch.Entries {
		if e.Name == entryName {
			return e.Data, nil
		}
	}
	return nil, fmt.Errorf("entry not found in archive: %s", entryName)
}
