// Package update проверяет и скачивает новые версии приложения из
// GitHub Releases. Это — серверная «часть» кнопки «Обновить из облака»,
// которая появится в окне программы на этапе подключения Wails/Vue.
//
// Пакет не зависит от Wails и не трогает UI: он умеет спросить у GitHub
// «какая последняя версия?», сравнить её с текущей и скачать артефакт
// релиза. Установка нового бинарника и перезапуск отложены на слой UI:
// заменить работающий исполняемый файл нельзя напрямую (он открыт процессом),
// и это требует привилегий / вспомогательного процесса — типичная задача
// платформенно-специфичного кода.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Asset — один файл из релиза (атрибут GitHub Releases).
type Asset struct {
	Name        string `json:"name"`                 // имя файла
	Size        int64  `json:"size"`                 // размер в байтах
	DownloadURL string `json:"browser_download_url"` // прямая ссылка на скачивание
}

// ReleaseInfo — метаданные последнего публичного релиза.
type ReleaseInfo struct {
	TagName string  `json:"tag_name"` // тег, например "v1.2.3"
	Name    string  `json:"name"`     // заголовок релиза
	Body    string  `json:"body"`     // release notes
	HTMLURL string  `json:"html_url"` // страница релиза в браузере
	Assets  []Asset `json:"assets"`   // файлы релиза
}

// Checker — клиент, который спрашивает GitHub о новых версиях.
type Checker struct {
	Owner   string       // GitHub-организация или пользователь (например "User-GMN")
	Repo    string       // имя репозитория (например "heatnode")
	BaseURL string       // для тестов: пусто → https://api.github.com
	Client  *http.Client // желаемый http‑клиент (nil — стандартный)
}

// NewChecker создаёт Checker для репозитория owner/repo.
func NewChecker(owner, repo string) *Checker {
	return &Checker{Owner: owner, Repo: repo}
}

// LatestRelease запрашивает информацию о последнем (самом свежем) релизе
// из репозитория. Использует публичный REST‑API GitHub:
//
//	GET https://api.github.com/repos/{owner}/{repo}/releases/latest
//
// При BaseURL != "" (чаще всего в тестах) запрос идёт туда — это упрощает
// проверку без реального сетевого доступа к github.com.
func (c *Checker) LatestRelease(ctx context.Context) (*ReleaseInfo, error) {
	endpoint := c.releaseEndpoint()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("update: сбор запроса: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "heatnode")

	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("update: запрос к GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update: GitHub вернул %s", resp.Status)
	}

	var rel ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("update: разбор ответа GitHub: %w", err)
	}
	return &rel, nil
}

// releaseEndpoint возвращает URL последнего релиза.
func (c *Checker) releaseEndpoint() string {
	const path = "/repos/%s/%s/releases/latest"
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/") + fmt.Sprintf(path, c.Owner, c.Repo)
	}
	return "https://api.github.com/repos/" + fmt.Sprintf(path, c.Owner, c.Repo)[1:]
}

// PickAsset ищет в релизе артефакат для Windows‑64 бит. Возвращает ошибку,
// если подходящего файла нет.
//
// Соглашение об именах артефактов (устанавливается на этапе публикации релиза):
//
//	heatnode_<версия>_windows_amd64.zip
//
// Достаточно, чтобы имя содержало "windows" и "amd64" (без учёта регистра).
func PickAsset(rel *ReleaseInfo) (*Asset, error) {
	for i := range rel.Assets {
		a := &rel.Assets[i]
		low := strings.ToLower(a.Name)
		if strings.Contains(low, "windows") && strings.Contains(low, "amd64") {
			return a, nil
		}
	}
	return nil, fmt.Errorf("update: в релизе %s нет артефаката для windows/amd64", rel.TagName)
}

// IsNewer возвращает true, если текущая версия current старее версии из релиза
// candidate. Версии сравниваются как семантичные: ожидаются теги вида
// "v1.2.3" (префикс v — необязателен). Предрелизы не учитываются.
func IsNewer(current, candidate string) bool {
	return compareVersions(candidate, current) > 0
}

// compareVersions возвращает -1, 0 или 1 аналогично a <=> b.
func compareVersions(a, b string) int {
	va := parseVersion(a)
	vb := parseVersion(b)
	for i := 0; i < 3; i++ {
		if va[i] != vb[i] {
			if va[i] > vb[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

// parseVersion разбирает тег "v1.2.3" → [1 2 3].
// Префикс "v" и любые суффиксы после patch (например "-rc1") отбрасываются.
func parseVersion(s string) [3]uint {
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, ".", 3)
	var out [3]uint
	for i, p := range parts {
		if j := strings.IndexAny(p, "-+"); j >= 0 {
			p = p[:j]
		}
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return out // "грязная" версия → остальные компоненты = 0
		}
		out[i] = uint(n)
	}
	return out
}

// Download скачивает файл по url и потоково записывает его в dst.
// Возвращает число скачанных байт. Параметр ctx управляет таймаутом.
//
// GitHub требует User-Agent; прямые ссылки на артефакты редиректят —
// стандартный клиент следует за перенаправлениями автоматически.
func Download(ctx context.Context, url string, dst io.Writer) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("update: сбор запроса: %w", err)
	}
	req.Header.Set("User-Agent", "heatnode")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("update: скачивание: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("update: сервер вернул %s", resp.Status)
	}
	// TODO: прогресс‑бар через io.TeeReader / канал сообщений.
	return io.Copy(dst, resp.Body)
}
