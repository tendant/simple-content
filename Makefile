SOURCES := $(shell find . -mindepth 2 -name "main.go")
DESTS := $(patsubst ./%/main.go,dist/%,$(SOURCES))
ALL := $(DESTS)

GOARCH ?= amd64
GOOS ?= linux

all: $(ALL)
	@echo $@: Building Targets $^

dist/%: %/main.go
	@echo $@: Building $^ to $@
	GOARCH=$(GOARCH) GOOS=$(GOOS) go build -buildvcs -o $@ $^

run: dist/cmd/server
	@echo "Running dist/cmd/server..."
	./dist/cmd/server

dep:
	go mod tidy		

docker-build:
	docker build -t simple-content .

clean:
	go clean
	rm -f $(ALL)

.PHONY: migrate-postgres-up migrate-postgres-down migrate-postgres-status

# Database migrations using goose (Postgres)
migrate-postgres-up:
	goose -dir ./migrations/postgres postgres "$(DATABASE_URL)" up

migrate-postgres-down:
	goose -dir ./migrations/postgres postgres "$(DATABASE_URL)" down

migrate-postgres-status:
	goose -dir ./migrations/postgres postgres "$(DATABASE_URL)" status

.PHONY: clean
