package utils

import (
	"crypto/rand"
	"encoding/binary"
)

func SafeRand() int64 {
	var safeRndVal int64
	err := binary.Read(rand.Reader, binary.LittleEndian, &safeRndVal)
	if err != nil {
		LavaFormatInfo(err.Error(), nil)
	}
	safeRndVal &= 0x7fffffffffffffff // set the highest bit to 0 to get a positive number
	for safeRndVal == 0 {            // we don't allow 0
		err = binary.Read(rand.Reader, binary.LittleEndian, &safeRndVal)
		if err != nil {
			LavaFormatInfo(err.Error(), nil)
		}
		safeRndVal &= 0x7fffffffffffffff // set the highest bit to 0 to get a positive number
	}
	return safeRndVal
}
