// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package main

import (
	"embed"
	"fmt"
	iofs "io/fs"
	"path"
	"regexp"
	"strings"
)

//go:embed lib
var libFS embed.FS

const nativePreamble = `val native   = @import("internal.native")
val io       = native.io
val fs       = native.fs
val http     = native.http
val crypto   = native.crypto
val db       = native.db
val env      = native.env
val ws       = native.ws
val mail     = native.mail
val ai       = native.ai
val utils    = native.utils
val validate = native.validate
val xml      = native.xml
val os       = native.os
val jwt      = native.jwt
val redis    = native.redis
val postgres = native.postgres
val mysql    = native.mysql
val stripe   = native.stripe
val oauth2   = native.oauth2
val graphql  = native.graphql
val rabbitmq = native.rabbitmq
val excel    = native.excel
val pdf      = native.pdf
val csv      = native.csv
val yaml     = native.yaml
val toml     = native.toml
val markdown = native.markdown
val mustache = native.mustache
val alloc    = native.alloc
val zip      = native.zip
val regex    = native.regex
val math     = native.math
val datetime = native.datetime
val path     = native.path
val compress = native.compress

`

var libNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)

var trustedSourcePrefixes = []string{
	"github.com/Megamexlevi2/lunex-modules",
	"https://github.com/Megamexlevi2/lunex-modules",
	"raw.githubusercontent.com/Megamexlevi2/lunex-modules",
	"https://raw.githubusercontent.com/Megamexlevi2/lunex-modules",
	"embedded:",
}

func validateLibName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("empty library name")
	}

	name = strings.ReplaceAll(name, "\\", "/")

	if strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("absolute paths are not allowed")
	}

	cleaned := path.Clean(name)

	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("invalid library name")
	}

	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("path traversal is not allowed")
	}

	if !libNamePattern.MatchString(cleaned) {
		return "", fmt.Errorf("invalid library name")
	}

	if !iofs.ValidPath("lib/" + cleaned + ".lx") {
		return "", fmt.Errorf("invalid library path")
	}

	return cleaned, nil
}

func isTrustedSource(source string) bool {
	source = strings.TrimSpace(source)
	if source == "" {
		return false
	}
	for _, prefix := range trustedSourcePrefixes {
		if strings.HasPrefix(source, prefix) {
			return true
		}
	}
	return false
}

func shouldGrantNativeAccess(source string) bool {
	return isTrustedSource(source)
}

func wrapPrivileged(source, src string) string {
	if shouldGrantNativeAccess(source) {
		return nativePreamble + src
	}
	return src
}

func loadEmbeddedLib(name string) (string, bool) {
	cleanName, err := validateLibName(name)
	if err != nil {
		return "", false
	}

	data, err := libFS.ReadFile("lib/" + cleanName + ".lx")
	if err != nil {
		return "", false
	}

	return wrapPrivileged("embedded:"+cleanName, string(data)), true
}
