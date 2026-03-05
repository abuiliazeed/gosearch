#!/bin/bash
# Mock HTTP server that returns 429 (Too Many Requests)
# Used for testing block detection and backoff

# Configuration
PORT="${BLOCK_SERVER_PORT:-9999}"

# Colors for output
GREEN='\033[0;32m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

# Check if nc (netcat) is available
if ! command -v nc &> /dev/null; then
    echo "Error: nc (netcat) is required but not installed"
    echo "Install with: brew install netcat (macOS) or apt-get install netcat (Linux)"
    exit 1
fi

log_info "Starting mock block server on port $PORT..."
log_info "This server returns 429 (Too Many Requests) for all requests"
log_info "Press Ctrl+C to stop"

# Simple HTTP server using netcat
while true; do
    {
        # Read the request line
        read -r request_line

        # Skip headers
        while read -r line && [ "$line" != $'\r' ] && [ -n "$line" ]; do
            :
        done

        # Send 429 response
        echo "HTTP/1.1 429 Too Many Requests"
        echo "Content-Type: text/html"
        echo "Retry-After: 5"
        echo "Connection: close"
        echo ""
        echo "<html><body>Too Many Requests - Rate limit exceeded</body></html>"
    } | nc -l "$PORT"

    log_info "Request handled, server restarting..."
done
