package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
)

type Message struct {
	UserID    int64  `json:"user_id"`
	Sender    string `json:"sender"`
	Content   string `json:"content"`
	FilePath  string `json:"file_path,omitempty"`
	Timestamp string `json:"timestamp"`
}
// 配置结构体
type Config struct {
	BotToken     string `json:"bot_token"`
	StartMessage string `json:"start_message"`
	HTTPUser     string `json:"http_user"`
	HTTPPassword string `json:"http_password"`
}

var config Config

// 读取配置文件
func LoadConfig() {
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatalf("无法读取配置文件: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Fatalf("配置文件解析失败: %v", err)
	}
}
var (
	db       *sql.DB
	bot      *tgbotapi.BotAPI
)

// 初始化数据库
func InitDB() {
	var err error
	db, err = sql.Open("sqlite3", "./messages.db")
	if err != nil {
		log.Fatal("无法连接数据库:", err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY,
		user_id INTEGER,
		sender TEXT,
		content TEXT,
		file_path TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
}

// 保存消息
func SaveMessage(msg Message) {
    // 如果 sender 为空，设置默认值
    if msg.Sender == "" {
        msg.Sender = fmt.Sprintf("用户ID - %d", msg.UserID)
    }

    // 插入消息记录
    _, err := db.Exec(`INSERT INTO messages (user_id, sender, content, file_path) VALUES (?, ?, ?, ?)`,
        msg.UserID, msg.Sender, msg.Content, msg.FilePath)
    if err != nil {
        log.Printf("保存消息失败: %v", err)
    }
}


// 获取历史消息
func GetMessagesByUserID(userID int64) []Message {
	rows, _ := db.Query(`SELECT sender, content, file_path, timestamp FROM messages WHERE user_id = ? ORDER BY timestamp`, userID)
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		rows.Scan(&msg.Sender, &msg.Content, &msg.FilePath, &msg.Timestamp)

		if msg.FilePath != "" {
			filename := msg.FilePath[strings.LastIndex(msg.FilePath, "/")+1:]
			msg.Content = fmt.Sprintf("发送文件 [%s] [<a href='%s' target='_blank'>点击下载</a>]", filename, msg.FilePath)
		}
		messages = append(messages, msg)
	}
	return messages
}

// 获取客户列表
func GetAllCustomers() []string {
    rows, _ := db.Query(`
        SELECT user_id, MAX(sender)
        FROM messages
        WHERE sender != 'admin'
        GROUP BY user_id
    `)
    defer rows.Close()

    var customers []string
    for rows.Next() {
        var userID int64
        var sender string
        rows.Scan(&userID, &sender)
        customers = append(customers, fmt.Sprintf("%s - %d", sender, userID))
    }
    return customers
}
//鉴权
func BasicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != config.HTTPUser || pass != config.HTTPPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "未授权访问", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}


func StartBot() {
    var err error
    bot, err = tgbotapi.NewBotAPI(config.BotToken)
    if err != nil {
        log.Panicf("启动 Bot 失败: %v", err)
    }
    log.Printf("Bot 已启动: %s", bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    updates := bot.GetUpdatesChan(u)

    for update := range updates {
        if update.Message != nil {
            user := update.Message.From
            fullName := strings.TrimSpace(user.FirstName + " " + user.LastName)
            userInfo := fmt.Sprintf("%s@%s", fullName, user.UserName)
		   if update.Message.Text == "/start" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, config.StartMessage)
				bot.Send(msg)
			}
            if update.Message.Text != "" {
                SaveMessage(Message{
                    UserID:   user.ID,
                    Sender:   userInfo,
                    Content:  update.Message.Text,
                })
            }

            //文件、图片、视频...
            if update.Message.Document != nil || update.Message.Photo != nil {
                fileID := ""
                filename := ""

                if update.Message.Document != nil { // 处理文件
                    fileID = update.Message.Document.FileID
                    filename = update.Message.Document.FileName
                } else if len(update.Message.Photo) > 0 { // 处理图片
                    fileID = update.Message.Photo[len(update.Message.Photo)-1].FileID
                    filename = fmt.Sprintf("photo_%d.jpg", update.Message.Photo[len(update.Message.Photo)-1].FileSize)
                }

                // 下载文件
                fileURL, _ := bot.GetFileDirectURL(fileID)
                savePath := "./uploads/" + filename
                downloadFile(fileURL, savePath)

                SaveMessage(Message{
                    UserID:   user.ID,
                    Sender:   userInfo,
                    Content:  fmt.Sprintf("发送文件 [%s] [<a href='/uploads/%s' target='_blank'>点击下载</a>]", filename, filename),
                    FilePath: savePath,
                })
            }
        }
    }
}

func downloadFile(url, filePath string) {
    resp, err := http.Get(url)
    if err != nil {
        log.Println("文件下载失败:", err)
        return
    }
    defer resp.Body.Close()

    out, err := os.Create(filePath)
    if err != nil {
        log.Println("无法创建文件:", err)
        return
    }
    defer out.Close()

    _, err = io.Copy(out, resp.Body)
    if err != nil {
        log.Println("文件保存失败:", err)
    }
}


func main() {
	LoadConfig()
	InitDB()
	go StartBot()
    http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))
	http.HandleFunc("/", BasicAuth(ChatPageHandler))
	http.HandleFunc("/customers", BasicAuth(CustomersHandler))
	http.HandleFunc("/history", BasicAuth(HistoryHandler))
	http.HandleFunc("/send", BasicAuth(SendMessageHandler))
	http.HandleFunc("/upload", BasicAuth(UploadHandler))
	//这里可以自己改端口
	log.Println("Web 服务运行中: http://0.0.0.0:8080")
	http.ListenAndServe("0.0.0.0:8080", nil)
}

func ChatPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/index.html")
}

func CustomersHandler(w http.ResponseWriter, r *http.Request) {
	customers := GetAllCustomers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customers)
}

func HistoryHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
	messages := GetMessagesByUserID(userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}



func SendMessageHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == "POST" {
        userID, _ := strconv.ParseInt(r.FormValue("user_id"), 10, 64)
        content := r.FormValue("content")

        // 获取用户最新的昵称和用户名
        var sender string
        row := db.QueryRow(`SELECT sender FROM messages WHERE user_id = ? ORDER BY timestamp DESC LIMIT 1`, userID)
        row.Scan(&sender)
        if sender == "" { // 如果没有历史记录，设置默认值
            sender = fmt.Sprintf("用户ID - %d", userID)
        }

        // 保存消息
        SaveMessage(Message{UserID: userID, Sender: "admin", Content: content})
        msg := tgbotapi.NewMessage(userID, content)
        bot.Send(msg)

        fmt.Fprintf(w, "消息发送成功")
    }
}


func UploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "文件上传失败", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filePath := "./uploads/" + handler.Filename
	dst, _ := os.Create(filePath)
	defer dst.Close()
	io.Copy(dst, file)

	userID, _ := strconv.ParseInt(r.FormValue("user_id"), 10, 64)
	fileToSend := tgbotapi.NewDocument(userID, tgbotapi.FilePath(filePath))
	bot.Send(fileToSend)

	SaveMessage(Message{
		UserID:   userID,
		Sender:   "admin",
		FilePath: "/uploads/" + handler.Filename,
	})

	fmt.Fprintf(w, "文件上传成功")
}
