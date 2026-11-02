# Standard development checks.

test *dateien:
    #!/usr/bin/env bash
    pakete=$(for f in {{dateien}}; do echo "./$(dirname "$f")"; done | sort -u)
    out=$(SOPSY_REQUIRE_SOPS=1 go test ${pakete:-./...} 2>&1) || { printf '%s\n' "$out"; exit 1; }; printf '%s\n' "$out" | tail -n 4


lint *dateien:
    #!/usr/bin/env bash
    pakete=$(for f in {{dateien}}; do echo "./$(dirname "$f")"; done | sort -u)
    out=$(golangci-lint run ${pakete:-./...} 2>&1) || { printf '%s\n' "$out"; exit 1; }
    echo "lint ok"


cov:
    #!/usr/bin/env bash
    mkdir -p coverage
    out=$(SOPSY_REQUIRE_SOPS=1 go test -coverprofile=coverage/cover.out ./... 2>&1) || { printf '%s\n' "$out"; exit 1; }; printf '%s\n' "$out" | tail -n 4
    go run github.com/jandelgado/gcov2lcov@latest -infile coverage/cover.out -outfile coverage/lcov.info
