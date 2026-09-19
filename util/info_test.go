package util

import (
	"testing"
	// "srun-auth/util"
)

func TestInfo(t *testing.T) {
	tests := []struct {
		info  InfoData
		token string
		want  string
	}{
		{
			info: InfoData{
				Username: "111111@dx",
				Password: "12345",
				IP:       "10.0.1.1",
				ACID:     "3",
				EncVer:   "srun_bx1",
			},
			token: "7f31ebcccbc14318005e0ba573ce3e681f0ae52c20e156bf6520f209b68308b7",
			want:  "{SRBX1}sL2W5810S9ESa7V96wCXmUQc51xM7RhXGndKke3TCXKXQopHTetb5tmQEEkkjQAdswAB1fN3QYdt9FGSfmuXv/JU/3AjtkLdoaRNGzPw//Wz8I5GVS406tSQtMZLayVJ",
		},
	}

	for _, tt := range tests {
		result, err := Info(tt.info, tt.token)
		if err != nil {
			t.Fatal(err)
		}
		if result != tt.want {
			t.Errorf("Info = %s, want = %s", result, tt.want)
		}
	}
}
