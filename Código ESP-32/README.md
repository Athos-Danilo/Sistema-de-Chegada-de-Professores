# SCP — Sistema de Chegada de Professores (Firmware IoT)

Este diretório contém o código-fonte principal desenvolvido para o microcontrolador **ESP32** (originado na etapa **R5 - Programação e Leitura dos Sensores** e evoluído para a montagem física real de apresentação da disciplina de Sistemas Embarcados).

## Montagem Física Real vs. Escopo Planejado

Na versão física real para apresentação, o hardware embarcado foi consolidado para máxima confiabilidade em bancada:
* **Microcontrolador:** **ESP32 DevKit v1 (ESP-WROOM-32, 30 pinos)** *(substituindo o ESP32-S3 inicial, que apresentou defeito físico)*.
* **Sensor RFID:** **Módulo MFRC522 via I2C (13.56 MHz, endereço `0x28`)**, conectado diretamente nos pinos **`D27` (SDA)**, **`D26` (SCL)**, **`VIN` (5V)** e **`GND`**.
* **Display OLED 1.3" I2C (SH1106/SSD1306, `0x3C`) — *Planejado*:** Foi projetado e validado em código para exibir relógio ao vivo e confirmação visual na entrada da sala (`D21`/`D22`), mas sofreu queima física do painel durante os testes de bancada. Por isso, a operação física final utiliza **apenas o ESP32 + Leitor RFID I2C**, mantendo a documentação do display preservada como especificação de projeto.
* **Integração Externa (Escopo Enxuto):** Em vez de Frontend web, banco poliglota e WhatsApp, o sistema embarcado comunica-se via Wi-Fi com um **Backend REST** ligado a um **Banco de Dados único**, disparando alertas automáticos via **Bot do Telegram**.

## Descrição do Funcionamento (Firmware)

Ao aproximar o crachá/tag do leitor RFID, o sistema:
1. Extrai o **UID** da tag e aplica um filtro de _debounce_ temporal (para evitar leituras duplicadas acidentais).
2. Registra o horário exato sincronizado via **NTP / relógio interno**.
3. Envia os dados da leitura via Wi-Fi (**HTTP REST / JSON**) para o Backend, que valida o registro no banco de dados e notifica a turma via **Telegram**.
4. *(Referência R5)* No código inicial da entrega R5, também constam comentadas as estruturas para Display OLED, LEDs (Verde, Amarelo, Vermelho) e Buzzer passivo, organizadas nos requisitos **A a F**:
   * **A.** Configuração dos pinos (`pinMode`).
   * **B.** Inicialização dos sensores (`Wire.begin` / `rfid.PCD_Init()`).
   * **C.** Variáveis utilizadas (flags de estado, strings e temporizadores).
   * **D.** Comandos responsáveis pela leitura (`PICC_IsNewCardPresent()`, `PICC_ReadCardSerial()`).
   * **E.** Estruturas condicionais (`if/else` e `while` para validações e rede).
   * **F.** Acionamento de atuadores/feedback (Serial, LEDs/Buzzer na referência R5).

---
*Projeto desenvolvido por Athos, Alessandro, Joseph e José E. para o IFPE - Campus Garanhuns.*
