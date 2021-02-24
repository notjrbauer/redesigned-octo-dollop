	GOBIN=$(shell pwd)/bin
	GOFILES=$(wildcard ./cmd/*.go)
	GONAME=$(shell basename "$(PWD)")
	PID=/tmp/go-$(GONAME).pid

build:
	@echo "Building $(GOFILES) to ./bin"
	@GOBIN=$(GOBIN) go build -o bin/$(GONAME) $(GOFILES)

get:
	@GOBIN=$(GOBIN) go get .

install:
	@GOBIN=$(GOBIN) go install $(GOFILES)

run:
	@GOBIN=$(GOBIN) go run $(GOFILES)

watch:
	@$(MAKE) restart &
	@fswatch -o . -e 'bin/.*' | xargs -n1 -I{}  make restart

restart: clear stop clean build start

start:
	@echo "Starting bin/$(GONAME)"
	@./bin/$(GONAME) & echo $$! > $(PID)

stop:
	@echo "Stopping bin/$(GONAME) if it's running"
	@-kill `[[ -f $(PID) ]] && cat $(PID)` 2>/dev/null || true

clear:
	@clear

clean:
	@echo "Cleaning"
	@GOBIN=$(GOBIN) go clean

.PHONY: build get install run watch start stop restart clean
