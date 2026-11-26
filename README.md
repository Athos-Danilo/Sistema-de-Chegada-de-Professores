# 📡 SCP — Sistema de Chegada de Professores

> **Projeto Multidisciplinar — IFPE Campus Garanhuns**  
> **Disciplinas Envolvidas:** Sistemas Embarcados | Web II | Banco de Dados II  
> **Professor Orientador:** Lauro Alves  
> **Equipe:** Athos, Joseph, José, Alessandro. 

---

## Visão Geral do Projeto

O **SCP (Sistema de Chegada de Professores)** é uma solução IoT projetada para automatizar o registro de presença e a localização dos docentes em salas de aula e laboratórios do campus em tempo real.

O sistema realiza a leitura sem contato de crachás/tags RFID via microcontrolador **ESP32**, enviando os dados via **API REST/JSON** para um backend em **Go** hospedado na **Vercel**, que persiste os registros em um banco **PostgreSQL** na **Neon** e dispara notificações automáticas para as turmas via **Bot do Telegram**.

```
+--------------------+      Wi-Fi      +--------------------+     Bot API     +--------------------+
|  Leitor Embarcado  | --------------> |  Backend em Go     | --------------> |   Bot Telegram     |
| (ESP32 + RFID I2C) |                 |     (Vercel)       |                 | (Alertas / Turmas) |
+--------------------+                 +---------+----------+                 +--------------------+
                                                 |
                                                 v
                                       +--------------------+
                                       |  PostgreSQL (Neon) |
                                       +--------------------+
```

---

## Arquitetura do Sistema Embarcado

### Hardware do Protótipo
* **Microcontrolador:** **ESP32 DevKit v1 (ESP-WROOM-32, 30 pinos)** com Wi-Fi integrado e Dual-Core 240 MHz.
* **Leitor RFID/NFC:** **Módulo RFID MFRC522 I2C (13.56 MHz, endereço `0x28`)**, conectado diretamente ao barramento I2C do ESP32.
* **Tags/Crachás:** Cartões/tags 13.56 MHz com UID único (Mifare 1KB / NTAG215).
* **Display Gráfico *(Especificação de Interface)*:** OLED 1.3" I2C 128x64 (Controlador SH1106/SSD1306, endereço `0x3C`), documentado para exibição de relógio, status de rede e confirmação de leitura (`PONTO REGISTRADO!`, UID e horário). *(Observação física: o painel de vidro do módulo OLED sofreu queima durante os testes de bancada, operando o protótipo físico com o ESP32 + Leitor RFID I2C).*
* **Feedback e Monitoramento:** Monitor Serial (115200 baud), LED onboard do ESP32 e notificação imediata no **Telegram**.

### Mapeamento de Pinos (ESP32 DevKit v1)

| Periférico | Pino Componente | Pino ESP32 (GPIO) | Protocolo / Função |
| :--- | :--- | :--- | :--- |
| **RFID MFRC522 (I2C)** | `5V` / `G` | **`VIN` (5V) / `GND`** | Alimentação (lado direito) |
| **RFID MFRC522 (I2C)** | `SDA` / `SCL` | **`GPIO 27 (D27)` / `GPIO 26 (D26)`** | I2C (`Wire1`, endereço `0x28`) |
| **Display OLED *(Especificação)*** | `VCC` / `GND` | `3V3` / `GND` | Alimentação (lado esquerdo) |
| **Display OLED *(Especificação)*** | `SDA` / `SCL` | **`GPIO 21 (D21)` / `GPIO 22 (D22)`** | I2C (`Wire`, endereço `0x3C`) |

> *(Para a montagem virtual simulada no Wokwi com LEDs, Buzzer e RFID em SPI, consulte [`Wokwi_project/diagram.json`](./Wokwi_project/diagram.json)).*

---

## Prototipagem e Simulação Virtual (Etapa R4)

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

## Estrutura do Repositório

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

## Licença e Entrega

Trabalho prático desenvolvido para as etapas **R3 (Componentes e Orçamento)** e **R4 (Montagem Virtual e Testes de Viabilidade)** da disciplina de **Sistemas Embarcados** do IFPE.
