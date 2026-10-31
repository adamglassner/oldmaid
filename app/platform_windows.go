//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	pMessageBox    = user32.NewProc("MessageBoxW")
	pFindWindow    = user32.NewProc("FindWindowW")
	pShowWindow    = user32.NewProc("ShowWindow")
	pSetForeground = user32.NewProc("SetForegroundWindow")
	pCreateMutex   = kernel32.NewProc("CreateMutexW")
)

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

func fatal(text string) {
	pMessageBox.Call(0, uintptr(unsafe.Pointer(u16(text))), uintptr(unsafe.Pointer(u16(productName))), 0x10)
	os.Exit(1)
}

func singleInstance() bool {
	_, _, err := pCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(u16(`Local\IroniteOldMaid`))))
	if err == syscall.Errno(183) { // already running: bring its window forward
		if h, _, _ := pFindWindow.Call(0, uintptr(unsafe.Pointer(u16("Old Maid "+version)))); h != 0 {
			pShowWindow.Call(h, 9)
			pSetForeground.Call(h)
		}
		return false
	}
	return true
}

func findBrowser() string {
	var roots []string
	for _, k := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		if v := os.Getenv(k); v != "" {
			roots = append(roots, v)
		}
	}
	for _, rel := range []string{`Microsoft\Edge\Application\msedge.exe`, `Google\Chrome\Application\chrome.exe`, `BraveSoftware\Brave-Browser\Application\brave.exe`} {
		for _, r := range roots {
			p := filepath.Join(r, rel)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

func openWindow(url string) (func(), bool) {
	b := findBrowser()
	if b == "" {
		return nil, false
	}
	profile := filepath.Join(cacheDir(), "WebView")
	_ = os.MkdirAll(profile, 0o755)
	cmd := exec.Command(b, "--app="+url, "--user-data-dir="+profile, "--window-size=1366,900",
		"--no-first-run", "--no-default-browser-check", "--disable-extensions", "--autoplay-policy=no-user-gesture-required",
		"--disable-features=Translate,msEdgeSidebarV2,msUndersideButton,msHubApps", "--disable-pinch", "--overscroll-history-navigation=0")
	if err := cmd.Start(); err != nil {
		return nil, false
	}
	return func() { _ = cmd.Wait() }, true
}

func openDefaultBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
