package ttr01

import (
	"fmt"
	"math"
)

// Clock — время и дата, прочитанные из регистров прибора.
type Clock struct {
	Year   int
	Month  int
	Day    int
	Hour   int
	Minute int
	Second int
}

// String форматирует время в виде «2006-01-02 15:04:05».
func (c Clock) String() string {
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d",
		c.Year, c.Month, c.Day, c.Hour, c.Minute, c.Second)
}

// CurrentData — распарсенные «текущие данные» контроллера TTR-01
// (область регистров, начиная с 43101).
type CurrentData struct {
	// Температуры, °C. Каждое значение занимает 2 регистра
	// (вещественное число одинарной точности IEEE 754).
	TOutdoor float64
	TIndoor  float64
	TSupply  float64
	TReturn  float64
	TDHW     float64

	// Status — битовая маска флагов состояния (см. Status*).
	Status uint16

	// DeviceTime — время прибора.
	DeviceTime Clock

	// RawRegisters — «сырые» регистры ответа (удобно для отладки).
	RawRegisters []uint16
}

// Has проверяет, установлены ли все биты mask в статусе прибора.
// Удобен для проверки отдельных флагов, например d.Has(StatusHeating).
func (d *CurrentData) Has(mask uint16) bool {
	return d.Status&mask == mask
}

// tempFromRegisters преобразует два регистра в температуру.
//
// Предполагаемый формат — вещественное число одинарной точности IEEE 754,
// занимающее 2 регистра: старший регистр — первым (big-endian порядок слов).
// Формат будет уточнён по документации TTR-01.
func tempFromRegisters(hi, lo uint16) float64 {
	bits := uint32(hi)<<16 | uint32(lo)
	return float64(math.Float32frombits(bits))
}

// TempToRegisters упаковывает температуру в два регистра в формате прибора.
// Пригодится для заполнения MockTransport в тестах и отладке.
func TempToRegisters(temp float64) (uint16, uint16) {
	bits := math.Float32bits(float32(temp))
	return uint16(bits >> 16), uint16(bits & 0xFFFF)
}
