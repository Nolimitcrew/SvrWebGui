package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"SvrWebGui/internal/modbus"
	"SvrWebGui/internal/ttr01"
	"SvrWebGui/internal/update"
)

// App — основная структура приложения Wails.
type App struct {
	ctx       context.Context
	transport *modbus.MockTransport
	driver    *ttr01.Driver
	version   string // текущая версия приложения (тег релиза)
}

// ReleaseCheck — результат проверки наличия обновления.
type ReleaseCheck struct {
	// Newer — true, если в релизе версия новее текущей.
	Newer bool `json:"newer"`
	// TagName — тег релиза (например "v0.1.0").
	TagName string `json:"tagName"`
	// Name — название релиза.
	Name string `json:"name"`
	// Body — описание изменений (release notes).
	Body string `json:"body"`
	// HTMLURL — ссылка на страницу релиза.
	HTMLURL string `json:"htmlUrl"`
	// AssetName — имя подходящего артефакта для Windows amd64.
	AssetName string `json:"assetName"`
	// AssetSize — размер артефакта в байтах.
	AssetSize int64 `json:"assetSize"`
	// AssetURL — прямая ссылка на скачивание артефакта.
	AssetURL string `json:"assetUrl"`
	// HasAsset — есть ли подходящий артефакт в релизе.
	HasAsset bool `json:"hasAsset"`
	// Error — текст ошибки, если проверка не удалась.
	Error string `json:"error"`
}

// DownloadResult — результат скачивания обновления.
type DownloadResult struct {
	// Success — true, если файл успешно скачан.
	Success bool `json:"success"`
	// Path — полный путь к сохранённому файлу.
	Path string `json:"path"`
	// Size — размер скачанного файла в байтах.
	Size int64 `json:"size"`
	// Error — текст ошибки, если скачивание не удалось.
	Error string `json:"error"`
}

// NewApp создаёт новое приложение Wails.
func NewApp() *App {
	return &App{
		version: "v0.1.0", // текущая версия; при публикации релиза меняем на соответствующий тег
	}
}

// startup вызывается при старте приложения. Сохраняем контекст
// и инициализируем транспорт с драйвером TTR-01 (используется MockTransport).
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Создаём эмулятор Modbus (MockTransport). Позже можно заменить
	// на RTUTransport/TCPTransport без изменения UI.
	a.transport = modbus.NewMockTransport()
	a.driver = ttr01.NewDriver(a.transport, 1)
}

// GetCurrentData читает текущие данные контроллера TTR-01.
//
// Данные заполняются тестовыми значениями (эмулятор), чтобы интерфейс
// сразу что-то показывал. При подключении реального прибора достаточно
// заменить транспорт в startup().
func (a *App) GetCurrentData() (*ttr01.CurrentData, error) {
	if a.driver == nil || a.transport == nil {
		return nil, fmt.Errorf("драйвер TTR-01 не инициализирован")
	}

	// Наполняем эмулятор свежими тестовыми показаниями (включая текущее время).
	a.fillMockRegisters()

	// Чтение с небольшим таймаутом, чтобы UI не зависал.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := a.driver.ReadCurrentData(ctx)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать данные TTR-01: %w", err)
	}
	return data, nil
}

// CheckForUpdate проверяет наличие новой версии в GitHub Releases.
// Репозиторий: Nolimitcrew/SvrWebGui.
func (a *App) CheckForUpdate() ReleaseCheck {
	const owner = "Nolimitcrew"
	const repo = "SvrWebGui"

	checker := update.NewChecker(owner, repo)
	// Используем короткий таймаут, чтобы не блокировать UI.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rel, err := checker.LatestRelease(ctx)
	if err != nil {
		return ReleaseCheck{
			Newer: false,
			Error: err.Error(),
		}
	}

	res := ReleaseCheck{
		Newer:   update.IsNewer(a.version, rel.TagName),
		TagName: rel.TagName,
		Name:    rel.Name,
		Body:    rel.Body,
		HTMLURL: rel.HTMLURL,
	}

	// Ищем подходящий артефакт для Windows amd64.
	asset, err := update.PickAsset(rel)
	if err != nil {
		res.HasAsset = false
		// Ошибку не показываем как критическую — можно открыть страницу релиза.
		return res
	}

	res.HasAsset = true
	res.AssetName = asset.Name
	res.AssetSize = asset.Size
	res.AssetURL = asset.DownloadURL
	return res
}

// DownloadUpdate скачивает артефакт релиза по assetURL и сохраняет
// его в папку «Загрузки» (Downloads) текущего пользователя.
// Возвращает путь к сохранённому файлу.
func (a *App) DownloadUpdate(assetURL, fileName string) DownloadResult {
	if assetURL == "" {
		return DownloadResult{Success: false, Error: "не указана ссылка на скачивание"}
	}
	if fileName == "" {
		// Попробуем вырезать имя из URL, если не передали.
		fileName = filepath.Base(assetURL)
		if fileName == "." || fileName == "/" || fileName == "" {
			fileName = "SvrWebGui_update.zip"
		}
	}

	// Определяем папку Downloads.
	home, err := os.UserHomeDir()
	if err != nil {
		return DownloadResult{Success: false, Error: fmt.Sprintf("не удалось определить домашнюю папку: %v", err)}
	}
	downloads := filepath.Join(home, "Downloads")
	if err := os.MkdirAll(downloads, 0o755); err != nil {
		return DownloadResult{Success: false, Error: fmt.Sprintf("не удалось создать папку Downloads: %v", err)}
	}

	destPath := filepath.Join(downloads, fileName)
	f, err := os.Create(destPath)
	if err != nil {
		return DownloadResult{Success: false, Error: fmt.Sprintf("не удалось создать файл: %v", err)}
	}
	defer f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	n, err := update.Download(ctx, assetURL, f)
	if err != nil {
		// Удаляем частично скачанный файл, чтобы не оставлять битый архив.
		_ = os.Remove(destPath)
		return DownloadResult{Success: false, Error: err.Error()}
	}

	return DownloadResult{Success: true, Path: destPath, Size: n}
}

// fillMockRegisters заполняет эмулятор Modbus тестовыми значениями.
func (a *App) fillMockRegisters() {
	const base = ttr01.RegCurrentDataStart

	writeTemp := func(off uint16, temp float64) {
		hi, lo := ttr01.TempToRegisters(temp)
		a.transport.SetRegisters(base+off, hi, lo)
	}

	writeTemp(ttr01.OffTOutdoor, -5.5) // уличная
	writeTemp(ttr01.OffTIndoor, 22.0)  // комнатная
	writeTemp(ttr01.OffTSupply, 45.0)  // подача
	writeTemp(ttr01.OffTReturn, 35.0)  // обратка
	writeTemp(ttr01.OffTDHW, 55.0)     // ГВС

	// Флаги: отопление + насос включены.
	a.transport.SetRegister(base+ttr01.OffStatus, ttr01.StatusHeating|ttr01.StatusPump)

	now := time.Now()
	a.transport.SetRegister(base+ttr01.OffYear, uint16(now.Year()))
	a.transport.SetRegister(base+ttr01.OffMonth, uint16(int(now.Month())))
	a.transport.SetRegister(base+ttr01.OffDay, uint16(now.Day()))
	a.transport.SetRegister(base+ttr01.OffHour, uint16(now.Hour()))
	a.transport.SetRegister(base+ttr01.OffMinute, uint16(now.Minute()))
	a.transport.SetRegister(base+ttr01.OffSecond, uint16(now.Second()))
}
