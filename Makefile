.PHONY: data-get build-api run-api web-dev mobile-build clean

export AVD_NAME := Expo_Device

data-get:
	go run ./backend/cmd/fetchRawData.go

clean:
	rm -rf data/raw

run-api:
	go run ./backend/main.go

start-android:
	QT_QPA_PLATFORM=xcb nohup emulator -avd Expo_Device -gpu host > /tmp/emulator.log 2>&1 &
	adb wait-for-device

run-android:
	cd android && ./gradlew assembleDebug --build-cache
	adb install -r android/app/build/outputs/apk/debug/app-debug.apk
	adb shell am start -n com.octara.maps/.MainActivity -S

android-compileSources:
	cd android && ./gradlew compileDebugSources
	
data-build:
	go run ./backend/cmd/builder/main.go --input ./data/raw --output ./data/processed/network.bin

deploy:
	go build -o maps ./backend/main.go
	orion deploy