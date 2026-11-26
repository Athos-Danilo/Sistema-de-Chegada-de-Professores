// Firmware IoT do Sistema de Chegada de Professores (SCP) para ESP32 DevKit v1.
// Realiza leitura de crachás RFID via barramento I2C (MFRC522), sincroniza relógio NTP,
// mantém fila off-line em caso de queda de Wi-Fi e envia eventos via HTTPS REST para a API na Vercel.
// Projeto desenvolvido para o IFPE Campus Garanhuns (Athos, Joseph, José e Alessandro).

#include <WiFi.h>
#include <WiFiClientSecure.h>
#include <HTTPClient.h>
#include <Wire.h>
#include <MFRC522_I2C.h>
#include <time.h>

// --- CONFIGURAÇÕES DE REDE E NUVEM ---
const char* WIFI_SSID         = "SEU_WIFI_AQUI";
const char* WIFI_SENHA        = "SUA_SENHA_AQUI";

const char* URL_BASE_API      = "https://scp-three-blue.vercel.app";
const char* ID_DISPOSITIVO    = "EMB-LAB-01";
const char* TOKEN_DISPOSITIVO = "scp-token-seguro-2026";
const char* VERSAO_FIRMWARE   = "scp-esp32-i2c-1.0.0";

const char* SERVIDOR_NTP      = "pool.ntp.org";
const long  FUSO_HORARIO_SEG  = -10800; // GMT-3 (America/Recife)

// --- A. configuração dos pinos ---
const uint8_t PINO_SDA_I2C      = 27;   // Pino SDA conectado ao leitor MFRC522 I2C
const uint8_t PINO_SCL_I2C      = 26;   // Pino SCL conectado ao leitor MFRC522 I2C
const uint8_t PINO_RST_RFID     = 25;   // Pino auxiliar mantido em LOW antes do PCD_Init()
const uint8_t ENDERECO_RFID_I2C = 0x28; // Endereço I2C padrão do módulo MFRC522
const uint8_t PINO_LED_STATUS   = 2;    // LED azul integrado na placa ESP32 DevKit v1

// --- B. inicialização dos sensores ---
MFRC522_I2C leitorRfid(ENDERECO_RFID_I2C, PINO_RST_RFID, &Wire);

// --- C. variáveis utilizadas ---
struct RegistroPendente {
  String uidTag;
  int64_t carimboTempo;
};

const size_t MAX_FILA_OFFLINE              = 32;
const unsigned long TEMPO_DEBOUNCE_MS      = 10000; // Evita leituras repetidas do mesmo crachá por 10s
const unsigned long INTERVALO_HEARTBEAT_MS = 60000; // Telemetria a cada 60 segundos
const unsigned long INTERVALO_WIFI_MS      = 15000; // Tentativa de reconexão Wi-Fi a cada 15s

RegistroPendente filaOffline[MAX_FILA_OFFLINE];
size_t totalPendentesOffline = 0;

String ultimoUidLido                  = "";
unsigned long instanteUltimaLeituraMs = 0;
unsigned long ultimoHeartbeatMs       = 0;
unsigned long ultimaTentativaWifiMs   = 0;
bool ntpSincronizado                  = false;

// --- F. acionamento de atuadores/feedback ---
// Pisca o LED integrado da placa para confirmar leitura, envio ou erro sem travar o sistema.
void sinalizarLedStatus(uint8_t repeticoes, int duracaoMs) {
  for (uint8_t i = 0; i < repeticoes; i++) {
    digitalWrite(PINO_LED_STATUS, HIGH);
    delay(duracaoMs);
    digitalWrite(PINO_LED_STATUS, LOW);
    delay(duracaoMs);
  }
}

// Formata os bytes do UID lido pelo MFRC522 no padrão hexadecimal AA:BB:CC:DD.
String formatarUidTag() {
  String uidFormatado = "";
  for (byte i = 0; i < leitorRfid.uid.size; i++) {
    if (i > 0) {
      uidFormatado += ":";
    }
    if (leitorRfid.uid.uidByte[i] < 0x10) {
      uidFormatado += "0";
    }
    uidFormatado += String(leitorRfid.uid.uidByte[i], HEX);
  }
  uidFormatado.toUpperCase();
  return uidFormatado;
}

// Obtém o timestamp Unix atual via NTP (retorna 0 caso o relógio ainda não esteja sincronizado).
int64_t obterTimestampUnixAtual() {
  time_t agora = time(nullptr);
  if (agora < 1700000000LL) {
    return 0;
  }
  return static_cast<int64_t>(agora);
}

// Conecta ao Wi-Fi configurado e sincroniza o horário via NTP.
void conectarWifiESincronizarNtp() {
  if (WiFi.status() == WL_CONNECTED) {
    return;
  }

  Serial.printf("[WIFI] Conectando na rede \"%s\"...\n", WIFI_SSID);
  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_SENHA);

  unsigned long inicio = millis();
  while (WiFi.status() != WL_CONNECTED && millis() - inicio < 12000) {
    delay(300);
    Serial.print(".");
  }
  Serial.println();

  if (WiFi.status() == WL_CONNECTED) {
    Serial.print("[WIFI] Conectado! IP local: ");
    Serial.println(WiFi.localIP());

    if (!ntpSincronizado) {
      configTime(FUSO_HORARIO_SEG, 0, SERVIDOR_NTP, "a.st1.ntp.br");
      Serial.print("[NTP] Sincronizando relogio");
      for (int tentativa = 0; tentativa < 20; tentativa++) {
        if (obterTimestampUnixAtual() > 0) {
          ntpSincronizado = true;
          break;
        }
        delay(250);
        Serial.print(".");
      }
      Serial.println(ntpSincronizado ? " OK!" : " aguardando em segundo plano.");
    }
  } else {
    Serial.println("[WIFI] Sem conexao no momento. Operando em modo off-line.");
  }
}

// Executa requisição HTTPS POST autenticada para a API REST em Go na Vercel.
// Retorna o código HTTP recebido (> 0) ou valor negativo em caso de erro de rede/Wi-Fi.
int enviarRequisicaoPostJson(const String& rota, const String& payloadJson, String& respostaCorpo) {
  if (WiFi.status() != WL_CONNECTED) {
    return -1;
  }

  WiFiClientSecure clienteSeguro;
  clienteSeguro.setInsecure();

  HTTPClient http;
  String urlCompleta = String(URL_BASE_API) + rota;
  if (!http.begin(clienteSeguro, urlCompleta)) {
    Serial.println("[HTTP] Falha ao iniciar cliente HTTPS.");
    return -2;
  }

  http.setTimeout(10000);
  http.addHeader("Content-Type", "application/json");
  http.addHeader("X-Device-Token", TOKEN_DISPOSITIVO);

  int codigoHttp = http.POST(payloadJson);
  if (codigoHttp > 0) {
    respostaCorpo = http.getString();
    Serial.printf("[HTTP] POST %s -> %d | %s\n", rota.c_str(), codigoHttp, respostaCorpo.c_str());
  } else {
    Serial.printf("[HTTP] Erro na requisicao POST %s: %s\n", rota.c_str(), http.errorToString(codigoHttp).c_str());
  }

  http.end();
  return codigoHttp;
}

// Enfileira uma leitura em memória caso a conexão Wi-Fi ou nuvem falhe temporariamente.
void enfileirarLeituraOffline(const String& uidTag, int64_t carimboTempo) {
  if (totalPendentesOffline >= MAX_FILA_OFFLINE) {
    Serial.println("[OFFLINE] Fila cheia! Descartando leitura mais antiga.");
    for (size_t i = 1; i < MAX_FILA_OFFLINE; i++) {
      filaOffline[i - 1] = filaOffline[i];
    }
    totalPendentesOffline = MAX_FILA_OFFLINE - 1;
  }

  filaOffline[totalPendentesOffline].uidTag = uidTag;
  filaOffline[totalPendentesOffline].carimboTempo = carimboTempo;
  totalPendentesOffline++;
  Serial.printf("[OFFLINE] Leitura salva na fila local (%u pendentes).\n", static_cast<unsigned>(totalPendentesOffline));
}

// Envia todas as leituras armazenadas na fila off-line em lote para a Vercel.
void sincronizarFilaOffline() {
  if (totalPendentesOffline == 0 || WiFi.status() != WL_CONNECTED) {
    return;
  }

  String payload = "{";
  payload += "\"device_id\":\"" + String(ID_DISPOSITIVO) + "\",";
  payload += "\"batch_count\":" + String(totalPendentesOffline) + ",";
  payload += "\"records\":[";

  for (size_t i = 0; i < totalPendentesOffline; i++) {
    if (i > 0) {
      payload += ",";
    }
    payload += "{\"tag_uid\":\"" + filaOffline[i].uidTag + "\",";
    payload += "\"read_timestamp\":" + String(static_cast<long long>(filaOffline[i].carimboTempo)) + "}";
  }
  payload += "]}";

  String resposta;
  int codigoHttp = enviarRequisicaoPostJson("/api/v1/attendance/sync-batch", payload, resposta);
  if (codigoHttp >= 200 && codigoHttp < 300) {
    Serial.printf("[OFFLINE] Lote sincronizado com sucesso (%u registros).\n", static_cast<unsigned>(totalPendentesOffline));
    totalPendentesOffline = 0;
    sinalizarLedStatus(2, 100);
  }
}

// Envia uma leitura em tempo real para a API ou armazena na fila off-line se houver queda de rede.
void processarEnvioPresenca(const String& uidTag) {
  int64_t carimboTempo = obterTimestampUnixAtual();
  int rssi = (WiFi.status() == WL_CONNECTED) ? WiFi.RSSI() : 0;

  String payload = "{";
  payload += "\"device_id\":\"" + String(ID_DISPOSITIVO) + "\",";
  payload += "\"tag_uid\":\"" + uidTag + "\",";
  payload += "\"read_timestamp\":" + String(static_cast<long long>(carimboTempo)) + ",";
  payload += "\"rssi\":" + String(rssi) + ",";
  payload += "\"is_offline_record\":false";
  payload += "}";

  String resposta;
  int codigoHttp = enviarRequisicaoPostJson("/api/v1/attendance", payload, resposta);
  if (codigoHttp >= 200 && codigoHttp < 300) {
    Serial.println("[SCP] Presenca confirmada na nuvem e notificada no Telegram!");
    sinalizarLedStatus(3, 80);
  } else if (codigoHttp >= 400 && codigoHttp < 500) {
    Serial.printf("[SCP] Leitura recusada pela API (HTTP %d). Verifique cadastro da tag/token.\n", codigoHttp);
    sinalizarLedStatus(5, 60);
  } else {
    enfileirarLeituraOffline(uidTag, carimboTempo);
    sinalizarLedStatus(1, 450);
  }
}

// Envia telemetria periódica de saúde (heartbeat) do ESP32 para monitoramento no PostgreSQL.
void enviarTelemetriaHeartbeat() {
  if (WiFi.status() != WL_CONNECTED) {
    return;
  }

  unsigned long tempoAtivoSeg = millis() / 1000;
  String payload = "{";
  payload += "\"device_id\":\"" + String(ID_DISPOSITIVO) + "\",";
  payload += "\"firmware_version\":\"" + String(VERSAO_FIRMWARE) + "\",";
  payload += "\"uptime_seconds\":" + String(tempoAtivoSeg) + ",";
  payload += "\"wifi_rssi\":" + String(WiFi.RSSI()) + ",";
  payload += "\"free_heap_bytes\":" + String(ESP.getFreeHeap()) + ",";
  payload += "\"offline_pending_count\":" + String(totalPendentesOffline);
  payload += "}";

  String resposta;
  enviarRequisicaoPostJson("/api/v1/telemetry/heartbeat", payload, resposta);
}

void setup() {
  Serial.begin(115200);
  delay(200);
  Serial.println("\n=== SCP - Sistema de Chegada de Professores (ESP32 + MFRC522 I2C) ===");

  // --- A. configuração dos pinos ---
  pinMode(PINO_LED_STATUS, OUTPUT);
  digitalWrite(PINO_LED_STATUS, LOW);

  // Mantém PINO_RST_RFID em LOW antes do PCD_Init para evitar travamento no SoftReset I2C.
  pinMode(PINO_RST_RFID, OUTPUT);
  digitalWrite(PINO_RST_RFID, LOW);

  // --- B. inicialização dos sensores ---
  Wire.begin(PINO_SDA_I2C, PINO_SCL_I2C);
  leitorRfid.PCD_Init();
  Serial.print("[RFID] Versao do leitor I2C: ");
  leitorRfid.PCD_DumpVersionToSerial();

  conectarWifiESincronizarNtp();
  ultimoHeartbeatMs = millis();

  if (WiFi.status() == WL_CONNECTED) {
    enviarTelemetriaHeartbeat();
  }
  Serial.println("[SCP] Sistema pronto! Aproxime um cracha RFID para registrar presenca.");
}

void loop() {
  unsigned long agoraMs = millis();

  // --- E. estruturas condicionais de reconexão e sincronização ---
  if (WiFi.status() != WL_CONNECTED && (agoraMs - ultimaTentativaWifiMs >= INTERVALO_WIFI_MS)) {
    ultimaTentativaWifiMs = agoraMs;
    conectarWifiESincronizarNtp();
  }

  if (WiFi.status() == WL_CONNECTED && totalPendentesOffline > 0) {
    sincronizarFilaOffline();
  }

  if (WiFi.status() == WL_CONNECTED && (agoraMs - ultimoHeartbeatMs >= INTERVALO_HEARTBEAT_MS)) {
    ultimoHeartbeatMs = agoraMs;
    enviarTelemetriaHeartbeat();
  }

  // --- D. comandos responsáveis pela leitura do sensor RFID ---
  if (!leitorRfid.PICC_IsNewCardPresent()) {
    delay(50);
    return;
  }
  if (!leitorRfid.PICC_ReadCardSerial()) {
    delay(50);
    return;
  }

  String uidLido = formatarUidTag();
  leitorRfid.PICC_HaltA();
  leitorRfid.PCD_StopCrypto1();

  // --- E. estrutura condicional para filtro de debounce (10 segundos) ---
  if (uidLido == ultimoUidLido && (agoraMs - instanteUltimaLeituraMs < TEMPO_DEBOUNCE_MS)) {
    Serial.printf("[DEBOUNCE] Tag %s ignorada (leitura repetida em menos de 10s).\n", uidLido.c_str());
    delay(300);
    return;
  }

  ultimoUidLido = uidLido;
  instanteUltimaLeituraMs = agoraMs;

  Serial.printf("\n[RFID] Cracha detectado! UID: %s\n", uidLido.c_str());
  processarEnvioPresenca(uidLido);
}
