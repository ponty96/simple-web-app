.PHONY: build run test clean

build:
	go build -ldflags "-s -w" -o simple-web-app cmd/simple-web-app/main.go

run: build
	./simple-web-app

test:
	go test -v ./...

clean:
	rm -rf simple-web-app
