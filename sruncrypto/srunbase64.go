package sruncrypto

const alpha = "LVoJPiCN2R8G90yg+hmFHuacZ1OWMnrsSTXkYpUq/3dlbfKwv6xztjI7DeBE45QA"
const padChar = '='

func stringToBytes(s string) []byte {
	bytes := make([]byte, len(s))
	for i, r := range s {
		bytes[i] = byte(r & 0xFF)
	}
	return bytes
}

// 自定义 Base64 编码
func Base64Encode(s string) string {
	// data := stringToBytes(s)
	data := []byte(s)
	imax := len(data) - len(data)%3

	x := []rune{}
	for i := 0; i < imax; i += 3 {
		b10 := (int(data[i]) << 16) | (int(data[i+1]) << 8) | int(data[i+2])
		x = append(x,
			rune(alpha[b10>>18]),
			rune(alpha[(b10>>12)&63]),
			rune(alpha[(b10>>6)&63]),
			rune(alpha[b10&63]),
		)
	}
	switch len(data) - imax {
	case 1:
		b10 := int(data[imax]) << 16
		x = append(x,
			rune(alpha[b10>>18]),
			rune(alpha[(b10>>12)&63]),
			padChar,
			padChar,
		)
	case 2:
		b10 := (int(data[imax]) << 16) | (int(data[imax+1]) << 8)
		x = append(x,
			rune(alpha[b10>>18]),
			rune(alpha[(b10>>12)&63]),
			rune(alpha[(b10>>6)&63]),
			padChar,
		)
	}
	return string(x)
}
