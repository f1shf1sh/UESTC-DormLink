# srun-auto-login

电子科技大学 SRun 校园网认证工具，可在 Linux 路由器上运行。当前执行一次认证，自动重连尚未实现。

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
  "portal_ip": "1.1.1.1"
}
```

`username` 不带运营商后缀；`carrier` 按实际登录页面填写，例如电信 `dx`、移动 `cmcc`，无需后缀时填空字符串。入口无法跳转到门户时，把 `portal_ip` 换成浏览器实际登录页的完整 URL，保留查询参数。

在电脑上上传，将示例地址 `192.168.1.1` 替换为路由器地址：

```sh
ssh root@192.168.1.1 'mkdir -p /root/srun-auto-login'
ssh root@192.168.1.1 'cat > /root/srun-auto-login/srun-auto-login' < bin/srun-auto-login
ssh root@192.168.1.1 'cat > /root/srun-auto-login/config.json' < config.json
```

登录路由器后运行：

```sh
cd /root/srun-auto-login
chmod +x srun-auto-login
./srun-auto-login
```

程序读取当前目录的 `config.json`，因此运行前需要先 `cd`。认证失败时输出错误并返回退出码 1；宿舍实际认证仍需现场验证。
