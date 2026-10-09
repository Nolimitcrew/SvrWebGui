package modbus

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
)

// MockTransport — эмулятор Modbus-устройства для разработки и тестов
// без реального железа.
//
// Умеет отвечать на запросы чтения holding-регистров (функция 0x03),
// отдавая значения из собственного хранилища. Все переданные и полученные
// кадры печатаются в консоль — так видно, что именно «уходит в провод».
//
// Эмулятор не знает ничего о конкретном приборе: он просто хранит регистры
// по их адресу. Смысловое наполнение регистров задаёт вызывающий (тесты
// драйвера ttr01 или cmd/heatnode при отладке).
type MockTransport struct {
	// Verbose — печатать ли кадры в консоль (по умолчанию включено).
	Verbose bool

	mu        sync.Mutex        // защищает registers от одновременного доступа
	registers map[uint16]uint16 // адрес регистра → значение
}

// NewMockTransport создаёт пустой эмулятор с включённым выводом кадров.
func NewMockTransport() *MockTransport {
	return &MockTransport{
		Verbose:   true,
		registers: make(map[uint16]uint16),
	}
}

// SetRegister задаёт значение одного holding-регистра эмулятора.
func (m *MockTransport) SetRegister(addr, value uint16) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registers[addr] = value
}

// SetRegisters задаёт значения нескольких регистров подряд,
// начиная с addr. Регистры, отсутствующие в хранилище, при чтении
// возвращаются как 0.
func (m *MockTransport) SetRegisters(addr uint16, values ...uint16) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, v := range values {
		m.registers[addr+uint16(i)] = v
	}
}

// Send реализует интерфейс Transport: разбирает запрос и отвечает,
// как ответил бы настоящий прибор.
func (m *MockTransport) Send(ctx context.Context, adu []byte) ([]byte, error) {
	m.log("Запрос  →", adu)

	// Уважаем внешний контекст: если он уже отменён, не «общаемся» с прибором.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(adu) < 2 {
		return nil, fmt.Errorf("mock: кадр слишком короткий (%d байт)", len(adu))
	}

	unit := adu[0] // адрес запрошенного устройства
	fc := adu[1]   // код функции

	var resp []byte
	switch fc {
	case FuncReadHoldingRegisters: // 0x03
		resp = m.handleReadHolding(unit, adu)
	default:
		// На все остальные функции отвечаем исключением Modbus 0x01
		// «недопустимая функция».
		resp = exceptionResponse(unit, fc, ExceptionIllegalFunction)
	}

	m.log("Ответ   ←", resp)
	return resp, nil
}

// Close у эмулятора нет открытых дескрипторов, но метод нужен для
// удовлетворения интерфейсу Transport.
func (m *MockTransport) Close() error {
	return nil
}

// handleReadHolding обрабатывает запрос функции 0x03.
//
// Запрос:  [адрес][0x03][старший байт адреса регистра][младший][кол-во старший][кол-во младший]
// Ответ:   [адрес][0x03][счётчик байт][значения регистров, 2 байта каждый]
func (m *MockTransport) handleReadHolding(unit byte, adu []byte) []byte {
	// Минимум: адрес(1) + функция(1) + адрес регистра(2) + количество(2) = 6 байт.
	if len(adu) < 6 {
		return exceptionResponse(unit, FuncReadHoldingRegisters, ExceptionIllegalDataValue)
	}

	addr := binary.BigEndian.Uint16(adu[2:4])
	count := binary.BigEndian.Uint16(adu[4:6])

	// В одном запросе Modbus допускает чтение от 1 до 125 регистров.
	if count == 0 || count > 125 {
		return exceptionResponse(unit, FuncReadHoldingRegisters, ExceptionIllegalDataValue)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	resp := make([]byte, 0, 5+2*int(count))
	resp = append(resp, unit, FuncReadHoldingRegisters, byte(2*count))
	for i := uint16(0); i < count; i++ {
		v := m.registers[addr+i] // пропущенные регистры читаются как 0
		resp = append(resp, byte(v>>8), byte(v))
	}
	return resp
}

// log печатает кадр в виде строки из шестнадцатеричных пар:
// "01 03 0C 1C 00 20".
func (m *MockTransport) log(action string, frame []byte) {
	if !m.Verbose {
		return
	}
	var b strings.Builder
	for i, v := range frame {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%02X", v)
	}
	fmt.Printf("[mock] %s %s\n", action, b.String())
}

// exceptionResponse формирует кадр ответа-исключения:
// [адрес][функция | 0x80][код исключения].
func exceptionResponse(unit, fc, code byte) []byte {
	return []byte{unit, fc | 0x80, code}
}
