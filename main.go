package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	outputFile     = "current_song.txt"
	checkInterval  = 2 * time.Second
	spotifyAppName = "Spotify"
	pausedTitle    = "Spotify Premium"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowTextLen         = user32.NewProc("GetWindowTextLengthW")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procGetModuleFileNameExW     = kernel32.NewProc("K32GetModuleFileNameExW")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
)

// isSpotifyWindow checks if a window belongs to the Spotify executable
func isSpotifyWindow(hwnd syscall.Handle) bool {
	var processID uint32

	// Get process ID from window handle
	_, _, _ = procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&processID)))
	if processID == 0 {
		return false
	}

	// Windows process access rights
	const PROCESS_QUERY_INFORMATION = 0x0400
	const PROCESS_VM_READ = 0x0010

	// Open process to get more information
	processHandle, _, _ := procOpenProcess.Call(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ, 0, uintptr(processID))
	if processHandle == 0 {
		return false
	}

	defer procCloseHandle.Call(processHandle)

	// Get the executable path
	exePath := make([]uint16, syscall.MAX_PATH)
	_, _, _ = procGetModuleFileNameExW.Call(
		processHandle,
		0,
		uintptr(unsafe.Pointer(&exePath[0])),
		uintptr(len(exePath)),
	)

	path := syscall.UTF16ToString(exePath)
	return strings.Contains(strings.ToLower(path), "spotify.exe")
}

// getWindowTitle gets the title of a window
func getWindowTitle(hwnd syscall.Handle) string {
	textLen, _, _ := procGetWindowTextLen.Call(uintptr(hwnd))
	if textLen == 0 {
		return ""
	}

	buf := make([]uint16, textLen+1)
	procGetWindowTextW.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&buf[0])),
		textLen+1)

	return syscall.UTF16ToString(buf)
}

// isSpotifyPaused checks if Spotify is paused based on its window title
func isSpotifyPaused(title string) bool {
	return title == pausedTitle || title == spotifyAppName
}

// getCurrentSong finds the Spotify window and returns the song information from its title
// Returns empty string when Spotify is paused or not playing
func getCurrentSong() (string, bool, error) {
	var spotifyTitle string

	// Callback function for EnumWindows
	cb := syscall.NewCallback(func(hwnd syscall.Handle, lParam uintptr) uintptr {
		// Skip invisible windows
		isVisible, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
		if isVisible == 0 {
			return 1 // Continue enumeration
		}

		// Check if it's a Spotify window
		if isSpotifyWindow(hwnd) {
			title := getWindowTitle(hwnd)

			if title != "" {
				spotifyTitle = title
				return 0 // Stop enumeration
			}
		}

		return 1 // Continue enumeration
	})

	// Enumerate all windows
	procEnumWindows.Call(cb, 0)

	if spotifyTitle == "" {
		return "", false, fmt.Errorf("spotify window not found")
	}

	// If Spotify is paused, return empty string with isPaused flag
	if isSpotifyPaused(spotifyTitle) {
		return "", true, nil
	}

	return spotifyTitle, false, nil
}

// writeSongToFile writes the given song information to a file
func writeSongToFile(songInfo string) error {
	return os.WriteFile(outputFile, []byte(songInfo), 0644)
}

func main() {
	fmt.Println("Spotify Song Tracker started")
	fmt.Println("Monitoring Spotify and writing current song to:", outputFile)
	fmt.Println("Press Ctrl+C to exit")

	var lastSong string

	// Main loop - check Spotify every few seconds
	for {
		songInfo, isPaused, err := getCurrentSong()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else if isPaused {
			// Do nothing when Spotify is paused
		} else if songInfo != lastSong {
			lastSong = songInfo
			if err := writeSongToFile(songInfo); err != nil {
				fmt.Printf("Error writing to file: %v\n", err)
			} else {
				fmt.Printf("Now playing: %s\n", songInfo)
			}
		}

		time.Sleep(checkInterval)
	}
}
