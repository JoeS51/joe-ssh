# Joe SSH Portfolio

A terminal portfolio you can SSH into. Built with Go, Wish, and Bubble Tea.

## Quick start (local)

1. Generate a host key (one time):

```bash
make keys
```

2. Run the server:

```bash
make run
```

3. Connect from another terminal:

```bash
ssh -p 2222 localhost
```

## Make targets

- `make run`: run locally with `go run .`
- `make build`: build a local binary at `bin/joe-ssh`
- `make build-linux`: build a Linux binary at `bin/joe-ssh-linux`
- `make keys`: generate `.ssh/host_ed25519`
- `make fmt`: run `go fmt ./...`
- `make clean`: remove the `bin/` directory

## Configuration (current defaults)

Right now the server listens on:

- Address: `0.0.0.0:2222`
- Host key path: `.ssh/host_ed25519`

These are currently hardcoded in `main.go`.

## Notes

- This repo intentionally ignores `.ssh/` because it contains private keys.
- Built binaries are ignored; use `bin/` for artifacts.
