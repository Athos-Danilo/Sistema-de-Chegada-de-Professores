/* ==========================================================================
   SCP — Sistema de Chegada de Professores (Simulador IoT Web)
   Lógica JavaScript Modularizada & Sintetizador de Áudio Premium
   ========================================================================== */

// --- AUDIO CONTEXT PARA SINTETIZADOR DO BUZZER PASSIVO DE ALTA FIDELIDADE ---
const AudioContext = window.AudioContext || window.webkitAudioContext;
let audioCtx = null;

function getAudioContext() {
  if (!audioCtx) {
    audioCtx = new AudioContext();
  }
  if (audioCtx.state === 'suspended') {
    audioCtx.resume();
  }
  return audioCtx;
}

/**
 * Toca tom com curva ADSR (Envelope de volume) para som mais bonito e natural
 */
function playTone(freq, durationMs, type = 'sine', volume = 0.08) {
  const ctx = getAudioContext();
  const osc = ctx.createOscillator();
  const gain = ctx.createGain();

  osc.type = type;
  osc.frequency.setValueAtTime(freq, ctx.currentTime);

  // Envelope ADSR suave
  gain.gain.setValueAtTime(0, ctx.currentTime);
  gain.gain.linearRampToValueAtTime(volume, ctx.currentTime + 0.01);
  gain.gain.exponentialRampToValueAtTime(0.0001, ctx.currentTime + (durationMs / 1000));

  osc.connect(gain);
  gain.connect(ctx.destination);

  osc.start();
  osc.stop(ctx.currentTime + (durationMs / 1000));
}

/**
 * Emite som no buzzer simulando padrões sonoros de IoT Premium
 */
function playAudioFeedback(mode) {
  const buzzerEl = document.getElementById('buzzer');
  buzzerEl.classList.add('beeping');

  setTimeout(() => {
    buzzerEl.classList.remove('beeping');
  }, 400);

  switch (mode) {
    case 'success':
      // Chime duplo harmônico (C6 -> E6)
      playTone(1046.5, 100, 'sine', 0.08);
      setTimeout(() => playTone(1318.51, 140, 'sine', 0.08), 80);
      break;

    case 'offline_save':
      // Tom marimba duplo de gravação em Flash (880Hz -> 659Hz)
      playTone(880, 90, 'triangle', 0.09);
      setTimeout(() => playTone(659.25, 110, 'triangle', 0.09), 90);
      break;

    case 'error':
      // Tom grave de erro/acesso negado (220Hz -> 180Hz)
      playTone(220, 200, 'sawtooth', 0.07);
      setTimeout(() => playTone(180, 250, 'sawtooth', 0.07), 100);
      break;

    case 'wifi_drop':
      // Alerta sonoro de queda de rede (800Hz -> 500Hz)
      playTone(800, 150, 'square', 0.06);
      setTimeout(() => playTone(500, 200, 'square', 0.06), 120);
      break;

    case 'wifi_connect':
      // Arpejo ascendente de reconexão (C5 -> E5 -> G5)
      playTone(523.25, 80, 'sine', 0.07);
      setTimeout(() => playTone(659.25, 80, 'sine', 0.07), 70);
      setTimeout(() => playTone(783.99, 120, 'sine', 0.07), 140);
      break;

    case 'reset':
      // Som grave de acionamento do botão reset (600Hz)
      playTone(600, 300, 'triangle', 0.08);
      break;
  }
}

// --- ESTADO GLOBAL DO EMBARCADO ESP32 ---
let isOnline = true;
let offlineBuffer = [];
let lastUID = "";
let lastTime = 0;
const DEBOUNCE_MS = 5000;
const stateMap = {};

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
 * Retorna a saudação baseada no horário do dia
 */
function getTimeBasedGreeting() {
  const hour = new Date().getHours();
  if (hour >= 5 && hour < 12) {
    return "Bom dia";
  } else if (hour >= 12 && hour < 18) {
    return "Boa tarde";
  } else {
    return "Boa noite";
  }
}

/**
 * Gera mensagem rica e dinâmica de boas-vindas para Entrada ou Saída
 */
function generateDynamicWelcomeMessage(profName, isEntry) {
  const greeting = getTimeBasedGreeting();
  
  const entryGreetings = [
    `${greeting}, ${profName}!<br>Seja bem-vindo(a)!<br><span style="color:#86efac;">ENTRADA REGISTRADA</span>`,
    `${greeting}, ${profName}!<br>Ótima aula no campus!<br><span style="color:#86efac;">PRESENCA CONFIRMADA</span>`,
    `${greeting}, ${profName}!<br>Bom trabalho hoje!<br><span style="color:#86efac;">ENTRADA EM TEMPO REAL</span>`,
    `${greeting}, ${profName}!<br>Turma notificada.<br><span style="color:#86efac;">SALA ATIVA</span>`
  ];

  const exitGreetings = [
    `Até logo, ${profName}!<br>Tenha um bom descanso!<br><span style="color:#38bdf8;">SAIDA CONFIRMADA</span>`,
    `${greeting}, ${profName}!<br>Aula finalizada com sucesso!<br><span style="color:#38bdf8;">SAIDA REGISTRADA</span>`,
    `Bom descanso, ${profName}!<br>Até a próxima aula!<br><span style="color:#38bdf8;">SAIDA REGISTRADA</span>`,
    `Até breve, ${profName}!<br>Presença concluída.<br><span style="color:#38bdf8;">SALA LIBERADA</span>`
  ];

  const list = isEntry ? entryGreetings : exitGreetings;
  return list[Math.floor(Math.random() * list.length)];
}

/**
 * Retorna o horário atual formatado (HH:MM:SS)
 */
function getCurrentTimeStr() {
  const now = new Date();
  return now.toTimeString().split(' ')[0];
}

/**
 * Atualiza o relógio NTP do display OLED a cada 1 segundo
 */
function updateOledClock() {
  const clockEl = document.getElementById('oledClock');
  const greetingEl = document.getElementById('oledGreeting');
  if (clockEl) {
    clockEl.innerText = getCurrentTimeStr();
  }
  if (greetingEl) {
    greetingEl.innerText = `${getTimeBasedGreeting()}!`;
  }
}
setInterval(updateOledClock, 1000);

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
 */
function toggleWifi() {
  isOnline = !isOnline;
  const toggleBtn = document.getElementById('wifiToggleBtn');
  const oledFooter = document.getElementById('oledFooter');
  const ledYellow = document.getElementById('ledYellow');

  if (isOnline) {
    toggleBtn.classList.remove('offline');
    toggleBtn.innerHTML = `
      <svg style="width:18px;height:18px;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0"></path></svg>
      Rede Wi-Fi: ONLINE
    `;
    oledFooter.innerText = 'Wi-Fi: ONLINE';
    
    ledYellow.classList.remove('active');
    playAudioFeedback('wifi_connect');
    logSerial(`[WIFI_MANAGER] Conexão Wi-Fi Reestabelecida com o Roteador!`, 'api');

    if (offlineBuffer.length > 0) {
      syncOfflineBatch();
    }
    resetOledScreen();

  } else {
    toggleBtn.classList.add('offline');
    toggleBtn.innerHTML = `
      <svg style="width:18px;height:18px;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 5.636a9 9 0 010 12.728m0 0l-2.829-2.829m2.829 2.829L21 21M15.536 8.464a5 5 0 010 7.072m0 0l-2.829-2.829m-4.243 2.829a4.978 4.978 0 01-1.414-2.83m-1.414 5.657a9 9 0 01-2.121-3.536m3.536 3.536L3 21m0 0l2.829-2.829"></path></svg>
      Rede Wi-Fi: OFFLINE (Queda de Sinal)
    `;
    oledFooter.innerText = 'Wi-Fi: OFFLINE';
    
    ledYellow.classList.add('active');
    playAudioFeedback('wifi_drop');

    updateOledBody(`
      <span style="color:#fef08a;">QUEDA DE SINAL!</span><br>
      REDE INDISPONIVEL<br>
      MODO OFFLINE ATIVO
    `);
    
    logSerial(`==========================================`);
    logSerial(`[WIFI_MANAGER] Queda de conexão Wi-Fi detectada!`, 'err');
    logSerial(`[SISTEMA] Alerta sonoro emitido e LED Amarelo ativado (Modo Flash LittleFS).`, 'err');

    setTimeout(() => {
      resetOledScreen();
    }, 3000);
  }
}

/**
 * Atualiza o corpo do display OLED com efeito suave de transição
 */
function updateOledBody(htmlContent) {
  const bodyEl = document.getElementById('oledBody');
  if (!bodyEl) return;
  
  bodyEl.classList.add('updating');
  setTimeout(() => {
    bodyEl.innerHTML = htmlContent;
    bodyEl.classList.remove('updating');
  }, 120);
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
  
  offlineBuffer = [];
}

/**
 * Processa a leitura de uma Tag RFID e exibe saudações personalizadas
 */
function swipeTag(uid, name = null) {
  const now = Date.now();
  
  if (uid === lastUID && (now - lastTime < DEBOUNCE_MS)) {
    logSerial(`[DEBOUNCE] Leitura duplicada da Tag ${uid} ignorada (${Math.round((DEBOUNCE_MS - (now - lastTime))/1000)}s restantes).`, 'err');
    return;
  }

  lastUID = uid;
  lastTime = now;

  if (!registeredTags[uid] && name !== "Desconhecida") {
    triggerErrorTag(uid);
    return;
  }

  const profName = registeredTags[uid] || name || "Professor";
  stateMap[uid] = !stateMap[uid];
  const isEntry = stateMap[uid];
  const eventType = isEntry ? "ENTRADA" : "SAÍDA";
  const timeStr = getCurrentTimeStr();

  const welcomeMsg = generateDynamicWelcomeMessage(profName, isEntry);

  logSerial(`==========================================`);
  logSerial(`[PRESENÇA] Tag RFID Lida: ${uid} (${profName})`);

  if (isOnline) {
    document.getElementById('ledGreen').classList.add('active');
    playAudioFeedback('success');

    updateOledBody(welcomeMsg);
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
    }, 3200);

  } else {
    document.getElementById('ledYellow').classList.add('active');
    playAudioFeedback('offline_save');

    updateOledBody(`
      <span style="color:#fef08a;">SALVO OFFLINE</span><br>
      ${profName}<br>
      ${eventType} na Flash
    `);
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
    }, 3200);
  }
}

/**
 * Simula a leitura de uma Tag Desconhecida / Não Cadastrada (Cenário Erro)
 */
function triggerErrorTag(uid = "EE99AA11") {
  const timeStr = getCurrentTimeStr();
  document.getElementById('ledRed').classList.add('active');
  playAudioFeedback('error');

  updateOledBody(`
    <span style="color:#f87171;">ERRO DE LEITURA!</span><br>
    TAG NAO REGISTRADA<br>
    UID: ${uid}
  `);
  document.getElementById('oledFooter').innerText = `Acesso Negado`;

  logSerial(`==========================================`);
  logSerial(`[ERRO_RFID] Tag Desconhecida / Não Autorizada detectada: ${uid}`, 'err');
  logSerial(`[SISTEMA] Registro de acesso negado emitido.`, 'err');

  setTimeout(() => {
    document.getElementById('ledRed').classList.remove('active');
    resetOledScreen();
  }, 3000);
}

/**
 * Simula o acionamento do Botão de Reset Wi-Fi (GPIO 14) / Portal Cativo
 */
function triggerResetButton() {
  document.getElementById('ledRed').classList.add('active');
  playAudioFeedback('reset');

  updateOledBody(`
    <span style="color:#f87171;">RESET WI-FI</span><br>
    Modo Portal Cativo<br>
    Aguardando Setup
  `);
  document.getElementById('oledFooter').innerText = `IP: 192.168.4.1`;

  logSerial(`[GPIO 14] Botão de Reset Wi-Fi Pressionado!`, 'err');
  logSerial(`[ESP32] Entrando em Modo Access Point (SSID: Presenca-Device-Setup)...`, 'err');

  setTimeout(() => {
    document.getElementById('ledRed').classList.remove('active');
    resetOledScreen();
  }, 3500);
}

/**
 * Restaura a tela de repouso com relógio e saudações dinâmicas
 */
function resetOledScreen() {
  const greeting = getTimeBasedGreeting();
  updateOledBody(`
    SISTEMA PRONTO<br>
    Aproxime o cartão<br>
    na entrada da sala
  `);
  document.getElementById('oledGreeting').innerText = `${greeting}!`;
  document.getElementById('oledFooter').innerText = isOnline ? `Wi-Fi: ONLINE` : `Wi-Fi: OFFLINE`;
  
  if (!isOnline) {
    document.getElementById('ledYellow').classList.add('active');
  } else {
    document.getElementById('ledYellow').classList.remove('active');
  }
}

// Inicialização da tela ao carregar
updateOledClock();
resetOledScreen();
