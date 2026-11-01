.PHONY: data-get build-api run-api web-dev mobile-build clean

data-get:
	go run ./backend/cmd/fetchRawData.go

clean:
	rm -rf data/raw

run-api:
	go run ./backend/main.go

data-build:
	go run ./backend/cmd/builder/main.go --input ./data/raw --output ./data/processed/network.bin

deploy:
	go build -o maps ./backend/main.go
	orion deploy