# 📡 SCP — Sistema de Chegada de Professores

> **Projeto Multidisciplinar — IFPE Campus Garanhuns**  
> **Disciplinas Envolvidas:** Sistemas Embarcados | Web II | Banco de Dados II  
> **Professor Orientador:** Lauro Alves  
> **Equipe:** Alessandro, Athos Danilo, Joseph e José E.  

---

## 📌 Visão Geral do Projeto

O **SCP (Sistema de Chegada de Professores)** é uma solução IoT inteligente projetada para automatizar o registro de presença e a localização dos docentes em salas de aula e laboratórios do campus em tempo real.

O sistema consiste na leitura sem contato de crachás/tags RFID via microcontrolador **ESP32**, comunicando-se via **API REST/JSON** com um backend central para cruzamento de grade horária e envio automático de notificações via **WhatsApp** para as turmas.

```
+-------------------+       Wi-Fi       +-------------------+       API       +-------------------+
| Leitor Embarcado  | ----------------> |  Backend (Go/Py)  | --------------> |  WhatsApp Notify  |
| (RFID+OLED+Buzzer)|                   +---------+---------+                 +-------------------+
+-------------------+                             |
                                                  v
                                        +-------------------+
                                        |  Banco Poliglota  |
                                        |   (SQL + NoSQL)   |
                                        +---------+---------+
```

---

## 🚀 Arquitetura do Sistema Embarcado

### Hardware Utilizado (Etapa R3)
* **Microcontrolador:** ESP32-S3 DevKitC-1 (Dual-Core 240 MHz, 520KB SRAM, Wi-Fi e suporte nativo a FreeRTOS)
* **Leitor RFID/NFC:** Módulo MFRC522 (13.56 MHz, Comunicação SPI)
* **Tags/Crachás:** Cartões PVC NTAG215 (13.56 MHz com UID único)
* **Display Gráfico:** OLED 1.3" I2C 128x64 (Controlador SSD1306/SH1106)
* **Feedback Visual e Sonoro:** LEDs indicativos (Verde: Sucesso; Amarelo: Offline; Vermelho: Erro) e Buzzer Passivo (PWM)
* **Entradas Auxiliares:** Botão Físico no GPIO 14 (Simulação de Reset Wi-Fi / Portal Cativo)

### Mapeamento de Pinos (ESP32 Pinout)

| Periférico | Pino Componente | Pino ESP32 (GPIO) | Protocolo / Função |
| :--- | :--- | :--- | :--- |
| **RFID MFRC522** | SDA (SS) / RST / SCK / MOSI / MISO | **GPIO 5, 4, 18, 23, 19** | SPI Master |
| **Display OLED** | SDA / SCL | **GPIO 21, 22** | I2C Data & Clock |
| **LEDs Feedback** | Verde / Amarelo / Vermelho | **GPIO 13, 2, 15** | Saída Digital (PWM) |
| **Buzzer Passivo** | Signal | **GPIO 12** | Gerador de Tom (PWM) |
| **Botão Reset** | Terminal A | **GPIO 14** | Entrada Digital (Pull-up) |

---

## 🖥️ Prototipagem e Simulação Virtual (Etapa R4)

Este repositório contém a validação virtual completa do protótipo antes da montagem dos componentes físicos:

### 1. Simulador Web Interativo (HTML5 + CSS + JS)
Localizado na pasta [`/simulador`](./simulador), permite executar e demonstrar a solução diretamente no navegador com:
* **Audio Synthesizer (Web Audio API):** Som real do buzzer no som do PC.
* **Display OLED Animado:** Atualização dinâmica de horário NTP, alternância de Entrada/Saída e UID lido.
* **Terminal Serial em Tempo Real:** Visualização da carga útil JSON transmitida via REST API.
* 🔗 **Para testar localmente:** Basta abrir o arquivo [`simulador/index.html`](./simulador/index.html) em qualquer navegador.

### 2. Projeto de Simulação no Wokwi
Localizado na pasta [`/wokwi_project`](./wokwi_project):
* `sketch.ino`: Firmware C++ completo desenvolvido sobre ESP-IDF / Arduino ESP32.
* `diagram.json`: Esquema elétrico e ligações dos componentes no simulador Wokwi.
* `libraries.txt`: Lista de bibliotecas utilizadas.

---

## 📂 Estrutura do Repositório

```text
.
├── DOCUMENTACAO.md               # Documentação técnica completa e detalhada da arquitetura
├── simulador/                    # Simulador Web Interativo em HTML5/CSS/JS (Etapa R4)
│   ├── index.html
│   ├── style.css
│   └── app.js
├── wokwi_project/                # Projeto de simulação física do Wokwi
│   ├── sketch.ino
│   ├── diagram.json
│   ├── libraries.txt
│   └── wokwi.toml
└── README.md                     # Este arquivo de apresentação
```

---

## 📄 Licença e Entrega

Trabalho prático desenvolvido para as etapas **R3 (Componentes e Orçamento)** e **R4 (Montagem Virtual e Testes de Viabilidade)** da disciplina de **Sistemas Embarcados** do IFPE.
