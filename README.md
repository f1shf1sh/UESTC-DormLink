# srun-auto-login

电子科技大学 SRun 校园网认证工具，支持单次登录和掉线自动重连，可在 Linux 路由器上运行。

## 1. 确认路由器架构

在路由器上执行 `uname -m`，结合固件的架构说明选择编译参数。

| 路由器架构 | Go 编译参数 |
|---|---|
| ARM64 / aarch64 | `GOARCH=arm64` |
| ARMv7 / armv7l | `GOARCH=arm GOARM=7` |
| MIPS 大端 | `GOARCH=mips GOMIPS=softfloat` |
| MIPS 小端 | `GOARCH=mipsle GOMIPS=softfloat` |
| x86_64 | `GOARCH=amd64` |
| x86 32 位 | `GOARCH=386` |

MIPS 还需确认大小端，小端通常标为 `mipsel` 或 `mipsle`。ARMv7 参数不适用于所有 32 位 ARM 设备。参数说明见 [Go ARM](https://go.dev/wiki/GoArm) 和 [Go MIPS](https://go.dev/wiki/GoMips)。

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

将编译好的 `srun-auto-login` 和 `config.json` 上传到路由器上你选择的目录，进入该目录运行：

```sh
# 单次认证
chmod +x ./srun-auto-login
./srun-auto-login

# 持续监测，默认每轮检查完成后等待 30 秒
./srun-auto-login -watch
```

可用 `-interval 1m` 调整监测间隔，或用 `-config /路径/config.json` 指定配置文件；默认读取当前目录的 `config.json`。

仅在门户明确返回离线时重连；请求失败或状态未知会等待下一轮。用户名/密码错误、账号锁定或要求修改密码时暂停尝试，修改配置后重启程序。按 Ctrl+C 可停止。

程序以前台方式运行。后台运行、开机启动和进程托管由使用者自行配置。

已通过模拟掉线测试及上述六种架构的交叉编译；宿舍实际认证与路由器实机运行仍需现场验证。
