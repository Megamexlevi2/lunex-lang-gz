// NT-IDE — Lunex Integrated Development Environment
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package ide

import (
	"sort"
	"strings"
)

// CompletionKind describes what type of completion an item is
type CompletionKind int

const (
	KindKeyword   CompletionKind = iota
	KindFunction
	KindVariable
	KindModule
	KindMethod
	KindSnippet
	KindBuiltin
	KindProperty
)

func (k CompletionKind) Icon() string {
	switch k {
	case KindKeyword:
		return "kw"
	case KindFunction:
		return "fn"
	case KindVariable:
		return "va"
	case KindModule:
		return "md"
	case KindMethod:
		return "me"
	case KindSnippet:
		return "sn"
	case KindBuiltin:
		return "bi"
	case KindProperty:
		return "pr"
	default:
		return "  "
	}
}

func (k CompletionKind) Color() string {
	switch k {
	case KindKeyword:
		return colorKeyword
	case KindFunction:
		return colorFnName
	case KindVariable:
		return colorIdent
	case KindModule:
		return colorBuiltin
	case KindMethod:
		return colorCyan
	case KindSnippet:
		return colorPurple
	case KindBuiltin:
		return colorBuiltin
	case KindProperty:
		return colorTeal
	default:
		return colorIdent
	}
}

// CompletionItem is a single autocomplete suggestion
type CompletionItem struct {
	Label       string
	Detail      string
	Kind        CompletionKind
	InsertText  string
	TriggerChar string // char that triggered this (e.g. ".")
}

// AutocompleteState holds the current autocomplete UI state
type AutocompleteState struct {
	Active   bool
	Items    []CompletionItem
	Selected int
	Word     string     // the current partial word being typed
	Receiver string     // receiver before dot (e.g. "io" in "io.l")
	CursorX  int
	CursorY  int
	MaxItems int
	ScrollY  int
}

func newAutocompleteState() AutocompleteState {
	return AutocompleteState{MaxItems: 10}
}

func (ac *AutocompleteState) Reset() {
	ac.Active = false
	ac.Items = nil
	ac.Selected = 0
	ac.Word = ""
	ac.Receiver = ""
	ac.ScrollY = 0
}

func (ac *AutocompleteState) MoveUp() {
	if len(ac.Items) == 0 {
		return
	}
	ac.Selected--
	if ac.Selected < 0 {
		ac.Selected = len(ac.Items) - 1
	}
	ac.adjustScroll()
}

func (ac *AutocompleteState) MoveDown() {
	if len(ac.Items) == 0 {
		return
	}
	ac.Selected = (ac.Selected + 1) % len(ac.Items)
	ac.adjustScroll()
}

func (ac *AutocompleteState) adjustScroll() {
	if ac.Selected < ac.ScrollY {
		ac.ScrollY = ac.Selected
	}
	if ac.Selected >= ac.ScrollY+ac.MaxItems {
		ac.ScrollY = ac.Selected - ac.MaxItems + 1
	}
}

func (ac *AutocompleteState) Current() *CompletionItem {
	if !ac.Active || len(ac.Items) == 0 {
		return nil
	}
	if ac.Selected >= len(ac.Items) {
		ac.Selected = 0
	}
	return &ac.Items[ac.Selected]
}

// --- Completion sources ---

// Lunex keywords
var ntlKeywords = []CompletionItem{
	{Label: "val", Detail: "immutable binding", Kind: KindKeyword, InsertText: "val "},
	{Label: "var", Detail: "mutable binding", Kind: KindKeyword, InsertText: "var "},
	{Label: "fn", Detail: "function declaration", Kind: KindKeyword, InsertText: "fn "},
	{Label: "if", Detail: "conditional", Kind: KindSnippet, InsertText: "if  {\n  \n}"},
	{Label: "else", Detail: "else branch", Kind: KindKeyword, InsertText: "else {\n  \n}"},
	{Label: "elif", Detail: "else if", Kind: KindKeyword, InsertText: "elif  {\n  \n}"},
	{Label: "unless", Detail: "unless condition", Kind: KindKeyword, InsertText: "unless  {\n  \n}"},
	{Label: "while", Detail: "while loop", Kind: KindSnippet, InsertText: "while  {\n  \n}"},
	{Label: "for", Detail: "for loop", Kind: KindKeyword, InsertText: "for "},
	{Label: "each", Detail: "iterate over collection", Kind: KindSnippet, InsertText: "each item in  {\n  \n}"},
	{Label: "in", Detail: "in operator", Kind: KindKeyword, InsertText: "in "},
	{Label: "of", Detail: "of operator", Kind: KindKeyword, InsertText: "of "},
	{Label: "break", Detail: "break loop", Kind: KindKeyword, InsertText: "break"},
	{Label: "continue", Detail: "continue loop", Kind: KindKeyword, InsertText: "continue"},
	{Label: "return", Detail: "return value (last expr auto-returned)", Kind: KindKeyword, InsertText: "return "},
	{Label: "match", Detail: "pattern matching", Kind: KindSnippet, InsertText: "match  {\n  case  => \n  default => \n}"},
	{Label: "case", Detail: "match case", Kind: KindKeyword, InsertText: "case  => "},
	{Label: "default", Detail: "default case", Kind: KindKeyword, InsertText: "default => "},
	{Label: "try", Detail: "error handling", Kind: KindSnippet, InsertText: "try {\n  \n} catch err {\n  \n}"},
	{Label: "catch", Detail: "catch error", Kind: KindKeyword, InsertText: "catch err {\n  \n}"},
	{Label: "finally", Detail: "finally block", Kind: KindKeyword, InsertText: "finally {\n  \n}"},
	{Label: "throw", Detail: "throw error", Kind: KindKeyword, InsertText: "throw "},
	{Label: "raise", Detail: "raise error", Kind: KindKeyword, InsertText: "raise "},
	{Label: "struct", Detail: "define a struct type", Kind: KindSnippet, InsertText: "struct {\n  fn new() {\n    {}\n  }\n}"},
	{Label: "new", Detail: "create instance", Kind: KindKeyword, InsertText: "new "},
	{Label: "this", Detail: "current instance", Kind: KindKeyword, InsertText: "this"},
	{Label: "super", Detail: "parent struct", Kind: KindKeyword, InsertText: "super"},
	{Label: "static", Detail: "static member", Kind: KindKeyword, InsertText: "static "},
	{Label: "abstract", Detail: "abstract member", Kind: KindKeyword, InsertText: "abstract "},
	{Label: "override", Detail: "override method", Kind: KindKeyword, InsertText: "override "},
	{Label: "import", Detail: "import module", Kind: KindSnippet, InsertText: "import "},
	{Label: "export", Detail: "export symbol", Kind: KindKeyword, InsertText: "export "},
	{Label: "from", Detail: "import from", Kind: KindKeyword, InsertText: "from "},
	{Label: "as", Detail: "alias", Kind: KindKeyword, InsertText: "as "},
	{Label: "null", Detail: "null value", Kind: KindKeyword, InsertText: "null"},
	{Label: "true", Detail: "boolean true", Kind: KindKeyword, InsertText: "true"},
	{Label: "false", Detail: "boolean false", Kind: KindKeyword, InsertText: "false"},
	{Label: "and", Detail: "logical and", Kind: KindKeyword, InsertText: "and "},
	{Label: "or", Detail: "logical or", Kind: KindKeyword, InsertText: "or "},
	{Label: "not", Detail: "logical not", Kind: KindKeyword, InsertText: "not "},
	{Label: "range", Detail: "range(start, end, step)", Kind: KindBuiltin, InsertText: "range("},
	{Label: "sleep", Detail: "sleep(ms)", Kind: KindBuiltin, InsertText: "sleep("},
	{Label: "spawn", Detail: "spawn goroutine", Kind: KindKeyword, InsertText: "spawn "},
	{Label: "channel", Detail: "create channel", Kind: KindBuiltin, InsertText: "channel()"},
	{Label: "loop", Detail: "infinite loop", Kind: KindSnippet, InsertText: "loop {\n  \n}"},
	{Label: "repeat", Detail: "repeat N times", Kind: KindSnippet, InsertText: "repeat  {\n  \n}"},
	{Label: "guard", Detail: "guard condition", Kind: KindSnippet, InsertText: "guard  else { }"},
	{Label: "defer", Detail: "defer execution", Kind: KindKeyword, InsertText: "defer "},
	{Label: "typeof", Detail: "get type string", Kind: KindBuiltin, InsertText: "typeof "},
	{Label: "instanceof", Detail: "check instance", Kind: KindKeyword, InsertText: "instanceof "},
	{Label: "delete", Detail: "delete property", Kind: KindKeyword, InsertText: "delete "},
}

// Standard module methods by receiver
var moduleCompletions = map[string][]CompletionItem{
	"io": {
		{Label: "log", Detail: "log(msg) — print to stdout", Kind: KindMethod, InsertText: "log("},
		{Label: "print", Detail: "print(msg) — print without newline", Kind: KindMethod, InsertText: "print("},
		{Label: "error", Detail: "error(msg) — print to stderr", Kind: KindMethod, InsertText: "error("},
		{Label: "warn", Detail: "warn(msg) — print warning", Kind: KindMethod, InsertText: "warn("},
		{Label: "readLine", Detail: "readLine() — read a line from stdin", Kind: KindMethod, InsertText: "readLine()"},
		{Label: "readAll", Detail: "readAll() — read all stdin", Kind: KindMethod, InsertText: "readAll()"},
		{Label: "write", Detail: "write(msg) — write to stdout", Kind: KindMethod, InsertText: "write("},
	},
	"fs": {
		{Label: "read", Detail: "read(path) — read file contents", Kind: KindMethod, InsertText: "read("},
		{Label: "write", Detail: "write(path, data) — write file", Kind: KindMethod, InsertText: "write("},
		{Label: "append", Detail: "append(path, data) — append to file", Kind: KindMethod, InsertText: "append("},
		{Label: "exists", Detail: "exists(path) — check if file exists", Kind: KindMethod, InsertText: "exists("},
		{Label: "mkdir", Detail: "mkdir(path) — create directory", Kind: KindMethod, InsertText: "mkdir("},
		{Label: "rm", Detail: "rm(path) — remove file/dir", Kind: KindMethod, InsertText: "rm("},
		{Label: "stat", Detail: "stat(path) — get file info", Kind: KindMethod, InsertText: "stat("},
		{Label: "readDir", Detail: "readDir(path) — list directory", Kind: KindMethod, InsertText: "readDir("},
		{Label: "copy", Detail: "copy(src, dst) — copy file", Kind: KindMethod, InsertText: "copy("},
		{Label: "move", Detail: "move(src, dst) — move file", Kind: KindMethod, InsertText: "move("},
		{Label: "readJSON", Detail: "readJSON(path) — read and parse JSON", Kind: KindMethod, InsertText: "readJSON("},
		{Label: "writeJSON", Detail: "writeJSON(path, obj) — write JSON", Kind: KindMethod, InsertText: "writeJSON("},
	},
	"http": {
		{Label: "get", Detail: "get(url, opts?) — HTTP GET", Kind: KindMethod, InsertText: "get("},
		{Label: "post", Detail: "post(url, body, opts?) — HTTP POST", Kind: KindMethod, InsertText: "post("},
		{Label: "put", Detail: "put(url, body, opts?) — HTTP PUT", Kind: KindMethod, InsertText: "put("},
		{Label: "delete", Detail: "delete(url, opts?) — HTTP DELETE", Kind: KindMethod, InsertText: "delete("},
		{Label: "patch", Detail: "patch(url, body, opts?) — HTTP PATCH", Kind: KindMethod, InsertText: "patch("},
		{Label: "serve", Detail: "serve(port, handler) — start HTTP server", Kind: KindMethod, InsertText: "serve("},
		{Label: "router", Detail: "router() — create router", Kind: KindMethod, InsertText: "router()"},
		{Label: "listen", Detail: "listen(port) — listen on port", Kind: KindMethod, InsertText: "listen("},
	},
	"math": {
		{Label: "abs", Detail: "abs(n) — absolute value", Kind: KindMethod, InsertText: "abs("},
		{Label: "ceil", Detail: "ceil(n) — round up", Kind: KindMethod, InsertText: "ceil("},
		{Label: "floor", Detail: "floor(n) — round down", Kind: KindMethod, InsertText: "floor("},
		{Label: "round", Detail: "round(n) — round to nearest", Kind: KindMethod, InsertText: "round("},
		{Label: "max", Detail: "max(...n) — maximum value", Kind: KindMethod, InsertText: "max("},
		{Label: "min", Detail: "min(...n) — minimum value", Kind: KindMethod, InsertText: "min("},
		{Label: "sqrt", Detail: "sqrt(n) — square root", Kind: KindMethod, InsertText: "sqrt("},
		{Label: "pow", Detail: "pow(base, exp) — power", Kind: KindMethod, InsertText: "pow("},
		{Label: "random", Detail: "random() — random 0..1", Kind: KindMethod, InsertText: "random()"},
		{Label: "PI", Detail: "π constant", Kind: KindProperty, InsertText: "PI"},
		{Label: "E", Detail: "Euler's number", Kind: KindProperty, InsertText: "E"},
		{Label: "log", Detail: "log(n) — natural log", Kind: KindMethod, InsertText: "log("},
		{Label: "log2", Detail: "log2(n) — log base 2", Kind: KindMethod, InsertText: "log2("},
		{Label: "log10", Detail: "log10(n) — log base 10", Kind: KindMethod, InsertText: "log10("},
	},
	"Math": {
		{Label: "abs", Detail: "abs(n)", Kind: KindMethod, InsertText: "abs("},
		{Label: "ceil", Detail: "ceil(n)", Kind: KindMethod, InsertText: "ceil("},
		{Label: "floor", Detail: "floor(n)", Kind: KindMethod, InsertText: "floor("},
		{Label: "round", Detail: "round(n)", Kind: KindMethod, InsertText: "round("},
		{Label: "max", Detail: "max(...n)", Kind: KindMethod, InsertText: "max("},
		{Label: "min", Detail: "min(...n)", Kind: KindMethod, InsertText: "min("},
		{Label: "sqrt", Detail: "sqrt(n)", Kind: KindMethod, InsertText: "sqrt("},
		{Label: "pow", Detail: "pow(base, exp)", Kind: KindMethod, InsertText: "pow("},
		{Label: "random", Detail: "random()", Kind: KindMethod, InsertText: "random()"},
		{Label: "PI", Detail: "π", Kind: KindProperty, InsertText: "PI"},
	},
	"JSON": {
		{Label: "stringify", Detail: "stringify(obj, null?, indent?) — to JSON string", Kind: KindMethod, InsertText: "stringify("},
		{Label: "parse", Detail: "parse(str) — parse JSON string", Kind: KindMethod, InsertText: "parse("},
	},
	"env": {
		{Label: "get", Detail: "get(key) — get env var", Kind: KindMethod, InsertText: "get("},
		{Label: "set", Detail: "set(key, val) — set env var", Kind: KindMethod, InsertText: "set("},
		{Label: "getAll", Detail: "getAll() — get all env vars", Kind: KindMethod, InsertText: "getAll()"},
		{Label: "has", Detail: "has(key) — check env var", Kind: KindMethod, InsertText: "has("},
	},
	"crypto": {
		{Label: "hash", Detail: "hash(algo, data) — hash data", Kind: KindMethod, InsertText: "hash("},
		{Label: "md5", Detail: "md5(data) — MD5 hash", Kind: KindMethod, InsertText: "md5("},
		{Label: "sha1", Detail: "sha1(data) — SHA1 hash", Kind: KindMethod, InsertText: "sha1("},
		{Label: "sha256", Detail: "sha256(data) — SHA256 hash", Kind: KindMethod, InsertText: "sha256("},
		{Label: "sha512", Detail: "sha512(data) — SHA512 hash", Kind: KindMethod, InsertText: "sha512("},
		{Label: "randomBytes", Detail: "randomBytes(n) — random bytes", Kind: KindMethod, InsertText: "randomBytes("},
		{Label: "uuid", Detail: "uuid() — generate UUID", Kind: KindMethod, InsertText: "uuid()"},
		{Label: "bcrypt", Detail: "bcrypt(pass, cost?) — hash password", Kind: KindMethod, InsertText: "bcrypt("},
		{Label: "verify", Detail: "verify(pass, hash) — verify bcrypt", Kind: KindMethod, InsertText: "verify("},
		{Label: "base64encode", Detail: "base64encode(data)", Kind: KindMethod, InsertText: "base64encode("},
		{Label: "base64decode", Detail: "base64decode(data)", Kind: KindMethod, InsertText: "base64decode("},
		{Label: "aesEncrypt", Detail: "aesEncrypt(data, key)", Kind: KindMethod, InsertText: "aesEncrypt("},
		{Label: "aesDecrypt", Detail: "aesDecrypt(data, key)", Kind: KindMethod, InsertText: "aesDecrypt("},
	},
	"os": {
		{Label: "exit", Detail: "exit(code) — exit process", Kind: KindMethod, InsertText: "exit("},
		{Label: "args", Detail: "args() — command line args", Kind: KindMethod, InsertText: "args()"},
		{Label: "exec", Detail: "exec(cmd, args?) — run command", Kind: KindMethod, InsertText: "exec("},
		{Label: "platform", Detail: "platform() — OS name", Kind: KindMethod, InsertText: "platform()"},
		{Label: "homedir", Detail: "homedir() — home directory", Kind: KindMethod, InsertText: "homedir()"},
		{Label: "cwd", Detail: "cwd() — current working directory", Kind: KindMethod, InsertText: "cwd()"},
		{Label: "getenv", Detail: "getenv(key) — get env variable", Kind: KindMethod, InsertText: "getenv("},
	},
	"path": {
		{Label: "join", Detail: "join(...parts) — join path segments", Kind: KindMethod, InsertText: "join("},
		{Label: "basename", Detail: "basename(path) — file name", Kind: KindMethod, InsertText: "basename("},
		{Label: "dirname", Detail: "dirname(path) — directory name", Kind: KindMethod, InsertText: "dirname("},
		{Label: "ext", Detail: "ext(path) — file extension", Kind: KindMethod, InsertText: "ext("},
		{Label: "resolve", Detail: "resolve(path) — absolute path", Kind: KindMethod, InsertText: "resolve("},
	},
	"db": {
		{Label: "connect", Detail: "connect(url) — connect to database", Kind: KindMethod, InsertText: "connect("},
		{Label: "query", Detail: "query(sql, ...args) — run query", Kind: KindMethod, InsertText: "query("},
		{Label: "exec", Detail: "exec(sql, ...args) — execute statement", Kind: KindMethod, InsertText: "exec("},
		{Label: "close", Detail: "close() — close connection", Kind: KindMethod, InsertText: "close()"},
	},
	"test": {
		{Label: "describe", Detail: "describe(name, fn) — test suite", Kind: KindMethod, InsertText: "describe("},
		{Label: "it", Detail: "it(name, fn) — test case", Kind: KindMethod, InsertText: "it("},
		{Label: "expect", Detail: "expect(val) — assertion", Kind: KindMethod, InsertText: "expect("},
		{Label: "run", Detail: "run() — run all tests", Kind: KindMethod, InsertText: "run()"},
	},
	"logger": {
		{Label: "info", Detail: "info(msg, ...args)", Kind: KindMethod, InsertText: "info("},
		{Label: "warn", Detail: "warn(msg, ...args)", Kind: KindMethod, InsertText: "warn("},
		{Label: "error", Detail: "error(msg, ...args)", Kind: KindMethod, InsertText: "error("},
		{Label: "debug", Detail: "debug(msg, ...args)", Kind: KindMethod, InsertText: "debug("},
		{Label: "create", Detail: "create(opts) — create logger", Kind: KindMethod, InsertText: "create("},
	},
	"validate": {
		{Label: "email", Detail: "email(str) — validate email", Kind: KindMethod, InsertText: "email("},
		{Label: "url", Detail: "url(str) — validate URL", Kind: KindMethod, InsertText: "url("},
		{Label: "required", Detail: "required(val) — check not null/empty", Kind: KindMethod, InsertText: "required("},
		{Label: "min", Detail: "min(val, n) — minimum value/length", Kind: KindMethod, InsertText: "min("},
		{Label: "max", Detail: "max(val, n) — maximum value/length", Kind: KindMethod, InsertText: "max("},
		{Label: "pattern", Detail: "pattern(str, regex) — match pattern", Kind: KindMethod, InsertText: "pattern("},
	},
	"events": {
		{Label: "on", Detail: "on(event, handler) — listen to event", Kind: KindMethod, InsertText: "on("},
		{Label: "emit", Detail: "emit(event, ...args) — emit event", Kind: KindMethod, InsertText: "emit("},
		{Label: "off", Detail: "off(event, handler) — remove listener", Kind: KindMethod, InsertText: "off("},
		{Label: "once", Detail: "once(event, handler) — one-time listener", Kind: KindMethod, InsertText: "once("},
	},
	"cache": {
		{Label: "get", Detail: "get(key) — get cached value", Kind: KindMethod, InsertText: "get("},
		{Label: "set", Detail: "set(key, val, ttl?) — set cached value", Kind: KindMethod, InsertText: "set("},
		{Label: "has", Detail: "has(key) — check cache", Kind: KindMethod, InsertText: "has("},
		{Label: "delete", Detail: "delete(key) — remove from cache", Kind: KindMethod, InsertText: "delete("},
		{Label: "clear", Detail: "clear() — clear all cache", Kind: KindMethod, InsertText: "clear()"},
	},
	"ws": {
		{Label: "connect", Detail: "connect(url) — connect WebSocket", Kind: KindMethod, InsertText: "connect("},
		{Label: "serve", Detail: "serve(port, handler) — start WS server", Kind: KindMethod, InsertText: "serve("},
		{Label: "send", Detail: "send(msg) — send message", Kind: KindMethod, InsertText: "send("},
		{Label: "close", Detail: "close() — close connection", Kind: KindMethod, InsertText: "close()"},
	},
	"ai": {
		{Label: "chat", Detail: "chat(prompt, opts?) — AI chat completion", Kind: KindMethod, InsertText: "chat("},
		{Label: "complete", Detail: "complete(prompt, opts?) — text completion", Kind: KindMethod, InsertText: "complete("},
		{Label: "embed", Detail: "embed(text) — get text embedding", Kind: KindMethod, InsertText: "embed("},
	},
	"datetime": {
		{Label: "now", Detail: "now() — current date/time", Kind: KindMethod, InsertText: "now()"},
		{Label: "parse", Detail: "parse(str, fmt?) — parse date string", Kind: KindMethod, InsertText: "parse("},
		{Label: "format", Detail: "format(date, fmt) — format date", Kind: KindMethod, InsertText: "format("},
		{Label: "add", Detail: "add(date, amount, unit) — add duration", Kind: KindMethod, InsertText: "add("},
		{Label: "diff", Detail: "diff(a, b, unit) — difference between dates", Kind: KindMethod, InsertText: "diff("},
		{Label: "timestamp", Detail: "timestamp() — Unix timestamp (ms)", Kind: KindMethod, InsertText: "timestamp()"},
		{Label: "fromTimestamp", Detail: "fromTimestamp(ms) — date from timestamp", Kind: KindMethod, InsertText: "fromTimestamp("},
	},
	"compress": {
		{Label: "gzip", Detail: "gzip(data) — compress with gzip", Kind: KindMethod, InsertText: "gzip("},
		{Label: "gunzip", Detail: "gunzip(data) — decompress gzip", Kind: KindMethod, InsertText: "gunzip("},
		{Label: "zip", Detail: "zip(files) — create zip archive", Kind: KindMethod, InsertText: "zip("},
		{Label: "unzip", Detail: "unzip(data, dest) — extract zip", Kind: KindMethod, InsertText: "unzip("},
		{Label: "deflate", Detail: "deflate(data) — deflate compress", Kind: KindMethod, InsertText: "deflate("},
		{Label: "inflate", Detail: "inflate(data) — deflate decompress", Kind: KindMethod, InsertText: "inflate("},
	},
	"regex": {
		{Label: "match", Detail: "match(pattern, str) — test regex match", Kind: KindMethod, InsertText: "match("},
		{Label: "find", Detail: "find(pattern, str) — find first match", Kind: KindMethod, InsertText: "find("},
		{Label: "findAll", Detail: "findAll(pattern, str) — find all matches", Kind: KindMethod, InsertText: "findAll("},
		{Label: "replace", Detail: "replace(pattern, str, repl) — replace matches", Kind: KindMethod, InsertText: "replace("},
		{Label: "split", Detail: "split(pattern, str) — split by regex", Kind: KindMethod, InsertText: "split("},
		{Label: "groups", Detail: "groups(pattern, str) — capture groups", Kind: KindMethod, InsertText: "groups("},
	},
	"xml": {
		{Label: "parse", Detail: "parse(str) — parse XML string", Kind: KindMethod, InsertText: "parse("},
		{Label: "stringify", Detail: "stringify(obj) — convert to XML", Kind: KindMethod, InsertText: "stringify("},
		{Label: "query", Detail: "query(doc, xpath) — XPath query", Kind: KindMethod, InsertText: "query("},
	},
	"alloc": {
		{Label: "create", Detail: "create(size) — allocate buffer", Kind: KindMethod, InsertText: "create("},
		{Label: "free", Detail: "free(buf) — free buffer", Kind: KindMethod, InsertText: "free("},
		{Label: "read", Detail: "read(buf, offset, size) — read bytes", Kind: KindMethod, InsertText: "read("},
		{Label: "write", Detail: "write(buf, offset, data) — write bytes", Kind: KindMethod, InsertText: "write("},
		{Label: "size", Detail: "size(buf) — buffer size", Kind: KindMethod, InsertText: "size("},
	},
	"queue": {
		{Label: "create", Detail: "create(opts?) — create queue", Kind: KindMethod, InsertText: "create("},
		{Label: "push", Detail: "push(item) — enqueue item", Kind: KindMethod, InsertText: "push("},
		{Label: "pop", Detail: "pop() — dequeue item", Kind: KindMethod, InsertText: "pop()"},
		{Label: "peek", Detail: "peek() — view front item", Kind: KindMethod, InsertText: "peek()"},
		{Label: "size", Detail: "size() — queue length", Kind: KindMethod, InsertText: "size()"},
		{Label: "clear", Detail: "clear() — empty the queue", Kind: KindMethod, InsertText: "clear()"},
	},
	"mail": {
		{Label: "send", Detail: "send(opts) — send email", Kind: KindMethod, InsertText: "send("},
		{Label: "configure", Detail: "configure(smtp) — configure SMTP", Kind: KindMethod, InsertText: "configure("},
		{Label: "template", Detail: "template(name, vars) — render template", Kind: KindMethod, InsertText: "template("},
	},
	"jwt": {
		{Label: "sign", Detail: "sign(payload, secret, opts?) — create JWT", Kind: KindMethod, InsertText: "sign("},
		{Label: "verify", Detail: "verify(token, secret) — verify JWT", Kind: KindMethod, InsertText: "verify("},
		{Label: "decode", Detail: "decode(token) — decode without verify", Kind: KindMethod, InsertText: "decode("},
	},
	"oauth2": {
		{Label: "createClient", Detail: "createClient(config) — create OAuth2 client", Kind: KindMethod, InsertText: "createClient("},
		{Label: "authUrl", Detail: "authUrl(client, scopes) — get auth URL", Kind: KindMethod, InsertText: "authUrl("},
		{Label: "exchange", Detail: "exchange(client, code) — exchange code for token", Kind: KindMethod, InsertText: "exchange("},
		{Label: "refresh", Detail: "refresh(client, token) — refresh access token", Kind: KindMethod, InsertText: "refresh("},
	},
	"stripe": {
		{Label: "charge", Detail: "charge(amount, currency, source) — charge card", Kind: KindMethod, InsertText: "charge("},
		{Label: "createCustomer", Detail: "createCustomer(opts) — create customer", Kind: KindMethod, InsertText: "createCustomer("},
		{Label: "createSubscription", Detail: "createSubscription(customerId, planId) — subscribe", Kind: KindMethod, InsertText: "createSubscription("},
		{Label: "refund", Detail: "refund(chargeId, amount?) — refund charge", Kind: KindMethod, InsertText: "refund("},
		{Label: "webhook", Detail: "webhook(secret, handler) — handle webhooks", Kind: KindMethod, InsertText: "webhook("},
	},
	"postgres": {
		{Label: "connect", Detail: "connect(url) — connect to PostgreSQL", Kind: KindMethod, InsertText: "connect("},
		{Label: "query", Detail: "query(sql, ...args) — run query", Kind: KindMethod, InsertText: "query("},
		{Label: "exec", Detail: "exec(sql, ...args) — execute statement", Kind: KindMethod, InsertText: "exec("},
		{Label: "transaction", Detail: "transaction(fn) — run in transaction", Kind: KindMethod, InsertText: "transaction("},
		{Label: "close", Detail: "close() — close connection", Kind: KindMethod, InsertText: "close()"},
	},
	"mysql": {
		{Label: "connect", Detail: "connect(url) — connect to MySQL", Kind: KindMethod, InsertText: "connect("},
		{Label: "query", Detail: "query(sql, ...args) — run query", Kind: KindMethod, InsertText: "query("},
		{Label: "exec", Detail: "exec(sql, ...args) — execute statement", Kind: KindMethod, InsertText: "exec("},
		{Label: "transaction", Detail: "transaction(fn) — run in transaction", Kind: KindMethod, InsertText: "transaction("},
		{Label: "close", Detail: "close() — close connection", Kind: KindMethod, InsertText: "close()"},
	},
	"redis": {
		{Label: "connect", Detail: "connect(url) — connect to Redis", Kind: KindMethod, InsertText: "connect("},
		{Label: "get", Detail: "get(key) — get value", Kind: KindMethod, InsertText: "get("},
		{Label: "set", Detail: "set(key, val, ttl?) — set value", Kind: KindMethod, InsertText: "set("},
		{Label: "del", Detail: "del(key) — delete key", Kind: KindMethod, InsertText: "del("},
		{Label: "exists", Detail: "exists(key) — check key exists", Kind: KindMethod, InsertText: "exists("},
		{Label: "expire", Detail: "expire(key, ttl) — set TTL", Kind: KindMethod, InsertText: "expire("},
		{Label: "incr", Detail: "incr(key) — increment counter", Kind: KindMethod, InsertText: "incr("},
		{Label: "lpush", Detail: "lpush(key, val) — push to list", Kind: KindMethod, InsertText: "lpush("},
		{Label: "rpop", Detail: "rpop(key) — pop from list", Kind: KindMethod, InsertText: "rpop("},
		{Label: "hset", Detail: "hset(key, field, val) — set hash field", Kind: KindMethod, InsertText: "hset("},
		{Label: "hget", Detail: "hget(key, field) — get hash field", Kind: KindMethod, InsertText: "hget("},
		{Label: "close", Detail: "close() — close connection", Kind: KindMethod, InsertText: "close()"},
	},
	"rabbitmq": {
		{Label: "connect", Detail: "connect(url) — connect to RabbitMQ", Kind: KindMethod, InsertText: "connect("},
		{Label: "publish", Detail: "publish(exchange, key, msg) — publish message", Kind: KindMethod, InsertText: "publish("},
		{Label: "subscribe", Detail: "subscribe(queue, handler) — consume messages", Kind: KindMethod, InsertText: "subscribe("},
		{Label: "declare", Detail: "declare(queue, opts?) — declare queue", Kind: KindMethod, InsertText: "declare("},
		{Label: "close", Detail: "close() — close connection", Kind: KindMethod, InsertText: "close()"},
	},
	"graphql": {
		{Label: "schema", Detail: "schema(typeDefs, resolvers) — create schema", Kind: KindMethod, InsertText: "schema("},
		{Label: "serve", Detail: "serve(port, schema) — start GraphQL server", Kind: KindMethod, InsertText: "serve("},
		{Label: "query", Detail: "query(url, gql, vars?) — run GraphQL query", Kind: KindMethod, InsertText: "query("},
	},
	"utils": {
		{Label: "uuid", Detail: "uuid() — generate UUID v4", Kind: KindMethod, InsertText: "uuid()"},
		{Label: "sleep", Detail: "sleep(ms) — sleep milliseconds", Kind: KindMethod, InsertText: "sleep("},
		{Label: "debounce", Detail: "debounce(fn, ms) — debounce function", Kind: KindMethod, InsertText: "debounce("},
		{Label: "throttle", Detail: "throttle(fn, ms) — throttle function", Kind: KindMethod, InsertText: "throttle("},
		{Label: "clamp", Detail: "clamp(val, min, max) — clamp value", Kind: KindMethod, InsertText: "clamp("},
		{Label: "deepEqual", Detail: "deepEqual(a, b) — deep equality check", Kind: KindMethod, InsertText: "deepEqual("},
		{Label: "pick", Detail: "pick(obj, keys) — pick object keys", Kind: KindMethod, InsertText: "pick("},
		{Label: "omit", Detail: "omit(obj, keys) — omit object keys", Kind: KindMethod, InsertText: "omit("},
		{Label: "merge", Detail: "merge(a, b) — deep merge objects", Kind: KindMethod, InsertText: "merge("},
		{Label: "flatten", Detail: "flatten(arr) — flatten nested array", Kind: KindMethod, InsertText: "flatten("},
		{Label: "chunk", Detail: "chunk(arr, size) — chunk array", Kind: KindMethod, InsertText: "chunk("},
		{Label: "unique", Detail: "unique(arr) — remove duplicates", Kind: KindMethod, InsertText: "unique("},
	},
	"pdf": {
		{Label: "create", Detail: "create(opts?) — create PDF document", Kind: KindMethod, InsertText: "create("},
		{Label: "addPage", Detail: "addPage() — add new page", Kind: KindMethod, InsertText: "addPage()"},
		{Label: "text", Detail: "text(x, y, str, opts?) — add text", Kind: KindMethod, InsertText: "text("},
		{Label: "image", Detail: "image(x, y, path, opts?) — add image", Kind: KindMethod, InsertText: "image("},
		{Label: "save", Detail: "save(path) — save PDF to file", Kind: KindMethod, InsertText: "save("},
		{Label: "parse", Detail: "parse(path) — parse existing PDF", Kind: KindMethod, InsertText: "parse("},
	},
	"excel": {
		{Label: "create", Detail: "create() — create workbook", Kind: KindMethod, InsertText: "create()"},
		{Label: "open", Detail: "open(path) — open Excel file", Kind: KindMethod, InsertText: "open("},
		{Label: "addSheet", Detail: "addSheet(name) — add worksheet", Kind: KindMethod, InsertText: "addSheet("},
		{Label: "setCell", Detail: "setCell(sheet, cell, val) — set cell value", Kind: KindMethod, InsertText: "setCell("},
		{Label: "getCell", Detail: "getCell(sheet, cell) — get cell value", Kind: KindMethod, InsertText: "getCell("},
		{Label: "save", Detail: "save(path) — save workbook", Kind: KindMethod, InsertText: "save("},
	},
}

// Standard library module names for @import suggestions
var stdModules = []CompletionItem{
	{Label: `@import("std.io")`, Detail: "I/O operations", Kind: KindModule, InsertText: `@import("std.io")`},
	{Label: `@import("std.fs")`, Detail: "File system", Kind: KindModule, InsertText: `@import("std.fs")`},
	{Label: `@import("std.http")`, Detail: "HTTP client & server", Kind: KindModule, InsertText: `@import("std.http")`},
	{Label: `@import("std.crypto")`, Detail: "Cryptography", Kind: KindModule, InsertText: `@import("std.crypto")`},
	{Label: `@import("std.db")`, Detail: "Database", Kind: KindModule, InsertText: `@import("std.db")`},
	{Label: `@import("std.env")`, Detail: "Environment variables", Kind: KindModule, InsertText: `@import("std.env")`},
	{Label: `@import("std.validate")`, Detail: "Input validation", Kind: KindModule, InsertText: `@import("std.validate")`},
	{Label: `@import("std.events")`, Detail: "Event emitter", Kind: KindModule, InsertText: `@import("std.events")`},
	{Label: `@import("std.cache")`, Detail: "In-memory cache", Kind: KindModule, InsertText: `@import("std.cache")`},
	{Label: `@import("std.logger")`, Detail: "Structured logging", Kind: KindModule, InsertText: `@import("std.logger")`},
	{Label: `@import("std.queue")`, Detail: "Message queue", Kind: KindModule, InsertText: `@import("std.queue")`},
	{Label: `@import("std.ws")`, Detail: "WebSockets", Kind: KindModule, InsertText: `@import("std.ws")`},
	{Label: `@import("std.mail")`, Detail: "Email sending", Kind: KindModule, InsertText: `@import("std.mail")`},
	{Label: `@import("std.ai")`, Detail: "AI/LLM integration", Kind: KindModule, InsertText: `@import("std.ai")`},
	{Label: `@import("std.test")`, Detail: "Testing framework", Kind: KindModule, InsertText: `@import("std.test")`},
	{Label: `@import("std.alloc")`, Detail: "Memory allocation", Kind: KindModule, InsertText: `@import("std.alloc")`},
	{Label: `@import("std.math")`, Detail: "Math functions", Kind: KindModule, InsertText: `@import("std.math")`},
	{Label: `@import("std.datetime")`, Detail: "Date/time", Kind: KindModule, InsertText: `@import("std.datetime")`},
	{Label: `@import("std.compress")`, Detail: "Compression", Kind: KindModule, InsertText: `@import("std.compress")`},
	{Label: `@import("std.regex")`, Detail: "Regular expressions", Kind: KindModule, InsertText: `@import("std.regex")`},
	{Label: `@import("std.os")`, Detail: "OS operations", Kind: KindModule, InsertText: `@import("std.os")`},
	{Label: `@import("std.path")`, Detail: "Path operations", Kind: KindModule, InsertText: `@import("std.path")`},
	{Label: `@import("std.xml")`, Detail: "XML parsing", Kind: KindModule, InsertText: `@import("std.xml")`},
	{Label: `@import("std.yaml")`, Detail: "YAML parsing", Kind: KindModule, InsertText: `@import("std.yaml")`},
	{Label: `@import("std.toml")`, Detail: "TOML parsing", Kind: KindModule, InsertText: `@import("std.toml")`},
	{Label: `@import("std.csv")`, Detail: "CSV parsing", Kind: KindModule, InsertText: `@import("std.csv")`},
	{Label: `@import("std.pdf")`, Detail: "PDF generation", Kind: KindModule, InsertText: `@import("std.pdf")`},
	{Label: `@import("std.excel")`, Detail: "Excel files", Kind: KindModule, InsertText: `@import("std.excel")`},
	{Label: `@import("std.jwt")`, Detail: "JSON Web Tokens", Kind: KindModule, InsertText: `@import("std.jwt")`},
	{Label: `@import("std.oauth2")`, Detail: "OAuth2", Kind: KindModule, InsertText: `@import("std.oauth2")`},
	{Label: `@import("std.stripe")`, Detail: "Stripe payments", Kind: KindModule, InsertText: `@import("std.stripe")`},
	{Label: `@import("std.postgres")`, Detail: "PostgreSQL", Kind: KindModule, InsertText: `@import("std.postgres")`},
	{Label: `@import("std.mysql")`, Detail: "MySQL", Kind: KindModule, InsertText: `@import("std.mysql")`},
	{Label: `@import("std.redis")`, Detail: "Redis", Kind: KindModule, InsertText: `@import("std.redis")`},
	{Label: `@import("std.rabbitmq")`, Detail: "RabbitMQ", Kind: KindModule, InsertText: `@import("std.rabbitmq")`},
	{Label: `@import("std.graphql")`, Detail: "GraphQL", Kind: KindModule, InsertText: `@import("std.graphql")`},
	{Label: `@import("std.utils")`, Detail: "Utility functions", Kind: KindModule, InsertText: `@import("std.utils")`},
}

// extractUserSymbols scans all lines for user-defined functions and variables
func extractUserSymbols(lines []string) []CompletionItem {
	items := make([]CompletionItem, 0, 16)
	seen := map[string]bool{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// fn name( or fn name {
		if strings.HasPrefix(trimmed, "fn ") {
			rest := strings.TrimPrefix(trimmed, "fn ")
			end := strings.IndexAny(rest, "({")
			if end > 0 {
				name := strings.TrimSpace(rest[:end])
				if name != "" && !seen[name] {
					seen[name] = true
					params := ""
					pStart := strings.Index(rest, "(")
					pEnd := strings.Index(rest, ")")
					if pStart >= 0 && pEnd > pStart {
						params = rest[pStart+1 : pEnd]
					}
					items = append(items, CompletionItem{
						Label:      name,
						Detail:     "fn " + name + "(" + params + ")",
						Kind:       KindFunction,
						InsertText: name + "(",
					})
				}
			}
		}

		// val/var name = ...
		for _, prefix := range []string{"val ", "var "} {
			if strings.HasPrefix(trimmed, prefix) {
				rest := strings.TrimPrefix(trimmed, prefix)
				end := strings.IndexAny(rest, " =:,({[")
				if end > 0 {
					name := strings.TrimSpace(rest[:end])
					if name != "" && !seen[name] && len(name) > 1 {
						seen[name] = true
						detail := prefix + name
						kind := KindVariable
						// If it's an @import, make it a module
						if strings.Contains(rest, "@import(") {
							kind = KindModule
							// Extract module name
							iStart := strings.Index(rest, `"std.`)
							iEnd := strings.LastIndex(rest, `"`)
							if iStart >= 0 && iEnd > iStart {
								modName := rest[iStart+5 : iEnd]
								detail = "module: std." + modName
							}
						}
						items = append(items, CompletionItem{
							Label:      name,
							Detail:     detail,
							Kind:       kind,
							InsertText: name,
						})
					}
				}
			}
		}
	}

	return items
}

// GetCompletions returns the list of completions for the current cursor context
func GetCompletions(lines []string, line string, cx int) []CompletionItem {
	receiver, partial := dotContextAtCursor(line, cx)

	// Dot-triggered completions (e.g. "io.lo")
	if receiver != "" {
		methods, ok := moduleCompletions[receiver]
		if !ok {
			return nil
		}
		if partial == "" {
			return methods
		}
		var filtered []CompletionItem
		lp := strings.ToLower(partial)
		for _, m := range methods {
			if strings.HasPrefix(strings.ToLower(m.Label), lp) {
				filtered = append(filtered, m)
			}
		}
		return filtered
	}

	// @import suggestions — trigger on "@", "@i", "@im", "@import", "@import("
	lineUpToCursor := ""
	if cx <= len([]rune(line)) {
		lineUpToCursor = string([]rune(line)[:cx])
	}
	trimmedUp := strings.TrimSpace(lineUpToCursor)

	isAtImport := strings.HasPrefix(trimmedUp, "@import(") ||
		strings.HasPrefix(trimmedUp, "@import") ||
		strings.HasPrefix(trimmedUp, "@im") ||
		strings.HasPrefix(trimmedUp, "@i") ||
		trimmedUp == "@"

	if isAtImport {
		// Extract partial module name after "std." if present
		importPartial := ""
		atStd := strings.LastIndex(lineUpToCursor, `"std.`)
		if atStd >= 0 {
			importPartial = lineUpToCursor[atStd+5:]
			importPartial = strings.TrimRight(importPartial, `"`)
		} else {
			// Still in the early part — check if there's a partial after @import("
			atQ := strings.LastIndex(lineUpToCursor, `"`)
			if atQ >= 0 {
				importPartial = lineUpToCursor[atQ+1:]
			}
		}

		var filtered []CompletionItem
		lp := strings.ToLower(importPartial)
		for _, m := range stdModules {
			label := strings.TrimPrefix(m.Label, `@import("std.`)
			label = strings.TrimSuffix(label, `")`)
			if strings.HasPrefix(strings.ToLower(label), lp) {
				filtered = append(filtered, m)
			}
		}
		return filtered
	}

	if partial == "" || len(partial) < 1 {
		return nil
	}

	lp := strings.ToLower(partial)

	// Collect from multiple sources
	var results []CompletionItem
	seen := map[string]bool{}

	// User symbols (highest priority)
	userSymbols := extractUserSymbols(lines)
	for _, item := range userSymbols {
		if strings.HasPrefix(strings.ToLower(item.Label), lp) && item.Label != partial {
			if !seen[item.Label] {
				seen[item.Label] = true
				results = append(results, item)
			}
		}
	}

	// Keywords
	for _, kw := range ntlKeywords {
		if strings.HasPrefix(strings.ToLower(kw.Label), lp) && kw.Label != partial {
			if !seen[kw.Label] {
				seen[kw.Label] = true
				results = append(results, kw)
			}
		}
	}

	// Sort by match quality: exact prefix first, then by label
	sort.SliceStable(results, func(i, j int) bool {
		li := strings.ToLower(results[i].Label)
		lj := strings.ToLower(results[j].Label)
		exactI := strings.HasPrefix(li, lp)
		exactJ := strings.HasPrefix(lj, lp)
		if exactI != exactJ {
			return exactI
		}
		return results[i].Kind < results[j].Kind
	})

	// Limit
	if len(results) > 20 {
		results = results[:20]
	}

	return results
}
