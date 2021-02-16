.PHONY: build

build:
	go build -o build/iso-stream cmd/main.go

clean:
	rm -rf build/*

run: clean build
	./build/iso-edit
