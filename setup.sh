#!/bin/bash
echo "正在安装项目依赖..."
go version || { echo "Go 未安装，请安装 Go >= 1.16"; exit 1; }
go mod init tgbot-1
go get github.com/go-telegram-bot-api/telegram-bot-api/v5
go get github.com/mattn/go-sqlite3
go mod tidy

echo "创建数据库文件..."
touch messages.db
echo "编译项目..."
go build -o tgbot .
