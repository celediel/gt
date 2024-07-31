binary := "gt"
version := "0.0.1"
build_dir := "bin"
dist_dir := "dist"
cmd := "."
output := "." / build_dir / binary
dist := "." / dist_dir / binary

# do the thing
default: test check install

# build binary
build:
	go build -o {{output}} {{cmd}}

package:
	go build -o {{binary}}
	distrobox enter alpine -- go build -o {{binary}}-musl {{cmd}}
	tar cafv {{dist_dir}}/{{binary}}-{{version}}-x86_64.tar.gz {{binary}} README.md LICENSE
	tar cafv {{dist_dir}}/{{binary}}-{{version}}-x86_64-musl.tar.gz {{binary}}-musl README.md LICENSE
	rm {{binary}} {{binary}}-musl

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
	-rm {{output}}
	-rm {{output}}-musl
	-rm {{dist_dir}}/*.tar.gz

# run go tests
test:
	gotestsum

# run linter
check:
	staticcheck ./...

# generate manpage
man:
	scdoc < {{binary}}.1.scd > {{binary}}.1
