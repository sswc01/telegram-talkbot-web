# 双向Bot

一个基于 **Telegram** 的双向通信 Bot，支持简单静态页面展示，快速部署，轻量稳定。

频道 https://t.me/sswcnet

作者 https://t.me/sswc01

博客部署教程 https://www.sswc.lol/T1746189556


## 功能特点

- **双向通信**：支持用户与 Bot 之间消息的实时双向转发。
- **静态网页**：内置轻量级静态页面（`static/index.html`）。
- **快速部署**：提供一键部署脚本 `setup.sh`。
- **简洁配置**：通过 `config.json` 文件轻松管理 Bot 配置。
- **高效稳定**：基于 Go 语言开发，性能优秀。

## 项目结构

```
.
├── config.json      # Bot 配置文件
├── main.go          # 主程序源代码
├── setup.sh         # 安装部署脚本
├── tgbot            # 可执行文件（编译版）
└── static/
    └── index.html   # 静态网页
```

## 使用方法

### 1. 克隆项目

```bash
git clone https://github.com/sswc01/telegram-talkbot-web.git
cd telegram-talkbot-web
```

### 2. 配置环境

编辑 `config.json` 文件，填入你的 Telegram Bot Token 和相关设置，例如：

```json
{
  "bot_token": "bot tokan",
  "start_message": "你好！请发送您的问题，客服会尽快回复您。",
  "http_user": "admin",
  "http_password": "123456"
}

```

### 3. 运行 Bot

#### 使用已有可执行文件（推荐）

```bash
chmod +x tgbot
./tgbot
```

#### 或自行编译源码

如果需要自己编译 `main.go`：

```bash
go build -o tgbot main.go
./tgbot
```

### 4. 使用一键部署脚本（可选）

```bash
chmod +x setup.sh
./setup.sh
```

## 注意事项

- 请确保你的服务器具备访问telegram能力。
- 部署前请正确配置 `config.json`。
- 若需修改网页内容，可直接编辑 `static/index.html`。

