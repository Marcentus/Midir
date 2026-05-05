//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

func messagebox(msg string) {
	// https://gist.github.com/NaniteFactory/0bd94e84bbe939cda7201374a0c261fd

	const (
		title = "Midir"
		MB_OK = 0
	)

	hwnd := 0

	titlePtr, _ := syscall.UTF16PtrFromString(title)
	msgPtr, _ := syscall.UTF16PtrFromString(msg)

	ret, _, _ := syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(msgPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(MB_OK),
	)

	_ = ret
}
