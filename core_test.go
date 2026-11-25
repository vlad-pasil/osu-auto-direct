package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct{ url, k, id string }{{"https://osu.ppy.sh/beatmapsets/320118#osu/10", "set", "320118"}, {"https://osu.ppy.sh/b/15", "beatmap", "15"}, {"http://osu.ppy.sh/s/7", "set", "7"}, {"https://osu.ppy.sh/beatmaps/55", "beatmap", "55"}, {"https://osu.ppy.sh/d/12", "set", "12"}, {"https://osu.ppy.sh/beatmapsets/12/download", "set", "12"}, {"https://osu.ppy.sh/users/15", "", ""}, {"https://osu.ppy.sh.evil.test/b/1", "", ""}, {"https://osu.ppy.sh@evil.test/b/1", "", ""}, {"https://osu.ppy.sh/b/0", "", ""}, {"https://osu.ppy.sh/beatmapsets/12/discussion", "", ""}, {"https://example.com", "", ""}}
	for _, v := range tests {
		k, id := parseLink(v.url)
		if k != v.k || id != v.id {
			t.Errorf("%s: %s %s", v.url, k, id)
		}
	}
}
func TestArchive(t *testing.T) {
	for _, v := range []struct {
		entry string
		ok    bool
	}{{"map.osu", true}, {"../map.osu", false}, {"skin.ini", false}, {"C:/map.osu", false}, {`..\map.osu`, false}} {
		p := filepath.Join(t.TempDir(), "test.osz")
		f, _ := os.Create(p)
		w := zip.NewWriter(f)
		x, _ := w.Create(v.entry)
		x.Write([]byte("test"))
		w.Close()
		f.Close()
		if (validateArchive(p) == nil) != v.ok {
			t.Errorf("%s", v.entry)
		}
	}
}
