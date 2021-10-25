package util

import (
	"fmt"
	"testing"
)

var testMap = make(map[string]*Bitmap)

func TestMarshal(t *testing.T) {
	// testMap["key1"] = NewBitmap(256)
	// testMap["key1"].Bits[0] |= 1
	// jsonContent, _ := json.Marshal(&testMap)
	// fmt.Println(string(jsonContent))
	// var testMap2 map[string]*Bitmap
	// json.Unmarshal(jsonContent, &testMap2)
	// fmt.Println(testMap2["key1"])
	// testMap2["key1"].GetAvailableAndSet()
	// testMap2["key1"].GetAvailableAndSet()
	// fmt.Println(testMap2["key1"])
	a := []byte{1, 2, 3, 4}
	b := []byte{1, 2, 3, 4}
	for i := 3; i >= 0; i-- {
		fmt.Println(int(a[i] - b[i]) << ((3 - i) * 8))
	}
}
