# WLED Simulator

A minimal but extensible desktop application that behaves like a real WLED node while running entirely on your workstation. It offers a Fyne-powered LED matrix, a WLED-compatible JSON REST API, and a full-speed DDP UDP listener, making it ideal for client development and automated integration tests.

## Features

* Two vertical columns of configurable LEDs displayed in a Fyne GUI.
* Full WLED JSON API (`/json`, `/json/state`, `/json/info`).
* DDP UDP listener on port 4048 for real-time LED streaming.
* Thread-safe shared LED state with power and brightness control.
* Command-line flags and optional `config.yaml` for easy configuration.
* Modular internal packages (`api`, `ddp`, `gui`, `state`).
* Basic unit tests.

## Quick Start

```bash
git clone https://github.com/13rac1/wled-simulator
cd wled-simulator
go mod tidy
go run ./cmd -leds 30 -http :9090 -init "#202020"
```

Open `http://localhost:9090/json` in your browser or point your WLED mobile app at the same address to test live control.

## Configuration Flags

| Flag        | Default | Description                          |
|-------------|---------|--------------------------------------|
| `-leds`     | 20      | LEDs per column                      |
| `-http`     | :8080   | HTTP listen address                  |
| `-ddp-port` | 4048    | UDP port for DDP                     |
| `-init`     | #000000 | Initial LED colour (hex)             |
| `-controls` | false   | Show power/brightness controls in UI |
| `-headless` | false   | Disable GUI (API/DDP only)           |
| `-v`        | false   | Verbose logging                      |

You can also create a `config.yaml` file with the same keys to persist defaults.

```yaml
leds: 30
http_address: ":9090"
ddp_port: 4048
init_color: "#202020"
```

## Testing

Run all unit tests:

```bash
go test ./...
```

## Extending

* **Add effects** – implement a ticker that mutates the `LEDState` slice, then expose the effect list in `/json`.
* **Additional protocols** – mirror the `ddp` package structure for protocols like E1.31 or MQTT.
* **Headless CI** – run with `-headless` to integrate in continuous integration pipelines.
