[![Build](https://github.com/lost-woods/random/actions/workflows/publish.yml/badge.svg)](https://github.com/lost-woods/random/actions/workflows/publish.yml)
# Random (TrueRNG-backed) HTTP API

This project exposes a small HTTP API that returns:
- uniform random integers (`/`)
- raw random bytes (`/bytes`)
- random playing cards without replacement (`/cards`)
- random strings from configurable character sets (`/strings`)
- exact probability rolls (`/percent`)

It is designed to read entropy from a TrueRNG device over a serial port.

## Output format (plain text vs JSON)

Responses are **plain text by default**.

If you want **JSON**, send an `Accept: application/json` header.

All **successful** responses include a **UUID v4** called `request_id` so you can correlate outcomes:
- JSON: `"request_id": "<uuid>"`
- Text: a final line `request_id: <uuid>`

## Endpoints

### `GET /`
Uniform integer in `[min, max]` (inclusive).

Query params:
- `min` (default `1`)
- `max` (default `100`)

Examples:
```bash
curl "http://localhost:777/?min=1&max=49"
curl -H "Accept: application/json" "http://localhost:777/?min=1&max=49"
```

### `GET /bytes`
Hex-encoded random bytes.

Query params:
- `size` (default `1`, max `256`)

```bash
curl "http://localhost:777/bytes?size=32"
```

### `GET /cards`
Draw cards *without replacement* from one or more decks.

Query params:
- `decks` (default `1`, max `100`)
- `jokers` (default `false`)
- `cards` (default `1`)

```bash
curl "http://localhost:777/cards?decks=1&jokers=false&cards=5"
curl -H "Accept: application/json" "http://localhost:777/cards?cards=5"
```

### `GET /strings`
Generate a random string from selected character sets.

Query params:
- `size` (default `10`, max `256`)
- `lowercase` (default `true`)
- `uppercase` (default `true`)
- `numbers` (default `true`)
- `symbols` (default `true`)

```bash
curl "http://localhost:777/strings?size=32&symbols=false"
```

### `GET /percent`
Rolls a probability exactly.

Query params:
- `percent` (default `25`)

Rules:
- accepts `0` and `100` (deterministic fail/pass)
- accepts up to **7 decimal places**
  (values with **8 or more decimal places**, e.g. `12.34567891`, are rejected)

```bash
curl "http://localhost:777/percent?percent=12.5"
curl "http://localhost:777/percent?percent=0"
curl "http://localhost:777/percent?percent=100"
```

### `GET /health`
Returns `200 OK` if the RNG is healthy, otherwise `503`.

## Configuration

The server expects these environment variables:

- `API_KEY` – required. Requests must include `X-API-KEY: <API_KEY>`.
- `SERIAL_DEVICE_NAME` – e.g. `/dev/TrueRNG`
- `SERIAL_BAUD_RATE` – TrueRNG baud rate (depends on device/OS)
- `SERIAL_READ_TIMEOUT` – read timeout (milliseconds)
- `RNG_HEALTH_INTERVAL_MS` – interval in milliseconds between background RNG health checks (default: `10000`).

## Running

```bash
export API_KEY="your-secret"
export SERIAL_DEVICE_NAME="/dev/TrueRNG"
export SERIAL_BAUD_RATE="300"
export SERIAL_READ_TIMEOUT="1000"

go run .
```

## Tests

All tests live under the `test/` folder to keep production packages clean.

Run:
```bash
go test ./...
```

The fairness/uniformity tests use deterministic pseudo-RNG readers so they are stable in CI.
