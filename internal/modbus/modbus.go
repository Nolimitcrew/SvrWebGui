// Package modbus реализует низкоуровневый транспортный слой протокола Modbus.
//
// Пакет ничего не знает о конкретных приборах (TTR-01, насосы Grundfos и
// т.п.): его задача — отправить кадр ADU в канал связи и получить ответ.
// Смысл данных (температуры, флаги, уставки) разбирается уровнем выше —
// в пакетах приборов (internal/ttr01 и т.п.).
package modbus

import (
	"context"
	"fmt"
)

// Коды функций Modbus, которые пакет работает с напрямую.
const (
	// FuncReadHoldingRegisters — функция 0x03: чтение группы
	// holding-регистров.
	FuncReadHoldingRegisters byte = 0x03
)

// Коды исключений Modbus. Возвращаются прибором в ответе с установленным
// в коде функции старшим битом (0x80).
const (
	// ExceptionIllegalFunction — запрошенная функция не поддерживается.
	ExceptionIllegalFunction byte = 0x01
	// ExceptionIllegalDataAddress — адрес регистра вне допустимого диапазона.
	ExceptionIllegalDataAddress byte = 0x02
	// ExceptionIllegalDataValue — значение параметра недопустимо.
	ExceptionIllegalDataValue byte = 0x03
	// ExceptionSlaveDeviceFailure — сбой в самом приборе.
	ExceptionSlaveDeviceFailure byte = 0x04
)

// Transport — абстракция физического канала связи с прибором.
//
// Один и тот же драйвер прибора работает с любой реализацией транспорта,
// поэтому вся специфика протокола связи (CRC, MBAP-заголовок, параметры
// COM-порта) спрятана внутри реализаций.
//
// ADU (Application Data Unit — кадр прикладного уровня) в рамках этого
// интерфейса имеет вид:
//
//	[адрес устройства, 1 байт][PDU: код функции + данные]
//
// Канальную часть кадра реализация транспорта добавляет и снимает
// самостоятельно:
//
//   - Modbus RTU  — CRC-16 в конце кадра;
//   - Modbus TCP  — MBAP-заголовок в начале кадра.
//
// Ответный кадр имеет ту же структуру: [адрес устройства][PDU ответа].
type Transport interface {
	// Send отправляет кадр ADU и возвращает ответный кадр ADU.
	//
	// Реализация обязана вернуть ошибку, если прибор не ответил, ответил
	// некорректно (битая контрольная сумма, исключение Modbus) либо истёк
	// таймаут. Параметр ctx позволяет отменить обмен по внешнему таймауту.
	Send(ctx context.Context, adu []byte) ([]byte, error)

	// Close закрывает канал связи и освобождает ресурсы
	// (закрывает COM-порт или TCP-соединение).
	Close() error
}

// ExceptionText возвращает русскоязычное описание кода исключения Modbus.
// Если код неизвестен — возвращает строку "неизвестное исключение 0xNN".
func ExceptionText(code byte) string {
	switch code {
	case ExceptionIllegalFunction:
		return "недопустимая функция (01)"
	case ExceptionIllegalDataAddress:
		return "недопустимый адрес данных (02)"
	case ExceptionIllegalDataValue:
		return "недопустимое значение данных (03)"
	case ExceptionSlaveDeviceFailure:
		return "сбой в устройстве-ведомом (04)"
	default:
		return fmt.Sprintf("неизвестное исключение 0x%02X", code)
	}
}
