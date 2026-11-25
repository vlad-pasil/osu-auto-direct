# osu! Auto Direct

[Documentación en español](README.md)

A lightweight, unofficial native Windows utility designed to simplify beatmap downloads, particularly when a map is missing in an osu! multiplayer lobby. It receives beatmap URLs through Windows' default HTTP/HTTPS handler, downloads an `.osz` archive from a third-party mirror, performs basic archive checks, and asks Windows to open it using the registered `.osz` association. Other web links are passed to the selected browser.

**Experimental beta:** the source has been built and unit-tested; Windows integration, live mirror availability and osu! import have not been verified end to end. The executable is unsigned.

## Getting started

1. Download the Windows x64 ZIP from the repository's Releases section and extract it.
2. Run `osu-Auto-Direct.exe` and confirm your fallback browser.
3. In Windows Settings > Apps > Default apps, select osu! Auto Direct for HTTP and HTTPS only. Leave `.html` and `.htm` unchanged.
4. Ensure `.osz` files are associated with your osu! client.
5. Try opening a missing beatmap from osu!. Also test a normal external URL to confirm browser forwarding.

## Important limitations

The utility receives all HTTP/HTTPS links dispatched to it by Windows, not just links originating in osu!. It cannot identify the originating process and does not intercept navigation within a browser. It starts on demand rather than running as a persistent service. It uses [Hinai](https://mirror.hinamizawa.ai/docs/beatmap-download) and [Catboy](https://catboy.best/) for downloads; beatmap IDs and normal connection data are shared with those providers, but no osu! credentials are requested. Catboy is a download fallback only. Files may remain in `%LOCALAPPDATA%\OsuAutoDirect\Downloads`. Downloads are limited to 512 MiB, and structural archive validation is not a malware scan. Respect service terms and only download content you are authorized to use.

## Build

Install a supported version of Go (minimum 1.23). Run `go test ./...` on Windows, then `COMPILAR.bat`. Alternatively:

```powershell
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -trimpath -ldflags="-H=windowsgui -s -w" -o osu-Auto-Direct.exe .
```

## Uninstall

Restore your usual browser as the default HTTP/HTTPS handler, run `DESINSTALAR.bat`, and optionally remove `%LOCALAPPDATA%\OsuAutoDirect`.

This is an independent community project, not affiliated with or endorsed by osu!, ppy or the mirror providers. It is not the official osu!direct service. No beatmaps or music are distributed with this repository. See the proposed [MIT license](LICENSE) and [contribution guidelines](CONTRIBUTING.md).
