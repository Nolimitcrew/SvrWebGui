package ttr01

import (
	"context"
	"encoding/binary"
	"fmt"

	"SvrWebGui/internal/modbus"
)

// Driver вЂ” РґСЂР°Р№РІРµСЂ РєРѕРЅС‚СЂРѕР»Р»РµСЂР° TTR-01 (РѕС‚РѕРїР»РµРЅРёРµ + Р“Р’РЎ).
//
// Р”СЂР°Р№РІРµСЂ РїРѕСЃС‚СЂРѕРµРЅ РїРѕРІРµСЂС… Р°Р±СЃС‚СЂР°РєС†РёРё modbus.Transport, Р±Р»Р°РіРѕРґР°СЂСЏ С‡РµРјСѓ
// РѕРґРёРЅ Рё С‚РѕС‚ Р¶Рµ РєРѕРґ СЂР°Р±РѕС‚Р°РµС‚ СЃ Serial RTU, TCP Рё РґР°Р¶Рµ СЃ Р·Р°РіР»СѓС€РєРѕР№
// (MockTransport) вЂ” РґРѕСЃС‚Р°С‚РѕС‡РЅРѕ РїРѕРґРјРµРЅРёС‚СЊ РѕР±СЉРµРєС‚ С‚СЂР°РЅСЃРїРѕСЂС‚Р°.
type Driver struct {
	transport modbus.Transport // РєР°РЅР°Р» СЃРІСЏР·Рё
	unitID    byte             // Р°РґСЂРµСЃ РїСЂРёР±РѕСЂР° РІ СЃРµС‚Рё Modbus (1..247)
}

// NewDriver СЃРѕР·РґР°С‘С‚ РґСЂР°Р№РІРµСЂ РґР»СЏ РїСЂРёР±РѕСЂР° СЃ Р°РґСЂРµСЃРѕРј unitID, РёСЃРїРѕР»СЊР·СѓСЋС‰РµРіРѕ
// РїРµСЂРµРґР°РЅРЅС‹Р№ С‚СЂР°РЅСЃРїРѕСЂС‚ (Serial RTU, TCP РёР»Рё Р·Р°РіР»СѓС€РєСѓ).
func NewDriver(transport modbus.Transport, unitID byte) *Driver {
	return &Driver{transport: transport, unitID: unitID}
}

// ReadCurrentData С‡РёС‚Р°РµС‚ РѕР±Р»Р°СЃС‚СЊ В«С‚РµРєСѓС‰РёРµ РґР°РЅРЅС‹РµВ» РїСЂРёР±РѕСЂР°.
//
// Р—Р°РїСЂРѕСЃ: С„СѓРЅРєС†РёСЏ 0x03, РЅР°С‡Р°Р»СЊРЅС‹Р№ Р°РґСЂРµСЃ 43101 (0x0C1C), RegCurrentDataCount
// СЂРµРіРёСЃС‚СЂРѕРІ. РћС‚РІРµС‚ СЂР°Р·Р±РёСЂР°РµС‚СЃСЏ РІ СЃС‚СЂСѓРєС‚СѓСЂСѓ CurrentData.
func (d *Driver) ReadCurrentData(ctx context.Context) (*CurrentData, error) {
	// 1. Р¤РѕСЂРјРёСЂСѓРµРј PDU: [0x03][СЃС‚Р°СЂС€РёР№ Р°РґСЂРµСЃ][РјР»Р°РґС€РёР№][СЃС‚Р°СЂС€РµРµ РєРѕР»-РІРѕ][РјР»Р°РґС€РµРµ].
	pdu := make([]byte, 5)
	pdu[0] = modbus.FuncReadHoldingRegisters
	binary.BigEndian.PutUint16(pdu[1:3], RegCurrentDataStart)
	binary.BigEndian.PutUint16(pdu[3:5], RegCurrentDataCount)

	// 2. РџРѕР»РЅС‹Р№ РєР°РґСЂ ADU = [Р°РґСЂРµСЃ РїСЂРёР±РѕСЂР°][PDU].
	adu := make([]byte, 0, 1+len(pdu))
	adu = append(adu, d.unitID)
	adu = append(adu, pdu...)

	// 3. РћС‚РїСЂР°РІР»СЏРµРј Р·Р°РїСЂРѕСЃ Рё РїРѕР»СѓС‡Р°РµРј РѕС‚РІРµС‚РЅС‹Р№ РєР°РґСЂ ADU.
	resp, err := d.transport.Send(ctx, adu)
	if err != nil {
		return nil, fmt.Errorf("ttr01: РѕР±РјРµРЅ Modbus РЅРµ СѓРґР°Р»СЃСЏ: %w", err)
	}

	// 4. Р Р°Р·Р±РёСЂР°РµРј РѕС‚РІРµС‚.
	return parseReadHoldingResponse(d.unitID, RegCurrentDataCount, resp)
}

// parseReadHoldingResponse РїСЂРѕРІРµСЂСЏРµС‚ Рё СЂР°Р·Р±РёСЂР°РµС‚ РѕС‚РІРµС‚ С„СѓРЅРєС†РёРё 0x03.
func parseReadHoldingResponse(unit byte, expectedCount uint16, resp []byte) (*CurrentData, error) {
	// РњРёРЅРёРјР°Р»СЊРЅР°СЏ РґР»РёРЅР° РѕС‚РІРµС‚Р°: Р°РґСЂРµСЃ(1) + С„СѓРЅРєС†РёСЏ(1) + СЃС‡С‘С‚С‡РёРє Р±Р°Р№С‚(1)
	// + 2*expectedCount Р±Р°Р№С‚ РґР°РЅРЅС‹С….
	minLen := 3 + int(2*expectedCount)
	if len(resp) < minLen {
		return nil, fmt.Errorf("ttr01: РѕС‚РІРµС‚ СЃР»РёС€РєРѕРј РєРѕСЂРѕС‚РєРёР№: %d Р±Р°Р№С‚, РѕР¶РёРґР°Р»РѕСЃСЊ в‰Ґ %d", len(resp), minLen)
	}

	// РђРґСЂРµСЃ СѓСЃС‚СЂРѕР№СЃС‚РІР° РІ РѕС‚РІРµС‚Рµ РґРѕР»Р¶РµРЅ СЃРѕРІРїР°РґР°С‚СЊ СЃ Р·Р°РїСЂРѕС€РµРЅРЅС‹Рј.
	if resp[0] != unit {
		return nil, fmt.Errorf("ttr01: РІ РѕС‚РІРµС‚Рµ Р°РґСЂРµСЃ %d, РѕР¶РёРґР°Р»СЃСЏ %d", resp[0], unit)
	}

	// Р•СЃР»Рё СЃС‚Р°СЂС€РёР№ Р±РёС‚ С„СѓРЅРєС†РёРё СѓСЃС‚Р°РЅРѕРІР»РµРЅ вЂ” РїСЂРёР±РѕСЂ РІРµСЂРЅСѓР» РёСЃРєР»СЋС‡РµРЅРёРµ.
	if resp[1]&0x80 != 0 {
		code := modbus.ExceptionIllegalFunction // В«Р·Р°РїР°СЃРЅРѕР№В» РєРѕРґ, РµСЃР»Рё РґР»РёРЅР° РѕС‚РІРµС‚Р° РјР°Р»Р°
		if len(resp) > 2 {
			code = resp[2]
		}
		return nil, fmt.Errorf("ttr01: РїСЂРёР±РѕСЂ РІРµСЂРЅСѓР» РёСЃРєР»СЋС‡РµРЅРёРµ Modbus: %s", modbus.ExceptionText(code))
	}

	// Р¤СѓРЅРєС†РёСЏ РІ РѕС‚РІРµС‚Рµ РґРѕР»Р¶РЅР° СЃРѕРІРїР°РґР°С‚СЊ СЃ Р·Р°РїСЂРѕСЃРѕРј.
	if resp[1] != modbus.FuncReadHoldingRegisters {
		return nil, fmt.Errorf("ttr01: РЅРµРѕР¶РёРґР°РЅРЅС‹Р№ РєРѕРґ С„СѓРЅРєС†РёРё РІ РѕС‚РІРµС‚Рµ: 0x%02X", resp[1])
	}

	// РЎС‡С‘С‚С‡РёРє Р±Р°Р№С‚ РґР°РЅРЅС‹С… (РєР°Р¶РґС‹Р№ СЂРµРіРёСЃС‚СЂ вЂ” 2 Р±Р°Р№С‚Р°).
	byteCount := int(resp[2])
	if byteCount != int(2*expectedCount) {
		return nil, fmt.Errorf("ttr01: СЃС‡С‘С‚С‡РёРє Р±Р°Р№С‚ %d в‰  РѕР¶РёРґР°РµРјРѕРјСѓ %d", byteCount, 2*expectedCount)
	}

	// РЎС‡РёС‚С‹РІР°РµРј СЂРµРіРёСЃС‚СЂС‹ big-endian.
	regs := make([]uint16, expectedCount)
	for i := uint16(0); i < expectedCount; i++ {
		regs[i] = binary.BigEndian.Uint16(resp[3+2*i:])
	}
	if len(regs) < CurrentDataMinRegisters {
		return nil, fmt.Errorf("ttr01: РЅРµ С…РІР°С‚Р°РµС‚ СЂРµРіРёСЃС‚СЂРѕРІ РґР»СЏ СЂР°Р·Р±РѕСЂР°: %d < %d", len(regs), CurrentDataMinRegisters)
	}

	return buildCurrentData(regs), nil
}

// buildCurrentData РїСЂРµРІСЂР°С‰Р°РµС‚ СЂРµРіРёСЃС‚СЂС‹ РІ РёРјРµРЅРѕРІР°РЅРЅСѓСЋ СЃС‚СЂСѓРєС‚СѓСЂСѓ.
// РўРѕС‡РЅР°СЏ РєР°СЂС‚Р° СЂРµРіРёСЃС‚СЂРѕРІ Р·Р°РґР°С‘С‚СЃСЏ РєРѕРЅСЃС‚Р°РЅС‚Р°РјРё Off* РІ registers.go.
func buildCurrentData(regs []uint16) *CurrentData {
	// get РІРѕР·РІСЂР°С‰Р°РµС‚ СЂРµРіРёСЃС‚СЂ РїРѕ СЃРјРµС‰РµРЅРёСЋ РёР»Рё 0, РµСЃР»Рё РІС‹С…РѕРґ Р·Р° РіСЂР°РЅРёС†С‹.
	get := func(off uint16) uint16 {
		if int(off) < len(regs) {
			return regs[off]
		}
		return 0
	}

	// temp С‡РёС‚Р°РµС‚ С‚РµРјРїРµСЂР°С‚СѓСЂСѓ РёР· РїР°СЂС‹ СЂРµРіРёСЃС‚СЂРѕРІ [off, off+1].
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
