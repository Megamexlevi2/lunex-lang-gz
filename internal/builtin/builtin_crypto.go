// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"lunex/internal/runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

// Hash object pools to avoid repeated allocations on hot paths.
var (
	md5Pool    = sync.Pool{New: func() any { return md5.New() }}
	sha1Pool   = sync.Pool{New: func() any { return sha1.New() }}
	sha256Pool = sync.Pool{New: func() any { return sha256.New() }}
	sha512Pool = sync.Pool{New: func() any { return sha512.New() }}
)

func hashBytes(algorithm string, data []byte) string {
	switch strings.ToLower(algorithm) {
	case "md5":
		h := md5Pool.Get().(hash.Hash)
		h.Reset()
		h.Write(data)
		sum := hex.EncodeToString(h.Sum(nil))
		md5Pool.Put(h)
		return sum
	case "sha1":
		h := sha1Pool.Get().(hash.Hash)
		h.Reset()
		h.Write(data)
		sum := hex.EncodeToString(h.Sum(nil))
		sha1Pool.Put(h)
		return sum
	case "sha256":
		h := sha256Pool.Get().(hash.Hash)
		h.Reset()
		h.Write(data)
		sum := hex.EncodeToString(h.Sum(nil))
		sha256Pool.Put(h)
		return sum
	case "sha512":
		h := sha512Pool.Get().(hash.Hash)
		h.Reset()
		h.Write(data)
		sum := hex.EncodeToString(h.Sum(nil))
		sha512Pool.Put(h)
		return sum
	default:
		h := sha256Pool.Get().(hash.Hash)
		h.Reset()
		h.Write(data)
		sum := hex.EncodeToString(h.Sum(nil))
		sha256Pool.Put(h)
		return sum
	}
}

func hashString(algorithm, data string) string {
	return hashBytes(algorithm, []byte(data))
}

func hmacHashFunc(algorithm string) func() hash.Hash {
	switch strings.ToLower(algorithm) {
	case "sha256":
		return sha256.New
	case "sha512":
		return sha512.New
	case "md5":
		return md5.New
	case "sha1":
		return sha1.New
	default:
		return sha256.New
	}
}

// hmacBytes computes HMAC and returns raw bytes.
func hmacBytes(algorithm string, key, data []byte) []byte {
	mac := hmac.New(hmacHashFunc(algorithm), key)
	mac.Write(data)
	return mac.Sum(nil)
}

func hmacString(algorithm, key, data string) string {
	return hex.EncodeToString(hmacBytes(algorithm, []byte(key), []byte(data)))
}

// jwtSign creates a standards-compliant JWT (HS256).
// The HMAC-SHA256 is computed over the raw signing input and base64url-encoded
// as raw bytes — matching the JWT spec (RFC 7519 / RFC 7515).
func jwtSign(payload map[string]*runtime.Value, secret string, expiresIn int64) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	now := time.Now().Unix()
	claims := make(map[string]*runtime.Value)
	for k, v := range payload {
		claims[k] = v
	}
	if _, ok := claims["iat"]; !ok {
		claims["iat"] = runtime.NumberVal(float64(now))
	}
	if expiresIn > 0 {
		if _, ok := claims["exp"]; !ok {
			claims["exp"] = runtime.NumberVal(float64(now + expiresIn))
		}
	}
	payloadStr := valueToJSON(runtime.ObjectVal(claims))
	payloadB64 := base64.RawURLEncoding.EncodeToString([]byte(payloadStr))
	sigInput := header + "." + payloadB64
	// Raw binary HMAC-SHA256, then base64url-encode — this is the JWT standard.
	rawSig := hmacBytes("sha256", []byte(secret), []byte(sigInput))
	sigB64 := base64.RawURLEncoding.EncodeToString(rawSig)
	return sigInput + "." + sigB64
}

func jwtVerify(token, secret string) (*runtime.Value, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}
	sigInput := parts[0] + "." + parts[1]
	rawSig := hmacBytes("sha256", []byte(secret), []byte(sigInput))
	expectedB64 := base64.RawURLEncoding.EncodeToString(rawSig)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedB64)) {
		return nil, fmt.Errorf("invalid signature")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding")
	}
	payload, err := parseJSON(string(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("invalid payload JSON")
	}
	if exp, ok := payload.ObjVal["exp"]; ok && exp != nil {
		if time.Now().Unix() > int64(exp.ToNumber()) {
			return nil, fmt.Errorf("token expired")
		}
	}
	return payload, nil
}

func aesEncrypt(plaintext, key string) (string, error) {
	k := make([]byte, 32)
	copy(k, []byte(key))
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func aesDecrypt(ciphertextB64, key string) (string, error) {
	k := make([]byte, 32)
	copy(k, []byte(key))
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// realPBKDF2 uses the industry-standard PBKDF2-HMAC-SHA256 algorithm.
func realPBKDF2(password, salt string, iterations, keyLen int) string {
	key := pbkdf2.Key([]byte(password), []byte(salt), iterations, keyLen, sha256.New)
	return hex.EncodeToString(key)
}

// CryptoModule returns the built-in crypto module.
// All hash operations use sync.Pool to minimise allocations on repeated calls.
func CryptoModule() *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"hash": runtime.FuncVal(&runtime.Function{Name: "hash", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(hashString(args[0].ToString(), args[1].ToString())), nil
		}}),

		"hmac": runtime.FuncVal(&runtime.Function{Name: "hmac", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(hmacString(args[0].ToString(), args[1].ToString(), args[2].ToString())), nil
		}}),

		"md5": runtime.FuncVal(&runtime.Function{Name: "md5", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(hashString("md5", args[0].ToString())), nil
		}}),

		"sha1": runtime.FuncVal(&runtime.Function{Name: "sha1", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(hashString("sha1", args[0].ToString())), nil
		}}),

		"sha256": runtime.FuncVal(&runtime.Function{Name: "sha256", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(hashString("sha256", args[0].ToString())), nil
		}}),

		"sha512": runtime.FuncVal(&runtime.Function{Name: "sha512", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(hashString("sha512", args[0].ToString())), nil
		}}),

		"randomBytes": runtime.FuncVal(&runtime.Function{Name: "randomBytes", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			n := 16
			if len(args) > 0 {
				n = int(args[0].ToNumber())
			}
			b := make([]byte, n)
			rand.Read(b)
			return runtime.StringVal(hex.EncodeToString(b)), nil
		}}),

		"randomHex": runtime.FuncVal(&runtime.Function{Name: "randomHex", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			n := 16
			if len(args) > 0 {
				n = int(args[0].ToNumber())
			}
			b := make([]byte, n)
			rand.Read(b)
			return runtime.StringVal(hex.EncodeToString(b)), nil
		}}),

		"randomUUID": runtime.FuncVal(&runtime.Function{Name: "randomUUID", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.StringVal(genUUID()), nil
		}}),

		"token": runtime.FuncVal(&runtime.Function{Name: "token", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			n := 32
			if len(args) > 0 {
				n = int(args[0].ToNumber())
			}
			b := make([]byte, n)
			rand.Read(b)
			return runtime.StringVal(hex.EncodeToString(b)), nil
		}}),

		"encrypt": runtime.FuncVal(&runtime.Function{Name: "encrypt", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.StringVal(""), nil
			}
			result, err := aesEncrypt(args[0].ToString(), args[1].ToString())
			if err != nil {
				return runtime.StringVal(""), err
			}
			return runtime.StringVal(result), nil
		}}),

		"decrypt": runtime.FuncVal(&runtime.Function{Name: "decrypt", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.StringVal(""), nil
			}
			result, err := aesDecrypt(args[0].ToString(), args[1].ToString())
			if err != nil {
				return runtime.Null, nil
			}
			return runtime.StringVal(result), nil
		}}),

		"toHex": runtime.FuncVal(&runtime.Function{Name: "toHex", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(hex.EncodeToString([]byte(args[0].ToString()))), nil
		}}),

		"fromHex": runtime.FuncVal(&runtime.Function{Name: "fromHex", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			b, err := hex.DecodeString(args[0].ToString())
			if err != nil {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(string(b)), nil
		}}),

		"base64Encode": runtime.FuncVal(&runtime.Function{Name: "base64Encode", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(base64.StdEncoding.EncodeToString([]byte(args[0].ToString()))), nil
		}}),

		"base64Decode": runtime.FuncVal(&runtime.Function{Name: "base64Decode", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			b, err := base64.StdEncoding.DecodeString(args[0].ToString())
			if err != nil {
				b, err = base64.RawStdEncoding.DecodeString(args[0].ToString())
				if err != nil {
					return runtime.StringVal(""), nil
				}
			}
			return runtime.StringVal(string(b)), nil
		}}),

		"base64UrlEncode": runtime.FuncVal(&runtime.Function{Name: "base64UrlEncode", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(base64.RawURLEncoding.EncodeToString([]byte(args[0].ToString()))), nil
		}}),

		"base64UrlDecode": runtime.FuncVal(&runtime.Function{Name: "base64UrlDecode", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			b, err := base64.RawURLEncoding.DecodeString(args[0].ToString())
			if err != nil {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(string(b)), nil
		}}),

		// pbkdf2 uses the standard PBKDF2-HMAC-SHA256 algorithm (RFC 2898).
		// Signature: pbkdf2(password, salt, iterations?, keyLen?)
		"pbkdf2": runtime.FuncVal(&runtime.Function{Name: "pbkdf2", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.StringVal(""), nil
			}
			password := args[0].ToString()
			salt := args[1].ToString()
			iterations := 100000
			keyLen := 32
			if len(args) > 2 {
				iterations = int(args[2].ToNumber())
			}
			if len(args) > 3 {
				keyLen = int(args[3].ToNumber())
			}
			return runtime.StringVal(realPBKDF2(password, salt, iterations, keyLen)), nil
		}}),

		// hashPassword uses bcrypt for secure password storage.
		// Returns a bcrypt hash that includes the salt automatically.
		"hashPassword": runtime.FuncVal(&runtime.Function{Name: "hashPassword", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			password := args[0].ToString()
			cost := bcrypt.DefaultCost
			if len(args) > 1 {
				c := int(args[1].ToNumber())
				if c >= bcrypt.MinCost && c <= bcrypt.MaxCost {
					cost = c
				}
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
			if err != nil {
				return runtime.StringVal(""), err
			}
			return runtime.StringVal(string(hash)), nil
		}}),

		// verifyPassword checks a bcrypt hash produced by hashPassword.
		"verifyPassword": runtime.FuncVal(&runtime.Function{Name: "verifyPassword", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.False, nil
			}
			password := args[0].ToString()
			hashed := args[1].ToString()
			err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
			return runtime.BoolVal(err == nil), nil
		}}),

		"jwt": runtime.ObjectVal(map[string]*runtime.Value{
			"sign": runtime.FuncVal(&runtime.Function{Name: "sign", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) < 2 {
					return runtime.StringVal(""), fmt.Errorf("jwt.sign: payload and secret required")
				}
				payload := make(map[string]*runtime.Value)
				if args[0].Tag == runtime.TypeObject {
					for k, v := range args[0].ObjVal {
						payload[k] = v
					}
				}
				secret := args[1].ToString()
				expiresIn := int64(3600)
				if len(args) > 2 && args[2].Tag == runtime.TypeObject {
					if exp, ok := args[2].ObjVal["expiresIn"]; ok {
						expiresIn = int64(exp.ToNumber())
					}
					if exp, ok := args[2].ObjVal["expires"]; ok {
						expiresIn = int64(exp.ToNumber())
					}
				}
				return runtime.StringVal(jwtSign(payload, secret, expiresIn)), nil
			}}),
			"verify": runtime.FuncVal(&runtime.Function{Name: "verify", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) < 2 {
					return runtime.Null, fmt.Errorf("jwt.verify: token and secret required")
				}
				payload, err := jwtVerify(args[0].ToString(), args[1].ToString())
				if err != nil {
					return runtime.Null, err
				}
				return payload, nil
			}}),
			"decode": runtime.FuncVal(&runtime.Function{Name: "decode", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) == 0 {
					return runtime.Null, nil
				}
				parts := strings.Split(args[0].ToString(), ".")
				if len(parts) != 3 {
					return runtime.Null, nil
				}
				payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
				if err != nil {
					return runtime.Null, nil
				}
				return parseJSON(string(payloadBytes))
			}}),
		}),

		// compare does a constant-time comparison to prevent timing attacks.
		"compare": runtime.FuncVal(&runtime.Function{Name: "compare", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.False, nil
			}
			a := []byte(args[0].ToString())
			b := []byte(args[1].ToString())
			return runtime.BoolVal(hmac.Equal(a, b)), nil
		}}),
	})
}
