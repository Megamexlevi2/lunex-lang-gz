// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.


// left aside for a while, don't call
package codegen

import "runtime"

type Arch string

const (
	ArchAMD64 Arch = "amd64"
	ArchARM64 Arch = "arm64"
	ArchX86   Arch = "386"
)

type OS string

const (
	OSLinux   OS = "linux"
	OSWindows OS = "windows"
	OSDarwin  OS = "darwin"
	OSAndroid OS = "android"
)

type Target struct {
	Arch Arch
	OS   OS
}

func HostTarget() Target {
	var arch Arch
	switch runtime.GOARCH {
	case "amd64":
		arch = ArchAMD64
	case "arm64":
		arch = ArchARM64
	default:
		arch = ArchAMD64
	}
	var os OS
	switch runtime.GOOS {
	case "linux":
		os = OSLinux
	case "windows":
		os = OSWindows
	case "darwin":
		os = OSDarwin
	case "android":
		os = OSAndroid
	default:
		os = OSLinux
	}
	return Target{Arch: arch, OS: os}
}

func ParseTarget(s string) (Target, bool) {
	known := []Target{
		{ArchAMD64, OSLinux},
		{ArchARM64, OSLinux},
		{ArchAMD64, OSWindows},
		{ArchARM64, OSWindows},
		{ArchARM64, OSAndroid},
		{ArchAMD64, OSDarwin},
		{ArchARM64, OSDarwin},
	}
	for _, t := range known {
		if string(t.OS)+"/"+string(t.Arch) == s {
			return t, true
		}
	}
	return Target{}, false
}
