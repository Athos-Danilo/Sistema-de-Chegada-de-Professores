# SCP — Sistema de Chegada de Professores (Firmware IoT)

Este diretório contém o código-fonte do firmware desenvolvido para o microcontrolador **ESP32** na disciplina de Sistemas Embarcados.

## Arquitetura do Hardware Embarcado

* **Microcontrolador:** **ESP32 DevKit v1 (ESP-WROOM-32, 30 pinos)** com Wi-Fi integrado.
* **Sensor RFID:** **Módulo MFRC522 via I2C (13.56 MHz, endereço `0x28`)**, conectado diretamente nos pinos **`D27` (SDA)**, **`D26` (SCL)**, **`VIN` (5V)** e **`GND`**.
* **Display OLED 1.3" I2C (SH1106/SSD1306, `0x3C`):** Especificado no projeto para exibição de relógio ao vivo e confirmação visual na entrada da sala (`D21`/`D22`). *(No protótipo físico em bancada, o módulo sofreu queima física do painel OLED, operando o conjunto com o ESP32 + Leitor RFID I2C).*
* **Integração em Nuvem:** O ESP32 envia as leituras via Wi-Fi (**HTTP REST / JSON**) para o **Backend em Go** hospedado na **Vercel**, que persiste os dados no **PostgreSQL (Neon)** e envia as notificações para as turmas via **Bot do Telegram**.

## Descrição do Funcionamento (Firmware)

Ao aproximar o crachá/tag do leitor RFID, o sistema:
1. Extrai o **UID** da tag e aplica um filtro de _debounce_ temporal para evitar leituras duplicadas acidentais.
2. Registra o horário sincronizado via **NTP / relógio interno**.
3. Transmite os dados da leitura via Wi-Fi (**HTTP REST / JSON**) para a API em **Go** na **Vercel**, que valida o registro no **PostgreSQL (Neon)** e notifica o grupo correspondente no **Telegram**.
4. *(Referência R5)* No código-base da entrega R5 também constam as marcações estruturais dos requisitos **A a F**:
   * **A.** Configuração dos pinos (`pinMode`).
   * **B.** Inicialização dos sensores (`Wire.begin` / `rfid.PCD_Init()`).
   * **C.** Variáveis utilizadas (flags de estado, strings e temporizadores).
   * **D.** Comandos responsáveis pela leitura (`PICC_IsNewCardPresent()`, `PICC_ReadCardSerial()`).
   * **E.** Estruturas condicionais (`if/else` e `while` para validações e rede).
   * **F.** Acionamento de feedback/atuadores.

---
*Projeto desenvolvido por Athos, Joseph, José e Alessandro para o IFPE - Campus Garanhuns.*
