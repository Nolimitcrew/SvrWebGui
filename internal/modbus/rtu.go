package modbus

import (
	"context"
	"fmt"
	"time"
)

// RTUTransport — Modbus RTU поверх последовательного порта (COM-порт,
// USB-RS485 адаптер).
//
// ЗАГОТОВКА (stub): реальная работа с портом здесь пока не реализована —
// для неё понадобится сторонняя библиотека последовательного порта (например,
// go.bug.st/serial или github.com/tarm/serial), которую мы не подключаем
// на этом этапе. Метод Send возвращает ошибку, чтобы не давать ложного
// ощущения работы.
//
// Когда библиотека появится, в этом файле появится полноценная реализация.
// Интерфейс Transport и драйвер ttr01 при этом меняться не будут — достаточно
// написать Send и Close.
type RTUTransport struct {
	// Port — имя COM-порта в системе, например "COM3".
	Port string
	// BaudRate — скорость обмена в бит/с (у TTR-01 обычно 9600).
	BaudRate int
	// DataBits — число битов данных (обычно 8).
	DataBits int
	// Parity — чётность: "N" — нет, "E" — чёт, "O" — нечёт.
	Parity string
	// StopBits — число стоп-битов (1 или 2).
	StopBits int
	// Timeout — максимум времени ожидания ответа прибора.
	Timeout time.Duration

	// port — дескриптор последовательного порта из внешней библиотеки.
	// Остаётся nil до первого обращения (ленивая инициализация).
	// TODO: заменить тип any на реальный serial.Port, как только
	//       появится выбранная библиотека.
	port any
}

// NewRTUTransport создаёт транспорт с типовыми настройками для TTR-01.
// Параметры подключения можно переопределить через поля структуры.
func NewRTUTransport(port string, baudRate int) *RTUTransport {
	return &RTUTransport{
		Port:     port,
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   "N",
		StopBits: 1,
		Timeout:  1 * time.Second,
	}
}

// Send реализует интерфейс Transport для Modbus RTU.
func (t *RTUTransport) Send(ctx context.Context, adu []byte) ([]byte, error) {
	// TODO: реализовать работу с COM-портом:
	//
	//  1. Открыть порт с настройками t.Port/t.BaudRate/... (лениво,
	//     при первом обращении; переиспользовать открытый порт).
	//  2. Собрать кадр RTU: запрос + CRC-16 младшим байтом вперёд:
	//         crc := CRC16(adu)
	//         frame := append(append([]byte{}, adu...), byte(crc), byte(crc>>8))
	//  3. Очистить входной буфер порта и записать frame.
	//  4. Читать ответ до t.Timeout или отмены ctx.
	//  5. Проверить: длина >= 4, совпадение CRC (контрольная сумма в конце
	//     кадра), адрес прибора в ответе совпадает с запрошенным.
	//  6. Отбросить CRC и вернуть [адрес][PDU].
	//  7. Если в ответе у функции установлен бит 0x80 — вернуть ошибку с
	//     описанием кода исключения Modbus (см. modbus.ExceptionText).
	return nil, fmt.Errorf("RTUTransport.Send: реализация требует сторонней библиотеки для работы с COM-портом (см. TODO)")
}

// Close закрывает COM-порт, если он был открыт.
func (t *RTUTransport) Close() error {
	// TODO: if t.port != nil { t.port.Close() }
	return nil
}
