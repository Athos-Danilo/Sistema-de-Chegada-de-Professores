# SCP — Sistema de Chegada de Professores (Firmware IoT)

Este diretório contém o código-fonte principal desenvolvido para o microcontrolador **ESP32** referente à etapa **R5 - Programação e Leitura dos Sensores** da disciplina de Sistemas Embarcados.

## Descrição do Funcionamento

O sistema realiza a leitura de identificação de professores utilizando um **Leitor RFID MFRC522** (sensor). Ao aproximar o crachá/tag, o sistema:
1. Extrai o UID da tag e verifica o _debounce_ temporal (para evitar leituras duplicadas acidentais).
2. Fornece feedback visual através do **Display OLED SSD1306** e **LEDs indicativos** (Verde, Amarelo, Vermelho).
3. Fornece feedback sonoro através de um **Buzzer passivo** utilizando sinais PWM.
4. Conecta-se via Wi-Fi para sincronização de data/hora (NTP) e interage através de um payload JSON formatado, alternando seu estado entre "Online" e "Offline" quando há falha na rede.

## Requisitos da Entrega (Itens A a F)

O arquivo `sketch.ino` está fortemente comentado em pontos-chave, indicando onde cada requisito foi implementado, seguindo estritamente as marcações:
* **A.** Configuração dos pinos (`pinMode`).
* **B.** Inicialização dos sensores (`SPI.begin`, `rfid.PCD_Init()`, `display.begin`).
* **C.** Variáveis utilizadas (flags de estado, strings e arrays, e controle de temporizadores).
* **D.** Comandos responsáveis pela leitura (`rfid.PICC_IsNewCardPresent()`, `rfid.PICC_ReadCardSerial()`).
* **E.** Estruturas condicionais (como `if/else` e `while` para validações e gerenciamento de rede).
* **F.** Acionamento de atuadores (`digitalWrite` nos LEDs e `tone` no Buzzer).

---
*Projeto desenvolvido por Athos, Alessandro, Joseph e José E. para o IFPE - Campus Garanhuns.*
