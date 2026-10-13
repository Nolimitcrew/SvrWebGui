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

// App вЂ” РѕСЃРЅРѕРІРЅР°СЏ СЃС‚СЂСѓРєС‚СѓСЂР° РїСЂРёР»РѕР¶РµРЅРёСЏ Wails.
type App struct {
	ctx       context.Context
	transport *modbus.MockTransport
	driver    *ttr01.Driver
	version   string // С‚РµРєСѓС‰Р°СЏ РІРµСЂСЃРёСЏ РїСЂРёР»РѕР¶РµРЅРёСЏ (С‚РµРі СЂРµР»РёР·Р°)
}

// ReleaseCheck вЂ” СЂРµР·СѓР»СЊС‚Р°С‚ РїСЂРѕРІРµСЂРєРё РЅР°Р»РёС‡РёСЏ РѕР±РЅРѕРІР»РµРЅРёСЏ.
type ReleaseCheck struct {
	// Newer вЂ” true, РµСЃР»Рё РІ СЂРµР»РёР·Рµ РІРµСЂСЃРёСЏ РЅРѕРІРµРµ С‚РµРєСѓС‰РµР№.
	Newer bool `json:"newer"`
	// TagName вЂ” С‚РµРі СЂРµР»РёР·Р° (РЅР°РїСЂРёРјРµСЂ "v0.1.0").
	TagName string `json:"tagName"`
	// Name вЂ” РЅР°Р·РІР°РЅРёРµ СЂРµР»РёР·Р°.
	Name string `json:"name"`
	// Body вЂ” РѕРїРёСЃР°РЅРёРµ РёР·РјРµРЅРµРЅРёР№ (release notes).
	Body string `json:"body"`
	// HTMLURL вЂ” СЃСЃС‹Р»РєР° РЅР° СЃС‚СЂР°РЅРёС†Сѓ СЂРµР»РёР·Р°.
	HTMLURL string `json:"htmlUrl"`
	// AssetName вЂ” РёРјСЏ РїРѕРґС…РѕРґСЏС‰РµРіРѕ Р°СЂС‚РµС„Р°РєС‚Р° РґР»СЏ Windows amd64.
	AssetName string `json:"assetName"`
	// AssetSize вЂ” СЂР°Р·РјРµСЂ Р°СЂС‚РµС„Р°РєС‚Р° РІ Р±Р°Р№С‚Р°С….
	AssetSize int64 `json:"assetSize"`
	// AssetURL вЂ” РїСЂСЏРјР°СЏ СЃСЃС‹Р»РєР° РЅР° СЃРєР°С‡РёРІР°РЅРёРµ Р°СЂС‚РµС„Р°РєС‚Р°.
	AssetURL string `json:"assetUrl"`
	// HasAsset вЂ” РµСЃС‚СЊ Р»Рё РїРѕРґС…РѕРґСЏС‰РёР№ Р°СЂС‚РµС„Р°РєС‚ РІ СЂРµР»РёР·Рµ.
	HasAsset bool `json:"hasAsset"`
	// Error вЂ” С‚РµРєСЃС‚ РѕС€РёР±РєРё, РµСЃР»Рё РїСЂРѕРІРµСЂРєР° РЅРµ СѓРґР°Р»Р°СЃСЊ.
	Error string `json:"error"`
}

// DownloadResult вЂ” СЂРµР·СѓР»СЊС‚Р°С‚ СЃРєР°С‡РёРІР°РЅРёСЏ РѕР±РЅРѕРІР»РµРЅРёСЏ.
type DownloadResult struct {
	// Success вЂ” true, РµСЃР»Рё С„Р°Р№Р» СѓСЃРїРµС€РЅРѕ СЃРєР°С‡Р°РЅ.
	Success bool `json:"success"`
	// Path вЂ” РїРѕР»РЅС‹Р№ РїСѓС‚СЊ Рє СЃРѕС…СЂР°РЅС‘РЅРЅРѕРјСѓ С„Р°Р№Р»Сѓ.
	Path string `json:"path"`
	// Size вЂ” СЂР°Р·РјРµСЂ СЃРєР°С‡Р°РЅРЅРѕРіРѕ С„Р°Р№Р»Р° РІ Р±Р°Р№С‚Р°С….
	Size int64 `json:"size"`
	// Error вЂ” С‚РµРєСЃС‚ РѕС€РёР±РєРё, РµСЃР»Рё СЃРєР°С‡РёРІР°РЅРёРµ РЅРµ СѓРґР°Р»РѕСЃСЊ.
	Error string `json:"error"`
}

// NewApp СЃРѕР·РґР°С‘С‚ РЅРѕРІРѕРµ РїСЂРёР»РѕР¶РµРЅРёРµ Wails.
func NewApp() *App {
	return &App{
		version: "v0.1.7", // С‚РµРєСѓС‰Р°СЏ РІРµСЂСЃРёСЏ; РїСЂРё РїСѓР±Р»РёРєР°С†РёРё СЂРµР»РёР·Р° РјРµРЅСЏРµРј РЅР° СЃРѕРѕС‚РІРµС‚СЃС‚РІСѓСЋС‰РёР№ С‚РµРі
	}
}

// startup РІС‹Р·С‹РІР°РµС‚СЃСЏ РїСЂРё СЃС‚Р°СЂС‚Рµ РїСЂРёР»РѕР¶РµРЅРёСЏ. РЎРѕС…СЂР°РЅСЏРµРј РєРѕРЅС‚РµРєСЃС‚
// Рё РёРЅРёС†РёР°Р»РёР·РёСЂСѓРµРј С‚СЂР°РЅСЃРїРѕСЂС‚ СЃ РґСЂР°Р№РІРµСЂРѕРј TTR-01 (РёСЃРїРѕР»СЊР·СѓРµС‚СЃСЏ MockTransport).
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// РЎРѕР·РґР°С‘Рј СЌРјСѓР»СЏС‚РѕСЂ Modbus (MockTransport). РџРѕР·Р¶Рµ РјРѕР¶РЅРѕ Р·Р°РјРµРЅРёС‚СЊ
	// РЅР° RTUTransport/TCPTransport Р±РµР· РёР·РјРµРЅРµРЅРёСЏ UI.
	a.transport = modbus.NewMockTransport()
	a.driver = ttr01.NewDriver(a.transport, 1)
}

// GetCurrentData С‡РёС‚Р°РµС‚ С‚РµРєСѓС‰РёРµ РґР°РЅРЅС‹Рµ РєРѕРЅС‚СЂРѕР»Р»РµСЂР° TTR-01.
//
// Р”Р°РЅРЅС‹Рµ Р·Р°РїРѕР»РЅСЏСЋС‚СЃСЏ С‚РµСЃС‚РѕРІС‹РјРё Р·РЅР°С‡РµРЅРёСЏРјРё (СЌРјСѓР»СЏС‚РѕСЂ), С‡С‚РѕР±С‹ РёРЅС‚РµСЂС„РµР№СЃ
// СЃСЂР°Р·Сѓ С‡С‚Рѕ-С‚Рѕ РїРѕРєР°Р·С‹РІР°Р». РџСЂРё РїРѕРґРєР»СЋС‡РµРЅРёРё СЂРµР°Р»СЊРЅРѕРіРѕ РїСЂРёР±РѕСЂР° РґРѕСЃС‚Р°С‚РѕС‡РЅРѕ
// Р·Р°РјРµРЅРёС‚СЊ С‚СЂР°РЅСЃРїРѕСЂС‚ РІ startup().
func (a *App) GetCurrentData() (*ttr01.CurrentData, error) {
	if a.driver == nil || a.transport == nil {
		return nil, fmt.Errorf("РґСЂР°Р№РІРµСЂ TTR-01 РЅРµ РёРЅРёС†РёР°Р»РёР·РёСЂРѕРІР°РЅ")
	}

	// РќР°РїРѕР»РЅСЏРµРј СЌРјСѓР»СЏС‚РѕСЂ СЃРІРµР¶РёРјРё С‚РµСЃС‚РѕРІС‹РјРё РїРѕРєР°Р·Р°РЅРёСЏРјРё (РІРєР»СЋС‡Р°СЏ С‚РµРєСѓС‰РµРµ РІСЂРµРјСЏ).
	a.fillMockRegisters()

	// Р§С‚РµРЅРёРµ СЃ РЅРµР±РѕР»СЊС€РёРј С‚Р°Р№РјР°СѓС‚РѕРј, С‡С‚РѕР±С‹ UI РЅРµ Р·Р°РІРёСЃР°Р».
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := a.driver.ReadCurrentData(ctx)
	if err != nil {
		return nil, fmt.Errorf("РЅРµ СѓРґР°Р»РѕСЃСЊ РїСЂРѕС‡РёС‚Р°С‚СЊ РґР°РЅРЅС‹Рµ TTR-01: %w", err)
	}
	return data, nil
}

// CheckForUpdate РїСЂРѕРІРµСЂСЏРµС‚ РЅР°Р»РёС‡РёРµ РЅРѕРІРѕР№ РІРµСЂСЃРёРё РІ GitHub Releases.
// Р РµРїРѕР·РёС‚РѕСЂРёР№: Nolimitcrew/SvrWebGui.
func (a *App) CheckForUpdate() ReleaseCheck {
	const owner = "Nolimitcrew"
	const repo = "SvrWebGui"

	checker := update.NewChecker(owner, repo)
	// РСЃРїРѕР»СЊР·СѓРµРј РєРѕСЂРѕС‚РєРёР№ С‚Р°Р№РјР°СѓС‚, С‡С‚РѕР±С‹ РЅРµ Р±Р»РѕРєРёСЂРѕРІР°С‚СЊ UI.
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

	// РС‰РµРј РїРѕРґС…РѕРґСЏС‰РёР№ Р°СЂС‚РµС„Р°РєС‚ РґР»СЏ Windows amd64.
	asset, err := update.PickAsset(rel)
	if err != nil {
		res.HasAsset = false
		// РћС€РёР±РєСѓ РЅРµ РїРѕРєР°Р·С‹РІР°РµРј РєР°Рє РєСЂРёС‚РёС‡РµСЃРєСѓСЋ вЂ” РјРѕР¶РЅРѕ РѕС‚РєСЂС‹С‚СЊ СЃС‚СЂР°РЅРёС†Сѓ СЂРµР»РёР·Р°.
		return res
	}

	res.HasAsset = true
	res.AssetName = asset.Name
	res.AssetSize = asset.Size
	res.AssetURL = asset.DownloadURL
	return res
}

// DownloadUpdate СЃРєР°С‡РёРІР°РµС‚ Р°СЂС‚РµС„Р°РєС‚ СЂРµР»РёР·Р° РїРѕ assetURL Рё СЃРѕС…СЂР°РЅСЏРµС‚
// РµРіРѕ РІ РїР°РїРєСѓ В«Р—Р°РіСЂСѓР·РєРёВ» (Downloads) С‚РµРєСѓС‰РµРіРѕ РїРѕР»СЊР·РѕРІР°С‚РµР»СЏ.
// Р’РѕР·РІСЂР°С‰Р°РµС‚ РїСѓС‚СЊ Рє СЃРѕС…СЂР°РЅС‘РЅРЅРѕРјСѓ С„Р°Р№Р»Сѓ.
func (a *App) DownloadUpdate(assetURL, fileName string) DownloadResult {
	if assetURL == "" {
		return DownloadResult{Success: false, Error: "РЅРµ СѓРєР°Р·Р°РЅР° СЃСЃС‹Р»РєР° РЅР° СЃРєР°С‡РёРІР°РЅРёРµ"}
	}
	if fileName == "" {
		// РџРѕРїСЂРѕР±СѓРµРј РІС‹СЂРµР·Р°С‚СЊ РёРјСЏ РёР· URL, РµСЃР»Рё РЅРµ РїРµСЂРµРґР°Р»Рё.
		fileName = filepath.Base(assetURL)
		if fileName == "." || fileName == "/" || fileName == "" {
			fileName = "SvrWebGui_update.zip"
		}
	}

	// РћРїСЂРµРґРµР»СЏРµРј РїР°РїРєСѓ Downloads.
	home, err := os.UserHomeDir()
	if err != nil {
		return DownloadResult{Success: false, Error: fmt.Sprintf("РЅРµ СѓРґР°Р»РѕСЃСЊ РѕРїСЂРµРґРµР»РёС‚СЊ РґРѕРјР°С€РЅСЋСЋ РїР°РїРєСѓ: %v", err)}
	}
	downloads := filepath.Join(home, "Downloads")
	if err := os.MkdirAll(downloads, 0o755); err != nil {
		return DownloadResult{Success: false, Error: fmt.Sprintf("РЅРµ СѓРґР°Р»РѕСЃСЊ СЃРѕР·РґР°С‚СЊ РїР°РїРєСѓ Downloads: %v", err)}
	}

	destPath := filepath.Join(downloads, fileName)
	f, err := os.Create(destPath)
	if err != nil {
		return DownloadResult{Success: false, Error: fmt.Sprintf("РЅРµ СѓРґР°Р»РѕСЃСЊ СЃРѕР·РґР°С‚СЊ С„Р°Р№Р»: %v", err)}
	}
	defer f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	n, err := update.Download(ctx, assetURL, f)
	if err != nil {
		// РЈРґР°Р»СЏРµРј С‡Р°СЃС‚РёС‡РЅРѕ СЃРєР°С‡Р°РЅРЅС‹Р№ С„Р°Р№Р», С‡С‚РѕР±С‹ РЅРµ РѕСЃС‚Р°РІР»СЏС‚СЊ Р±РёС‚С‹Р№ Р°СЂС…РёРІ.
		_ = os.Remove(destPath)
		return DownloadResult{Success: false, Error: err.Error()}
	}

	return DownloadResult{Success: true, Path: destPath, Size: n}
}

// fillMockRegisters Р·Р°РїРѕР»РЅСЏРµС‚ СЌРјСѓР»СЏС‚РѕСЂ Modbus С‚РµСЃС‚РѕРІС‹РјРё Р·РЅР°С‡РµРЅРёСЏРјРё.
func (a *App) fillMockRegisters() {
	const base = ttr01.RegCurrentDataStart

	writeTemp := func(off uint16, temp float64) {
		hi, lo := ttr01.TempToRegisters(temp)
		a.transport.SetRegisters(base+off, hi, lo)
	}

	writeTemp(ttr01.OffTOutdoor, -5.5) // СѓР»РёС‡РЅР°СЏ
	writeTemp(ttr01.OffTIndoor, 22.0)  // РєРѕРјРЅР°С‚РЅР°СЏ
	writeTemp(ttr01.OffTSupply, 45.0)  // РїРѕРґР°С‡Р°
	writeTemp(ttr01.OffTReturn, 35.0)  // РѕР±СЂР°С‚РєР°
	writeTemp(ttr01.OffTDHW, 55.0)     // Р“Р’РЎ

	// Р¤Р»Р°РіРё: РѕС‚РѕРїР»РµРЅРёРµ + РЅР°СЃРѕСЃ РІРєР»СЋС‡РµРЅС‹.
	a.transport.SetRegister(base+ttr01.OffStatus, ttr01.StatusHeating|ttr01.StatusPump)

	now := time.Now()
	a.transport.SetRegister(base+ttr01.OffYear, uint16(now.Year()))
	a.transport.SetRegister(base+ttr01.OffMonth, uint16(int(now.Month())))
	a.transport.SetRegister(base+ttr01.OffDay, uint16(now.Day()))
	a.transport.SetRegister(base+ttr01.OffHour, uint16(now.Hour()))
	a.transport.SetRegister(base+ttr01.OffMinute, uint16(now.Minute()))
	a.transport.SetRegister(base+ttr01.OffSecond, uint16(now.Second()))
}




