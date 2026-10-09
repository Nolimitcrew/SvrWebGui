// Command heatnode — точка входа в приложение «Мониторинг теплового узла».
//
// На этом этапе программа проверяет транспортный слой (internal/modbus) и
// драйвер TTR-01 (internal/ttr01) на заглушке MockTransport — без
// реального оборудования. Реальная связь через COM-порт или TCP появится
// позже, заменив заглушку на RTUTransport или TCPTransport.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"heatnode/internal/modbus"
	"heatnode/internal/ttr01"
)

func main() {
	// Таймаут на весь обмен — чтобы программа не зависала в случае
	// потери связи с прибором.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Создаём эмулятор транспорта и наполняем его «показаниями»,
	//    похожими на настоящие (это заменяется реальным COM-портом позже).
	transport := modbus.NewMockTransport()
	fillMockRegisters(transport)

	// 2. Драйвер контроллера. Адрес 1 — типичный Modbus-адрес по умолчанию.
	driver := ttr01.NewDriver(transport, 1)

	// 3. Читаем текущие данные.
	data, err := driver.ReadCurrentData(ctx)
	if err != nil {
		log.Fatalf("не удалось прочитать данные TTR-01: %v", err)
	}

	// 4. Выводим результат.
	printCurrentData(data)

	// 5. Закрываем транспорт (у заглушки это пустая операция,
	//    у реального — освободит COM-порт или сокет).
	if err := transport.Close(); err != nil {
		log.Printf("закрытие транспорта: %v", err)
	}
}

// fillMockRegisters пишет в эмулятор тестовые значения регистров,
// чтобы драйвер имел от чего спарсить структуру CurrentData.
func fillMockRegisters(t *modbus.MockTransport) {
	const base = ttr01.RegCurrentDataStart

	// Температуры записываются парой регистров в формате IEEE 754
	// (старший регистр — первым, big-endian порядок слов).
	writeTemp := func(off uint16, temp float64) {
		hi, lo := ttr01.TempToRegisters(temp)
		t.SetRegisters(base+off, hi, lo)
	}

	writeTemp(ttr01.OffTOutdoor, -5.5) // уличная  —5,5 °C
	writeTemp(ttr01.OffTIndoor, 22.0)  // комнатная 22,0 °C
	writeTemp(ttr01.OffTSupply, 45.0)  // подача   45,0 °C
	writeTemp(ttr01.OffTReturn, 35.0)  // обратка  35,0 °C
	writeTemp(ttr01.OffTDHW, 55.0)     // ГВС      55,0 °C

	// Флаги: отопление + насос включены, остальные выключены.
	t.SetRegister(base+ttr01.OffStatus, ttr01.StatusHeating|ttr01.StatusPump)

	// Время прибора — возьмём текущие значения хоста (для демонстрации).
	now := time.Now()
	t.SetRegister(base+ttr01.OffYear, uint16(now.Year()))
	t.SetRegister(base+ttr01.OffMonth, uint16(int(now.Month())))
	t.SetRegister(base+ttr01.OffDay, uint16(now.Day()))
	t.SetRegister(base+ttr01.OffHour, uint16(now.Hour()))
	t.SetRegister(base+ttr01.OffMinute, uint16(now.Minute()))
	t.SetRegister(base+ttr01.OffSecond, uint16(now.Second()))
}

// printCurrentData выводит данные в человекочитаемом виде.
func printCurrentData(d *ttr01.CurrentData) {
	fmt.Println("=== Текущие данные контроллера TTR-01 (адрес 1) ===")
	fmt.Printf("Уличная  температура: %6.1f °C\n", d.TOutdoor)
	fmt.Printf("Комнатная температура: %5.1f °C\n", d.TIndoor)
	fmt.Printf("Подача:  %6.1f °C\n", d.TSupply)
	fmt.Printf("Обратка: %6.1f °C\n", d.TReturn)
	fmt.Printf("ГВС:     %6.1f °C\n", d.TDHW)
	fmt.Printf("Время прибора: %s\n", d.DeviceTime)

	fmt.Printf("Флаги состояния: 0x%04X\n", d.Status)
	if d.Has(ttr01.StatusHeating) {
		fmt.Println("  • отопление включено")
	}
	if d.Has(ttr01.StatusDHW) {
		fmt.Println("  • нагрев ГВС включён")
	}
	if d.Has(ttr01.StatusPump) {
		fmt.Println("  • насос работает")
	}
	if d.Has(ttr01.StatusAlarm) {
		fmt.Println("  • АВАРИЯ")
	}
	fmt.Println("=== Конец блока данных ===")
}
