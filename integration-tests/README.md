# Japella integration tests

Browser integration tests for the Japella web UI, using Mocha and Selenium WebDriver.

## Prerequisites

- Node.js (npm)
- Docker (MariaDB for the test database)
- A built Japella binary at `../service/japella`
- A built frontend at `../frontend/dist`

From the repository root:

```bash
make frontend
make service
cd integration-tests && make
```

## Running tests

All tests:

```bash
cd integration-tests && make
```

Single suite:

```bash
cd integration-tests && npx mocha tests/general/general.mjs
```

## How it works

- `compose.yml` starts MariaDB on port `3307`.
- `runner.mjs` starts Japella as a local process on port `18080` with per-suite config from `tests/<name>/config.yaml`.
- `mochaSetup.mjs` launches headless Chrome and tears down Docker when the run finishes.

## Environment variables

| Variable | Description |
| --- | --- |
| `JAPELLA_TEST_RUNNER` | `local` (default) starts a local process; `container` waits for an existing endpoint via `IP` and `PORT`. |
| `JAPELLA_TEST_RUNNER_LOG_STDOUT` | Set to `1` to log Japella stdout/stderr during tests. |

## CI

GitHub Actions runs `make frontend`, `make service`, then `cd integration-tests && make` on pull requests and pushes to `main`.
