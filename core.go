package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const maxDownload int64 = 512 << 20
const mirror = "https://mirror.hinamizawa.ai"

var client = &http.Client{Timeout: 4 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 || req.URL.Scheme != "https" {
		return fmt.Errorf("redireccion no permitida")
	}
	return nil
}}

func validID(s string) bool {
	if len(s) == 0 || len(s) > 10 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	n, e := strconv.ParseUint(s, 10, 32)
	return e == nil && n > 0
}
func parseLink(raw string) (kind, id string) {
	u, e := url.Parse(raw)
	if e != nil || (u.Scheme != "https" && u.Scheme != "http") || !strings.EqualFold(u.Host, "osu.ppy.sh") || u.User != nil {
		return "", ""
	}
	p := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(p) < 2 || !validID(p[1]) {
		return "", ""
	}
	if len(p) > 2 && !(len(p) == 3 && p[0] == "beatmapsets" && p[2] == "download") {
		return "", ""
	}
	switch p[0] {
	case "beatmapsets", "s", "d":
		return "set", p[1]
	case "beatmaps", "b":
		return "beatmap", p[1]
	}
	return "", ""
}
func get(endpoint string) (*http.Response, error) {
	req, e := http.NewRequest("GET", endpoint, nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("User-Agent", "OsuAutoDirect/2.0 (Windows desktop)")
	r, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	if r.StatusCode != 200 {
		r.Body.Close()
		return nil, fmt.Errorf("HTTP %d", r.StatusCode)
	}
	return r, nil
}
func resolve(id string) (string, error) {
	r, e := get(mirror + "/api/get_beatmaps?b=" + id)
	if e != nil {
		return "", e
	}
	defer r.Body.Close()
	var rows []struct {
		Set json.RawMessage `json:"beatmapset_id"`
	}
	if e = json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&rows); e != nil || len(rows) == 0 {
		return "", fmt.Errorf("no se pudo resolver beatmap %s", id)
	}
	s := strings.Trim(string(rows[0].Set), `"`)
	if !validID(s) {
		return "", fmt.Errorf("ID de conjunto no valido")
	}
	return s, nil
}
func validateArchive(file string) error {
	z, e := zip.OpenReader(file)
	if e != nil {
		return e
	}
	defer z.Close()
	if len(z.File) > 15000 {
		return fmt.Errorf("demasiados archivos")
	}
	var total uint64
	found := false
	for _, f := range z.File {
		name := strings.ReplaceAll(f.Name, `\`, "/")
		clean := path.Clean(name)
		if strings.HasPrefix(name, "/") || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(name, ":") || f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("ruta insegura dentro del archivo")
		}
		if f.UncompressedSize64 > 1<<30 || total > (2<<30)-f.UncompressedSize64 {
			return fmt.Errorf("archivo descomprimido demasiado grande")
		}
		total += f.UncompressedSize64
		if strings.EqualFold(path.Ext(name), ".osu") && !f.FileInfo().IsDir() {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("no contiene mapas .osu")
	}
	return nil
}
func download(id, dir string) (string, error) {
	if !validID(id) {
		return "", fmt.Errorf("ID invalido")
	}
	if e := os.MkdirAll(dir, 0700); e != nil {
		return "", e
	}
	var last error
	for _, endpoint := range []string{mirror + "/api/v1/hinai/d/" + id, "https://catboy.best/d/" + id} {
		file, e := downloadOne(endpoint, id, dir)
		if e == nil {
			return file, nil
		}
		last = e
	}
	return "", last
}
func downloadOne(endpoint, id, dir string) (string, error) {
	r, e := get(endpoint)
	if e != nil {
		return "", e
	}
	defer r.Body.Close()
	if r.ContentLength > maxDownload {
		return "", fmt.Errorf("descarga demasiado grande")
	}
	f, e := os.CreateTemp(dir, id+"-*.part")
	if e != nil {
		return "", e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	n, e := io.Copy(f, io.LimitReader(r.Body, maxDownload+1))
	closeErr := f.Close()
	if e != nil {
		return "", e
	}
	if closeErr != nil {
		return "", closeErr
	}
	if n > maxDownload {
		return "", fmt.Errorf("descarga demasiado grande")
	}
	if r.ContentLength >= 0 && n != r.ContentLength {
		return "", fmt.Errorf("descarga incompleta")
	}
	if e = validateArchive(tmp); e != nil {
		return "", e
	}
	final := strings.TrimSuffix(tmp, filepath.Ext(tmp)) + ".osz"
	if e = os.Rename(tmp, final); e != nil {
		return "", e
	}
	return final, nil
}
