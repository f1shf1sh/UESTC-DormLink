package main

import (
	"fmt"
	"srun-auth/config"
	"srun-auth/portal"
)

func main() {
	cfg, err := config.LoadConfig("./config.json")
	if err != nil {
		fmt.Println("加载配置失败: ", err)
		return
	}
	fmt.Println("[+] 配置加载成功")
	fmt.Printf("Username: %s\nPassword: %s\n运营商: %s\n", cfg.Username, cfg.Password, cfg.Carrier)

	err = portal.Redirect(cfg)
	if err != nil {
		fmt.Println("[-] 重定向失败: ", err)
		return
	}
	fmt.Println("[+] 重定向成功,开始认证")

	err = portal.GetChallenge(cfg)
	if err != nil {
		fmt.Println("[-] get_challenge获取token失败: ", err)
		return
	}

	fmt.Println("[+] token获取成功")
	fmt.Println("token:", cfg.Token)

	err = portal.Auth(cfg)
	if err != nil {
		fmt.Println("[-] 认证失败")
		return
	}
	fmt.Println("[+] 认证成功")
}
