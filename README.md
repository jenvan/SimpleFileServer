# Simple File Server

A simple and powerful file server written in Go with support for browsing, uploading, downloading, moving, copying, deleting files with resume capability.

## Features

- 📁 **File Browsing** - Navigate through directories with a clean web interface
- 📤 **File Upload** - Upload files via web form or drag & drop
- 📥 **File Download** - Support files download
- 📄 **File Preview** - Support common files previews (such as text and image files, etc.)
- 🔃 **File Move 、Copy、Delete** - Also support files or directory move, copy and delete
- 🧰 **API Support** - Fully REST API support
- 🔒 **Security** - Path traversal protection to prevent accessing files outside the working directory
- 🎨 **Modern UI** - Clean and responsive interface
- ⚡ **Lightweight** - Single binary with embedded templates

## Installation

### Install (recommended)

Install the latest stable release with the `go` command:

```bash
go install github.com/jenvan/sfs@latest
```

Install the current tip of the default branch (useful for nightly/testing builds):

```bash
go install github.com/jenvan/sfs@master
```

### Download pre-compiled binaries

You can also download pre-compiled binaries from the [nightly releases](https://github.com/jenvan/sfs/releases):

1. Go to the [Releases page](https://github.com/jenvan/sfs/releases)
2. Download the appropriate binary for your platform (Windows, Linux, macOS)
3. Make the binary executable (on Unix-like systems): `chmod +x sfs`
4. Move it to a directory in your PATH or run it directly

Notes:
- `go install ...@latest` installs the latest released module version.
- Installing `@master` (or `@main`) fetches the tip of the branch — treat this as a nightly/edge build.
- The installed binary is placed in `$GOBIN` (if set) or `$(go env GOPATH)/bin`; make sure that directory is on your `PATH`:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Build from source

If you prefer to build locally:

```bash
git clone https://github.com/jenvan/sfs.git
cd sfs
go build -o sfs ./...
```

Requirements:
- Go 1.21 or newer (see `go.mod`).


## Usage

### Basic Usage

Run the server in the current directory on port 9527
```bash
./sfs
```

### Command-Line Options

```bash
./sfs [options]
```

Options:
- `-host <address>` - Address to listen on (default: 0.0.0.0)
- `-port <port>` - Port to listen on (default: 9527)
- `-dir <directory>` - Working directory to serve files from (default: current directory)

### Examples

Run on custom port:
```bash
./sfs -port 9000
```

Listen only on localhost:
```bash
./sfs -host 127.0.0.1
```

Serve files from a specific directory:
```bash
./sfs -dir /path/to/files
```

Combine options:
```bash
./sfs -host 192.168.1.100 -port 9000 -dir /path/to/files
```

## Features Details

### File Browsing
- Navigate through directories using the web interface
- View file sizes and modification times
- Breadcrumb navigation for easy path traversal

### File Upload
- Click "Upload File" button to upload
- Select a file or drag and drop onto the page
- Upload progress indicator shows transfer status

### File Preview / Download
- Support previewing file content, if the file extension is in text or image format, such as .txt, .png
- Click on any file to download it
- Automatic file name preservation

### Security
- Path traversal protection prevents accessing files outside the configured directory
- All paths are validated and sanitized
- No execution of uploaded files

## API Endpoints
- `Accept: application/json` or `Content-Type: application/json` in header or use query params `?format=json`
- `GET /<directory>` - Browse files in a specific directory
- `GET /<directory>/<filename>` - Get file content by base64 encode
- `POST /<directory>/<filename>` - Upload a file with json body `{content: base64_encode("xxx")}`
- `PUT /<path>` - Move or Copy file or directory with json body `{act: 'move|copy', dst: '/some/path'}`
- `DELETE /<path>` - Delete file or directory

## Technical Details

- **Language**: Go
- **Dependencies**: Standard library only
- **Templates**: Embedded in binary using `embed` package
- **API Features**: Support for REST style interfaces
- **Maximum upload size**: 100MB in memory

## License

See LICENSE file for details.
