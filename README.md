# srun-auto-login

电子科技大学 SRun 校园网认证工具，支持单次登录和掉线自动重连，可在 Linux 路由器上运行。

## 1. 确认路由器架构

在路由器上执行 `uname -m`。OpenWrt 还可查看 `cat /etc/openwrt_release` 中的 `DISTRIB_ARCH`。

| 路由器架构 | Go 编译参数 |
|---|---|
| ARM64 / aarch64 | `GOARCH=arm64` |
| ARMv7 / armv7l | `GOARCH=arm GOARM=7` |
| MIPS 大端 | `GOARCH=mips GOMIPS=softfloat` |
| MIPS 小端 | `GOARCH=mipsle GOMIPS=softfloat` |
| x86_64 | `GOARCH=amd64` |
| x86 32 位 | `GOARCH=386` |

MIPS 不能只看 `uname -m`：OpenWrt 的 `mipsel_*` 对应 `mipsle`，`mips_*` 对应 `mips`。ARMv7 参数不适用于所有 32 位 ARM 设备。参数说明见 [Go ARM](https://go.dev/wiki/GoArm) 和 [Go MIPS](https://go.dev/wiki/GoMips)。

## 2. 在电脑上编译

电脑安装 Go 1.25 或更高版本，路由器无需安装 Go。以下命令在 Linux、macOS 或 WSL 的项目根目录执行。

以 ARM64 为例；其他架构把 `GOARCH=arm64` 替换为上表对应的整组参数：

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
  go build -trimpath -ldflags="-s -w" -o bin/srun-auto-login .
```

例如 MIPS 小端：

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=mipsle GOMIPS=softfloat \
  go build -trimpath -ldflags="-s -w" -o bin/srun-auto-login .
```

生成文件为 `bin/srun-auto-login`。`CGO_ENABLED=0` 避免依赖路由器上的 C 动态库，`-s -w` 用于减小文件体积。

## 3. 配置并放到路由器运行

编辑项目根目录的 `config.json`：

```json
{
  "username": "你的学号",
  "password": "你的校园网密码",
  "carrier": "dx",
  "portal_ip": "http://10.253.0.235/srun_portal_pc?ac_id=3"
}
```

`username` 不带运营商后缀；`carrier` 按实际登录页面填写，例如电信 `dx`、移动 `cmcc`，无需后缀时填空字符串。上面的网关和 `ac_id` 是示例，请按宿舍实际门户填写。

自动重连需要实际门户地址，保留 `ac_id`，不要固定旧的 `wlanuserip`；不要用 `1.1.1.1` 等探测入口，因为已联网时它可能不再跳转到门户。

在电脑上上传，将示例地址 `192.168.1.1` 替换为路由器地址：

```sh
ssh root@192.168.1.1 'mkdir -p /etc/srun-auto-login'
ssh root@192.168.1.1 'cat > /usr/bin/srun-auto-login && chmod +x /usr/bin/srun-auto-login' < bin/srun-auto-login
ssh root@192.168.1.1 'cat > /etc/srun-auto-login/config.json' < config.json
```

登录路由器后运行：

```sh
# 单次认证
srun-auto-login -config /etc/srun-auto-login/config.json

# 持续监测，默认每轮检查完成后等待 30 秒
srun-auto-login -watch -config /etc/srun-auto-login/config.json
```

可用 `-interval 1m` 调整监测间隔。不指定 `-config` 时读取当前目录的 `config.json`。

仅在门户明确返回离线时重连；请求失败或状态未知会等待下一轮。用户名/密码错误、账号锁定或要求修改密码时暂停尝试，修改配置后重启程序。按 Ctrl+C 可停止。

## 4. OpenWrt 开机运行

在电脑上传启动脚本：

```sh
ssh root@192.168.1.1 'cat > /etc/init.d/srun-auto-login && chmod +x /etc/init.d/srun-auto-login' < openwrt/srun-auto-login
```

然后在路由器执行：

```sh
/etc/init.d/srun-auto-login enable
/etc/init.d/srun-auto-login start
logread -e srun-auto-login
```

修改配置后执行 `/etc/init.d/srun-auto-login restart`。停止并取消开机启动分别使用 `stop` 和 `disable`。脚本由 OpenWrt procd 托管，无需另外添加后台循环或 cron。

已通过模拟掉线测试及上述六种架构的交叉编译；宿舍实际认证与 OpenWrt 实机运行仍需现场验证。
