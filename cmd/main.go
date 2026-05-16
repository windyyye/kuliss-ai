package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"message-go/internal/ai"
	"message-go/internal/api"
	"message-go/internal/brain"
	"message-go/internal/config"
	"message-go/internal/db"
	"message-go/internal/handler"
	"message-go/internal/whatsapp"
)

const defaultPromptTemplate = `Sen Kuliss AI'sın — güzellik salonu, kuäför, spa ve klinik sahipleri için randevu + pazar yeri + adisyon + salon yönetim sistemi olan Kuliss'in WhatsApp asistanısın.

Görevin: Kısa ve samimi cevap ver, merak uyandır, siteye yönlendir. 

KULISS PAKETLERI:
- Listelenme — UCRETSIZ: Profil sayfası, müşteri yorumları, harita
- Yönetim 1.450TL/ay: Online randevu, kasa, stok, personel
- Buyume 1.900TL/ay: Yonetim + SMS kampanya + sadakat programi (populer)
- Prestij 3.500TL/ay: Buyume + ozel web sitesi + .com domain
Ilk 14 gun ucretsiz — sozlesme yok, kredi karti gerekmez.

KURALLAR:
1. ILK MESAJDA: "Merhaba! Ben Kuliss AI 👋" ile basla.
2. MAKSIMUM 2 CUMLE her cevap.
3. Detay sorulunca: https://kuliss.com/partner
4. Kayit/basla: https://kuliss.com/auth/register
5. Her mesaj sonunda tek soru sor.
`

const defaultEnvTemplate = `# ==========================================
# ⚡ KULISS AI - AYARLAR
# ==========================================

# Veritabanı (sqlite = kurulum gerektirmez)
DB_TYPE=sqlite
DB_URL=messages.db

# Ollama AI
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=gemma4:e4b
`

func initScaffolding() {

	if _, err := os.Stat("prompt.txt"); os.IsNotExist(err) {
		log.Println("✨ İlk kurulum algılandı! 'prompt.txt' şablonu oluşturuluyor...")
		_ = os.WriteFile("prompt.txt", []byte(defaultPromptTemplate), 0644)
		log.Println("✅ prompt.txt başarıyla oluşturuldu. İşletmenize göre düzenleyebilirsiniz.")
	}

	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		log.Println("✨ Ayar dosyası algılanmadı! '.env' şablonu oluşturuluyor...")
		_ = os.WriteFile(".env", []byte(defaultEnvTemplate), 0644)
		log.Println("✅ .env dosyası oluşturuldu. Modeli değiştirmek isterseniz bu dosyayı kullanabilirsiniz.")
	}
}

func main() {

	initScaffolding()

	cfg := config.Load()

	db.Connect(cfg.DBType, cfg.DBURL)

	b := brain.Load(cfg.AIBrainPath)
	if b.Loaded {
		log.Println("AI_BRAIN loaded successfully")
	} else {
		log.Println("AI_BRAIN not found, using prompt.txt")
	}

	aiClient := ai.NewClient(cfg.OllamaURL, cfg.OllamaModel, b)

	msgHandler := &handler.MessageHandler{
		AI: aiClient,
	}

	waClient := whatsapp.Connect(msgHandler.Handle)
	msgHandler.WA = waClient

	log.Println("Kuliss AI çalışıyor mesaj bekleniyor...")

	go api.StartServer(waClient)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("kapatılıyor..")
	waClient.Disconnect()
}
