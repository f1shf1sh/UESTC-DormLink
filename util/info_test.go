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
			want:  "{SRBX1}Kp4SjaY9Xmyk5rLMyoMRZ2dKtoP2MZDEcGQXx3hvShV3WE3hi4ZMhfQEKwIOwu6aHj670LW+aDD1x6q5qHqBSI1YoKde8myPpZGyOsKmmMhQCVDMo81In2r2y+hX/8/C8EM10lvxmS4=",
		},
	}

	for _, tt := range tests {
		result, _ := Info(tt.info, tt.token)
		if result == tt.want {
			t.Logf("info(Infodata, token) = %s, want = %s", result, tt.want)
		}
	}
}
