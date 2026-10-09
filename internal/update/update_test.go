// Package update: тесты работают без доступа к github.com — сервер
// заменяется httptest.Server, а сравнение версий проверяется на таблице.
package update

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// sampleRelease — типичный ответ GitHub /releases/latest с двумя артефактами.
const sampleRelease = `{
  "tag_name": "v1.2.3",
  "name": "Release 1.2.3",
  "body": "• Исправлены баги\n• Добавлена кнопка обновления",
  "html_url": "https://github.com/user/heatnode/releases/tag/v1.2.3",
  "assets": [
    {"name": "heatnode_1.2.3_windows_amd64.zip", "size": 10, "browser_download_url": "%s"},
    {"name": "heatnode_1.2.3_linux_amd64.tar.gz", "size": 9, "browser_download_url": "https://example/x.tar.gz"}
  ]
}`

func TestPickAssetWindows(t *testing.T) {
	t.Parallel()
	rel := &ReleaseInfo{TagName: "v1.2.3", Assets: []Asset{
		{Name: "heatnode_1.2.3_windows_amd64.zip", DownloadURL: "u1"},
		{Name: "heatnode_1.2.3_linux_amd64.tar.gz", DownloadURL: "u2"},
	}}
	a, err := PickAsset(rel)
	if err != nil || a.DownloadURL != "u1" {
		t.Fatalf("PickAsset = %+v, err=%v; ожидался windows‑артефакт", a, err)
	}
}

func TestPickAssetNone(t *testing.T) {
	t.Parallel()
	rel := &ReleaseInfo{TagName: "v1.0.0", Assets: []Asset{{Name: "linux.tar.gz", DownloadURL: "x"}}}
	if _, err := PickAsset(rel); err == nil {
		t.Error("ожидалась ошибка при отсутствии windows/amd64 артефакта")
	}
}

func TestLatestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("User-Agent не передан")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, sampleRelease, "http://example/assets/latest.zip")
	}))
	defer srv.Close()

	c := &Checker{Owner: "user", Repo: "heatnode", BaseURL: srv.URL, Client: srv.Client()}
	rel, err := c.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if rel.TagName != "v1.2.3" {
		t.Errorf("TagName = %q, ожидался v1.2.3", rel.TagName)
	}
	if rel.HTMLURL == "" {
		t.Error("html_url пуст")
	}
	if _, err := PickAsset(rel); err != nil {
		t.Fatalf("PickAsset в ответе GitHub: %v", err)
	}
}

func TestLatestReleaseHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := &Checker{Owner: "user", Repo: "heatnode", BaseURL: srv.URL, Client: srv.Client()}
	if _, err := c.LatestRelease(context.Background()); err == nil {
		t.Error("ожидалась ошибка при статусе 500")
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		current, candidate string
		want               bool
	}{
		{"v1.2.3", "v1.2.4", true},
		{"v1.2.3", "v1.2.3", false},
		{"v1.2.4", "v1.2.3", false},
		{"v1.3.0", "v1.2.3", false},
		{"1.2.3", "v1.3.0", true},
		{"v2.0.0", "v1.9.9", false},
		{"v1.2.3", "v2.0.0", true},
	}
	for _, c := range cases {
		if got := IsNewer(c.current, c.candidate); got != c.want {
			t.Errorf("IsNewer(%q,%q)=%v, ожидалось %v", c.current, c.candidate, got, c.want)
		}
	}
}

func TestDownload(t *testing.T) {
	wantPayload := []byte("heatnode-binary-bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.Copy(w, bytes.NewReader(wantPayload))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	n, err := Download(context.Background(), srv.URL, &buf)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if n != int64(len(wantPayload)) {
		t.Errorf("скачано %d байт, ожидалось %d", n, len(wantPayload))
	}
	if !bytes.Equal(buf.Bytes(), wantPayload) {
		t.Errorf("payload = %q, ожидалось %q", buf.String(), wantPayload)
	}
}
