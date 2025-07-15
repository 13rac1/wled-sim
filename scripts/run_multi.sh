#!/bin/bash

# Multi-instance WLED Simulator Runner
# Runs three instances with different ports and configurations

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration for each instance
INSTANCES=(
    "Instance-1 8080 4048 2 5 row #FF0000"
    "Instance-2 8081 4049 3 4 col #00FF00"
    "Instance-3 8082 4050 4 3 row #0000FF"
)

# Array to store PIDs
PIDS=()

# Function to print colored output
print_status() {
    echo -e "${GREEN}[MULTI-SIM]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to cleanup on exit
cleanup() {
    print_status "Shutting down all instances..."
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            print_status "Stopping process $pid"
            kill "$pid" 2>/dev/null || true
        fi
    done
    
    # Wait a moment for graceful shutdown
    sleep 2
    
    # Force kill if still running
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            print_warning "Force killing process $pid"
            kill -9 "$pid" 2>/dev/null || true
        fi
    done
    
    print_status "All instances stopped"
}

# Set up signal handlers
trap cleanup EXIT INT TERM

# Check if binary exists
BINARY="./build/wled-sim"
if [ ! -f "$BINARY" ]; then
    print_error "Binary not found at $BINARY"
    print_status "Building binary..."
    make build
    if [ ! -f "$BINARY" ]; then
        print_error "Failed to build binary"
        exit 1
    fi
fi

print_status "Starting multiple WLED simulator instances..."
echo

# Start each instance
for i in "${!INSTANCES[@]}"; do
    # Parse configuration
    IFS=' ' read -r name http_port ddp_port rows cols wiring color <<< "${INSTANCES[$i]}"
    
    print_status "Starting $name:"
    print_status "  HTTP: localhost:$http_port"
    print_status "  DDP:  localhost:$ddp_port"
    print_status "  Matrix: ${rows}x${cols} (${wiring}-major)"
    print_status "  Color: $color"
    
    # Start the instance in background
    $BINARY \
        -rows "$rows" \
        -cols "$cols" \
        -wiring "$wiring" \
        -http ":$http_port" \
        -ddp-port "$ddp_port" \
        -init "$color" \
        -v &
    
    # Store the PID
    pid=$!
    PIDS+=("$pid")
    
    print_status "  Started with PID: $pid"
    echo
    
    # Small delay to avoid startup conflicts
    sleep 1
done

print_status "All instances started successfully!"
print_status "Press Ctrl+C to stop all instances"
echo

# Display running instances
print_status "Running instances:"
for i in "${!INSTANCES[@]}"; do
    IFS=' ' read -r name http_port ddp_port rows cols wiring color <<< "${INSTANCES[$i]}"
    echo -e "  ${BLUE}$name${NC}: http://localhost:$http_port (PID: ${PIDS[$i]})"
done

echo
print_status "API endpoints (run these to get the state of the LED matrix):"
for i in "${!INSTANCES[@]}"; do
    IFS=' ' read -r name http_port ddp_port rows cols wiring color <<< "${INSTANCES[$i]}"
    echo "  curl http://localhost:$http_port/json/state"
done

echo
print_status "DDP test commands (run these to test the DDP protocol):"
for i in "${!INSTANCES[@]}"; do
    IFS=' ' read -r name http_port ddp_port rows cols wiring color <<< "${INSTANCES[$i]}"
    total_leds=$((rows * cols))
    echo "  python3 scripts/ddp_test.py --port $ddp_port --leds $total_leds --pattern chase"
done

# Wait for all processes to complete (or until interrupted)
wait 