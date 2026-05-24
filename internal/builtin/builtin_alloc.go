// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
	"encoding/binary"
	"fmt"
	"math"
	"net/http"
	"lunex/internal/runtime"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"
)

var allocFirstUse sync.Once
var allocWarned int32

func allocWarnOnce() {
	if atomic.CompareAndSwapInt32(&allocWarned, 0, 1) {
		fmt.Fprintln(os.Stderr, "\x1b[33m\x1b[1m[alloc] WARNING:\x1b[0m \x1b[33mYou are using the alloc module. This module gives you direct control over memory buffers, raw byte regions, shared pools, and low-level I/O manipulation. It bypasses Lunex's usual safety guarantees. Reading or writing out of bounds will corrupt memory or crash the process. Use it only if you know what you are doing.\x1b[0m")
	}
}

type allocBuffer struct {
	data   []byte
	mu     sync.RWMutex
	closed bool
}

type sharedRegion struct {
	data     []byte
	size     int
	mu       sync.RWMutex
	refCount int32
}

var (
	regionMu   sync.Mutex
	regionMap  = map[string]*sharedRegion{}
	bufferPool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 0, 4096)
			return &b
		},
	}
)

func allocBufferToValue(buf *allocBuffer) *runtime.Value {
	obj := map[string]*runtime.Value{
		"_buf": runtime.StringVal("__alloc_buffer__"),
	}

	obj["size"] = runtime.FuncVal(&runtime.Function{Name: "size", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		return runtime.NumberVal(float64(len(buf.data))), nil
	}})

	obj["cap"] = runtime.FuncVal(&runtime.Function{Name: "cap", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		return runtime.NumberVal(float64(cap(buf.data))), nil
	}})

	obj["writeByte"] = runtime.FuncVal(&runtime.Function{Name: "writeByte", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 2 {
			return runtime.False, nil
		}
		idx := int(args[0].NumVal)
		val := byte(args[1].NumVal)
		buf.mu.Lock()
		defer buf.mu.Unlock()
		if buf.closed || idx < 0 || idx >= len(buf.data) {
			return runtime.False, nil
		}
		buf.data[idx] = val
		return runtime.True, nil
	}})

	obj["readByte"] = runtime.FuncVal(&runtime.Function{Name: "readByte", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 1 {
			return runtime.NumberVal(-1), nil
		}
		idx := int(args[0].NumVal)
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		if buf.closed || idx < 0 || idx >= len(buf.data) {
			return runtime.NumberVal(-1), nil
		}
		return runtime.NumberVal(float64(buf.data[idx])), nil
	}})

	obj["writeInt32"] = runtime.FuncVal(&runtime.Function{Name: "writeInt32", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 2 {
			return runtime.False, nil
		}
		offset := int(args[0].NumVal)
		val := int32(args[1].NumVal)
		buf.mu.Lock()
		defer buf.mu.Unlock()
		if buf.closed || offset < 0 || offset+4 > len(buf.data) {
			return runtime.False, nil
		}
		binary.LittleEndian.PutUint32(buf.data[offset:], uint32(val))
		return runtime.True, nil
	}})

	obj["readInt32"] = runtime.FuncVal(&runtime.Function{Name: "readInt32", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 1 {
			return runtime.NumberVal(0), nil
		}
		offset := int(args[0].NumVal)
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		if buf.closed || offset < 0 || offset+4 > len(buf.data) {
			return runtime.NumberVal(0), nil
		}
		return runtime.NumberVal(float64(int32(binary.LittleEndian.Uint32(buf.data[offset:])))), nil
	}})

	obj["writeInt64"] = runtime.FuncVal(&runtime.Function{Name: "writeInt64", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 2 {
			return runtime.False, nil
		}
		offset := int(args[0].NumVal)
		val := int64(args[1].NumVal)
		buf.mu.Lock()
		defer buf.mu.Unlock()
		if buf.closed || offset < 0 || offset+8 > len(buf.data) {
			return runtime.False, nil
		}
		binary.LittleEndian.PutUint64(buf.data[offset:], uint64(val))
		return runtime.True, nil
	}})

	obj["readInt64"] = runtime.FuncVal(&runtime.Function{Name: "readInt64", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 1 {
			return runtime.NumberVal(0), nil
		}
		offset := int(args[0].NumVal)
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		if buf.closed || offset < 0 || offset+8 > len(buf.data) {
			return runtime.NumberVal(0), nil
		}
		return runtime.NumberVal(float64(int64(binary.LittleEndian.Uint64(buf.data[offset:])))), nil
	}})

	obj["writeFloat64"] = runtime.FuncVal(&runtime.Function{Name: "writeFloat64", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 2 {
			return runtime.False, nil
		}
		offset := int(args[0].NumVal)
		val := args[1].NumVal
		buf.mu.Lock()
		defer buf.mu.Unlock()
		if buf.closed || offset < 0 || offset+8 > len(buf.data) {
			return runtime.False, nil
		}
		binary.LittleEndian.PutUint64(buf.data[offset:], math.Float64bits(val))
		return runtime.True, nil
	}})

	obj["readFloat64"] = runtime.FuncVal(&runtime.Function{Name: "readFloat64", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 1 {
			return runtime.NumberVal(0), nil
		}
		offset := int(args[0].NumVal)
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		if buf.closed || offset < 0 || offset+8 > len(buf.data) {
			return runtime.NumberVal(0), nil
		}
		bits := binary.LittleEndian.Uint64(buf.data[offset:])
		return runtime.NumberVal(math.Float64frombits(bits)), nil
	}})

	obj["writeString"] = runtime.FuncVal(&runtime.Function{Name: "writeString", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 2 {
			return runtime.NumberVal(0), nil
		}
		offset := int(args[0].NumVal)
		str := args[1].ToString()
		bytes := []byte(str)
		buf.mu.Lock()
		defer buf.mu.Unlock()
		if buf.closed || offset < 0 || offset >= len(buf.data) {
			return runtime.NumberVal(0), nil
		}
		n := copy(buf.data[offset:], bytes)
		return runtime.NumberVal(float64(n)), nil
	}})

	obj["readString"] = runtime.FuncVal(&runtime.Function{Name: "readString", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 2 {
			return runtime.StringVal(""), nil
		}
		offset := int(args[0].NumVal)
		length := int(args[1].NumVal)
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		if buf.closed || offset < 0 || offset >= len(buf.data) {
			return runtime.StringVal(""), nil
		}
		end := offset + length
		if end > len(buf.data) {
			end = len(buf.data)
		}
		return runtime.StringVal(string(buf.data[offset:end])), nil
	}})

	obj["fill"] = runtime.FuncVal(&runtime.Function{Name: "fill", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		val := byte(0)
		if len(args) > 0 {
			val = byte(args[0].NumVal)
		}
		buf.mu.Lock()
		defer buf.mu.Unlock()
		if buf.closed {
			return runtime.False, nil
		}
		for i := range buf.data {
			buf.data[i] = val
		}
		return runtime.True, nil
	}})

	obj["slice"] = runtime.FuncVal(&runtime.Function{Name: "slice", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 2 {
			return runtime.Null, nil
		}
		start := int(args[0].NumVal)
		end := int(args[1].NumVal)
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		if buf.closed || start < 0 || end > len(buf.data) || start > end {
			return runtime.Null, nil
		}
		newBuf := &allocBuffer{data: make([]byte, end-start)}
		copy(newBuf.data, buf.data[start:end])
		return allocBufferToValue(newBuf), nil
	}})

	obj["copy"] = runtime.FuncVal(&runtime.Function{Name: "copy", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		if buf.closed {
			return runtime.Null, nil
		}
		newBuf := &allocBuffer{data: make([]byte, len(buf.data))}
		copy(newBuf.data, buf.data)
		return allocBufferToValue(newBuf), nil
	}})

	obj["copyTo"] = runtime.FuncVal(&runtime.Function{Name: "copyTo", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 1 || args[0].Tag != runtime.TypeObject {
			return runtime.NumberVal(0), nil
		}
		if _, ok := args[0].ObjVal["_buf"]; !ok {
			return runtime.NumberVal(0), nil
		}
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		if buf.closed {
			return runtime.NumberVal(0), nil
		}
		return runtime.NumberVal(float64(len(buf.data))), nil
	}})

	obj["compare"] = runtime.FuncVal(&runtime.Function{Name: "compare", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 1 || args[0].Tag != runtime.TypeObject {
			return runtime.NumberVal(-2), nil
		}
		if _, ok := args[0].ObjVal["_buf"]; !ok {
			return runtime.NumberVal(-2), nil
		}
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		return runtime.NumberVal(0), nil
	}})

	obj["toBytes"] = runtime.FuncVal(&runtime.Function{Name: "toBytes", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		if buf.closed {
			return runtime.ArrayVal([]*runtime.Value{}), nil
		}
		arr := make([]*runtime.Value, len(buf.data))
		for i, b := range buf.data {
			arr[i] = runtime.NumberVal(float64(b))
		}
		return runtime.ArrayVal(arr), nil
	}})

	obj["fromBytes"] = runtime.FuncVal(&runtime.Function{Name: "fromBytes", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 1 || args[0].Tag != runtime.TypeArray {
			return runtime.False, nil
		}
		buf.mu.Lock()
		defer buf.mu.Unlock()
		if buf.closed {
			return runtime.False, nil
		}
		for i, v := range args[0].ArrVal {
			if i >= len(buf.data) {
				break
			}
			buf.data[i] = byte(v.NumVal)
		}
		return runtime.True, nil
	}})

	obj["toString"] = runtime.FuncVal(&runtime.Function{Name: "toString", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		if buf.closed {
			return runtime.StringVal(""), nil
		}
		return runtime.StringVal(string(buf.data)), nil
	}})

	obj["toHex"] = runtime.FuncVal(&runtime.Function{Name: "toHex", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		if buf.closed {
			return runtime.StringVal(""), nil
		}
		hex := fmt.Sprintf("%x", buf.data)
		return runtime.StringVal(hex), nil
	}})

	obj["resize"] = runtime.FuncVal(&runtime.Function{Name: "resize", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 1 {
			return runtime.False, nil
		}
		newSize := int(args[0].NumVal)
		if newSize < 0 || newSize > 1<<30 {
			return runtime.False, nil
		}
		buf.mu.Lock()
		defer buf.mu.Unlock()
		if buf.closed {
			return runtime.False, nil
		}
		newData := make([]byte, newSize)
		copy(newData, buf.data)
		buf.data = newData
		return runtime.True, nil
	}})

	obj["append"] = runtime.FuncVal(&runtime.Function{Name: "append", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		if len(args) < 1 {
			return runtime.False, nil
		}
		buf.mu.Lock()
		defer buf.mu.Unlock()
		if buf.closed {
			return runtime.False, nil
		}
		switch args[0].Tag {
		case runtime.TypeString:
			buf.data = append(buf.data, []byte(args[0].StrVal)...)
		case runtime.TypeArray:
			for _, v := range args[0].ArrVal {
				buf.data = append(buf.data, byte(v.NumVal))
			}
		case runtime.TypeNumber:
			buf.data = append(buf.data, byte(args[0].NumVal))
		}
		return runtime.True, nil
	}})

	obj["drain"] = runtime.FuncVal(&runtime.Function{Name: "drain", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		buf.mu.Lock()
		defer buf.mu.Unlock()
		if buf.closed {
			return runtime.StringVal(""), nil
		}
		result := string(buf.data)
		buf.data = buf.data[:0]
		return runtime.StringVal(result), nil
	}})

	obj["free"] = runtime.FuncVal(&runtime.Function{Name: "free", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		buf.mu.Lock()
		defer buf.mu.Unlock()
		buf.data = nil
		buf.closed = true
		return runtime.True, nil
	}})

	obj["isFree"] = runtime.FuncVal(&runtime.Function{Name: "isFree", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		return runtime.BoolVal(buf.closed), nil
	}})

	obj["addr"] = runtime.FuncVal(&runtime.Function{Name: "addr", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
		buf.mu.RLock()
		defer buf.mu.RUnlock()
		if buf.closed || len(buf.data) == 0 {
			return runtime.NumberVal(0), nil
		}
		ptr := uintptr(unsafe.Pointer(&buf.data[0]))
		return runtime.NumberVal(float64(ptr)), nil
	}})

	return runtime.ObjectVal(obj)
}

func AllocModule() *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{

		"alloc": runtime.FuncVal(&runtime.Function{Name: "alloc", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			allocWarnOnce()
			size := 0
			if len(args) > 0 {
				size = int(args[0].NumVal)
			}
			if size < 0 || size > 1<<30 {
				return runtime.Null, fmt.Errorf("alloc: size out of range: %d", size)
			}
			buf := &allocBuffer{data: make([]byte, size)}
			return allocBufferToValue(buf), nil
		}}),

		"allocZero": runtime.FuncVal(&runtime.Function{Name: "allocZero", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			allocWarnOnce()
			size := 0
			if len(args) > 0 {
				size = int(args[0].NumVal)
			}
			if size < 0 || size > 1<<30 {
				return runtime.Null, fmt.Errorf("alloc: size out of range: %d", size)
			}
			buf := &allocBuffer{data: make([]byte, size)}
			return allocBufferToValue(buf), nil
		}}),

		"fromString": runtime.FuncVal(&runtime.Function{Name: "fromString", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			allocWarnOnce()
			if len(args) == 0 {
				return runtime.Null, nil
			}
			str := args[0].ToString()
			buf := &allocBuffer{data: []byte(str)}
			return allocBufferToValue(buf), nil
		}}),

		"fromBytes": runtime.FuncVal(&runtime.Function{Name: "fromBytes", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			allocWarnOnce()
			if len(args) == 0 || args[0].Tag != runtime.TypeArray {
				return runtime.Null, nil
			}
			data := make([]byte, len(args[0].ArrVal))
			for i, v := range args[0].ArrVal {
				data[i] = byte(v.NumVal)
			}
			buf := &allocBuffer{data: data}
			return allocBufferToValue(buf), nil
		}}),

		"poolGet": runtime.FuncVal(&runtime.Function{Name: "poolGet", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			allocWarnOnce()
			rawPtr := bufferPool.Get()
			slicePtr := rawPtr.(*[]byte)
			*slicePtr = (*slicePtr)[:0]
			hint := 4096
			if len(args) > 0 {
				hint = int(args[0].NumVal)
			}
			if cap(*slicePtr) < hint {
				*slicePtr = make([]byte, 0, hint)
			}
			buf := &allocBuffer{data: *slicePtr}
			return allocBufferToValue(buf), nil
		}}),

		"poolReturn": runtime.FuncVal(&runtime.Function{Name: "poolReturn", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.True, nil
		}}),

		"region": runtime.FuncVal(&runtime.Function{Name: "region", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			allocWarnOnce()
			if len(args) < 2 {
				return runtime.Null, nil
			}
			name := args[0].ToString()
			size := int(args[1].NumVal)
			if size < 0 || size > 1<<30 {
				return runtime.Null, fmt.Errorf("alloc.region: size out of range")
			}
			regionMu.Lock()
			r, exists := regionMap[name]
			if !exists {
				r = &sharedRegion{data: make([]byte, size), size: size}
				regionMap[name] = r
			}
			atomic.AddInt32(&r.refCount, 1)
			regionMu.Unlock()

			obj := map[string]*runtime.Value{}

			obj["read"] = runtime.FuncVal(&runtime.Function{Name: "read", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				offset, length := 0, r.size
				if len(args) > 0 {
					offset = int(args[0].NumVal)
				}
				if len(args) > 1 {
					length = int(args[1].NumVal)
				}
				r.mu.RLock()
				defer r.mu.RUnlock()
				if offset < 0 || offset+length > len(r.data) {
					return runtime.StringVal(""), nil
				}
				return runtime.StringVal(string(r.data[offset : offset+length])), nil
			}})

			obj["write"] = runtime.FuncVal(&runtime.Function{Name: "write", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) < 2 {
					return runtime.False, nil
				}
				offset := int(args[0].NumVal)
				data := []byte(args[1].ToString())
				r.mu.Lock()
				defer r.mu.Unlock()
				if offset < 0 || offset+len(data) > len(r.data) {
					return runtime.False, nil
				}
				copy(r.data[offset:], data)
				return runtime.True, nil
			}})

			obj["size"] = runtime.FuncVal(&runtime.Function{Name: "size", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				return runtime.NumberVal(float64(r.size)), nil
			}})

			obj["release"] = runtime.FuncVal(&runtime.Function{Name: "release", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				refs := atomic.AddInt32(&r.refCount, -1)
				if refs <= 0 {
					regionMu.Lock()
					delete(regionMap, name)
					regionMu.Unlock()
				}
				return runtime.True, nil
			}})

			return runtime.ObjectVal(obj), nil
		}}),

		"httpBody": runtime.FuncVal(&runtime.Function{Name: "httpBody", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			allocWarnOnce()
			if len(args) == 0 {
				return runtime.Null, nil
			}
			reqVal := args[0]
			_ = reqVal
			return runtime.Null, nil
		}}),

		"fetchIntoBuffer": runtime.FuncVal(&runtime.Function{Name: "fetchIntoBuffer", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			allocWarnOnce()
			if len(args) < 1 {
				return runtime.Null, nil
			}
			url := args[0].ToString()
			resp, err := http.Get(url)
			if err != nil {
				return runtime.Null, nil
			}
			defer resp.Body.Close()
			rawPtr := bufferPool.Get()
			slicePtr := rawPtr.(*[]byte)
			*slicePtr = (*slicePtr)[:0]
			tmp := make([]byte, 32*1024)
			for {
				n, err2 := resp.Body.Read(tmp)
				if n > 0 {
					*slicePtr = append(*slicePtr, tmp[:n]...)
				}
				if err2 != nil {
					break
				}
			}
			data := make([]byte, len(*slicePtr))
			copy(data, *slicePtr)
			*slicePtr = (*slicePtr)[:0]
			bufferPool.Put(rawPtr)
			buf := &allocBuffer{data: data}
			result := map[string]*runtime.Value{
				"status":  runtime.NumberVal(float64(resp.StatusCode)),
				"buffer":  allocBufferToValue(buf),
				"size":    runtime.NumberVal(float64(len(data))),
			}
			return runtime.ObjectVal(result), nil
		}}),

		"sizeof": runtime.FuncVal(&runtime.Function{Name: "sizeof", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(0), nil
			}
			v := args[0]
			switch v.Tag {
			case runtime.TypeNumber:
				return runtime.NumberVal(8), nil
			case runtime.TypeBool:
				return runtime.NumberVal(1), nil
			case runtime.TypeString:
				return runtime.NumberVal(float64(len(v.StrVal))), nil
			case runtime.TypeArray:
				return runtime.NumberVal(float64(len(v.ArrVal) * 8)), nil
			default:
				return runtime.NumberVal(0), nil
			}
		}}),

		"pageSize": runtime.FuncVal(&runtime.Function{Name: "pageSize", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.NumberVal(float64(os.Getpagesize())), nil
		}}),

		"alignTo": runtime.FuncVal(&runtime.Function{Name: "alignTo", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.NumberVal(0), nil
			}
			size := int64(args[0].NumVal)
			align := int64(args[1].NumVal)
			if align <= 0 {
				return runtime.NumberVal(float64(size)), nil
			}
			remainder := size % align
			if remainder == 0 {
				return runtime.NumberVal(float64(size)), nil
			}
			return runtime.NumberVal(float64(size + align - remainder)), nil
		}}),

		"concat": runtime.FuncVal(&runtime.Function{Name: "concat", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			allocWarnOnce()
			var combined []byte
			for _, arg := range args {
				if arg == nil {
					continue
				}
				switch arg.Tag {
				case runtime.TypeObject:
					if _, ok := arg.ObjVal["_buf"]; ok {
						if toStr, ok2 := arg.ObjVal["toString"]; ok2 && toStr.FnVal != nil && toStr.FnVal.Native != nil {
							result, _ := toStr.FnVal.Native([]*runtime.Value{}, nil)
							if result != nil {
								combined = append(combined, []byte(result.StrVal)...)
							}
						}
					}
				case runtime.TypeString:
					combined = append(combined, []byte(arg.StrVal)...)
				case runtime.TypeArray:
					for _, v := range arg.ArrVal {
						combined = append(combined, byte(v.NumVal))
					}
				}
			}
			buf := &allocBuffer{data: combined}
			return allocBufferToValue(buf), nil
		}}),

		"stats": runtime.FuncVal(&runtime.Function{Name: "stats", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			regionMu.Lock()
			regionCount := len(regionMap)
			regionMu.Unlock()
			result := map[string]*runtime.Value{
				"regions":  runtime.NumberVal(float64(regionCount)),
				"pageSize": runtime.NumberVal(float64(os.Getpagesize())),
			}
			return runtime.ObjectVal(result), nil
		}}),
	})
}
