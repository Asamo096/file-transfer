# File Transfer - beta

简单文件传输工具，集成了 STUN 穿透、DHT 网络发现和二维码连接功能，无需外部服务器中继（beta）。

## 特点

- 🌐 **跨网段传输**：支持 STUN NAT 穿透，突破内网限制
- 🔗 **P2P 直连**：设备间直接传输，不依赖外网服务器
- 📱 **扫码连接**：生成二维码，扫码即可建立连接
- 🔐 **端到端加密**：支持 AES-256-GCM 加密传输
- 📂 **大文件支持**：支持断点续传，传输状态持久化
- 📋 **文件管理**：上传、下载、删除、重命名、在线预览
- ⚡ **高性能传输**：使用 Go 语言和 Gin 框架
- 🌐 **多地址显示**：自动检测并显示 IPv6、局域网、公网地址
- 🌐 **全平台支持**：Windows、Linux、macOS

## 快速开始

### 下载和运行

直接下载对应平台的二进制文件，或从源码编译。

### 基本命令

```bash
# 启动服务（两者模式，同时支持发送和接收）
./file-transfer

# 发送方模式 - 分享文件给其他设备
./file-transfer sender

# 接收方模式 - 接收他人文件
./file-transfer receiver

# 指定端口和共享目录
./file-transfer -p 8888 -d /path/to/share

# 启用加密模式
./file-transfer -e

# 仅接收模式
./file-transfer -r
```

### 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-p` | 服务端口 | 8888 |
| `-d` | 共享/输出目录 | 当前目录 |
| `-e` | 启用端到端加密 | false |
| `-r` | 仅接收模式 | false |
| `-h` | 显示帮助 | - |

### 角色模式

| 角色 | 说明 |
|------|------|
| `sender` | 发送方模式 - 分享文件给其他设备 |
| `receiver` | 接收方模式 - 接收他人文件 |
| `both` | 两者模式 - 同时支持发送和接收（默认） |

## 工作原理

### 网络发现

1. **UDP 广播发现**：同一局域网内的设备自动发现
2. **DHT 网络**：通过分布式哈希表实现跨网段设备发现
3. **STUN 穿透**：获取公网 IP 和端口，实现 NAT 穿透

### 连接流程

1. 设备加入 DHT 网络，获取 6 位配对码
2. 发送方分享配对码给接收方
3. 接收方通过配对码查询发送方地址
4. 双方建立 P2P 连接，开始传输

### 二维码地址

程序启动时自动检测并显示：
- **IPv6 地址**：支持 IPv6 的网络
- **局域网地址**：本地网络地址
- **STUN 公网地址**：通过 STUN 服务器获取的公网地址

每个地址都会生成独立的 ASCII 二维码，方便不同网络环境下的设备扫描连接。

## 使用说明

### 1. 启动服务

运行程序，选择角色模式，终端将显示：
- 服务器信息
- 所有可访问地址及 ASCII 二维码
- 访问 URL

### 2. 连接方式

- **扫描二维码**：用手机扫描终端显示的二维码
- **直接访问**：在浏览器中输入显示的 URL
- **配对码连接**：通过 6 位配对码建立 P2P 连接

### 3. 文件操作

- **上传文件**：点击上传按钮或拖放文件
- **下载文件**：点击下载按钮
- **删除文件**：点击删除按钮并确认
- **重命名**：点击重命名按钮
- **断点续传**：中断后可继续传输，支持大文件

### 4. 视图切换

- **列表视图**：显示详细信息（大小、修改时间）
- **网格视图**：大图标视图，适合图片

### 5. 加密模式

在加密模式下：
- 密钥包含在 URL 的 hash 中（#key=...）
- 密钥永远不会发送到服务器
- 所有文件传输通过 AES-256-GCM 加密

## 从源码编译

### 环境要求

- Go 1.21 或更高版本

### 编译步骤

```bash
# 克隆项目
git clone https://github.com/Asamo096/file-transfer.git
cd file-transfer

# 编译
cd backend
go build -o file-transfer.exe .

# 或使用 go mod
go build -mod=mod -o file-transfer.exe .
```

### 跨平台编译

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o file-transfer.exe ./backend

# Linux
GOOS=linux GOARCH=amd64 go build -o file-transfer ./backend

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o file-transfer-darwin-amd64 ./backend

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o file-transfer-darwin-arm64 ./backend
```

## 项目结构

```
File-deliver/
├── backend/
│   ├── main.go           # 程序入口
│   ├── config/            # 配置模块
│   ├── handler/           # HTTP 处理器
│   │   ├── file.go       # 文件操作
│   │   ├── p2p.go        # P2P 连接
│   │   ├── qrcode.go     # 二维码生成
│   │   └── crypto.go     # 加密处理
│   ├── middleware/        # 中间件
│   ├── service/          # 业务逻辑
│   └── utils/            # 工具函数
├── frontend/
│   ├── index.html        # 主页面
│   ├── css/
│   │   └── styles.css    # 样式文件
│   └── js/
│       ├── app.js        # 应用主逻辑
│       ├── api.js        # API 调用封装
│       ├── crypto.js     # 加密功能
│       ├── fileManager.js # 文件管理
│       └── p2p.js        # P2P 功能
├── pkg/
│   └── xfer/
│       ├── stun/         # STUN 客户端
│       ├── dht/          # DHT 网络
│       ├── discovery/    # 设备发现
│       └── types.go      # 类型定义
├── go.mod                # Go 模块文件
└── README.md             # 本文件
```

## 技术栈

- **后端**：Go + Gin 框架
- **前端**：原生 HTML5 + CSS3 + JavaScript
- **加密**：AES-256-GCM (Web Crypto API)
- **二维码**：skip2/go-qrcode
- **STUN**：pion/stun
- **DHT**：go-libp2p-kad-dht

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

---

**享受简单快捷的文件传输！** 🚀
