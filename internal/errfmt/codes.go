// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package errfmt

type ErrorCode struct {
	Code       string
	Title      string
	Suggestion string
	English    string
}

const (
	ErrUndefinedVar   = "E0001"
	ErrUndefinedFunc  = "E0002"
	ErrConstReassign  = "E0003"
	ErrNotCallable    = "E0004"
	ErrNullAccess     = "E0005"
	ErrDivisionByZero = "E0006"
	ErrTypeMismatch   = "E0007"
	ErrModuleNotFound = "E0008"
	ErrIndexOutOfRange = "E0009"
	ErrStackOverflow  = "E0010"
	ErrInvalidArg     = "E0011"
	ErrUnexpectedToken = "E0012"
	ErrMissingToken   = "E0013"
	ErrInvalidSyntax  = "E0014"
	ErrDuplicateDecl  = "E0015"
	ErrInvalidReturn  = "E0016"
	ErrInvalidBreak   = "E0017"
	ErrInvalidContinue = "E0018"
	ErrCircularImport = "E0019"
	ErrIOFailure      = "E0020"
	ErrAssertFailed   = "E0021"
	ErrInvalidPattern = "E0022"
	ErrKeyNotFound    = "E0023"
	ErrReadonly       = "E0024"
	ErrNetworkFailure = "E0025"
	ErrTimeout        = "E0026"
	ErrPermission     = "E0027"
	ErrNotImplemented = "E0028"
	ErrDeadlock       = "E0029"
	ErrInvalidRegex   = "E0030"
	ErrParseJSON      = "E0031"
	ErrParseXML       = "E0032"
	ErrParseYAML      = "E0033"
	ErrParseTOML      = "E0034"
	ErrInvalidURL     = "E0035"
	ErrInvalidEmail   = "E0036"
	ErrCryptoFailure  = "E0037"
	ErrDBConnection   = "E0038"
	ErrDBQuery        = "E0039"
	ErrAuthFailure    = "E0040"
	ErrRateLimited    = "E0041"
	ErrFileNotFound   = "E0042"
	ErrInvalidFormat  = "E0043"

	ErrUnexpectedTokenGeneric = "E1000"
	ErrUnexpectedComma        = "E1001"
	ErrUnexpectedCloseParen   = "E1002"
	ErrUnexpectedCloseBrace   = "E1003"
	ErrUnexpectedCloseBracket = "E1004"
	ErrUnexpectedAssign       = "E1005"
	ErrUnexpectedSemicolon    = "E1006"
	ErrExpectedToken          = "E1010"

	// Self-hosted type system errors (E0044–E0059)
	ErrUnknownType        = "E0044"
	ErrReturnTypeMismatch = "E0045"
	ErrArgTypeMismatch    = "E0046"
	ErrNullableViolation  = "E0047"
	ErrUninitializedConst = "E0048"

	// Suspect / suspicious-pattern errors (S0001–S0020)
	ErrSuspectForOfNonIterable  = "S0001"
	ErrSuspectMatchNoArm        = "S0002"
	ErrSuspectNaNResult         = "S0003"
	ErrSuspectIndexOutOfBounds  = "S0004"
	ErrSuspectSpreadNonIterable = "S0005"
	ErrSuspectNullSpread        = "S0006"
	ErrSuspectCallUndefined     = "S0007"
)

func newErrorCode(code, title, suggestion string) ErrorCode {
	return ErrorCode{
		Code:       code,
		Title:      title,
		Suggestion: suggestion,
		English:    title,
	}
}

var errorCodes = map[string]ErrorCode{
	ErrUndefinedVar: newErrorCode(
		ErrUndefinedVar,
		"Undefined variable",
		"Declare the variable before using it with 'val name = ...' or 'var name = ...'.",
	),
	ErrUndefinedFunc: newErrorCode(
		ErrUndefinedFunc,
		"Undefined function",
		"Define the function before calling it with 'fn name(...) { ... }'.",
	),
	ErrConstReassign: newErrorCode(
		ErrConstReassign,
		"Cannot reassign constant",
		"Use 'var' instead of 'val' if the value needs to change.",
	),
	ErrNotCallable: newErrorCode(
		ErrNotCallable,
		"Value is not callable",
		"Make sure the value is a function before calling it with '()'.",
	),
	ErrNullAccess: newErrorCode(
		ErrNullAccess,
		"Null or undefined access",
		"Guard the value with 'if x != null { ... }' or use optional chaining like 'x?.prop'.",
	),
	ErrDivisionByZero: newErrorCode(
		ErrDivisionByZero,
		"Division by zero",
		"Check the divisor before dividing: 'if b != 0 { a / b }'.",
	),
	ErrTypeMismatch: newErrorCode(
		ErrTypeMismatch,
		"Type mismatch",
		"Convert the value to the expected type, such as Number(x) or String(x).",
	),
	ErrModuleNotFound: newErrorCode(
		ErrModuleNotFound,
		"Module not found",
		"Install the module with 'lunex add <module>' or verify that it exists in the standard library.",
	),
	ErrIndexOutOfRange: newErrorCode(
		ErrIndexOutOfRange,
		"Index out of range",
		"Check array bounds before accessing an element: 'if i < arr.length { arr[i] }'.",
	),
	ErrStackOverflow: newErrorCode(
		ErrStackOverflow,
		"Stack overflow",
		"Add a base case to recursive functions to stop infinite recursion.",
	),
	ErrInvalidArg: newErrorCode(
		ErrInvalidArg,
		"Invalid argument",
		"Verify that the function received the correct number and type of arguments.",
	),
	ErrUnexpectedToken: newErrorCode(
		ErrUnexpectedToken,
		"Unexpected token",
		"Check for missing brackets, commas, or keywords near this position.",
	),
	ErrMissingToken: newErrorCode(
		ErrMissingToken,
		"Missing token",
		"A required symbol or keyword is missing near this position.",
	),
	ErrInvalidSyntax: newErrorCode(
		ErrInvalidSyntax,
		"Invalid syntax",
		"Review the expression or statement structure near this location.",
	),
	ErrDuplicateDecl: newErrorCode(
		ErrDuplicateDecl,
		"Duplicate declaration",
		"This name was already declared in the current scope. Use a different name or remove the duplicate.",
	),
	ErrInvalidReturn: newErrorCode(
		ErrInvalidReturn,
		"Invalid return statement",
		"'return' can only be used inside a function.",
	),
	ErrInvalidBreak: newErrorCode(
		ErrInvalidBreak,
		"Invalid break statement",
		"'break' can only be used inside a loop.",
	),
	ErrInvalidContinue: newErrorCode(
		ErrInvalidContinue,
		"Invalid continue statement",
		"'continue' can only be used inside a loop.",
	),
	ErrCircularImport: newErrorCode(
		ErrCircularImport,
		"Circular import",
		"Remove the import cycle by refactoring shared code into a separate module.",
	),
	ErrIOFailure: newErrorCode(
		ErrIOFailure,
		"I/O failure",
		"Check file permissions, disk access, and the target path.",
	),
	ErrAssertFailed: newErrorCode(
		ErrAssertFailed,
		"Assertion failed",
		"The assertion evaluated to false. Review the condition and the surrounding logic.",
	),
	ErrInvalidPattern: newErrorCode(
		ErrInvalidPattern,
		"Invalid pattern",
		"The pattern format is not valid for this operation.",
	),
	ErrKeyNotFound: newErrorCode(
		ErrKeyNotFound,
		"Key not found",
		"Verify that the key exists before accessing it.",
	),
	ErrReadonly: newErrorCode(
		ErrReadonly,
		"Readonly value",
		"This value cannot be modified.",
	),
	ErrNetworkFailure: newErrorCode(
		ErrNetworkFailure,
		"Network failure",
		"Check the connection, endpoint, and any firewall or proxy settings.",
	),
	ErrTimeout: newErrorCode(
		ErrTimeout,
		"Operation timed out",
		"The operation took too long to complete.",
	),
	ErrPermission: newErrorCode(
		ErrPermission,
		"Permission denied",
		"Run the command with the required permissions or update access settings.",
	),
	ErrNotImplemented: newErrorCode(
		ErrNotImplemented,
		"Not implemented",
		"This feature has not been implemented yet.",
	),
	ErrDeadlock: newErrorCode(
		ErrDeadlock,
		"Deadlock detected",
		"Review locking order and concurrency flow to avoid circular waits.",
	),
	ErrInvalidRegex: newErrorCode(
		ErrInvalidRegex,
		"Invalid regular expression",
		"Check the regex pattern for syntax errors.",
	),
	ErrParseJSON: newErrorCode(
		ErrParseJSON,
		"Failed to parse JSON",
		"Verify that the JSON text is valid.",
	),
	ErrParseXML: newErrorCode(
		ErrParseXML,
		"Failed to parse XML",
		"Verify that the XML text is well-formed.",
	),
	ErrParseYAML: newErrorCode(
		ErrParseYAML,
		"Failed to parse YAML",
		"Verify that the YAML text is correctly indented and valid.",
	),
	ErrParseTOML: newErrorCode(
		ErrParseTOML,
		"Failed to parse TOML",
		"Verify that the TOML text follows the expected format.",
	),
	ErrInvalidURL: newErrorCode(
		ErrInvalidURL,
		"Invalid URL",
		"Check the URL format and ensure it includes a valid scheme.",
	),
	ErrInvalidEmail: newErrorCode(
		ErrInvalidEmail,
		"Invalid email address",
		"Check that the email address follows the correct format.",
	),
	ErrCryptoFailure: newErrorCode(
		ErrCryptoFailure,
		"Cryptographic failure",
		"Review the key, algorithm, or input data used in the operation.",
	),
	ErrDBConnection: newErrorCode(
		ErrDBConnection,
		"Database connection failed",
		"Check the database host, credentials, and network availability.",
	),
	ErrDBQuery: newErrorCode(
		ErrDBQuery,
		"Database query failed",
		"Review the query syntax and the database schema.",
	),
	ErrAuthFailure: newErrorCode(
		ErrAuthFailure,
		"Authentication failed",
		"Verify the credentials, token, or authentication flow.",
	),
	ErrRateLimited: newErrorCode(
		ErrRateLimited,
		"Rate limited",
		"Wait before retrying or reduce the request frequency.",
	),
	ErrFileNotFound: newErrorCode(
		ErrFileNotFound,
		"File not found",
		"Check the path and make sure the file exists.",
	),
	ErrInvalidFormat: newErrorCode(
		ErrInvalidFormat,
		"Invalid format",
		"Check whether the value matches the expected format.",
	),

	// Self-hosted type system errors
	ErrUnknownType: newErrorCode(
		ErrUnknownType,
		"Unknown type",
		"Use a built-in type (int, float, string, bool, any, []T) or declare a struct alias.",
	),
	ErrReturnTypeMismatch: newErrorCode(
		ErrReturnTypeMismatch,
		"Return type mismatch",
		"The returned value does not match the declared return type of the function.",
	),
	ErrArgTypeMismatch: newErrorCode(
		ErrArgTypeMismatch,
		"Argument type mismatch",
		"The argument type does not match the expected parameter type.",
	),
	ErrNullableViolation: newErrorCode(
		ErrNullableViolation,
		"Nullable violation",
		"A non-nullable type received null. Use 'T?' to declare a nullable type.",
	),
	ErrUninitializedConst: newErrorCode(
		ErrUninitializedConst,
		"Uninitialized constant",
		"Constants declared with 'val' must be assigned a value immediately.",
	),
	ErrUnexpectedTokenGeneric: newErrorCode(
		ErrUnexpectedTokenGeneric,
		"Unexpected token",
		"Check for missing brackets, commas, or keywords near this position.",
	),
	ErrUnexpectedComma: newErrorCode(
		ErrUnexpectedComma,
		"Unexpected comma",
		"A comma appeared where the parser did not expect one. Check for trailing commas or a missing expression.",
	),
	ErrUnexpectedCloseParen: newErrorCode(
		ErrUnexpectedCloseParen,
		"Unexpected closing parenthesis",
		"Make sure every '(' has a matching ')' and that no extra closing parenthesis was added.",
	),
	ErrUnexpectedCloseBrace: newErrorCode(
		ErrUnexpectedCloseBrace,
		"Unexpected closing brace",
		"Make sure every '{' block is properly closed and that no stray '}' exists.",
	),
	ErrUnexpectedCloseBracket: newErrorCode(
		ErrUnexpectedCloseBracket,
		"Unexpected closing bracket",
		"Make sure every '[' has a matching ']'.",
	),
	ErrUnexpectedAssign: newErrorCode(
		ErrUnexpectedAssign,
		"Unexpected assignment operator",
		"Use '==' for comparison, or move the assignment into its own statement.",
	),
	ErrUnexpectedSemicolon: newErrorCode(
		ErrUnexpectedSemicolon,
		"Unexpected semicolon",
		"Remove the extra semicolon or place it in a valid statement position.",
	),
	ErrExpectedToken: newErrorCode(
		ErrExpectedToken,
		"Expected token",
		"A required keyword or delimiter is missing before this position.",
	),

	// ── Suspect / suspicious-pattern errors ──
	ErrSuspectForOfNonIterable: newErrorCode(
		ErrSuspectForOfNonIterable,
		"for-of over non-iterable value",
		"Only arrays, strings, and objects are iterable. Check the value type before iterating.",
	),
	ErrSuspectMatchNoArm: newErrorCode(
		ErrSuspectMatchNoArm,
		"match expression produced no result",
		"No case matched the subject. Add a default arm: `_ => { ... }`.",
	),
	ErrSuspectNaNResult: newErrorCode(
		ErrSuspectNaNResult,
		"arithmetic produced NaN",
		"One of the operands could not be converted to a number. Check for undefined, null, or non-numeric strings.",
	),
	ErrSuspectIndexOutOfBounds: newErrorCode(
		ErrSuspectIndexOutOfBounds,
		"array index out of bounds",
		"The index is outside the valid range [0, length-1]. Guard with: if i < arr.length { ... }",
	),
	ErrSuspectSpreadNonIterable: newErrorCode(
		ErrSuspectSpreadNonIterable,
		"spreading a non-array/non-object value",
		"Spread (...) only works on arrays and objects. Wrap the value in an array first if needed.",
	),
	ErrSuspectNullSpread: newErrorCode(
		ErrSuspectNullSpread,
		"spreading null or undefined",
		"The spread target is null or undefined — nothing will be spread. Guard with: if x != null { ...x }",
	),
	ErrSuspectCallUndefined: newErrorCode(
		ErrSuspectCallUndefined,
		"calling result of expression that returned undefined",
		"The called value is undefined. The function may have forgotten to return a value.",
	),
}

func LookupCode(code string) (ErrorCode, bool) {
	ec, ok := errorCodes[code]
	return ec, ok
}

func CodeSuggestion(code string) string {
	if ec, ok := errorCodes[code]; ok {
		return ec.Suggestion
	}
	return ""
}

func CodeTitle(code string) string {
	if ec, ok := errorCodes[code]; ok {
		return ec.Title
	}
	return ""
}