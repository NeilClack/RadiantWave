# GEMINI.md

## Project Overview

This project, named RadiantWave, is a graphical application written in Go. It utilizes the `veandco/go-sdl2` library for creating windows, handling events, and rendering graphics with OpenGL. The application is structured around a "page" system, where different screens like a welcome page, settings page, and scroller page are managed as a stack.

The application starts by initializing various managers for shaders, fonts, and audio. It also connects to a SQLite database (`data.db`) to store configuration and other data. Before the main loop begins, the application validates the network connection, license key, email address, and subscription status.

The core of the application is an event loop that processes user input, updates the current page's state, and renders the scene.

## Building and Running

The `build.sh` script is the recommended way to build the project. It handles the complexities of versioning, packaging, and path management.

To build the project, run the following command:

```sh
./build.sh
```

This will create a `.tar.xz` archive in the root directory containing the application binary and all necessary assets. The script also supports different system and release types. For more information, run `./build.sh --help`.

For simple development builds, you can also use the `go build` command:

```sh
go build -o radiantwave .
```

To run the application, execute the compiled binary:

```sh
./radiantwave
```

### Dependencies

The project has the following dependencies, as defined in `go.mod`:

*   `github.com/veandco/go-sdl2/sdl`
*   `github.com/go-gl/gl`
*   `github.com/go-gl/mathgl`
*   `gorm.io/gorm`
*   `gorm.io/driver/sqlite`
*   `github.com/google/uuid`
*   `github.com/godbus/dbus/v5`

These dependencies will be automatically downloaded and installed when you build the project.

## Build Script

The `build.sh` script automates the build and deployment process. Here are some of the key features:

*   **System Types:** The script supports two system types: `home` and `commercial`. This is controlled by the first argument to the script.
*   **Release Types:** The script supports three release types: `dev`, `beta`, and `release`. This is controlled by the second argument to the script and is automatically determined from the Git branch.
*   **Versioning:** The version is automatically determined from Git tags or commit hashes.
*   **Packaging:** The script packages the application binary, assets, and other system files into a `pkgroot` directory before creating a `.tar.xz` archive.
*   **User-Specific Paths:** The script handles different usernames (`nclack` for development, `localuser` for release) and adjusts paths accordingly.
*   **Updater:** The script templates a `radiantwave-updater` script with the release channel and system type.

For more information on how to use the build script, run `./build.sh --help`.

## Development Conventions

### Project Structure

The project is organized into several packages within the `internal` directory:

*   `application`: Contains the main application logic, including the event loop and page management.
*   `audio`: Handles audio playback.
*   `colors`: Defines a color palette.
*   `db`: Manages the SQLite database connection.
*   `fontManager`: Manages loading and using fonts.
*   `graphics`: Provides graphics-related utility functions.
*   `keybinds`: Manages keyboard shortcuts.
*   `logger`: Implements a logger that writes to the database.
*   `mixer`: Controls audio mixing.
*   `network`: Handles network connectivity checks.
*   `page`: Defines the `Page` interface and contains the implementations of the different pages.
*   `pattern`: Contains implementations of different visual patterns.
*   `shaderManager`: Manages loading and using OpenGL shaders.

### Page System

The application uses a page stack to manage different screens. Each page must implement the `page.Page` interface, which defines the following methods:

*   `Init(app ApplicationInterface) error`: Initializes the page.
*   `HandleEvent(event *sdl.Event) error`: Handles user input events.
*   `Update(dt float32) error`: Updates the page's state.
*   `Render() error`: Renders the page to the screen.
*   `Destroy() error`: Cleans up the page's resources.

A `page.Base` struct is provided to implement common functionality that can be embedded in page implementations.

### Database

The application uses a SQLite database (`data.db`) located in `~/.local/share/radiantwave/` to store configuration and other data. The `gorm` library is used as an ORM.
