# categraf-win7-server2008

给 **Windows 7 / Server 2008 R2 / Server 2012** 用的 categraf 包，对接夜莺 9.1。官方新包在这些系统上起不来；能跑起来的旧包，元信息又不完整。本包把两件事一起补上。

## 下载

[Releases：categraf-v0.3.87-windows7-amd64.zip](https://github.com/glj2369/categraf-win7-server2008/releases/download/v0.3.87/categraf-v0.3.87-windows7-amd64.zip)

解压后得到 `categraf.exe` + `conf` + `scripts`。

**不支持**原版 Windows Server 2008（非 R2）。

## 旧版本的问题

夜莺上机器有心跳，但元信息基本不可用：

- PLATFORM / CPU / MEMORY / NETWORK / FILESYSTEM 经常整页「暂无数据」
- 中文 Windows 网卡解析失败后，网卡信息出不来，严重时整份元数据都不报
- 列表里 CPU 看起来像没采到（低负载会显示成 0.0%）
- AGENT 版本号过长，带着一串 hash

![元信息全空](docs/bugs/1.png)

![列表 CPU 显示为空](docs/bugs/2.png)

![网卡仍空、版本号过长](docs/bugs/3.png)

## 本版本解决了什么

- 能在 Win7 / 2008 R2 / 2012 上安装并运行
- 元信息容错：一块采集失败不再拖垮整页，平台 / CPU / 内存 / 文件系统能正常显示
- 中文 Windows 先切 UTF-8 再读网卡，NETWORK 能报出网卡和 IP
- AGENT 版本固定为短号 **`v0.3.87`**

个别未就绪盘（光驱、空卡）仍可能显示 NaN Bytes，夜莺前端把 `Unknown` 当数字乘，官方同样如此。

## 安装

先改 `conf/config.toml` 里的 `[[writers]]` 指向夜莺，再：

```text
categraf.exe --install
categraf.exe --start
```

已有旧版时，只换 exe，保留原 `conf`：

```text
categraf.exe --stop
:: 覆盖 categraf.exe
categraf.exe --start
```

也可用 `--stop` / `--remove` / `--status`。`[global]` / `[[writers]]` 可沿用现有夜莺地址，不要整份拷官方新版 conf。

## 源码

修改后的源码在 [`categraf/`](categraf/)，基于官方 [v0.3.87](https://github.com/flashcatcloud/categraf/tree/v0.3.87)。相对官方只动了这些：

- `go.mod`：语言版本改为 Go 1.20，才能在 Win7 / 2008 R2 上跑
- `heartbeat/network/network_windows.go`：中文 Windows 先 `chcp 65001` 再读网卡
- `inputs/mtail/internal/tailer/logstream/reader.go`：去掉 Go 1.21 才有的 `min()`，否则 1.20 编不过
- 版本号写成短的 `v0.3.87`

## 许可

上游 [categraf](https://github.com/flashcatcloud/categraf) 为 MIT，见 [LICENSE](LICENSE)。
