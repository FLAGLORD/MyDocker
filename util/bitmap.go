package util

type Bitmap struct {
	Bits   []byte
	Length uint64
}

func NewBitmap(n uint64) *Bitmap {
	return &Bitmap{
		Bits:   make([]byte, (n+7)/8),
		Length: n,
	}
}

func (bitmap *Bitmap) GetAvailableAndSet() (uint64, bool) {
	for i := 0; i < int(bitmap.Length); i++ {
		bitIndex, bitPos := i/8, i%8
		if bitmap.Bits[bitIndex]&(1<<bitPos) == 0 {
			bitmap.Bits[bitIndex] |= 1 << bitPos
			return uint64(i), true
		}
	}
	return 0, false
}

func (bitmap *Bitmap) Remove(num uint64) {
	bitIndex, bitPos := num/8, num%8
	bitmap.Bits[bitIndex] |= 1 << bitPos
	bitmap.Bits[bitIndex] ^= 1 << bitPos
}
