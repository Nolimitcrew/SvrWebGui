package modbus

import (
	"context"
	"fmt"
	"net"
	"time"
)

// TCPTransport — Modbus TCP (соединение через шлюз RS-485-to-Ethernet).
//
// В отличие от RTU здесь каждому кадру добавляется MBAP-заголовок (7 байт),
// а контрольная сумма CRC не используется — надёжность канала обеспечивает
// TCP. Для сетевого ввода-вывода достаточно средств стандартной библиотеки
// (net), поэтому «заглушкой» здесь считается лишь логика обмена, а не
// отсутствие зависимостей.
//
// ЗАГОТОВКА (stub): Send пока возвращает ошибку.
type TCPTransport struct {
	// Host — IP- или DNS-адрес шлюза, например "192.168.1.50".
	Host string
	// Port — порт шлюза (обычно 502).
	Port int
	// UnitID — адрес устройства «внутри» шлюза: 0, если шлюз сам
	// отвечает, иначе — конкретный ведомый (обычно 1..247).
	UnitID byte
	// Timeout — максимум времени на один обмен.
	Timeout time.Duration

	// conn — TCP-соединение, переиспользуемое между запросами.
	conn net.Conn
	// txID — счётчик транзакций, растёт с каждым запросом и позволяет
	// сопоставлять запросы с ответами.
	txID uint16
}

// NewTCPTransport создаёт транспорт к указанному шлюзу (порт по умолчанию 502).
func NewTCPTransport(host string, port int) *TCPTransport {
	if port <= 0 {
		port = 502
	}
	return &TCPTransport{
		Host:    host,
		Port:    port,
		Timeout: 2 * time.Second,
	}
}

// Send реализует интерфейс Transport для Modbus TCP.
func (t *TCPTransport) Send(ctx context.Context, adu []byte) ([]byte, error) {
	// TODO: реализовать:
	//
	//  1. Подключиться к Host:Port через net.Dialer с ctx; переиспользовать
	//     соединение или переподключаться при разрыве.
	//  2. Сформировать MBAP-заголовок (7 байт) + PDU из входного ADU:
	//         Transaction ID  — t.txID++; 2 байта big-endian
	//         Protocol ID     — 0x0000; 2 байта
	//         Length          — 1 (unit) + len(pdu); 2 байта
	//         Unit ID         — adu[0]; 1 байт
	//         PDU             — adu[1:]
	//  3. Записать MBAP+PDU, прочитать ответ.
	//  4. Из ответа извлечь MBAP: проверить совпадение Transaction ID и Length,
	//     затем вернуть [unit][PDU].
	//  5. Если функция в ответе имеет бит 0x80 — исключение Modbus.
	t.txID++
	return nil, fmt.Errorf("TCPTransport.Send: реализация пока не написана (TODO); шлюз %s:%d", t.Host, t.Port)
}

// Close закрывает TCP-соединение, если оно открыто.
func (t *TCPTransport) Close() error {
	if t.conn != nil {
		return t.conn.Close()
	}
	return nil
}
