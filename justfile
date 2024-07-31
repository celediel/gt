binary := "gt"
version := "0.0.2"
build_dir := "bin"
dist_dir := "dist"
cmd := "."
output := "." / build_dir / binary
dist := "." / dist_dir / binary
contrib_dir := "." / "contrib"
screenshots_dir := "screenshots"
archive_base := dist_dir / binary + "-" + version
linux_archive := archive_base + "-x86_64.tar"
arm64_archive := archive_base + "-arm64.tar"
arm_archive := archive_base + "-arm.tar"

# do the thing
default: test check install

# build binary
build:
	go build -o {{output}} {{cmd}}

package $CGO_ENABLED="0": man
	go build -o {{binary}} {{cmd}}
	tar cafv {{linux_archive}} {{binary}} {{binary}}.1 README.md LICENSE {{screenshots_dir}}
	tar rafv {{linux_archive}} -C {{contrib_dir}} "completions"
	gzip -f {{linux_archive}}
	rm {{binary}}

	GOARCH="arm" GOARM="7" go build -o {{binary}} {{cmd}}
	tar cafv {{arm_archive}} {{binary}} {{binary}}.1 README.md LICENSE {{screenshots_dir}}
	tar rafv {{arm_archive}} -C {{contrib_dir}} "completions"
	gzip -f {{arm_archive}}
	rm {{binary}}

	GOARCH="arm64" go build -o {{binary}} {{cmd}}
	tar cafv {{arm64_archive}} {{binary}} {{binary}}.1 README.md LICENSE {{screenshots_dir}}
	tar rafv {{arm64_archive}} -C {{contrib_dir}} "completions"
	gzip -f {{arm64_archive}}
	rm {{binary}}

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
