# Spotify Song Tracker

A lightweight Windows application that reads the currently playing song from the Spotify desktop application and writes it to a text file. Perfect for streamers, content creators, or anyone who wants to display or log their currently playing music.

## Features

- Monitors the Spotify application window for song changes
- Writes the current song information to a text file (`current_song.txt`)
- Updates the file only when the song changes
- Ignores paused state (keeps the last song information)
- Works without requiring Spotify API credentials
- Low resource usage
- No installation needed - single executable file

## Download

You can download the latest release from the [Releases](https://github.com/Arkkis/GoSpotifySong/releases/latest) page.

## Requirements

- Windows operating system
- Spotify desktop application

## Usage

1. Start the Spotify desktop application
2. Run `SpotifySongTracker.exe`
3. The current song information will be written to `current_song.txt` in the same directory
4. Press Ctrl+C in the console window to exit the application

### Running at Startup

To have the application run automatically when Windows starts:

1. Create a shortcut to `SpotifySongTracker.exe`
2. Press `Win+R`, type `shell:startup` and press Enter
3. Move the shortcut to the Startup folder that opens

## Using with OBS or Streaming Software

1. In OBS, add a Text source
2. Select "Read from file" and browse to the `current_song.txt` file
3. Customize the text appearance as desired
4. The text will update automatically when the song changes in Spotify

## How It Works

The application:

1. Locates the Spotify application window by identifying its executable
2. Reads the window title, which contains the song and artist information
3. Writes this information to a text file
4. Checks for changes every 2 seconds (configurable)

## Building from Source

### Prerequisites

- Go 1.24.3 or higher
- Git

### Build Instructions

1. Clone the repository:

   ```
   git clone https://github.com/Arkkis/GoSpotifySong.git
   cd GoSpotifySong
   ```

2. Build the executable:
   ```
   go build -o SpotifySongTracker.exe
   ```

### Automated Builds

This project uses GitHub Actions to automatically build and release the application whenever changes are pushed to the main branch. Each new push creates a release with:

- Date-based versioning (YYYY.MM.DD format)
- Windows executable ready to download and use
- Release notes containing the commit message

## Customization

You can modify the following constants in the code:

- `outputFile`: Change the output file location/name (default: `current_song.txt`)
- `checkInterval`: Change how frequently the application checks for song changes (default: every 5 seconds)

## Contributing

Contributions are welcome! Feel free to submit a Pull Request.

## License

This project is open source and available under the MIT License.
