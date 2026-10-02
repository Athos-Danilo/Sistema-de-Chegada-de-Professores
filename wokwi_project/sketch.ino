/*
 * SCP - Sistema de Chegada de Professores (Firmware IoT ESP32)
 * Disciplina: Sistemas Embarcados - Prof. Lauro Alves
 * Alunos: Athos, Alessandro, Joseph e José E.
 * 
 * Periféricos simulados:
 * - ESP32 DevKit v1
 * - Leitor RFID MFRC522 (SPI: SS=5, RST=4, SCK=18, MISO=19, MOSI=23)
 * - Display OLED SSD1306 128x64 (I2C: SDA=21, SCL=22)
 * - LED Verde (GPIO 13) - Sucesso / Presença Confirmada
 * - LED Amarelo (GPIO 2) - Registro Salvo Offline
 * - LED Vermelho (GPIO 15) - Erro de Leitura / Tag Desconhecida
 * - Buzzer Passivo (GPIO 12) - Feedback Sonoro
 * - Botão Reset Wi-Fi (GPIO 14) - Simulação de Portal Cativo / Erro
 */

#include <SPI.h>
#include <MFRC522.h>
#include <Wire.h>
#include <Adafruit_GFX.h>
#include <Adafruit_SSD1306.h>
#include <WiFi.h>
#include <time.h>

// --- MAPEAMENTO DE PINOS ---
#define SS_PIN          5
#define RST_PIN         4
#define LED_GREEN_PIN   13
#define LED_YELLOW_PIN  2
#define LED_RED_PIN     15
#define BUZZER_PIN      12
#define BTN_RESET_PIN   14

#define SCREEN_WIDTH 128
#define SCREEN_HEIGHT 64
#define OLED_RESET -1
Adafruit_SSD1306 display(SCREEN_WIDTH, SCREEN_HEIGHT, &Wire, OLED_RESET);
MFRC522 rfid(SS_PIN, RST_PIN);

// --- REGRAS DE NEGÓCIO ---
String last_uid = "";
unsigned long last_read_time = 0;
const unsigned long DEBOUNCE_DELAY = 10000; // 10s para testes
bool is_registered_in = false; // Alterna Entrada/Saída
bool wasOffline = false;

// --- CONFIGURAÇÃO DE REDE ---
const char* ssid = "Wokwi-GUEST";
const char* password = "";
const char* ntpServer = "pool.ntp.org";
const long gmtOffset_sec = -10800; // Horário de Brasília (-3h)
const int daylightOffset_sec = 0;

void playBuzzerBeep(int frequency, int durationMs, int times = 1) {
  for (int i = 0; i < times; i++) {
    tone(BUZZER_PIN, frequency, durationMs);
    delay(durationMs + 50);
  }
}

void showScreenMessage(String title, String line1, String line2, String line3) {
  display.clearDisplay();
  display.setTextSize(1);
  display.setTextColor(SSD1306_WHITE);
  
  // Cabeçalho
  display.setCursor(0, 0);
  display.print("== ");
  display.print(title);
  display.println(" ==");
  display.drawFastHLine(0, 10, 128, SSD1306_WHITE);
  
  // Conteúdo
  display.setCursor(0, 16);
  display.println(line1);
  display.setCursor(0, 30);
  display.println(line2);
  display.setCursor(0, 44);
  display.println(line3);
  
  display.display();
}

void setup() {
  Serial.begin(115200);
  Serial.println("\n[SCP] Inicializando Sistema de Chegada de Professores...");

  // Configuração dos Pinos
  pinMode(LED_GREEN_PIN, OUTPUT);
  pinMode(LED_YELLOW_PIN, OUTPUT);
  pinMode(LED_RED_PIN, OUTPUT);
  pinMode(BUZZER_PIN, OUTPUT);
  pinMode(BTN_RESET_PIN, INPUT_PULLUP);

  digitalWrite(LED_GREEN_PIN, LOW);
  digitalWrite(LED_YELLOW_PIN, LOW);
  digitalWrite(LED_RED_PIN, LOW);

  // Inicialização do OLED
  if (!display.begin(SSD1306_SWITCHCAPVCC, 0x3C)) {
    Serial.println(F("[ERRO] Falha ao inicializar o Display OLED SSD1306!"));
    for (;;);
  }
  
  showScreenMessage("SCP - IFPE", "Inicializando...", "Conectando Wi-Fi", "Aguarde...");

  // Inicialização do SPI e MFRC522
  SPI.begin();
  rfid.PCD_Init();
  Serial.println("[OK] Leitor RFID RC522 Inicializado!");

  // Conexão Wi-Fi (Simulação Wokwi)
  WiFi.begin(ssid, password);
  int wifi_attempts = 0;
  while (WiFi.status() != WL_CONNECTED && wifi_attempts < 10) {
    delay(300);
    Serial.print(".");
    wifi_attempts++;
  }

  if (WiFi.status() == WL_CONNECTED) {
    Serial.println("\n[OK] Wi-Fi Conectado!");
    Serial.print("IP: ");
    Serial.println(WiFi.localIP());
    configTime(gmtOffset_sec, daylightOffset_sec, ntpServer);
    wasOffline = false;
  } else {
    Serial.println("\n[AVISO] Conexao Wi-Fi falhou! Operando em modo offline.");
    digitalWrite(LED_YELLOW_PIN, HIGH);
    playBuzzerBeep(1200, 100, 2);
    wasOffline = true;
  }

  playBuzzerBeep(1000, 100, 2); // 2 bipes indicando inicialização completa
  if (wasOffline) {
    showScreenMessage("SISTEMA PRONTO", "Aproxime seu cartao", "na entrada da sala", "Status: OFFLINE");
  } else {
    showScreenMessage("SISTEMA PRONTO", "Aproxime seu cartao", "na entrada da sala", "Status: ONLINE");
  }
}

String getFormattedTime() {
  struct tm timeinfo;
  if (!getLocalTime(&timeinfo)) {
    return "19:00:00";
  }
  char timeStringBuff[15];
  strftime(timeStringBuff, sizeof(timeStringBuff), "%H:%M:%S", &timeinfo);
  return String(timeStringBuff);
}

void processTag(String uid) {
  unsigned long now = millis();
  
  // Filtro de Debounce
  if (uid == last_uid && (now - last_read_time < DEBOUNCE_DELAY)) {
    Serial.println("[DEBOUNCE] Leitura duplicada ignorada.");
    return;
  }

  last_uid = uid;
  last_read_time = now;
  is_registered_in = !is_registered_in; // Alterna entre Entrada e Saída

  String hora = getFormattedTime();
  String tipo = is_registered_in ? "ENTRADA" : "SAIDA";
  
  Serial.println("==========================================");
  Serial.print("[PRESENCA] UID da Tag: ");
  Serial.println(uid);
  Serial.print("[PRESENCA] Tipo: ");
  Serial.println(tipo);
  Serial.print("[PRESENCA] Horario: ");
  Serial.println(hora);

  // Payload JSON formatado direto sem biblioteca externa (ultra leve)
  String jsonPayload = "{\"device_id\":\"EMB-LAB-101\",\"tag_uid\":\"" + uid + "\",\"event_type\":\"" + tipo + "\",\"timestamp\":\"" + hora + "\"}";
  Serial.print("[API REST PAYLOAD] ");
  Serial.println(jsonPayload);

  if (WiFi.status() == WL_CONNECTED) {
    // Sinalização Visual e Sonora de Sucesso (Verde + 1 bipe curto)
    digitalWrite(LED_GREEN_PIN, HIGH);
    playBuzzerBeep(2000, 150, 1);
    
    showScreenMessage("REGISTRO OK!", "UID: " + uid, tipo + " registrada", "Hora: " + hora);
    delay(2000);
    digitalWrite(LED_GREEN_PIN, LOW);
  } else {
    // Sinalização de Registro Salvo Offline (Amarelo + 2 bipes)
    digitalWrite(LED_YELLOW_PIN, HIGH);
    playBuzzerBeep(1200, 100, 2);
    
    showScreenMessage("OFFLINE SAVED", "UID: " + uid, tipo + " (Offline)", "Hora: " + hora);
    delay(2000);
    digitalWrite(LED_YELLOW_PIN, HIGH); // Mantém o amarelo aceso no modo offline
  }

  if (WiFi.status() == WL_CONNECTED) {
    showScreenMessage("SISTEMA PRONTO", "Aproxime seu cartao", "na entrada da sala", "Status: ONLINE");
  } else {
    showScreenMessage("SISTEMA PRONTO", "Aproxime seu cartao", "na entrada da sala", "Status: OFFLINE");
  }
}

void loop() {
  // Gerenciamento do estado do Wi-Fi
  bool isOffline = (WiFi.status() != WL_CONNECTED);
  
  if (isOffline && !wasOffline) {
    Serial.println("\n[ALERTA] Conexao Wi-Fi perdida! Entrando em modo offline.");
    digitalWrite(LED_YELLOW_PIN, HIGH);
    playBuzzerBeep(1200, 100, 2); // 2 bipes curtos (Offline)
    showScreenMessage("SISTEMA PRONTO", "Aproxime seu cartao", "na entrada da sala", "Status: OFFLINE");
    wasOffline = true;
  } else if (!isOffline && wasOffline) {
    Serial.println("\n[INFO] Conexao Wi-Fi restaurada!");
    digitalWrite(LED_YELLOW_PIN, LOW);
    playBuzzerBeep(2000, 150, 1); // 1 bipe curto (Restaurado)
    showScreenMessage("SISTEMA PRONTO", "Aproxime seu cartao", "na entrada da sala", "Status: ONLINE");
    wasOffline = false;
  }

  // Teste do Botão de Reset / Erro
  if (digitalRead(BTN_RESET_PIN) == LOW) {
    Serial.println("[BOTAO RESET] Botao de Reset Wi-Fi pressionado!");
    digitalWrite(LED_RED_PIN, HIGH);
    playBuzzerBeep(500, 800, 1); // 1 bipe longo para erro/reset
    showScreenMessage("RESET WI-FI", "Modo Portal Cativo", "Aguardando Setup", "IP: 192.168.4.1");
    delay(3000);
    digitalWrite(LED_RED_PIN, LOW);
    if (wasOffline) {
      showScreenMessage("SISTEMA PRONTO", "Aproxime seu cartao", "na entrada da sala", "Status: OFFLINE");
    } else {
      showScreenMessage("SISTEMA PRONTO", "Aproxime seu cartao", "na entrada da sala", "Status: ONLINE");
    }
  }

  // Verifica se há novas tags RFID no leitor
  if (!rfid.PICC_IsNewCardPresent()) {
    return;
  }

  if (!rfid.PICC_ReadCardSerial()) {
    return;
  }

  // Converte o UID lido para String Hexadecimal
  String tagUID = "";
  for (byte i = 0; i < rfid.uid.size; i++) {
    tagUID += String(rfid.uid.uidByte[i] < 0x10 ? "0" : "");
    tagUID += String(rfid.uid.uidByte[i], HEX);
  }
  tagUID.toUpperCase();

  processTag(tagUID);

  // Finaliza a leitura da tag atual
  rfid.PICC_HaltA();
  rfid.PCD_StopCrypto1();
}
