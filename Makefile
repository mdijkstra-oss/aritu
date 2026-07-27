# aritu against this repository's own rules.
#
# FILES is a file or glob, or several separated by spaces. Quote them so aritu
# does the expanding rather than the shell: its ** crosses directories and sh's
# does not.
#
#   make apply
#   make apply FILES='internal/domain/config/*_test.go'
#   make apply FILES='cmd/**/*_test.go internal/**/*_test.go'

BINARY ?= ./aritu
FILES  ?=

.PHONY: build apply rulebook clean

build:
	go build -o $(BINARY) ./cmd/aritu

# apply judges the named files, or everything the enabled rules target when
# FILES is empty.
apply: build
	$(BINARY) apply $(FILES)

# rulebook prints the same rules as prose, for whoever is about to write a file
# rather than for whoever already did. It goes to stdout and nowhere else: the
# rules directory is the one copy, and a file written beside it would be a
# second one to keep in step. It calls no model.
rulebook: build
	@$(BINARY) rulebook

clean:
	rm -f $(BINARY)
