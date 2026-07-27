# aritu against this repository's own rules.
#
# FILES is a file or glob, or several separated by spaces. Quote them so aritu
# does the expanding rather than the shell: its ** crosses directories and sh's
# does not.
#
#   make apply
#   make apply FILES='internal/domain/config/*_test.go'
#   make apply FILES='cmd/**/*_test.go internal/**/*_test.go'

BINARY   ?= ./aritu
FILES    ?=
RULEBOOK ?= RULEBOOK.md

.PHONY: build apply rulebook clean

build:
	go build -o $(BINARY) ./cmd/aritu

# apply judges the named files, or everything the enabled rules target when
# FILES is empty.
apply: build
	$(BINARY) apply $(FILES)

# rulebook writes the same rules as prose, for whoever is about to write a file
# rather than for whoever already did. It calls no model.
rulebook: build
	$(BINARY) rulebook > $(RULEBOOK)
	@echo "wrote $(RULEBOOK)"

clean:
	rm -f $(BINARY)
