/*
 * SCP - Sistema de Chegada de Professores (Firmware IoT ESP32)
 * Disciplina: Sistemas Embarcados - Prof. Lauro Alves
 * Alunos: Athos, Alessandro, Joseph e José E.
 *
 * Entrega R5: Programação e leitura dos sensores
 * Código fonte atendendo aos requisitos A até F do PDF de entrega.
 */

#include <SPI.h>
#include <MFRC522.h>
#include <Wire.h>
#include <Adafruit_GFX.h>
#include <Adafruit_SSD1306.h>
#include <WiFi.h>
#include <time.h>

// --- A. configuração dos pinos ---
// Definição dos pinos para os sensores, atuadores e interface
#define SS_PIN          5   // Pino SDA (SS) do RFID
#define RST_PIN         4   // Pino RST do RFID
#define LED_GREEN_PIN   13  // Atuador: LED Verde
#define LED_YELLOW_PIN  2   // Atuador: LED Amarelo
#define LED_RED_PIN     15  // Atuador: LED Vermelho
#define BUZZER_PIN      12  // Atuador: Buzzer Passivo
#define BTN_RESET_PIN   14  // Sensor/Entrada: Botão de Reset/Simulação

#define SCREEN_WIDTH 128
#define SCREEN_HEIGHT 64
#define OLED_RESET -1

// Instanciação de objetos do Display e Leitor RFID
Adafruit_SSD1306 display(SCREEN_WIDTH, SCREEN_HEIGHT, &Wire, OLED_RESET);
MFRC522 rfid(SS_PIN, RST_PIN);

// --- C. variáveis utilizadas ---
// Variáveis para regras de negócio e controle de estado
String last_uid = "";
unsigned long last_read_time = 0;
const unsigned long DEBOUNCE_DELAY = 10000; // Tempo de filtro para evitar múltiplas leituras seguidas (10s)
bool is_registered_in = false;              // Variável booleana para alternar entre Entrada e Saída
bool wasOffline = false;                    // Controla o estado anterior da conexão Wi-Fi

// Variáveis para configuração de rede e NTP
const char* ssid = "Wokwi-GUEST";
const char* password = "";
const char* ntpServer = "pool.ntp.org";
const long gmtOffset_sec = -10800; // GMT-3 (Horário de Brasília)
const int daylightOffset_sec = 0;

// Função para acionamento do atuador sonoro
void playBuzzerBeep(int frequency, int durationMs, int times = 1) {
  for (int i = 0; i < times; i++) {
    // --- F. acionamento de atuadores, quando houver ---
    // Acionamento do Buzzer via PWM (função tone)
    tone(BUZZER_PIN, frequency, durationMs);
    delay(durationMs + 50);
  }
}

// Função auxiliar para mostrar mensagens no display OLED
void showScreenMessage(String title, String line1, String line2, String line3) {
  display.clearDisplay();
  display.setTextSize(1);
  display.setTextColor(SSD1306_WHITE);
  display.setCursor(0, 0);
  display.print("== ");
  display.print(title);
  display.println(" ==");
  display.drawFastHLine(0, 10, 128, SSD1306_WHITE);
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
  Serial.println("\n[SCP] Inicializando Sistema...");

  // --- A. configuração dos pinos ---
  // Definindo se os pinos atuarão como entrada (sensores/botões) ou saída (atuadores)
  pinMode(LED_GREEN_PIN, OUTPUT);
  pinMode(LED_YELLOW_PIN, OUTPUT);
  pinMode(LED_RED_PIN, OUTPUT);
  pinMode(BUZZER_PIN, OUTPUT);
  pinMode(BTN_RESET_PIN, INPUT_PULLUP);

  // Inicializando atuadores desligados
  digitalWrite(LED_GREEN_PIN, LOW);
  digitalWrite(LED_YELLOW_PIN, LOW);
  digitalWrite(LED_RED_PIN, LOW);

  // --- B. inicialização dos sensores ---
  // Inicialização do barramento I2C e Display OLED
  if (!display.begin(SSD1306_SWITCHCAPVCC, 0x3C)) {
    Serial.println(F("[ERRO] Falha ao inicializar o Display OLED SSD1306!"));
    for (;;);
  }
  
  showScreenMessage("SCP - IFPE", "Inicializando...", "Conectando Wi-Fi", "Aguarde...");

  // --- B. inicialização dos sensores ---
  // Inicialização do barramento SPI e do Leitor RFID MFRC522
  SPI.begin();
  rfid.PCD_Init();
  Serial.println("[OK] Leitor RFID MFRC522 Inicializado com sucesso!");

  // Inicialização do Wi-Fi
  WiFi.begin(ssid, password);
  int wifi_attempts = 0;
  
  // --- E. estruturas condicionais utilizadas ---
  // Estrutura de repetição condicional para aguardar conexão
  while (WiFi.status() != WL_CONNECTED && wifi_attempts < 10) {
    delay(300);
    Serial.print(".");
    wifi_attempts++;
  }

  // --- E. estruturas condicionais utilizadas ---
  // Verifica se o dispositivo está conectado à internet
  if (WiFi.status() == WL_CONNECTED) {
    Serial.println("\n[OK] Wi-Fi Conectado!");
    configTime(gmtOffset_sec, daylightOffset_sec, ntpServer);
    wasOffline = false;
  } else {
    Serial.println("\n[AVISO] Conexão Wi-Fi falhou! Operando em modo offline.");
    // --- F. acionamento de atuadores, quando houver ---
    // Acionamento do LED Amarelo para indicar estado Offline
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

// Retorna o horário formatado utilizando NTP
String getFormattedTime() {
  struct tm timeinfo;
  if (!getLocalTime(&timeinfo)) {
    return "19:00:00"; // Fallback para demonstração sem internet
  }
  char timeStringBuff[15];
  strftime(timeStringBuff, sizeof(timeStringBuff), "%H:%M:%S", &timeinfo);
  return String(timeStringBuff);
}

// Processa a leitura da tag RFID
void processTag(String uid) {
  unsigned long now = millis();
  
  // --- E. estruturas condicionais utilizadas ---
  // Condicional de validação: Filtro de Debounce para não ler a mesma tag múltiplas vezes muito rápido
  if (uid == last_uid && (now - last_read_time < DEBOUNCE_DELAY)) {
    Serial.println("[DEBOUNCE] Leitura duplicada ignorada.");
    return;
  }

  last_uid = uid;
  last_read_time = now;
  is_registered_in = !is_registered_in; // Alterna entre check-in e check-out

  String hora = getFormattedTime();
  String tipo = is_registered_in ? "ENTRADA" : "SAIDA";
  
  Serial.print("[PRESENCA] UID: ");
  Serial.print(uid);
  Serial.print(" | Tipo: ");
  Serial.print(tipo);
  Serial.print(" | Hora: ");
  Serial.println(hora);

  // --- E. estruturas condicionais utilizadas ---
  if (WiFi.status() == WL_CONNECTED) {
    // --- F. acionamento de atuadores, quando houver ---
    digitalWrite(LED_GREEN_PIN, HIGH);
    playBuzzerBeep(2000, 150, 1); // 1 bipe de sucesso
    showScreenMessage("REGISTRO OK!", "UID: " + uid, tipo + " registrada", "Hora: " + hora);
    delay(2000);
    digitalWrite(LED_GREEN_PIN, LOW); // Desliga o atuador
  } else {
    // --- F. acionamento de atuadores, quando houver ---
    digitalWrite(LED_YELLOW_PIN, HIGH);
    playBuzzerBeep(1200, 100, 2); // 2 bipes para offline
    showScreenMessage("OFFLINE SAVED", "UID: " + uid, tipo + " (Offline)", "Hora: " + hora);
    delay(2000);
  }

  // Retorna tela padrão
  if (WiFi.status() == WL_CONNECTED) {
    showScreenMessage("SISTEMA PRONTO", "Aproxime seu cartao", "na entrada da sala", "Status: ONLINE");
  } else {
    showScreenMessage("SISTEMA PRONTO", "Aproxime seu cartao", "na entrada da sala", "Status: OFFLINE");
  }
}

void loop() {
  bool isOffline = (WiFi.status() != WL_CONNECTED);
  
  // --- E. estruturas condicionais utilizadas ---
  if (isOffline && !wasOffline) {
    digitalWrite(LED_YELLOW_PIN, HIGH);
    playBuzzerBeep(1200, 100, 2); 
    showScreenMessage("SISTEMA PRONTO", "Aproxime seu cartao", "Status", "OFFLINE");
    wasOffline = true;
  } else if (!isOffline && wasOffline) {
    digitalWrite(LED_YELLOW_PIN, LOW);
    playBuzzerBeep(2000, 150, 1); 
    showScreenMessage("SISTEMA PRONTO", "Aproxime seu cartao", "Status", "ONLINE");
    wasOffline = false;
  }

  // Verifica o botão de Reset (Simulando uma falha ou setup)
  if (digitalRead(BTN_RESET_PIN) == LOW) {
    // --- F. acionamento de atuadores, quando houver ---
    digitalWrite(LED_RED_PIN, HIGH);
    playBuzzerBeep(500, 800, 1); 
    showScreenMessage("RESET WI-FI", "Modo Portal Cativo", "Aguardando Setup", "IP: 192.168.4.1");
    delay(3000);
    digitalWrite(LED_RED_PIN, LOW);
    showScreenMessage("SISTEMA PRONTO", "Aproxime seu cartao", "na entrada da sala", isOffline ? "Status: OFFLINE" : "Status: ONLINE");
  }

  // --- D. comandos responsáveis pela leitura ---
  // PICC_IsNewCardPresent() é o comando que verifica se há um sensor/tag RFID na área de leitura
  if (!rfid.PICC_IsNewCardPresent()) {
    return;
  }

  // --- D. comandos responsáveis pela leitura ---
  // PICC_ReadCardSerial() é o comando que realiza a extração dos dados (UID) do cartão
  if (!rfid.PICC_ReadCardSerial()) {
    return;
  }

  String tagUID = "";
  for (byte i = 0; i < rfid.uid.size; i++) {
    tagUID += String(rfid.uid.uidByte[i] < 0x10 ? "0" : "");
    tagUID += String(rfid.uid.uidByte[i], HEX);
  }
  tagUID.toUpperCase();

  // Aciona a lógica de negócio principal
  processTag(tagUID);

  // --- D. comandos responsáveis pela leitura ---
  // Finaliza a leitura da tag atual para permitir novas leituras futuras
  rfid.PICC_HaltA();
  rfid.PCD_StopCrypto1();
}
