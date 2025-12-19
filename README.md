# livedom

A fast and efficient live domain/URL checker written in Go. Similar to httpx, but optimized for high-performance bulk processing using fasthttp.

## Features

- ⚡ **Blazing Fast**: Powered by [fasthttp](https://github.com/valyala/fasthttp) for maximum performance
- 🔄 **Streaming Processing**: Processes URLs as they come in (no buffering)
- 🌐 **Smart URL Detection**: Handles both domains and full URLs
- 🎨 **Color-coded Output**: Status codes and fields are color-coded for easy identification
- 📊 **Rich Information**: Display status codes, content type, hash, title, server, IP, CNAME, and content length
- 🔀 **HTTP/HTTPS Fallback**: Tries HTTPS first, falls back to HTTP
- 🚀 **Concurrent Processing**: Configurable thread count for optimal performance
- ✅ **Any Response = Live**: Accepts 2xx, 3xx, 4xx, 5xx as "live" (matches httpx behavior)
- 💾 **File Output Support**: Colors work even when redirecting output to files

## Installation

### Prerequisites

- Go 1.21 or higher

### Install via go install (Recommended)

```bash
go install github.com/hackruler/livedom@latest
```

Make sure `$GOPATH/bin` is in your `$PATH`:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Build from source

```bash
git clone https://github.com/hackruler/livedom.git
cd livedom
go build -o livedom main.go
```

Or build and install:

```bash
go build -o livedom main.go
sudo mv livedom /usr/local/bin/
```

## Usage

### Basic Usage

Check live domains from stdin:

```bash
echo "example.com" | livedom
```

Check multiple domains:

```bash
cat domains.txt | livedom
```

### With Status Codes

Display status codes with color-coded output:

```bash
cat domains.txt | livedom -sc
```

### Custom Threads

Increase concurrent threads for faster processing:

```bash
cat domains.txt | livedom -sc -t 200
```

### Custom Timeout

Set custom request timeout (default: 5s):

```bash
cat domains.txt | livedom -sc -timeout 2s
```

### Input from File

Read domains from a file:

```bash
livedom -f domains.txt -sc
```

### Update Tool

Update livedom to the latest version:

```bash
livedom -up
```

This will run `go install github.com/hackruler/livedom@latest` to update the tool.

## Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-sc` | Show status code | `false` |
| `-ct` | Show content type | `false` |
| `-hash` | Show SHA256 hash of response body | `false` |
| `-title` | Show page title (extracted from HTML) | `false` |
| `-server` | Show server name from headers | `false` |
| `-ip` | Show IP address (DNS resolution) | `false` |
| `-cname` | Show CNAME record | `false` |
| `-cl` | Show content length | `false` |
| `-up` | Update livedom to the latest version | `false` |
| `-t` | Number of concurrent threads | `50` |
| `-timeout` | Request timeout duration | `5s` |
| `-f` | Input file (default: stdin) | `""` |

## Examples

### Basic Domain Check

```bash
$ echo "google.com" | livedom
https://google.com
```

### With Status Code

```bash
$ echo "google.com" | livedom -sc
https://google.com [301]
```

### With Multiple Flags

```bash
# Status code and content type
$ echo "google.com" | livedom -sc -ct
https://google.com [301] [text/html]

# Status code, IP, and server
$ echo "google.com" | livedom -sc -ip -server
https://google.com [301] [142.250.76.78] [gws]

# All information
$ echo "google.com" | livedom -sc -ct -hash -title -server -ip -cname -cl
https://google.com [301] [text/html] [a1b2c3...] [Google] [gws] [142.250.76.78] [google.com] [12345]
```

### Process Full URLs

```bash
$ cat urls.txt | livedom -sc
https://example.com/path?query=1 [200]
http://example.com/other [301]
```

### Chain with Other Tools

```bash
# With waybackurls
cat subdomains.txt | waybackurls | livedom -sc | anew live_urls.txt

# With subfinder
subfinder -d example.com | livedom -sc

# With amass
amass enum -d example.com | livedom -sc
```

### High-Performance Processing

For large-scale processing (millions of URLs):

```bash
cat urls.txt | livedom -sc -t 2000 -timeout 2s >> live_urls.txt
```

## Output Format

### Without Status Code (`-sc`)

```
https://example.com
http://example.com
```

### With Status Code (`-sc`)

```
https://example.com [200]
http://example.com [301]
https://example.com [404]
```

### With Multiple Flags

```
# Status code and content type
https://example.com [200] [text/html]

# Status code, IP, and server
https://example.com [200] [93.184.216.34] [nginx/1.18.0]

# Empty values show as empty brackets
https://example.com [200] [] [nginx/1.18.0]
```

### Color Coding

- **Status Codes**:
  - **Green**: 2xx (Success)
  - **Yellow**: 3xx (Redirect)
  - **Red**: 4xx (Client Error)
  - **Magenta**: 5xx (Server Error)
  - **White**: Other status codes
- **Content Type**: Yellow
- **Content Length**: Cyan
- **Hash**: Magenta
- **Title**: Blue
- **Server**: Green
- **IP**: Cyan
- **CNAME**: Yellow

### Empty Values

When a flag is set but the value is not available, empty brackets `[]` are displayed. This ensures consistent output format and makes it easy to parse results.

## How It Works

1. **URL Detection**: Automatically detects if input is a full URL or just a domain
2. **HTTPS First**: Tries HTTPS connection first, then falls back to HTTP
3. **Any Response = Live**: Any HTTP response (including 4xx/5xx) is considered "live"
4. **Streaming**: Processes URLs as they arrive, no buffering
5. **Connection Reuse**: Efficient connection pooling for better performance
6. **Data Extraction**: Extracts headers, body (limited to 8KB for performance), and performs DNS resolution
7. **Color Output**: Always outputs ANSI color codes, even when redirecting to files

## Performance

- **Streaming Processing**: No memory buffering, processes URLs in real-time
- **Fast HTTP Client**: Uses fasthttp for maximum throughput
- **Connection Pooling**: Reuses connections for better performance
- **Concurrent Workers**: Configurable thread pool for optimal speed

## Comparison with httpx

| Feature | livedom | httpx |
|---------|---------|-------|
| Speed | ⚡ Very Fast (fasthttp) | Fast (net/http) |
| Streaming | ✅ Yes | ✅ Yes |
| Status Codes | ✅ Yes | ✅ Yes |
| Content Type | ✅ Yes | ✅ Yes |
| Hash (SHA256) | ✅ Yes | ✅ Yes |
| Title Extraction | ✅ Yes | ✅ Yes |
| Server Header | ✅ Yes | ✅ Yes |
| IP Resolution | ✅ Yes | ✅ Yes |
| CNAME Resolution | ✅ Yes | ✅ Yes |
| Content Length | ✅ Yes | ✅ Yes |
| Color Output | ✅ Yes | ✅ Yes |
| Colors in Files | ✅ Yes | ✅ Yes |
| Full URL Support | ✅ Yes | ✅ Yes |
| Thread Control | ✅ Yes | ✅ Yes |

## Tips

1. **For large files**: Use higher thread count (`-t 1000` or more)
2. **For faster processing**: Reduce timeout (`-timeout 2s`)
3. **For streaming**: Works perfectly with tools like `waybackurls`, `subfinder`, etc.
4. **For output**: Use `anew` to avoid duplicates: `livedom -sc | anew live.txt`
5. **For file output**: Colors are preserved when redirecting to files: `livedom -sc -ct >> output.txt`
6. **For comprehensive info**: Combine multiple flags: `livedom -sc -ct -ip -server -hash`
7. **For title extraction**: Only reads first 8KB of response body for performance

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License

## Author

Created with ❤️ for the security community

