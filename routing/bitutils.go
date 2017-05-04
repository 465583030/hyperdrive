package routing

import (
	"errors"
)

const oneBitsMask uint = 0x55555555
const twoBitsMask uint = 0x33333333
const fourBitsMask uint = 0x0F0F0F0F
const eightBitsMask uint = 0x00FF00FF
const allBitsMask uint = 0xFFFFFFFF

// CTPop counts the number of 1 bits in a bitfield.
func CTPop(x uint) uint {
	x -= (x >> 1) & oneBitsMask
	x = (x & twoBitsMask) + ((x >> 2) & twoBitsMask)
	x = (x + (x >> 4)) & fourBitsMask
	x += x >> 8
	return (x + (x >> 16)) & 0x3F
}

// CTPopBelow counts the number os 1 bits below the given index.
func CTPopBelow(x uint, index uint) (uint, error) {
	if index > 31 || index < 0 {
		return 0, errors.New("Index out of bounds")
	}
	mask := allBitsMask >> (32 - index)
	return CTPop(x & mask), nil
}
