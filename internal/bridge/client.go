// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package bridge

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Result holds everything that came back from a Zig execution.
type Result struct {
	ExitCode   int
	Stdout     []byte
	Stderr     []byte
	RuntimeDir string    // where Zig keeps its files (set once at startup)
	Duration   time.Duration // wall time inside the Zig runtime; zero if not reported
}

// Client is the Go side of the NCP bridge.
// It writes to Zig's stdin and reads from Zig's stdout.
type Client struct {
	w     io.Writer
	r     *bufio.Reader
	mu    sync.Mutex
	seq   uint32
	debug bool
}

func NewClient(w io.Writer, r io.Reader) *Client {
	return &Client{
		w:     w,
		r:     bufio.NewReaderSize(r, 64*1024),
		debug: os.Getenv("NTL_DEBUG") == "1",
	}
}

func (c *Client) nextSeq() uint16 {
	return uint16(atomic.AddUint32(&c.seq, 1) & 0xFFFF)
}

func (c *Client) sendFrame(msgType, flags uint8, payload []byte) error {
	f := &Frame{MsgType: msgType, Flags: flags, Seq: c.nextSeq(), Payload: payload}
	encoded := f.Encode()
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.w.Write(encoded); err != nil {
		return fmt.Errorf("ncp: send type=0x%02X failed: %w", msgType, err)
	}
	if c.debug {
		fmt.Fprintf(os.Stderr, "[ncp] → 0x%02X seq=%d len=%d\n", msgType, f.Seq, len(payload))
	}
	return nil
}

func (c *Client) readFrame() (Frame, error) {
	hdr := make([]byte, frameHeaderSize)
	if _, err := io.ReadFull(c.r, hdr); err != nil {
		return Frame{}, fmt.Errorf("ncp: read header: %w", err)
	}
	msgType, flags, seq, payloadLen, payloadCRC, err := DecodeHeader(hdr)
	if err != nil {
		return Frame{}, err
	}
	var payload []byte
	if payloadLen > 0 {
		payload = make([]byte, payloadLen)
		if _, err := io.ReadFull(c.r, payload); err != nil {
			return Frame{}, fmt.Errorf("ncp: read payload (%d bytes): %w", payloadLen, err)
		}
		if err := ValidatePayload(payload, payloadCRC); err != nil {
			return Frame{}, err
		}
	}
	if c.debug {
		fmt.Fprintf(os.Stderr, "[ncp] ← 0x%02X seq=%d flags=0x%02X len=%d\n", msgType, seq, flags, payloadLen)
	}
	return Frame{MsgType: msgType, Flags: flags, Seq: seq, Payload: payload}, nil
}

// ExecPipe sends bytecode directly — no temp files, no disk cache.
// This is the hot path used by `lunex run`.
func (c *Client) ExecPipe(ncData []byte) (Result, error) {
	flags := FlagNoCache
	if c.debug {
		flags |= FlagDebug
	}
	if err := c.sendFrame(MsgExecPipe, flags, ncData); err != nil {
		return Result{}, err
	}
	return c.collectResult()
}

// ExecFile asks Zig to run a .nc file on disk.
func (c *Client) ExecFile(absPath string) (Result, error) {
	payload := append([]byte(absPath), 0x00) // null-terminated
	flags := FlagNone
	if c.debug {
		flags |= FlagDebug
	}
	if err := c.sendFrame(MsgExecFile, flags, payload); err != nil {
		return Result{}, err
	}
	return c.collectResult()
}

// RuntimeInfo asks the Zig process for its version/platform string.
func (c *Client) RuntimeInfo() (string, error) {
	if err := c.sendFrame(MsgRtInfo, FlagNone, nil); err != nil {
		return "", err
	}
	for {
		f, err := c.readFrame()
		if err != nil {
			return "", err
		}
		switch f.MsgType {
		case MsgRtInfoResp:
			return string(f.Payload), nil
		case MsgRespErr:
			ef, err := DecodeErrorFrame(f.Payload)
			if err != nil {
				return "", fmt.Errorf("ncp: malformed error frame: %w", err)
			}
			return "", fmt.Errorf("%s", ef.Msg)
		case MsgEnd:
			return "", fmt.Errorf("ncp: runtime closed without responding to info request")
		}
	}
}

// Kill sends a graceful shutdown. Use defer zigrt.Shutdown() instead of calling this directly.
func (c *Client) Kill() error {
	return c.sendFrame(MsgKill, FlagNone, nil)
}

// collectResult loops over incoming frames until we get an exit or end signal.
func (c *Client) collectResult() (Result, error) {
	var res Result
	for {
		f, err := c.readFrame()
		if err != nil {
			return res, fmt.Errorf("ncp: read error mid-execution: %w", err)
		}
		switch f.MsgType {
		case MsgStdout:
			res.Stdout = append(res.Stdout, f.Payload...)
			os.Stdout.Write(f.Payload)
		case MsgStderr:
			res.Stderr = append(res.Stderr, f.Payload...)
			os.Stderr.Write(f.Payload)
		case MsgRuntimeDirInfo:
			if info, err := DecodeRuntimeDirInfo(f.Payload); err == nil {
				res.RuntimeDir = info.Dir
			}
		case MsgRespExit:
			if len(f.Payload) > 0 {
				res.ExitCode = int(f.Payload[0])
			}
			return res, nil
		case MsgRespErr:
			ef, err := DecodeErrorFrame(f.Payload)
			if err != nil {
				return res, fmt.Errorf("ncp: malformed error frame: %w", err)
			}
			return res, &ZigError{Frame: ef}
		case MsgEnd:
			return res, nil
		case MsgRespOK:
			// just an ack, keep waiting
		default:
			if c.debug {
				fmt.Fprintf(os.Stderr, "[ncp] unknown frame 0x%02X — ignored\n", f.MsgType)
			}
		}
	}
}

// ZigError wraps a runtime ErrorFrame so callers can inspect the details.
type ZigError struct {
	Frame ErrorFrame
}

func (e *ZigError) Error() string {
	if e.Frame.Line > 0 {
		return fmt.Sprintf("runtime error [E%04d] at line %d col %d: %s",
			e.Frame.Code, e.Frame.Line, e.Frame.Column, e.Frame.Msg)
	}
	return fmt.Sprintf("runtime error [E%04d]: %s", e.Frame.Code, e.Frame.Msg)
}

func (e *ZigError) Hint() string { return e.Frame.Hint }

// CodeBadBCFormat is the Zig VM error code returned when the NC container
// holds source text instead of compiled opcodes. The Go interpreter handles
// this case directly.
const CodeBadBCFormat uint16 = 7003

