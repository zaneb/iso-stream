.PHONY: build

build:
	go build -o build/iso-stream main.go

clean:
	rm -rf build/*

run:
	go run main.go

test:
	go test ./...
