package main

const (
    iBm = 0xA001
)

type tABLe [256]uint16

var ibmTable = makeTable(iBm)

func makeTable(poly uint16) *tABLe {
    t := new(tABLe)
    for i := 0; i < 256; i++ {
        crc := uint16(i)
        for j := 0; j < 8; j++ {
            if crc & 1 == 1 {
                crc = (crc >> 1) ^ poly
            } else {
                crc >>= 1
            }
        }
        t[i] = crc
    }
    return t
}

func update(crc uint16, tab *tABLe, p []byte) uint16 {
    crc = ^crc
    for _, v := range p {
        crc = tab[byte(crc) ^ v] ^ (crc >> 8)
    }
    return ^crc
}

func CRC16(data []byte) uint16 {
    return update(0, ibmTable, data)
}

