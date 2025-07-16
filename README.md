# WLED Simulator

A minimal but extensible desktop application that behaves like a real WLED node while running entirely on your workstation. It offers a Fyne-powered LED matrix, a WLED-compatible JSON REST API, and a full-speed DDP UDP listener, making it ideal for client development and automated integration tests.

## Features

* Configurable LED matrix display in a Fyne GUI.
* Full WLED JSON API (`/json`, `/json/state`, `/json/info`) with `live` field support.
* DDP UDP listener on port 4048 for real-time LED streaming.
* Thread-safe shared LED state with power and brightness control.
* Command-line flags and optional `config.yaml` for easy configuration.
* Indicators for JSON and DDP activity, green for success and red for error.

## Screenshot

![WLED Simulator Screenshot](/docs/wled-sim-demo.png)

5x4 Matrix displays a rainbow during a DDP request.

## Quick Start

```bash
git clone https://github.com/13rac1/wled-simulator
cd wled-simulator
go mod tidy
go run ./cmd -rows 10 -cols 3 -http :9090 -init "#00FF00"
```

Open `http://localhost:9090/json` in your browser or point your WLED mobile app at the same address to test live control.

## Configuration Flags

| Flag        | Default | Description                          |
|-------------|---------|--------------------------------------|
| `-rows`     | 10      | Number of LED rows                   |
| `-cols`     | 2       | Number of LED columns                |
| `-wiring`   | row     | LED wiring pattern: 'row' or 'col'   |
| `-http`     | :8080   | HTTP listen address                  |
| `-ddp-port` | 4048    | UDP port for DDP                     |
| `-init`     | #000000 | Initial LED colour (hex)           |
| `-controls` | false   | Show power/brightness controls in UI |
| `-headless` | false   | Disable GUI for CI (API/DDP only)    |
| `-v`        | false   | Verbose logging                      |

You can also create a `config.yaml` file with the same keys to persist defaults.

```yaml
rows: 10
cols: 3
wiring: "col"
http_address: ":9090"
ddp_port: 4048
init_color: "#202020"
```

### LED Wiring Patterns

The simulator supports two common LED matrix wiring patterns:

- **Row-major (`-wiring row`)**: LEDs are wired left-to-right, then top-to-bottom
  ```
  0 → 1 → 2
  3 → 4 → 5
  ```

- **Column-major (`-wiring col`)**: LEDs are wired top-to-bottom, then left-to-right
  ```
  0   2   4
  ↓   ↓   ↓
  1   3   5
  ```

## Testing

Run all unit tests:

```bash
go test ./...
```

### Manual Testing with curl

Test the HTTP API endpoints with these curl commands (assumes simulator running on localhost:8080):

**Set all LEDs to blue:**
```bash
curl -X POST http://localhost:8080/json/state -H "Content-Type: application/json" -d '{"on":true,"bri":255,"seg":[{"col":[[0,0,255]]}]}'
```

**Set all LEDs to red:**
```bash
curl -X POST http://localhost:8080/json/state -H "Content-Type: application/json" -d '{"seg":[{"col":[[255,0,0]]}]}'
```

**Set all LEDs to green:**
```bash
curl -X POST http://localhost:8080/json/state -H "Content-Type: application/json" -d '{"seg":[{"col":[[0,255,0]]}]}'
```

**Set all LEDs to white:**
```bash
curl -X POST http://localhost:8080/json/state -H "Content-Type: application/json" -d '{"seg":[{"col":[[255,255,255]]}]}'
```

**Get current state:**
```bash
curl http://localhost:8080/json/state
```

**Get device info:**
```bash
curl http://localhost:8080/json/info
```

The API responses include a `live` field that indicates when DDP data is actively being received (matches real WLED behavior).

### Manual Testing with DDP

Test the DDP protocol using the included Python script (assumes simulator running on localhost:4048):

**Set all LEDs to blue:**
```bash
python3 scripts/ddp_test.py --color blue
```

**Set all LEDs to red:**
```bash
python3 scripts/ddp_test.py --color red
```

**Display rainbow pattern:**
```bash
python3 scripts/ddp_test.py --pattern rainbow
```

**Display gradient pattern:**
```bash
python3 scripts/ddp_test.py --pattern gradient
```

**Run chase pattern (great for testing wiring):**
```bash
python3 scripts/ddp_test.py --pattern chase --delay 0.5
```

**Cycle through colors:**
```bash
python3 scripts/ddp_test.py --pattern cycle --delay 0.5
```

**Test with custom LED count:**
```bash
python3 scripts/ddp_test.py --color green --leds 30
```

**Test with remote host:**
```bash
python3 scripts/ddp_test.py --color white --host 192.168.1.100
```

## License

AGPL