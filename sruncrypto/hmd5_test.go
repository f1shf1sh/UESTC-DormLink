package sruncrypto

import (
	"testing"
)

func TestHMD5(t *testing.T) {
	tests := []struct {
		pwd      string
		salt     string
		want_len int
		want     string
	}{

		{
			"12345",
			"2fc722c79ad2f4e9ae06d77da62fe664561e4221c6e7f669c0717c8b1dedbd20",
			32,
			"d8205694714616ad5e8ca7510957df82"},
	}

	for _, tt := range tests {
		got := HMD5(tt.pwd, tt.salt)
		if tt.want != got || len(got) != tt.want_len {
			t.Errorf("HMD5 = %s, want = %s (length %d)", got, tt.want, tt.want_len)
		}
	}
}
