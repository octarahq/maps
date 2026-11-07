.PHONY: data-get build-api run-api web-dev mobile-build clean build-android run-android release

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

build-android:
	cd android && ./gradlew assembleRelease
	@echo "==> APK Release généré avec succès dans android/app/build/outputs/apk/release/app-release.apk"

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

.PHONY: data-get build-api run-api web-dev mobile-build clean build-android run-android release android api all

release:
	@TARGET="$(filter-out release,$(MAKECMDGOALS))"; \
	TARGET_NAME=$$(echo $$TARGET | awk '{print $$1}'); \
	VERSION=$$(echo $$TARGET | awk '{print $$2}'); \
	if [ -z "$$TARGET_NAME" ] || [ -z "$$VERSION" ]; then \
		echo "Erreur d'utilisation : make release <android|api|all> <x.y.z>"; \
		echo "Exemple : make release android 1.0.0"; \
		exit 1; \
	fi; \
	case "$$TARGET_NAME" in \
		android|api|all) ;; \
		*) echo "Cible invalide '$$TARGET_NAME'. Choisissez parmi: android, api, all"; exit 1 ;; \
	esac; \
	TAG="$$TARGET_NAME-v$$VERSION"; \
	echo "Création et envoi du tag $$TAG vers GitHub..."; \
	git tag "$$TAG" && git push origin "$$TAG" && \
	echo "Tag $$TAG poussé ! GitHub Actions va construire et publier la release automatiquement."

android api all:
	@:

%:
	@: