/* ==========================================================================
   SCP — Sistema de Chegada de Professores (Simulador IoT Web)
   Lógica JavaScript Modularizada & Simulação de Firmware ESP32
   ========================================================================== */

// --- AUDIO CONTEXT PARA SINTETIZADOR DO BUZZER ---
const AudioContext = window.AudioContext || window.webkitAudioContext;
let audioCtx = null;

/**
 * Emite som no buzzer do navegador usando Web Audio API
 * @param {number} freq Frequência em Hz
 * @param {number} durationMs Duração em milissegundos
 * @param {number} count Número de bipes
 */
function playBeep(freq, durationMs, count = 1) {
  if (!audioCtx) {
    audioCtx = new AudioContext();
  }

  const buzzerEl = document.getElementById('buzzer');
  buzzerEl.classList.add('beeping');

  let current = 0;
  function step() {
    if (current >= count) {
      buzzerEl.classList.remove('beeping');
      return;
    }

    const osc = audioCtx.createOscillator();
    const gain = audioCtx.createGain();
    osc.type = 'square';
    osc.frequency.value = freq;
    gain.gain.value = 0.05; // Volume confortável
    
    osc.connect(gain);
    gain.connect(audioCtx.destination);
    
    osc.start();
    setTimeout(() => {
      osc.stop();
      current++;
      setTimeout(step, 60);
    }, durationMs);
  }

  step();
}

// --- ESTADO DO FIRMWARE ESP32 ---
let lastUID = "";
let lastTime = 0;
const DEBOUNCE_MS = 5000; // 5 segundos para facilidade de testes
const stateMap = {}; // Guarda estado Entrada/Saida por tag UID

/**
 * Retorna o horário atual formatado (HH:MM:SS)
 */
function getCurrentTimeStr() {
  const now = new Date();
  return now.toTimeString().split(' ')[0];
}

/**
 * Imprime mensagens formatadas no Monitor Serial Virtual
 */
function logSerial(msg, type = 'normal') {
  const logContainer = document.getElementById('serialLog');
  const line = document.createElement('div');
  line.className = `serial-line ${type}`;
  line.innerText = msg;
  logContainer.appendChild(line);
  logContainer.scrollTop = logContainer.scrollHeight;
}

/**
 * Limpa o console serial
 */
function clearSerial() {
  document.getElementById('serialLog').innerHTML = 
    '<div class="serial-line">[SISTEMA] Console limpo. Aguardando leitura de tag RFID...</div>';
}

/**
 * Simula a leitura e aproximação de uma Tag RFID no módulo RC522
 * @param {string} uid UID Hexadecimal da Tag
 * @param {string} name Nome do Professor associado
 */
function swipeTag(uid, name) {
  const now = Date.now();
  
  // Filtro de Debounce para ignorar leituras consecutivas instantâneas
  if (uid === lastUID && (now - lastTime < DEBOUNCE_MS)) {
    logSerial(`[DEBOUNCE] Leitura duplicada da Tag ${uid} ignorada (${Math.round((DEBOUNCE_MS - (now - lastTime))/1000)}s restantes).`, 'err');
    return;
  }

  lastUID = uid;
  lastTime = now;

  // Alterna o estado do registro (Entrada vs Saída)
  stateMap[uid] = !stateMap[uid];
  const eventType = stateMap[uid] ? "ENTRADA" : "SAÍDA";
  const timeStr = getCurrentTimeStr();

  // Ativa o LED Verde e o Buzzer (1 bipe curto)
  document.getElementById('ledGreen').classList.add('active');
  playBeep(2000, 120, 1);

  // Atualiza a tela OLED
  document.getElementById('oledBody').innerHTML = `
    REGISTRO OK!<br>
    UID: ${uid}<br>
    ${eventType} CONFIRMADA
  `;
  document.getElementById('oledFooter').innerText = `Hora: ${timeStr}`;

  // Logs detalhados no Terminal Serial
  logSerial(`==========================================`);
  logSerial(`[PRESENÇA] Tag RFID Detectada: ${uid} (${name})`);
  logSerial(`[PRESENÇA] Tipo de Evento: ${eventType}`);
  logSerial(`[PRESENÇA] Timestamp NTP: ${timeStr}`);
  
  // Payload JSON REST formatado
  const jsonPayload = JSON.stringify({
    device_id: "EMB-LAB-101",
    tag_uid: uid,
    event_type: eventType,
    read_timestamp: Math.floor(Date.now() / 1000),
    timestamp_str: timeStr
  }, null, 2);

  logSerial(`[HTTP REST POST /api/v1/attendance] Payload:\n${jsonPayload}`, 'api');
  logSerial(`[API RESPONSE] HTTP 200 OK - Presença confirmada no Backend!`, 'api');

  // Retorna o estado normal do display e LEDs após 2.5 segundos
  setTimeout(() => {
    document.getElementById('ledGreen').classList.remove('active');
    document.getElementById('oledBody').innerHTML = `
      SISTEMA PRONTO<br>
      Aproxime o cartão<br>
      na entrada da sala
    `;
    document.getElementById('oledFooter').innerText = `Wi-Fi: ONLINE`;
  }, 2500);
}

/**
 * Simula o acionamento do Botão de Reset Wi-Fi (GPIO 14)
 */
function triggerResetButton() {
  document.getElementById('ledRed').classList.add('active');
  playBeep(600, 300, 1);

  document.getElementById('oledBody').innerHTML = `
    RESET WI-FI<br>
    Modo Portal Cativo<br>
    Aguardando Setup
  `;
  document.getElementById('oledFooter').innerText = `IP: 192.168.4.1`;

  logSerial(`[GPIO 14] Botão de Reset Wi-Fi Pressionado!`, 'err');
  logSerial(`[ESP32] Entrando em Modo Access Point (SSID: Presenca-Device-Setup)...`, 'err');

  setTimeout(() => {
    document.getElementById('ledRed').classList.remove('active');
    document.getElementById('oledBody').innerHTML = `
      SISTEMA PRONTO<br>
      Aproxime o cartão<br>
      na entrada da sala
    `;
    document.getElementById('oledFooter').innerText = `Wi-Fi: ONLINE`;
  }, 3000);
}
