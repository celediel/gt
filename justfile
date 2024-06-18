binary := "gt"
build_dir := "bin"
cmd := "."
output := "." / build_dir / binary

# do the thing
default: test check install

# build binary
build:
	go build -o {{output}} {{cmd}}

# build windows binary
build-windows:
	GOOS=windows GOARCH=amd64 go build -o {{output}}.exe {{cmd}}

# run from source
run:
	go run {{cmd}}

# build 'n run
run-binary: build
	exec {{output}}

# run with args
run-args args:
	go run {{cmd}} {{args}}

# install binary into $GOPATH
install:
	go install {{cmd}}

# clean up after yourself
clean:
	rm {{output}}

# run go tests
test:
	gotestsum

# run linter
check:
	staticcheck ./...
