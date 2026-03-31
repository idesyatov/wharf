# Example Compose Stacks

Lightweight compose stacks for testing Wharf. All images are small (alpine/busybox, 5-30 MB).

## Quick Start

```bash
# Start all examples
cd examples/simple-web && docker compose up -d && cd ../..
cd examples/multi-service && docker compose up -d && cd ../..
cd examples/with-volumes && docker compose up -d && cd ../..

# Then run wharf
./bin/wharf-linux-amd64

# Stop all examples
cd examples/simple-web && docker compose down && cd ../..
cd examples/multi-service && docker compose down && cd ../..
cd examples/with-volumes && docker compose down && cd ../..
```

## Stacks

### simple-web
Single nginx container on port 8080. Minimal test case.

### multi-service
4 services: app (busybox loop), worker (busybox loop), redis, api (nginx).
Good for testing start/stop/restart, logs, filtering.

### with-volumes
2 redis instances with named volumes and a custom network.
Good for testing volumes view, networks view, prune.
