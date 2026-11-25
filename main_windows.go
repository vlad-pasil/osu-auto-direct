package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const appName = "osu! Auto Direct"
const exeName = "osu-Auto-Direct.exe"
const capKey = `Software\Clients\StartMenuInternet\OsuAutoDirect`
const progID = "OsuAutoDirect.URL"

var (
	user32     = syscall.NewLazyDLL("user32.dll")
	shell32    = syscall.NewLazyDLL("shell32.dll")
	shlwapi    = syscall.NewLazyDLL("shlwapi.dll")
	comdlg     = syscall.NewLazyDLL("comdlg32.dll")
	msgBox     = user32.NewProc("MessageBoxW")
	shellExec  = shell32.NewProc("ShellExecuteW")
	assocQuery = shlwapi.NewProc("AssocQueryStringW")
	openDialog = comdlg.NewProc("GetOpenFileNameW")
)

func wide(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func message(s string, flags uintptr) uintptr {
	r, _, _ := msgBox.Call(0, uintptr(unsafe.Pointer(wide(s))), uintptr(unsafe.Pointer(wide(appName))), flags)
	return r
}
func shellOpen(s string) error {
	r, _, _ := shellExec.Call(0, uintptr(unsafe.Pointer(wide("open"))), uintptr(unsafe.Pointer(wide(s))), 0, 0, 1)
	if r <= 32 {
		return fmt.Errorf("Windows ShellExecute: %d", r)
	}
	return nil
}
func assoc(protocol string) string {
	b := make([]uint16, 32768)
	n := uint32(len(b))
	r, _, _ := assocQuery.Call(0, 2, uintptr(unsafe.Pointer(wide(protocol))), uintptr(unsafe.Pointer(wide("open"))), uintptr(unsafe.Pointer(&b[0])), uintptr(unsafe.Pointer(&n)))
	if r != 0 {
		return ""
	}
	return syscall.UTF16ToString(b)
}
func appDir() string        { return filepath.Join(os.Getenv("LOCALAPPDATA"), "OsuAutoDirect") }
func installed() string     { return filepath.Join(appDir(), exeName) }
func same(a, b string) bool { return strings.EqualFold(filepath.Clean(a), filepath.Clean(b)) }
func validBrowser(p string) bool {
	if p == "" || !strings.EqualFold(filepath.Ext(p), ".exe") || same(p, installed()) || strings.EqualFold(filepath.Base(p), exeName) {
		return false
	}
	s, e := os.Stat(p)
	return e == nil && !s.IsDir()
}

type config struct {
	Browser string `json:"browser"`
}

func load() config {
	var c config
	b, _ := os.ReadFile(filepath.Join(appDir(), "config.json"))
	json.Unmarshal(b, &c)
	return c
}
func save(c config) error {
	b, e := json.MarshalIndent(c, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(filepath.Join(appDir(), "config.json"), b, 0600)
}

// Native Windows file picker. The structure uses Windows pointer alignment on x64.
type ofn struct {
	Size                                         uint32
	Owner, Instance                              uintptr
	Filter, CustomFilter                         *uint16
	MaxCustom, FilterIndex                       uint32
	File                                         *uint16
	MaxFile                                      uint32
	FileTitle                                    *uint16
	MaxFileTitle                                 uint32
	InitialDir, Title                            *uint16
	Flags                                        uint32
	FileOffset, FileExtension                    uint16
	DefExt, CustomData, Hook, Template, Reserved uintptr
	Reserved2, FlagsEx                           uint32
}

func chooseBrowser() string {
	b := make([]uint16, 32768)
	filter := []uint16{'E', 'j', 'e', 'c', 'u', 't', 'a', 'b', 'l', 'e', 's', 0, '*', '.', 'e', 'x', 'e', 0, 0}
	t := ofn{File: &b[0], MaxFile: uint32(len(b)), Filter: &filter[0], FilterIndex: 1, Title: wide("Selecciona el ejecutable de tu navegador"), Flags: 0x1000 | 0x800 | 0x80000 | 0x8}
	t.Size = uint32(unsafe.Sizeof(t))
	r, _, _ := openDialog.Call(uintptr(unsafe.Pointer(&t)))
	if r == 0 {
		return ""
	}
	p := syscall.UTF16ToString(b)
	if validBrowser(p) {
		return p
	}
	return ""
}
func hiddenCommand(name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return c
}
func regAdd(key, name, value string) error {
	a := []string{"add", key}
	if name == "" {
		a = append(a, "/ve")
	} else {
		a = append(a, "/v", name)
	}
	a = append(a, "/t", "REG_SZ", "/d", value, "/f")
	b, e := hiddenCommand(filepath.Join(os.Getenv("SystemRoot"), "System32", "reg.exe"), a...).CombinedOutput()
	if e != nil {
		return fmt.Errorf("registro: %v %s", e, b)
	}
	return nil
}
func register() error {
	base := `HKCU\` + capKey
	cls := `HKCU\Software\Classes\` + progID
	cmd := `"` + installed() + `" "%1"`
	entries := [][3]string{
		{cls, "", "osu! Auto Direct URL"}, {cls, "URL Protocol", ""}, {cls + `\DefaultIcon`, "", installed() + ",0"}, {cls + `\shell\open\command`, "", cmd},
		{base, "", appName}, {base + `\DefaultIcon`, "", installed() + ",0"}, {base + `\shell\open\command`, "", `"` + installed() + `"`},
		{base + `\Capabilities`, "ApplicationName", appName}, {base + `\Capabilities`, "ApplicationDescription", "Descarga beatmaps de osu! y reenvia otros enlaces al navegador elegido."}, {base + `\Capabilities`, "ApplicationIcon", installed() + ",0"},
		{base + `\Capabilities\URLAssociations`, "http", progID}, {base + `\Capabilities\URLAssociations`, "https", progID},
		{`HKCU\Software\RegisteredApplications`, appName, capKey + `\Capabilities`},
	}
	for _, v := range entries {
		if e := regAdd(v[0], v[1], v[2]); e != nil {
			return e
		}
	}
	shell32.NewProc("SHChangeNotify").Call(0x08000000, 0, 0, 0)
	return nil
}
func copySelf() error {
	p, e := os.Executable()
	if e != nil {
		return e
	}
	if same(p, installed()) {
		return nil
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return e
	}
	return os.WriteFile(installed(), b, 0700)
}
func settings() {
	if e := shellOpen("ms-settings:defaultapps"); e != nil {
		message("Abre Configuración > Aplicaciones > Aplicaciones predeterminadas.", 0x40)
	}
}
func setup(change bool) error {
	if os.Getenv("LOCALAPPDATA") == "" {
		return fmt.Errorf("LOCALAPPDATA no disponible")
	}
	if e := os.MkdirAll(appDir(), 0700); e != nil {
		return e
	}
	c := load()
	if change || !validBrowser(c.Browser) {
		p := assoc("https")
		if change || !validBrowser(p) || message("Navegador detectado:\n"+p+"\n\n¿Usarlo para todos los enlaces que no sean beatmaps?", 0x24) != 6 {
			p = chooseBrowser()
		}
		if !validBrowser(p) {
			return fmt.Errorf("No se seleccionó un navegador válido. No se cambiaron las asociaciones.")
		}
		c.Browser = p
	}
	if e := copySelf(); e != nil {
		return e
	}
	if e := save(c); e != nil {
		return e
	}
	if e := register(); e != nil {
		return e
	}
	message("Programa registrado.\n\nEn Aplicaciones predeterminadas busca osu! Auto Direct y asígnale HTTP y HTTPS (no .html).\n\nNavegador de respaldo:\n"+c.Browser+"\n\nWindows exige que confirmes este cambio manualmente.", 0x40)
	settings()
	return nil
}
func forward(raw string) error {
	c := load()
	if !validBrowser(c.Browser) {
		return fmt.Errorf("El navegador de respaldo no está disponible. Abre el programa y cambia el navegador.")
	}
	return exec.Command(c.Browser, raw).Start()
}
func uninstall() error {
	if same(assoc("http"), installed()) || same(assoc("https"), installed()) {
		message("Primero restaura tu navegador como predeterminado para HTTP y HTTPS en Windows. Luego vuelve a ejecutar DESINSTALAR.bat.", 0x30)
		settings()
		return nil
	}
	r := filepath.Join(os.Getenv("SystemRoot"), "System32", "reg.exe")
	for _, a := range [][]string{{"delete", `HKCU\Software\RegisteredApplications`, "/v", appName, "/f"}, {"delete", `HKCU\` + capKey, "/f"}, {"delete", `HKCU\Software\Classes\` + progID, "/f"}} {
		hiddenCommand(r, a...).Run()
	}
	shell32.NewProc("SHChangeNotify").Call(0x08000000, 0, 0, 0)
	message("Registro eliminado. Puedes borrar manualmente la carpeta:\n"+appDir()+"\n\nLos mapas importados en osu! no se eliminan.", 0x40)
	return nil
}
func runURL(raw string) error {
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return fmt.Errorf("Solo se admiten enlaces HTTP/HTTPS")
	}
	kind, id := parseLink(raw)
	if kind == "" {
		return forward(raw)
	}
	if kind == "beatmap" {
		id, e = resolve(id)
	}
	if e == nil {
		var p string
		p, e = download(id, filepath.Join(appDir(), "Downloads"))
		if e == nil {
			e = shellOpen(p)
			if e != nil {
				message("Mapa descargado, pero Windows no pudo abrirlo con osu!.\n\n"+p+"\n\nAsocia la extensión .osz con osu!.", 0x30)
				return nil
			}
			return nil
		}
	}
	if message("No se pudo descargar el beatmap.\n"+e.Error()+"\n\n¿Abrir la página oficial en tu navegador?", 0x24) == 6 {
		return forward(raw)
	}
	return nil
}
func main() {
	var e error
	switch {
	case len(os.Args) == 1:
		if validBrowser(load().Browser) && same(mustExe(), installed()) {
			r := message("¿Abrir configuración de aplicaciones predeterminadas?\n\nSí: abrir configuración.\nNo: cambiar navegador de respaldo.\nCancelar: cerrar.", 0x23)
			if r == 6 {
				settings()
			} else if r == 7 {
				e = setup(true)
			}
		} else {
			e = setup(false)
		}
	case os.Args[1] == "--browser":
		e = setup(true)
	case os.Args[1] == "--uninstall":
		e = uninstall()
	default:
		e = runURL(os.Args[1])
	}
	if e != nil {
		message(e.Error(), 0x10)
	}
}
func mustExe() string { p, _ := os.Executable(); return p }
