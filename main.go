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
	youtubePrefix  = " - YouTube"
)

// BrowserInfo holds information about browser window title patterns
type BrowserInfo struct {
	name        string
	titleMarker string
	separator   string // Character sequence between YouTube and browser name
}

// Define supported browsers and their title formats
var supportedBrowsers = []BrowserInfo{
	{"Edge", "Microsoft Edge", " and"},
	{"Edge", "Edge", " and"},
	{"Chrome", "Google Chrome", " - "},
	{"Firefox", "Mozilla Firefox", " — "},
	{"Brave", "Brave", " - "},
	{"Opera", "Opera", " - "},
}

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

// isAppWindow checks if a window belongs to the specified executable
func isAppWindow(hwnd syscall.Handle, exeNameContains string) bool {
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
	return strings.Contains(strings.ToLower(path), strings.ToLower(exeNameContains))
}

// isSpotifyWindow checks if a window belongs to the Spotify executable
func isSpotifyWindow(hwnd syscall.Handle) bool {
	return isAppWindow(hwnd, "spotify.exe")
}

// isBrowserWindow checks if a window belongs to a browser
func isBrowserWindow(hwnd syscall.Handle) bool {
	// Check for common browsers
	return isAppWindow(hwnd, "chrome.exe") ||
		isAppWindow(hwnd, "firefox.exe") ||
		isAppWindow(hwnd, "msedge.exe") ||
		isAppWindow(hwnd, "brave.exe") ||
		isAppWindow(hwnd, "opera.exe")
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

// isYoutubeMusic checks if the window title is from YouTube and contains a music video
func isYoutubeMusic(title string) bool {
	// Simple case: title ends with "- YouTube"
	if strings.HasSuffix(title, youtubePrefix) && title != youtubePrefix {
		return true
	}

	// Check all supported browser patterns
	for _, browser := range supportedBrowsers {
		if strings.Contains(title, youtubePrefix) && strings.Contains(title, browser.titleMarker) {
			return true
		}
	}

	return false
}

// extractYoutubeTitle extracts the actual title from a YouTube window title
func extractYoutubeTitle(title string) string {
	// Standard YouTube suffix - simplest case
	if strings.HasSuffix(title, youtubePrefix) {
		return strings.TrimSuffix(title, youtubePrefix)
	}

	// Check for known browser patterns
	for _, browser := range supportedBrowsers {
		pattern := youtubePrefix + browser.separator + browser.titleMarker
		if strings.Contains(title, pattern) {
			parts := strings.Split(title, youtubePrefix)
			if len(parts) > 0 {
				return parts[0]
			}
		}
	}

	// Generic case - try to extract title before YouTube prefix
	if strings.Contains(title, youtubePrefix) {
		parts := strings.Split(title, youtubePrefix)
		if len(parts) > 0 {
			return parts[0]
		}
	}

	// Fallback - return original title
	return title
}

// extractSongInfo extracts song information from window title
func extractSongInfo(title string, source string) string {
	if source == "spotify" {
		return title
	} else if source == "youtube" {
		return extractYoutubeTitle(title)
	}
	return ""
}

// MediaInfo holds information about a detected media playing window
type MediaInfo struct {
	title  string
	source string
}

// getMediaInfo finds music playing windows and returns the song information
func getMediaInfo() (string, string, bool, error) {
	var mediaInfos []MediaInfo

	// Callback function for EnumWindows
	cb := syscall.NewCallback(func(hwnd syscall.Handle, lParam uintptr) uintptr {
		// Skip invisible windows
		isVisible, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
		if isVisible == 0 {
			return 1 // Continue enumeration
		}

		title := getWindowTitle(hwnd)
		if title == "" {
			return 1 // Continue enumeration
		}

		// Check if it's a Spotify window
		if isSpotifyWindow(hwnd) {
			if !isSpotifyPaused(title) {
				mediaInfos = append(mediaInfos, MediaInfo{
					title:  title,
					source: "spotify",
				})
			}
		}

		// Check if it's a browser window with YouTube music
		if isBrowserWindow(hwnd) && isYoutubeMusic(title) {
			mediaInfos = append(mediaInfos, MediaInfo{
				title:  title,
				source: "youtube",
			})
		}

		return 1 // Continue enumeration to check all windows
	})

	// Enumerate all windows
	procEnumWindows.Call(cb, 0)

	if len(mediaInfos) == 0 {
		return "", "", false, fmt.Errorf("no music window found")
	}

	// Prioritize Spotify over YouTube if both are found
	var selectedMedia MediaInfo

	for _, info := range mediaInfos {
		if info.source == "spotify" {
			selectedMedia = info
			break
		}
		selectedMedia = info // Use whatever was found if no Spotify
	}

	songInfo := extractSongInfo(selectedMedia.title, selectedMedia.source)
	// Don't return isPaused in this function as we already filtered out paused Spotify windows
	// and there's no reliable way to detect paused state for YouTube
	return songInfo, selectedMedia.source, false, nil
}

// writeSongToFile writes the given song information to a file
func writeSongToFile(songInfo string) error {
	return os.WriteFile(outputFile, []byte(songInfo), 0644)
}

func main() {
	fmt.Println("Media Player Tracker started")
	fmt.Println("Monitoring Spotify and YouTube and writing current song to:", outputFile)
	fmt.Println("Press Ctrl+C to exit")

	var lastSong string
	var lastSource string

	// Main loop - check media players every few seconds
	for {
		songInfo, source, _, err := getMediaInfo()

		// Only update file when song changes and is valid
		if err == nil && songInfo != "" && (songInfo != lastSong || source != lastSource) {
			lastSong = songInfo
			lastSource = source

			if err := writeSongToFile(songInfo); err != nil {
				fmt.Printf("Error writing to file: %v\n", err)
			} else {
				fmt.Printf("Now playing (%s): %s\n", source, songInfo)
			}
		}

		time.Sleep(checkInterval)
	}
}
