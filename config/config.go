package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config 保存登录相关配置
type Config struct {
	Username     string `json:"username"` // 学号/用户名
	Password     string `json:"password"` // 明文密码
	Carrier      string `json:"carrier"`  // 运营商 (电信 & 移动)
	ACID         string // Portal ac_id
	PortalIP     string `json:"portal_ip"`
	WlanACIP     string
	OnlineIP     string // DHCP分配的主机ip
	AuthURL      string // 登录请求 URL
	ChallengeURL string // 获取 challenge URL
	Token        string // 获取challenge
	UserAgent    string // 模拟浏览器 UA
}

// 默认值
var defaultConfig = &Config{
	PortalIP:     "172.25.249.64", // 固定门户 IP
	AuthURL:      "http://10.253.0.235/cgi-bin/srun_portal_pc",
	ChallengeURL: "http://10.253.0.235/cgi-bin/get_challenge",
	UserAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0 Safari/537.36",
}

// LoadConfig 从 JSON 文件加载配置
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	cfg := *defaultConfig // 先拷贝默认值，包括固定 PortalIP

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("解析 JSON 配置失败: %v", err)
	}

	return &cfg, nil
}
