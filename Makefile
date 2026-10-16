.PHONY: data-get build-api run-api web-dev mobile-build clean

data-get:
	go run ./api/cmd/fetchRawData.go

build-api:
	go build -o bin/server ./api/cmd/server
	go build -o bin/builder ./api/cmd/builder

run-api:
	go run ./api/cmd/server/main.go

data-build:
	go run ./api/cmd/builder/main.go --input ./data/raw --output ./data/processed/network.bin

web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

mobile-build:
	cd mobile && ./gradlew assembleDebug