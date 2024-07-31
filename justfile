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

# install manpages into $HOME/.local/share/man
install-man:
	mkdir -p $HOME/.local/share/man/man1
	cp {{binary}}.1 $HOME/.local/share/man/man1/{{binary}}.1

# clean up after yourself
clean:
	rm {{output}}

# run go tests
test:
	gotestsum

# run linter
check:
	staticcheck ./...

# generate manpage
man:
	scdoc < {{binary}}.1.scd > {{binary}}.1
