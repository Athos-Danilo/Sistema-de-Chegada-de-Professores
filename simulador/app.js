/* ==========================================================================
   SCP — Sistema de Chegada de Professores (Simulador IoT Web)
   Lógica JavaScript Modularizada & Simulação Completa de Firmware ESP32
   ========================================================================== */

// --- AUDIO CONTEXT PARA SINTETIZADOR DO BUZZER PASSIVO ---
const AudioContext = window.AudioContext || window.webkitAudioContext;
let audioCtx = null;

/**
 * Emite tom/frequência no buzzer do navegador usando Web Audio API
 * @param {number} freq Frequência em Hz (ex: 2000Hz agudo, 500Hz grave)
 * @param {number} durationMs Duração de cada bipe em milissegundos
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
    osc.type = 'square'; // Onda quadrada para som de buzzer piezoelétrico
    osc.frequency.value = freq;
    gain.gain.value = 0.05; // Volume seguro
    
    osc.connect(gain);
    gain.connect(audioCtx.destination);
    
    osc.start();
    setTimeout(() => {
      osc.stop();
      current++;
      setTimeout(step, 80);
    }, durationMs);
  }

  step();
}

// --- ESTADO GLOBAL DO EMBARCADO ESP32 ---
let isOnline = true; // Estado da Conexão Wi-Fi
let offlineBuffer = []; // Fila FIFO de leituras mantidas em Flash LittleFS
let lastUID = "";
let lastTime = 0;
const DEBOUNCE_MS = 5000; // 5s para facilidade de testes
const stateMap = {}; // Guarda estado Entrada/Saida por tag UID

// Lista de Tags Cadastradas no Banco de Dados
const registeredTags = {
  "4A8B12F0": "Prof. Lauro Alves",
  "9C3D45E6": "Prof. Leonardo",
  "1F8A92D4": "Prof. Eduardo",
  "3B7C11E9": "Profa. Simone",
  "7D5E88A2": "Prof. Carlos",
  "2E9F44C1": "Profa. Luciana"
};

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
 * Alterna o estado da rede Wi-Fi (Modo Online vs Modo Offline)
 * Quando a rede cai: Emite alerta sonoro no Buzzer e mantém o LED Amarelo ACESO.
 */
function toggleWifi() {
  isOnline = !isOnline;
  const toggleBtn = document.getElementById('wifiToggleBtn');
  const oledFooter = document.getElementById('oledFooter');
  const ledYellow = document.getElementById('ledYellow');

  if (isOnline) {
    toggleBtn.classList.remove('offline');
    toggleBtn.innerHTML = '<span>📶</span> Rede Wi-Fi: ONLINE';
    oledFooter.innerText = 'Wi-Fi: ONLINE';
    
    // Apaga o LED Amarelo ao restabelecer a conexão
    ledYellow.classList.remove('active');
    
    // Bipe sonoro de reconexão bem-sucedida (1500Hz, 150ms)
    playBeep(1500, 150, 1);
    logSerial(`[WIFI_MANAGER] Conexão Wi-Fi Reestabelecida com o Roteador!`, 'api');

    // Se houver registros acumulados na Flash, sincroniza em lote
    if (offlineBuffer.length > 0) {
      syncOfflineBatch();
    }
    resetOledScreen();

  } else {
    toggleBtn.classList.add('offline');
    toggleBtn.innerHTML = '<span>🌐</span> Rede Wi-Fi: OFFLINE (Queda de Sinal)';
    oledFooter.innerText = 'Wi-Fi: OFFLINE';
    
    // MANTÉM O LED AMARELO ACESO CONTINUAMENTE ENQUANTO ESTIVER OFFLINE!
    ledYellow.classList.add('active');
    
    // AVISO SONORO DE QUEDA DE REDE (2 bipes de alerta grave - 800Hz, 200ms)
    playBeep(800, 200, 2);

    document.getElementById('oledBody').innerHTML = `
      QUEDA DE SINAL!<br>
      REDE INDISPONIVEL<br>
      MODO OFFLINE ATIVO
    `;
    
    logSerial(`==========================================`);
    logSerial(`[WIFI_MANAGER] Queda de conexão Wi-Fi detectada!`, 'err');
    logSerial(`[SISTEMA] Alerta sonoro emitido e LED Amarelo ativado (Modo Flash LittleFS).`, 'err');

    setTimeout(() => {
      resetOledScreen();
    }, 3000);
  }
}

/**
 * Sincroniza em lote os registros salvos em memória Flash durante o período desconectado
 */
function syncOfflineBatch() {
  logSerial(`==========================================`);
  logSerial(`[BATCH_SYNC] Retransmitindo ${offlineBuffer.length} registros offline acumulados na Flash LittleFS...`, 'api');
  
  const batchPayload = JSON.stringify({
    device_id: "EMB-LAB-101",
    batch_count: offlineBuffer.length,
    records: offlineBuffer
  }, null, 2);

  logSerial(`[HTTP REST POST /api/v1/attendance/sync-batch] Payload:\n${batchPayload}`, 'api');
  logSerial(`[API RESPONSE] HTTP 200 OK - Lote offline sincronizado com sucesso!`, 'api');
  
  offlineBuffer = []; // Limpa a fila
}

/**
 * Processa a leitura de uma Tag RFID e aplica as regras de negócio de LED e Buzzer
 * @param {string} uid Hexadecimal da Tag
 * @param {string} name Nome do Professor (opcional)
 */
function swipeTag(uid, name = null) {
  const now = Date.now();
  
  // 1. FILTRO DE DEBOUNCE
  if (uid === lastUID && (now - lastTime < DEBOUNCE_MS)) {
    logSerial(`[DEBOUNCE] Leitura duplicada da Tag ${uid} ignorada (${Math.round((DEBOUNCE_MS - (now - lastTime))/1000)}s restantes).`, 'err');
    return;
  }

  lastUID = uid;
  lastTime = now;

  // 2. VALIDAÇÃO DE TAG CADASTRADA (CENÁRIO ERRO DE LEITURA / UNKNOWN TAG)
  if (!registeredTags[uid] && name !== "Desconhecida") {
    triggerErrorTag(uid);
    return;
  }

  const profName = registeredTags[uid] || name || "Professor";
  stateMap[uid] = !stateMap[uid]; // Alterna Entrada / Saída
  const eventType = stateMap[uid] ? "ENTRADA" : "SAÍDA";
  const timeStr = getCurrentTimeStr();

  logSerial(`==========================================`);
  logSerial(`[PRESENÇA] Tag RFID Lida: ${uid} (${profName})`);

  // 3. CENÁRIO ONLINE (VERDE + 1 BIPE CURTO - 2000Hz, 120ms)
  if (isOnline) {
    document.getElementById('ledGreen').classList.add('active');
    playBeep(2000, 120, 1);

    document.getElementById('oledBody').innerHTML = `
      REGISTRO OK!<br>
      UID: ${uid}<br>
      ${eventType} CONFIRMADA
    `;
    document.getElementById('oledFooter').innerText = `Hora: ${timeStr}`;

    const jsonPayload = JSON.stringify({
      device_id: "EMB-LAB-101",
      tag_uid: uid,
      event_type: eventType,
      is_offline_record: false,
      timestamp_str: timeStr
    }, null, 2);

    logSerial(`[HTTP REST POST /api/v1/attendance] ${jsonPayload}`, 'api');
    logSerial(`[API RESPONSE] HTTP 200 OK - Presença transmitida em tempo real!`, 'api');

    setTimeout(() => {
      document.getElementById('ledGreen').classList.remove('active');
      resetOledScreen();
    }, 2500);

  } else {
    // 4. CENÁRIO OFFLINE (AMARELO PISCA + 2 BIPES CURTOS - 1200Hz, 90ms)
    document.getElementById('ledYellow').classList.add('active');
    playBeep(1200, 90, 2);

    document.getElementById('oledBody').innerHTML = `
      SALVO OFFLINE<br>
      UID: ${uid}<br>
      Flash LittleFS
    `;
    document.getElementById('oledFooter').innerText = `Hora: ${timeStr}`;

    const record = {
      tag_uid: uid,
      event_type: eventType,
      read_timestamp: Math.floor(Date.now() / 1000),
      timestamp_str: timeStr
    };
    offlineBuffer.push(record);

    logSerial(`[LITTLEFS] Gravação em memória Flash interna executada com sucesso. CRC16 OK.`, 'err');
    logSerial(`[FLASH BUFFER] Registros pendentes aguardando reconexão: ${offlineBuffer.length}`, 'err');

    setTimeout(() => {
      resetOledScreen();
    }, 2500);
  }
}

/**
 * Simula a leitura de uma Tag Desconhecida / Não Cadastrada (Cenário Erro)
 * LED Vermelho + 1 Bipe longo (500Hz, 450ms)
 */
function triggerErrorTag(uid = "EE99AA11") {
  const timeStr = getCurrentTimeStr();
  document.getElementById('ledRed').classList.add('active');
  playBeep(500, 450, 1);

  document.getElementById('oledBody').innerHTML = `
    ERRO DE LEITURA!<br>
    TAG NAO REGISTRADA<br>
    UID: ${uid}
  `;
  document.getElementById('oledFooter').innerText = `Acesso Negado`;

  logSerial(`==========================================`);
  logSerial(`[ERRO_RFID] Tag Desconhecida / Não Autorizada detectada: ${uid}`, 'err');
  logSerial(`[SISTEMA] Registro de acesso negado emitido.`, 'err');

  setTimeout(() => {
    document.getElementById('ledRed').classList.remove('active');
    resetOledScreen();
  }, 2500);
}

/**
 * Simula o acionamento do Botão de Reset Wi-Fi (GPIO 14) / Portal Cativo
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
    resetOledScreen();
  }, 3000);
}

/**
 * Restaura o estado de repouso da tela OLED e mantém os LEDs de estado (ex: LED Amarelo se offline)
 */
function resetOledScreen() {
  document.getElementById('oledBody').innerHTML = `
    SISTEMA PRONTO<br>
    Aproxime o cartão<br>
    na entrada da sala
  `;
  document.getElementById('oledFooter').innerText = isOnline ? `Wi-Fi: ONLINE` : `Wi-Fi: OFFLINE`;
  
  // Mantém o LED Amarelo aceso continuamente enquanto a rede continuar offline!
  if (!isOnline) {
    document.getElementById('ledYellow').classList.add('active');
  } else {
    document.getElementById('ledYellow').classList.remove('active');
  }
}
