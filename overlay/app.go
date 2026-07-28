package main

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type MARGINS struct {
	CxLeftWidth    int32
	CxRightWidth   int32
	CyTopHeight    int32
	CyBottomHeight int32
}

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procFindWindow               = user32.NewProc("FindWindowW")
	dwmapi                       = syscall.NewLazyDLL("dwmapi.dll")
	procDwmExtendFrameIntoClient = dwmapi.NewProc("DwmExtendFrameIntoClientArea")
	procDwmSetWindowAttribute    = dwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	DWMWA_NCRENDERING_POLICY  uintptr = 2
	DWMNCRP_DISABLED          uint32  = 1
	DWMWA_BORDER_COLOR        uintptr = 34
	DWMWA_COLOR_NONE          uint32  = 0xFFFFFFFE
	DWMWA_SYSTEMBACKDROP_TYPE uintptr = 38
	DWMSBT_NONE               uint32  = 1
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Apply DwmExtendFrameIntoClientArea, DWMSBT_NONE, DWMWA_COLOR_NONE, and DWMNCRP_DISABLED
	go func() {
		for i := 0; i < 15; i++ {
			time.Sleep(100 * time.Millisecond)
			titlePtr, err := syscall.UTF16PtrFromString("midir-overlay")
			if err != nil {
				continue
			}
			hwnd, _, _ := procFindWindow.Call(0, uintptr(unsafe.Pointer(titlePtr)))
			if hwnd != 0 {
				// 1. Extend margins into client area for full alpha rendering
				margins := MARGINS{-1, -1, -1, -1}
				procDwmExtendFrameIntoClient.Call(hwnd, uintptr(unsafe.Pointer(&margins)))

				// 2. Set DWMSBT_NONE (1) to disable Windows 10/11 focus-dependent DWM backdrop tinting
				backdropNone := DWMSBT_NONE
				procDwmSetWindowAttribute.Call(
					hwnd,
					DWMWA_SYSTEMBACKDROP_TYPE,
					uintptr(unsafe.Pointer(&backdropNone)),
					uintptr(unsafe.Sizeof(backdropNone)),
				)

				// 3. Set DWMWA_BORDER_COLOR to DWMWA_COLOR_NONE (0xFFFFFFFE) to remove Windows 11 system 1px grey border
				borderColorNone := DWMWA_COLOR_NONE
				procDwmSetWindowAttribute.Call(
					hwnd,
					DWMWA_BORDER_COLOR,
					uintptr(unsafe.Pointer(&borderColorNone)),
					uintptr(unsafe.Sizeof(borderColorNone)),
				)

				// 4. Set DWMWA_NCRENDERING_POLICY to DWMNCRP_DISABLED (1) to remove OS system drop shadow
				ncPolicyDisabled := DWMNCRP_DISABLED
				procDwmSetWindowAttribute.Call(
					hwnd,
					DWMWA_NCRENDERING_POLICY,
					uintptr(unsafe.Pointer(&ncPolicyDisabled)),
					uintptr(unsafe.Sizeof(ncPolicyDisabled)),
				)
				break
			}
		}
	}()
}

// GetWindowSize returns [width, height] of the window
func (a *App) GetWindowSize() []int {
	w, h := runtime.WindowGetSize(a.ctx)
	return []int{w, h}
}

// SetWindowSize sets window dimensions
func (a *App) SetWindowSize(width int, height int) {
	runtime.WindowSetSize(a.ctx, width, height)
}

// SetAlwaysOnTop toggles pin state
func (a *App) SetAlwaysOnTop(alwaysOnTop bool) {
	runtime.WindowSetAlwaysOnTop(a.ctx, alwaysOnTop)
}

// SaveOverlaySettings saves local settings to overlay_settings.json
func (a *App) SaveOverlaySettings(jsonStr string) bool {
	configPath := filepath.Join(".", "overlay_settings.json")
	err := os.WriteFile(configPath, []byte(jsonStr), 0644)
	return err == nil
}

// LoadOverlaySettings loads local settings from overlay_settings.json
func (a *App) LoadOverlaySettings() string {
	configPath := filepath.Join(".", "overlay_settings.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	return string(data)
}

// MinimizeWindow minimizes overlay
func (a *App) MinimizeWindow() {
	runtime.WindowMinimise(a.ctx)
}

// QuitApp closes overlay
func (a *App) QuitApp() {
	runtime.Quit(a.ctx)
}
