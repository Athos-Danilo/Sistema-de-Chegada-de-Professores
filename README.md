# 📡 SCP — Sistema de Chegada de Professores

> **Projeto Multidisciplinar — IFPE Campus Garanhuns**  
> **Disciplinas Envolvidas:** Sistemas Embarcados | Web II | Banco de Dados II  
> **Professor Orientador:** Lauro Alves  
> **Equipe:** Athos, Joseph, José, Alessandro. 

---

## 📌 Visão Geral do Projeto

O **SCP (Sistema de Chegada de Professores)** é uma solução IoT inteligente projetada para automatizar o registro de presença e a localização dos docentes em salas de aula e laboratórios do campus em tempo real.

O sistema consiste na leitura sem contato de crachás/tags RFID via microcontrolador **ESP32**, comunicando-se via **API REST/JSON** com um backend central com **banco de dados único** para cruzamento de grade horária e envio automático de notificações via **Telegram (Bot API)** para as turmas.

> 🔄 **Atualização de Escopo (Versão Final para Apresentação):**  
> Em relação à proposta inicial, a arquitetura foi simplificada para focar em estabilidade e entrega funcional: foram removidos o **Frontend web** e a **Persistência Poliglota (SQL + NoSQL)**, adotando-se um **Banco de Dados único** e substituindo o **WhatsApp** pela **API de Bot do Telegram**. No hardware físico, o protótipo final opera com **ESP32 DevKit v1 + Leitor RFID I2C** *(o Display OLED 1.3" permanece documentado abaixo conforme planejado, pois sofreu dano físico no painel durante os testes de bancada)*.

```
+-------------------+       Wi-Fi       +-------------------+     Bot API     +-------------------+
| Leitor Embarcado  | ----------------> |    Backend REST   | --------------> |  Alerta Telegram  |
| (ESP32 + RFID I2C)|                   +---------+---------+                 +-------------------+
+-------------------+                             |
                                                  v
                                        +-------------------+
                                        |  Banco de Dados   |
                                        |      (Único)      |
                                        +-------------------+
```

---

## 🚀 Arquitetura do Sistema Embarcado

### Hardware Utilizado: Planejado (R3) vs. Protótipo Físico Real
* **Microcontrolador:** **ESP32 DevKit v1 (ESP-WROOM-32, 30 pinos)** com Wi-Fi e Dual-Core 240 MHz. *(Inicialmente previsto o ESP32-S3 DevKitC-1, que apresentou defeito físico durante os testes de bancada e foi substituído pelo DevKit v1)*.
* **Leitor RFID/NFC:** **Módulo RFID MFRC522 I2C (13.56 MHz, endereço `0x28`)**, conectado diretamente via barramento I2C. *(Nos documentos R3/R4 também constam referências à versão SPI simulada no Wokwi)*.
* **Tags/Crachás:** Cartões/tags 13.56 MHz com UID único (Mifare 1KB / NTAG215).
* **Display Gráfico *(Documentado / Não ativo no protótipo final)*:** OLED 1.3" I2C 128x64 (Controlador SH1106/SSD1306, endereço `0x3C`). Seria utilizado para exibir relógio ao vivo, status de conexão e confirmação visual (`PONTO REGISTRADO!`, UID e horário) ao aproximar a tag; porém, após queima física do painel/circuito de alta tensão durante os testes, optou-se por manter apenas o ESP32 + RFID no protótipo físico.
* **Feedback no Protótipo Final:** Monitor Serial (115200 baud), LED interno da placa e confirmação direta via notificação no **Telegram** *(LEDs externos RGB e buzzer passivo permaneceram apenas na simulação virtual R4/Wokwi)*.

### Mapeamento de Pinos — Protótipo Físico Real (ESP32 DevKit v1)

| Periférico | Pino Componente | Pino ESP32 (GPIO) | Protocolo / Função |
| :--- | :--- | :--- | :--- |
| **RFID MFRC522 (I2C)** | `5V` / `G` | **`VIN` (5V) / `GND`** | Alimentação (lado direito) |
| **RFID MFRC522 (I2C)** | `SDA` / `SCL` | **`GPIO 27 (D27)` / `GPIO 26 (D26)`** | I2C (`Wire1`, endereço `0x28`) |
| **Display OLED *(Planejado)*** | `VCC` / `GND` | `3V3` / `GND` | Alimentação (lado esquerdo) |
| **Display OLED *(Planejado)*** | `SDA` / `SCL` | **`GPIO 21 (D21)` / `GPIO 22 (D22)`** | I2C (`Wire`, endereço `0x3C`) |

> *(Para consultar o pinout original da simulação virtual no Wokwi com LEDs, Buzzer e RFID em SPI, veja [`wokwi_project/diagram.json`](./wokwi_project/diagram.json)).*

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
├── Código ESP-32/                # Firmware principal do ESP32 (Entrega R5 e versão física final)
├── Entregas/                     # PDFs das entregas R1/R2/R3/R4/R5 da disciplina
├── DOCUMENTACAO.md               # Documentação técnica completa e detalhada da arquitetura
├── Simulador/                    # Simulador Web Interativo em HTML5/CSS/JS (Etapa R4)
│   ├── index.html
│   ├── style.css
│   └── app.js
├── Wokwi_project/                # Projeto de simulação física do Wokwi
│   ├── sketch.ino
│   ├── diagram.json
│   ├── libraries.txt
│   └── wokwi.toml
└── README.md                     # Este arquivo de apresentação
```

---

## 📄 Licença e Entrega

Trabalho prático desenvolvido para as etapas **R3 (Componentes e Orçamento)** e **R4 (Montagem Virtual e Testes de Viabilidade)** da disciplina de **Sistemas Embarcados** do IFPE.
