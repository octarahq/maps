.PHONY: data-get build-api run-api web-dev mobile-build clean

data-get:
	go run ./backend/cmd/fetchRawData.go

clean:
	rm -rf data/raw

build-api:
	go build -o bin/server ./backend/cmd/server
	go build -o bin/builder ./backend/cmd/builder

run-api:
	go run ./backend/cmd/server/main.go

data-build:
	go run ./backend/cmd/builder/main.go --input ./data/raw --output ./data/processed/network.bin

web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

mobile-build:
	cd mobile && ./gradlew assembleDebug