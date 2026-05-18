package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/wailsapp/wails/v3/pkg/application"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"

	"google.golang.org/protobuf/proto"
	"message-go/frontend"
	"message-go/internal/ai"
	"message-go/internal/brain"
	"message-go/internal/config"
	"message-go/internal/db"
	"message-go/internal/handler"
)

// initAppData seeds app data dir on first launch.
// .env yoksa bundled .env.example'ı kopyalar.
func initAppData() {
	envPath := config.DataPath(".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		exe, err := os.Executable()
		if err == nil {
			resDir := filepath.Join(filepath.Dir(exe), "..", "Resources")
			example := filepath.Join(resDir, ".env.example")
			if data, err := os.ReadFile(example); err == nil {
				_ = os.WriteFile(envPath, data, 0644)
			}
		}
	}
}


type Message struct {
	Phone		string	`json:"phone"`
	Role		string	`json:"role"`
	Content		string	`json:"content"`
	CreatedAt	string	`json:"created_at"`
}

type Contact struct {
	Phone		string	`json:"phone"`
	Blocked		bool	`json:"blocked"`
	LastMsg		string	`json:"last_msg"`
	CreatedAt	string	`json:"created_at"`
	ProfilePic	string	`json:"profile_pic"`
	ContactName	string	`json:"contact_name"`
}

type BotStatus struct {
	Running		bool	`json:"running"`
	Connected	bool	`json:"connected"`
	Model		string	`json:"model"`
}

type BotService struct {
	mu		sync.Mutex
	wa		*whatsmeow.Client
	container	*sqlstore.Container
	wailsApp	*application.App
	running		bool
	cfg		*config.Config
	brain		*brain.Brain
}

func NewBotService(app *application.App) *BotService {
	return &BotService{wailsApp: app}
}

func (s *BotService) Start() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return "already_running", nil
	}

	s.cfg = config.Load()

	if db.DB == nil {
		db.Connect(s.cfg.DBType, s.cfg.DBURL)
	}

	if s.container == nil {
		sessionDB := "file:" + config.DataPath("kuliss-session.db") + "?_foreign_keys=on"
		container, err := sqlstore.New(context.Background(), "sqlite3", sessionDB, waLog.Stdout("DB", "INFO", true))
		if err != nil {
			return "", fmt.Errorf("store hatası: %w", err)
		}
		s.container = container
	}

	deviceStore, err := s.container.GetFirstDevice(context.Background())
	if err != nil {
		return "", fmt.Errorf("device store: %w", err)
	}
	if deviceStore == nil {
		deviceStore = s.container.NewDevice()
	}

	aiClient := buildAIClient(s.cfg, s.brain)
	msgHandler := &handler.MessageHandler{
		AI:	aiClient,
		OnNewMessage: func(phone, role, text string) {
			nowRaw := time.Now().UTC().Format(time.RFC3339)
			s.wailsApp.Event.Emit("new_msg", Message{
				Phone:		phone,
				Role:		role,
				Content:	text,
				CreatedAt:	nowRaw,
			})
		},
		OnProfilePic: func(phone string) {
			s.wailsApp.Event.Emit("contact_updated", phone)
		},
	}
	waClient := whatsmeow.NewClient(deviceStore, waLog.Stdout("WhatsApp", "INFO", true))
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()
	store.DeviceProps.Os = proto.String("macOS")

	msgHandler.WA = waClient
	waClient.AddEventHandler(msgHandler.Handle)
	s.wa = waClient

	if waClient.Store.ID == nil {
		qrChan, _ := waClient.GetQRChannel(context.Background())
		if err = waClient.Connect(); err != nil {
			return "", fmt.Errorf("bağlantı: %w", err)
		}
		go func() {
			for evt := range qrChan {
				if evt.Event == "code" {
					if b64, qErr := generateQRBase64(evt.Code); qErr == nil {
						s.wailsApp.Event.Emit("qr_code", b64)
					}
				} else if evt.Event == "success" {
					s.mu.Lock()
					s.running = true
					s.mu.Unlock()
					s.wailsApp.Event.Emit("wa_connected", true)
					s.wailsApp.Event.Emit("status_change", s.GetStatus())
				} else if evt.Event == "timeout" {
					s.wailsApp.Event.Emit("qr_timeout", true)
				}
			}
		}()
		return "waiting_qr", nil
	}

	if err = waClient.Connect(); err != nil {
		return "", fmt.Errorf("reconnect: %w", err)
	}
	s.running = true
	s.wailsApp.Event.Emit("status_change", s.GetStatus())
	return "connected", nil
}

func (s *BotService) Stop() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.wa != nil {
		s.wa.Disconnect()
		s.wa = nil
	}
	s.running = false
	s.wailsApp.Event.Emit("status_change", s.GetStatus())
	return "stopped"
}

func (s *BotService) GetStatus() BotStatus {
	model := "gemma4:e4b"
	if s.cfg != nil {
		if s.cfg.AIProvider == "openrouter" && s.cfg.AIModel != "" {
			model = s.cfg.AIModel
		} else if s.cfg.OllamaModel != "" {
			model = s.cfg.OllamaModel
		}
	}
	return BotStatus{
		Running:	s.running,
		Connected:	s.wa != nil && s.wa.IsConnected(),
		Model:		model,
	}
}

func (s *BotService) LogoutWhatsApp() error {

	s.mu.Lock()
	client := s.wa
	oldContainer := s.container
	s.wa = nil
	s.running = false
	s.container = nil
	s.mu.Unlock()

	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if logoutErr := client.Logout(ctx); logoutErr != nil {
			log.Printf("Remote logout başarısız (yerel temizlemeye devam): %v", logoutErr)
		}
		cancel()
		client.Disconnect()
	}

	if oldContainer != nil {
		if dev, err := oldContainer.GetFirstDevice(context.Background()); err == nil && dev != nil {
			if delErr := oldContainer.DeleteDevice(context.Background(), dev); delErr != nil {
				log.Printf("DeleteDevice başarısız: %v", delErr)
			}
		}
	}

	if removeErr := os.Remove(config.DataPath("kuliss-session.db")); removeErr != nil {
		if !os.IsNotExist(removeErr) {
			log.Printf("Session DB silinemedi: %v", removeErr)
		}
	} else {
		log.Println("Session DB silindi, yeni QR üretilecek")
	}

	s.wailsApp.Event.Emit("status_change", s.GetStatus())
	return nil
}

func (s *BotService) GetPrompt() string {
	if s.brain != nil && s.brain.Loaded {
		return s.brain.GetFullPrompt()
	}
	data, _ := os.ReadFile(config.DataPath("prompt.txt"))
	return string(data)
}

func (s *BotService) SavePrompt(content string) error {
	if s.brain != nil && s.brain.Loaded {
		return brain.Reload(s.cfg.AIBrainPath, s.brain)
	}
	return os.WriteFile(config.DataPath("prompt.txt"), []byte(content), 0644)
}

func (s *BotService) TestPrompt(history []Message, message string) (string, error) {
	cfg := config.Load()
	aiClient := buildAIClient(cfg, s.brain)

	var mappedHistory []map[string]string

	for _, m := range history {
		mappedHistory = append(mappedHistory, map[string]string{
			"role":		m.Role,
			"content":	m.Content,
		})
	}

	return aiClient.Chat(mappedHistory, message)
}

type OllamaTagResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func (s *BotService) GetModels() ([]string, error) {
	cfg := config.Load()

	if cfg.AIProvider == "openrouter" {
		return getOpenRouterModels(cfg)
	}
	return getOllamaModels(cfg)
}

func getOllamaModels(cfg *config.Config) ([]string, error) {
	url := cfg.OllamaURL
	if url == "" {
		url = "http://localhost:11434"
	}
	url = strings.TrimRight(strings.TrimSuffix(url, "/v1"), "/")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("Ollama'ya ulaşılamadı: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama hata döndürdü: %d", resp.StatusCode)
	}

	var tags OllamaTagResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}

	var models []string
	for _, m := range tags.Models {
		models = append(models, m.Name)
	}
	return models, nil
}

type openRouterModel struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Pricing struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
	} `json:"pricing"`
}

type openRouterModelsResponse struct {
	Data []openRouterModel `json:"data"`
}

func getOpenRouterModels(cfg *config.Config) ([]string, error) {
	baseURL := "https://openrouter.ai/api/v1"

	req, err := http.NewRequest("GET", baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("request oluşturma: %w", err)
	}
	if cfg.AIApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.AIApiKey)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenRouter'a ulaşılamadı: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenRouter hata döndürdü: %d", resp.StatusCode)
	}

	var result openRouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var free, paid []string
	for _, m := range result.Data {
		if m.Pricing.Prompt == "0" && m.Pricing.Completion == "0" {
			free = append(free, m.ID)
		} else {
			paid = append(paid, m.ID)
		}
	}

	return append(free, paid...), nil
}

func (s *BotService) GetConfig() map[string]string {
	cfg := config.Load()
	return map[string]string{
		"AI_PROVIDER":  cfg.AIProvider,
		"AI_API_URL":   cfg.AIApiURL,
		"AI_MODEL":     cfg.AIModel,
		"AI_API_KEY":   cfg.AIApiKey,
		"OLLAMA_MODEL": cfg.OllamaModel,
		"OLLAMA_URL":   cfg.OllamaURL,
		"DB_TYPE":      cfg.DBType,
		"DB_URL":       cfg.DBURL,
		"LANGUAGE":     cfg.Language,
	}
}

func (s *BotService) SaveConfig(newCfg map[string]string) error {
	var sb strings.Builder
	for k, v := range newCfg {
		sb.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}
	err := os.WriteFile(config.DataPath(".env"), []byte(sb.String()), 0644)
	if err == nil {
		s.cfg = config.Load()
	}
	return err
}

func (s *BotService) GetContacts() ([]Contact, error) {
	if db.DB == nil {
		return []Contact{}, nil
	}
	ensureBlockedTable()
	rows, err := db.DB.Query(`
		SELECT c.phone,
		       COALESCE(b.phone,'') as blocked,
		       COALESCE((SELECT content FROM conversations WHERE phone=c.phone ORDER BY created_at DESC LIMIT 1),'') as last_msg,
		       c.created_at,
		       COALESCE(c.profile_pic, '') as profile_pic,
		       COALESCE(c.contact_name, '') as contact_name
		FROM contacts c
		LEFT JOIN blocked_contacts b ON c.phone = b.phone
		ORDER BY c.created_at DESC
		LIMIT 200
	`)
	if err != nil {
		return []Contact{}, err
	}
	defer rows.Close()

	var out []Contact
	for rows.Next() {
		var c Contact
		var blockedPhone string
		if err := rows.Scan(&c.Phone, &blockedPhone, &c.LastMsg, &c.CreatedAt, &c.ProfilePic, &c.ContactName); err != nil {
			continue
		}
		c.Blocked = blockedPhone != ""
		out = append(out, c)
	}
	return out, nil
}

func (s *BotService) GetConversation(phone string) ([]Message, error) {
	if db.DB == nil {
		return []Message{}, nil
	}
	rows, err := db.DB.Query(`
		SELECT phone, role, content, created_at
		FROM conversations WHERE phone = ? ORDER BY created_at ASC
	`, phone)
	if err != nil {
		return []Message{}, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.Phone, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (s *BotService) GetTotalMessages() (int, error) {
	if db.DB == nil {
		return 0, nil
	}
	var count int
	err := db.DB.QueryRow(`SELECT COUNT(*) FROM conversations`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *BotService) BlockContact(phone string) error {
	if db.DB == nil {
		return fmt.Errorf("db bağlı değil")
	}
	ensureBlockedTable()
	_, err := db.DB.Exec(`INSERT OR IGNORE INTO blocked_contacts (phone) VALUES (?)`, phone)
	return err
}

func (s *BotService) UnblockContact(phone string) error {
	if db.DB == nil {
		return fmt.Errorf("db bağlı değil")
	}
	_, err := db.DB.Exec(`DELETE FROM blocked_contacts WHERE phone = ?`, phone)
	return err
}

func ensureBlockedTable() {
	if db.DB != nil {
		_, _ = db.DB.Exec(`CREATE TABLE IF NOT EXISTS blocked_contacts (
			phone TEXT PRIMARY KEY,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`)
	}
}

type wailsLogWriter struct {
	app        *application.App
	fileWriter io.Writer
}

func (w *wailsLogWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	if w.app != nil {
		w.app.Event.Emit("sys_log", msg)
	}
	if w.fileWriter != nil {
		w.fileWriter.Write(p)
	}
	return os.Stdout.Write(p)
}

func generateQRBase64(code string) (string, error) {
	qr, err := qrcode.New(code, qrcode.High)
	if err != nil {
		return "", err
	}
	img := qr.Image(512)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func buildAIClient(cfg *config.Config, b *brain.Brain) *ai.Client {
	if cfg.AIProvider == "openrouter" {
		url := cfg.AIApiURL
		if url == "" {
			url = "https://openrouter.ai/api/v1"
		}
		model := cfg.AIModel
		if model == "" {
			model = "qwen/qwen3-next-80b-a3b-instruct:free"
		}
		return ai.NewProviderClient("openrouter", url, model, cfg.AIApiKey, b)
	}
	return ai.NewClient(cfg.OllamaURL, cfg.OllamaModel, b)
}

func init() {
	application.RegisterEvent[string]("qr_code")
	application.RegisterEvent[bool]("wa_connected")
	application.RegisterEvent[BotStatus]("status_change")
	application.RegisterEvent[string]("time")
	application.RegisterEvent[Message]("new_msg")
	application.RegisterEvent[string]("sys_log")
	application.RegisterEvent[bool]("qr_timeout")
}

func getFrontendAssets() fs.FS {
	subFS, err := fs.Sub(frontend.Assets, "dist")
	if err != nil {
		panic(err)
	}
	return subFS
}

func main() {
	// ── 1. App Data dizini oluştur ────────────────────────────────────────────
	initAppData()

	// ── 2. Log dosyasını aç (tüm uygulama boyunca aktif kalır) ───────────────
	logDir := filepath.Join(func() string { h, _ := os.UserHomeDir(); return h }(),
		"Library", "Logs", "Kuliss")
	_ = os.MkdirAll(logDir, 0755)
	logFilePath := filepath.Join(logDir, "kuliss.log")
	logFile, logErr := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	var fileWriter io.Writer
	if logErr == nil {
		fileWriter = logFile
		defer logFile.Close()
	} else {
		fileWriter = io.Discard
	}
	// log her zaman hem stderr hem dosyaya gider
	log.SetOutput(io.MultiWriter(os.Stderr, fileWriter))
	log.Printf("[1/6] Kuliss başlatılıyor. AppDataDir=%s", config.AppDataDir())

	// ── 3. BotService ─────────────────────────────────────────────────────────
	botService := &BotService{}
	log.Println("[2/6] BotService oluşturuldu")

	// ── 4. Wails uygulaması ───────────────────────────────────────────────────
	log.Println("[3/6] application.New() başlıyor...")
	app := application.New(application.Options{
		Name:        "Kuliss",
		Description: "WhatsApp AI Bot Yönetici",
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(getFrontendAssets()),
		},
		Services: []application.Service{
			application.NewService(botService),
		},
	})
	log.Println("[3/6] application.New() tamamlandı")

	botService.wailsApp = app

	// wailsLogger da dosyaya yazsın
	wailsLogger := &wailsLogWriter{app: app, fileWriter: fileWriter}
	log.SetOutput(wailsLogger)

	// ── 5. Veritabanı ─────────────────────────────────────────────────────────
	log.Println("[4/6] Veritabanı bağlanıyor...")
	cfg := config.Load()
	log.Printf("[4/6] DB: type=%s url=%s", cfg.DBType, cfg.DBURL)
	if db.DB == nil {
		db.Connect(cfg.DBType, cfg.DBURL)
	}
	log.Println("[4/6] Veritabanı bağlandı")

	botService.cfg = cfg
	b := brain.Load(cfg.AIBrainPath)
	botService.brain = b
	if b.Loaded {
		log.Println("[4.5/6] AI_BRAIN loaded")
	} else {
		log.Println("[4.5/6] AI_BRAIN not found, using prompt.txt")
	}

	// ── 6. Pencere ────────────────────────────────────────────────────────────
	log.Println("[5/6] Pencere oluşturuluyor...")
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:    "Kuliss",
		Width:    1100,
		Height:   740,
		MinWidth: 900,
		MinHeight: 600,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(255, 255, 255),
		URL:              "/",
	})
	log.Println("[5/6] Pencere oluşturuldu")

	stopTicker := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				app.Event.Emit("time", time.Now().Format("15:04:05"))
			case <-stopTicker:
				return
			}
		}
	}()

	if err := app.Run(); err != nil {
		log.Println("MAINGO ERROR:", err)
	}
	close(stopTicker)
}
