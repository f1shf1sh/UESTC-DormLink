package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	StatusURL    string // 已发现门户的在线状态接口
	Token        string // 获取challenge
	UserAgent    string // 模拟浏览器 UA
}

// 默认值
var defaultConfig = &Config{
	PortalIP:     "10.253.0.235", // 宿舍 SRun 门户；实际入口优先使用 config.json
	AuthURL:      "http://10.253.0.235/cgi-bin/srun_portal",
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
	if strings.TrimSpace(cfg.Username) == "" || cfg.Password == "" {
		return nil, fmt.Errorf("请在 config.json 中填写校园网 username 和 password")
	}
	if strings.TrimSpace(cfg.PortalIP) == "" {
		return nil, fmt.Errorf("portal_ip 不能为空")
	}

	return &cfg, nil
}

// LoginUsername 返回协议各个阶段共同使用的完整用户名。
func (cfg *Config) LoginUsername() string {
	if cfg.Carrier == "" {
		return cfg.Username
	}
	return cfg.Username + "@" + cfg.Carrier
}
