//go:build windows

// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package main

import _ "embed"

//go:embed bin/lunex-rt.exe
var embeddedZigRT []byte
