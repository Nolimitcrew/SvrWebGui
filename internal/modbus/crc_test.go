package modbus

import "testing"

// TestCRC16_KnownVector проверяет реализацию на эталонном значении из
// каталога CRC (Greg Cook): CRC-16/MODBUS от строки "123456789" = 0x4B37.
// Это «золотой образец», часто используемый для проверки таблиц и кода.
func TestCRC16_KnownVector(t *testing.T) {
	got := CRC16([]byte("123456789"))
	if got != 0x4B37 {
		t.Errorf("CRC16(\"123456789\") = 0x%04X, ожидалось 0x4B37", got)
	}
}

// TestCRC16_Empty проверяет, что для пустых данных возвращается начальное
// значение 0xFFFF (регистр CRC не меняется, если байтов нет).
func TestCRC16_Empty(t *testing.T) {
	if got := CRC16(nil); got != 0xFFFF {
		t.Errorf("CRC16(nil) = 0x%04X, ожидалось 0xFFFF", got)
	}
}
