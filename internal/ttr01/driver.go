package ttr01

import (
	"context"
	"encoding/binary"
	"fmt"

	"heatnode/internal/modbus"
)

// Driver — драйвер контроллера TTR-01 (отопление + ГВС).
//
// Драйвер построен поверх абстракции modbus.Transport, благодаря чему
// один и тот же код работает с Serial RTU, TCP и даже с заглушкой
// (MockTransport) — достаточно подменить объект транспорта.
type Driver struct {
	transport modbus.Transport // канал связи
	unitID    byte             // адрес прибора в сети Modbus (1..247)
}

// NewDriver создаёт драйвер для прибора с адресом unitID, использующего
// переданный транспорт (Serial RTU, TCP или заглушку).
func NewDriver(transport modbus.Transport, unitID byte) *Driver {
	return &Driver{transport: transport, unitID: unitID}
}

// ReadCurrentData читает область «текущие данные» прибора.
//
// Запрос: функция 0x03, начальный адрес 43101 (0x0C1C), RegCurrentDataCount
// регистров. Ответ разбирается в структуру CurrentData.
func (d *Driver) ReadCurrentData(ctx context.Context) (*CurrentData, error) {
	// 1. Формируем PDU: [0x03][старший адрес][младший][старшее кол-во][младшее].
	pdu := make([]byte, 5)
	pdu[0] = modbus.FuncReadHoldingRegisters
	binary.BigEndian.PutUint16(pdu[1:3], RegCurrentDataStart)
	binary.BigEndian.PutUint16(pdu[3:5], RegCurrentDataCount)

	// 2. Полный кадр ADU = [адрес прибора][PDU].
	adu := make([]byte, 0, 1+len(pdu))
	adu = append(adu, d.unitID)
	adu = append(adu, pdu...)

	// 3. Отправляем запрос и получаем ответный кадр ADU.
	resp, err := d.transport.Send(ctx, adu)
	if err != nil {
		return nil, fmt.Errorf("ttr01: обмен Modbus не удался: %w", err)
	}

	// 4. Разбираем ответ.
	return parseReadHoldingResponse(d.unitID, RegCurrentDataCount, resp)
}

// parseReadHoldingResponse проверяет и разбирает ответ функции 0x03.
func parseReadHoldingResponse(unit byte, expectedCount uint16, resp []byte) (*CurrentData, error) {
	// Минимальная длина ответа: адрес(1) + функция(1) + счётчик байт(1)
	// + 2*expectedCount байт данных.
	minLen := 3 + int(2*expectedCount)
	if len(resp) < minLen {
		return nil, fmt.Errorf("ttr01: ответ слишком короткий: %d байт, ожидалось ≥ %d", len(resp), minLen)
	}

	// Адрес устройства в ответе должен совпадать с запрошенным.
	if resp[0] != unit {
		return nil, fmt.Errorf("ttr01: в ответе адрес %d, ожидался %d", resp[0], unit)
	}

	// Если старший бит функции установлен — прибор вернул исключение.
	if resp[1]&0x80 != 0 {
		code := modbus.ExceptionIllegalFunction // «запасной» код, если длина ответа мала
		if len(resp) > 2 {
			code = resp[2]
		}
		return nil, fmt.Errorf("ttr01: прибор вернул исключение Modbus: %s", modbus.ExceptionText(code))
	}

	// Функция в ответе должна совпадать с запросом.
	if resp[1] != modbus.FuncReadHoldingRegisters {
		return nil, fmt.Errorf("ttr01: неожиданный код функции в ответе: 0x%02X", resp[1])
	}

	// Счётчик байт данных (каждый регистр — 2 байта).
	byteCount := int(resp[2])
	if byteCount != int(2*expectedCount) {
		return nil, fmt.Errorf("ttr01: счётчик байт %d ≠ ожидаемому %d", byteCount, 2*expectedCount)
	}

	// Считываем регистры big-endian.
	regs := make([]uint16, expectedCount)
	for i := uint16(0); i < expectedCount; i++ {
		regs[i] = binary.BigEndian.Uint16(resp[3+2*i:])
	}
	if len(regs) < CurrentDataMinRegisters {
		return nil, fmt.Errorf("ttr01: не хватает регистров для разбора: %d < %d", len(regs), CurrentDataMinRegisters)
	}

	return buildCurrentData(regs), nil
}

// buildCurrentData превращает регистры в именованную структуру.
// Точная карта регистров задаётся константами Off* в registers.go.
func buildCurrentData(regs []uint16) *CurrentData {
	// get возвращает регистр по смещению или 0, если выход за границы.
	get := func(off uint16) uint16 {
		if int(off) < len(regs) {
			return regs[off]
		}
		return 0
	}

	// temp читает температуру из пары регистров [off, off+1].
	temp := func(off uint16) float64 {
		return tempFromRegisters(get(off), get(off+1))
	}

	d := &CurrentData{RawRegisters: regs}

	d.TOutdoor = temp(OffTOutdoor)
	d.TIndoor = temp(OffTIndoor)
	d.TSupply = temp(OffTSupply)
	d.TReturn = temp(OffTReturn)
	d.TDHW = temp(OffTDHW)

	d.Status = get(OffStatus)

	d.DeviceTime = Clock{
		Year:   int(get(OffYear)),
		Month:  int(get(OffMonth)),
		Day:    int(get(OffDay)),
		Hour:   int(get(OffHour)),
		Minute: int(get(OffMinute)),
		Second: int(get(OffSecond)),
	}

	return d
}
