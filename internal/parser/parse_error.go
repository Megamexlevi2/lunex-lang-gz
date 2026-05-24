// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package parser

type ParseError struct {
	Message string
	File    string
	Line    int
	Col     int
}

func (e *ParseError) Error() string {
	return e.Message
}
