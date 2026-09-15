gobin := `go env GOBIN`
bindir := if gobin == "" { `go env GOPATH` / "bin" } else { gobin }

build:
    go build -o think ./cmd/think
    ln -sf think tnk

install: build
    install -m 0755 think {{bindir}}/think
    ln -sf {{bindir}}/think {{bindir}}/tnk

clean:
    rm -f think tnk
