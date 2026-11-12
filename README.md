# eLabFTW Lite

> ⚠️ [WARNING]
>
> This project is an **early experimental preview**.
>
> It is public for development visibility only and is **not ready for use**.
> Do **not** use it for real work, testing workflows, or production data.
>
> The recommended and supported way to run eLabFTW is the Docker-based web application.
> See the [official documentation](https://doc.elabftw.net/).

## Development Setup

---

## Prerequisites

### Go

**Minimum version:** Go **1.25+**

Check:

```bash
go version
```

---

### Node.js

- **Minimum version:** Node.js **20.19+**

```bash
node -v
npm -v
```

---

### Wails CLI

Install the Wails v3 CLI:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
```

Ensure it's in your PATH:

```bash
export PATH="$PATH:$HOME/go/bin"
```

Verify:

```bash
wails3 doctor
```

---

## Linux dependencies

Wails v3 requires system libraries (GTK 4 + WebKitGTK 6.0).

### Fedora (≥ 40)

```bash
sudo dnf install gtk4-devel webkitgtk6.0-devel pkgconf-pkg-config gcc gcc-c++
```

### Ubuntu (≥ 24.04) / Debian (≥ 13)

```bash
sudo apt install libgtk-4-dev libwebkitgtk-6.0-dev pkg-config
```

### Archlinux

```bash
sudo pacman -Syu gtk4 webkitgtk-6.0
```

---

## Setup

```bash
git clone https://github.com/elabftw/lite.git
cd lite
```

Frontend dependencies are installed automatically by `wails3` tasks.

---

## Run the application

```bash
wails3 dev
```

A **native desktop window** should open automatically. The app is **not meant to run in a browser**.

---

## Build

Outputs are written to `bin/`.

```bash
# desktop binary for the current OS
wails3 build
# macOS .app bundle / Windows installer / Linux packages
wails3 package
```

### Android

Requires a JDK (17+), the Android SDK and NDK (`ANDROID_HOME`, optionally `ANDROID_NDK_HOME`).
Run `wails3 task android:install:deps` to check them.

```bash
# release APK in bin/elabftw-lite.apk (debug-signed unless ANDROID_KEYSTORE_FILE is set)
wails3 task android:package
# run on a connected device or emulator
wails3 task android:run
```

### iOS

Requires macOS with Xcode. Run `wails3 task ios:install:deps` to check them.

```bash
# run in the iOS simulator
wails3 task ios:run
# signed device build
wails3 task ios:package:ipa IOS_PLATFORM=device CODESIGN_IDENTITY="Apple Development: ..."
```

On mobile, data is stored in the app sandbox. Saving uploads to a user chosen location is not supported there yet.

---

## Bindings

The frontend talks to Go through generated bindings in `frontend/bindings`.
Regenerate them after changing exported `App` methods or their types:

```bash
wails3 generate bindings -clean=true -ts -i
```
