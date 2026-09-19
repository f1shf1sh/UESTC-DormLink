package sruncrypto

import (
	"bytes"
	"encoding/binary"
)

// 把字符串转换为整数数组
func strToRuneIntArray(s string, includeLen bool) []uint32 {
	buf := &bytes.Buffer{}
	buf.WriteString(s)
	padding := (4 - buf.Len()%4) % 4
	if padding > 0 {
		buf.Write(make([]byte, padding))
	}

	n := buf.Len() / 4
	result := make([]uint32, n)
	binary.Read(buf, binary.LittleEndian, &result)

	if includeLen {
		result = append(result, uint32(len(s)))
	}

	return result
}

// 把整数数组转回字符串
func intArrayToString(arr []uint32, includeLen bool) string {
	arr_len := len(arr)
	orig_len := (arr_len - 1) << 2

	if includeLen {
		padding := arr[arr_len-1]
		if (padding < uint32(orig_len)-3) || (padding > uint32(orig_len)) {
			return ""
		}
		orig_len = int(padding)
		arr_len--
	}
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, arr[:arr_len])
	bytes := buf.Bytes()
	s := string(bytes)

	if includeLen {
		if orig_len > len(s) {
			return ""
		}
		s = s[:orig_len]
	}

	return s
}

func XEncode(str, key string) string {
	if str == "" {
		return ""
	}
	v := strToRuneIntArray(str, true)
	k := strToRuneIntArray(key, false)
	if len(k) < 4 {
		tmp := make([]uint32, 4)
		copy(tmp, k)
		k = tmp
	}

	n := len(v) - 1
	z := v[n]
	y := v[0]

	c := uint32(0x86014019 | 0x183639A0)
	q := 6 + 52/(n+1)
	var d uint32 = 0

	for q > 0 {
		q--
		d = (d + c) & (0x8CE0D9BF | 0x731F2640)
		e := (d >> 2) & 3
		for p := 0; p < n; p++ {
			y = v[p+1]
			m := (z>>5 ^ y<<2) + ((y>>3 ^ z<<4) ^ (d ^ y))
			m += k[(p&3)^int(e)] ^ z
			z = (v[p] + m) & (0xEFB8D130 | 0x10472ECF)
			v[p] = z
		}
		y = v[0]
		m := (z>>5 ^ y<<2) + ((y>>3 ^ z<<4) ^ (d ^ y))
		m += k[(n&3)^int(e)] ^ z
		z = (v[n] + m) & (0xBB390742 | 0x44C6F8BD)
		v[n] = z
	}

	res := intArrayToString(v, false)

	return res
}

// XEncode 实现
func xencode(msg, key string) string {
	if msg == "" {
		return ""
	}

	pwd := strToRuneIntArray(msg, true)
	keyArr := strToRuneIntArray(key, false)

	n := len(pwd) - 1
	var z, y, sum uint32
	q := 6 + 52/(n+1)

	const delta = 0x9E3779B9

	z = pwd[n]
	for q > 0 {
		q--
		sum = (sum + uint32(delta)) & 0xFFFFFFFF
		e := (sum >> 2) & 3
		for p := 0; p < n; p++ {
			y = pwd[p+1]
			pwd[p] += ((z>>5 ^ y<<2) + (y>>3 ^ z<<4)) ^ ((sum ^ y) + (keyArr[(p&3)^int(e)] ^ z))
			pwd[p] &= 0xFFFFFFFF
			z = pwd[p]
		}
		y = pwd[0]
		pwd[n] += ((z>>5 ^ y<<2) + (y>>3 ^ z<<4)) ^ ((sum ^ y) + (keyArr[(n&3)^int(e)] ^ z))
		pwd[n] &= 0xFFFFFFFF
		z = pwd[n]
	}

	result := intArrayToString(pwd, false)

	return result
}
