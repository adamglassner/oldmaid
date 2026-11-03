// Old Maid - A Victorian Parlor Game (c) 2026 Ironite.
package main

import (
	"syscall"
	"unsafe"
)

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	user32         = syscall.NewLazyDLL("user32.dll")
	pCreateMutex   = kernel32.NewProc("CreateMutexW")
	pMessageBox    = user32.NewProc("MessageBoxW")
	pFindWindow    = user32.NewProc("FindWindowW")
	pSetForeground = user32.NewProc("SetForegroundWindow")
	pShowWindow    = user32.NewProc("ShowWindow")
	pIsIconic      = user32.NewProc("IsIconic")
)

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

func fatal(text string) {
	pMessageBox.Call(0, uintptr(unsafe.Pointer(u16(text))), uintptr(unsafe.Pointer(u16("Old Maid"))), 0x10)
}

// singleInstance makes a second launch bring the open game to the front instead.
func singleInstance() bool {
	_, _, err := pCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(u16(`Local\IroniteOldMaid`))))
	if err == syscall.Errno(183) { // ERROR_ALREADY_EXISTS
		if h, _, _ := pFindWindow.Call(0, uintptr(unsafe.Pointer(u16(productName+" "+version)))); h != 0 {
			if ic, _, _ := pIsIconic.Call(h); ic != 0 {
				pShowWindow.Call(h, 9) // SW_RESTORE
			}
			pSetForeground.Call(h)
		}
		return false
	}
	return true
}
