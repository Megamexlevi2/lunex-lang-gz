package parser

import "lunex/internal/errfmt"

// ParseError is kept for backwards compatibility; internally the parser
// now produces *errfmt.LunexError directly so the full source view,
// contextual suggestions, and underline are always present.
type ParseError = errfmt.LunexError
