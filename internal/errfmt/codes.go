// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package errfmt

const (
        ErrUndefinedVar    = "E0001"
        ErrUndefinedFunc   = "E0002"
        ErrConstReassign   = "E0003"
        ErrNotCallable     = "E0004"
        ErrNullAccess      = "E0005"
        ErrDivisionByZero  = "E0006"
        ErrTypeMismatch    = "E0007"
        ErrModuleNotFound  = "E0008"
        ErrIndexOutOfRange = "E0009"
        ErrStackOverflow   = "E0010"
        ErrInvalidArg      = "E0011"
        ErrUnexpectedToken = "E0012"
        ErrMissingToken    = "E0013"
        ErrInvalidSyntax   = "E0014"
        ErrDuplicateDecl   = "E0015"
        ErrInvalidReturn   = "E0016"
        ErrInvalidBreak    = "E0017"
        ErrInvalidContinue = "E0018"
        ErrCircularImport  = "E0019"
        ErrIOFailure       = "E0020"
        ErrAssertFailed    = "E0021"
        ErrInvalidPattern  = "E0022"
        ErrKeyNotFound     = "E0023"
        ErrReadonly        = "E0024"
        ErrNetworkFailure  = "E0025"
        ErrTimeout         = "E0026"
        ErrPermission      = "E0027"
        ErrNotImplemented  = "E0028"
        ErrDeadlock        = "E0029"
        ErrInvalidRegex    = "E0030"
        ErrParseJSON       = "E0031"
        ErrParseXML        = "E0032"
        ErrParseYAML       = "E0033"
        ErrParseTOML       = "E0034"
        ErrInvalidURL      = "E0035"
        ErrInvalidEmail    = "E0036"
        ErrCryptoFailure   = "E0037"
        ErrDBConnection    = "E0038"
        ErrDBQuery         = "E0039"
        ErrAuthFailure     = "E0040"
        ErrRateLimited     = "E0041"
        ErrFileNotFound    = "E0042"
        ErrInvalidFormat   = "E0043"
)

type ErrorCode struct {
        Code       string
        Title      string
        Suggestion string
        English    string
}

var errorCodes = map[string]ErrorCode{
        ErrUndefinedVar: {
                Code:       ErrUndefinedVar,
                Title:      "Undefined variable",
                Suggestion: "Declare the variable with 'val name = ...' or 'var name = ...' before using it",
        },
        ErrUndefinedFunc: {
                Code:       ErrUndefinedFunc,
                Title:      "Undefined function",
                Suggestion: "Define the function with 'fn name(...) { ... }' before calling it",
        },
        ErrConstReassign: {
                Code:       ErrConstReassign,
                Title:      "Cannot reassign constant",
                Suggestion: "Use 'var' instead of 'val' if you need a mutable variable",
        },
        ErrNotCallable: {
                Code:       ErrNotCallable,
                Title:      "Value is not callable",
                Suggestion: "Make sure the value is a function before calling it with ()",
        },
        ErrNullAccess: {
                Code:       ErrNullAccess,
                Title:      "Null or undefined access",
                Suggestion: "Guard with 'if x != null { ... }' or use optional chaining 'x?.prop'",
        },
        ErrDivisionByZero: {
                Code:       ErrDivisionByZero,
                Title:      "Division by zero",
                Suggestion: "Check the divisor before dividing: 'if b != 0 { a / b }'",
        },
        ErrTypeMismatch: {
                Code:       ErrTypeMismatch,
                Title:      "Type mismatch",
                Suggestion: "Convert the value to the expected type, e.g. Number(x) or String(x)",
        },
        ErrModuleNotFound: {
                Code:       ErrModuleNotFound,
                Title:      "Module not found",
                Suggestion: "Install the module with 'lunex add <module>' or verify it exists in the stdlib",
        },
        ErrIndexOutOfRange: {
                Code:       ErrIndexOutOfRange,
                Title:      "Index out of range",
                Suggestion: "Check array bounds before accessing: 'if i < arr.length { arr[i] }'",
        },
        ErrStackOverflow: {
                Code:       ErrStackOverflow,
                Title:      "Stack overflow",
                Suggestion: "Add a base case to your recursive function to stop infinite recursion",
        },
        ErrUnexpectedToken: {
                Code:       ErrUnexpectedToken,
                Title:      "Unexpected token",
                Suggestion: "Check for missing brackets, colons, or keywords near this position",
        },
        ErrFileNotFound: {
                Code:       ErrFileNotFound,
                Title:      "File not found",
                Suggestion: "Check the file path and ensure the file exists",
        },
        ErrAssertFailed: {
                Code:       ErrAssertFailed,
                Title:      "Assertion failed",
                Suggestion: "The assertion condition evaluated to false; check your logic",
        },
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
