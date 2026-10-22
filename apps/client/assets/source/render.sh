#!/bin/sh
# Renders the app icons from the SVG sources. Needs rsvg-convert (librsvg).
# Run from apps/client: sh assets/source/render.sh
set -eu
src=assets/source
out=public/icons
mkdir -p "$out"

render() { rsvg-convert -w "$2" -h "$2" "$src/$1.svg" -o "$3"; }

# Web app (PWA) and browser tab
render icon-rounded 192 "$out/icon-192.png"
render icon-rounded 512 "$out/icon-512.png"
render icon 512 "$out/icon-maskable-512.png"
render icon 180 "$out/apple-touch-icon.png"
render icon-rounded 48 assets/favicon.png

# iOS and Android (app.json)
render icon 1024 assets/icon.png
render icon-rounded 1024 assets/splash-icon.png
render android-foreground 512 assets/android-icon-foreground.png
render android-monochrome 512 assets/android-icon-monochrome.png
render android-background 512 assets/android-icon-background.png
