
import { Events, Call as $Call } from '@wailsio/runtime';
import { BotService } from './bindings/message-go/cmd/gui/index.js';

// Import UI Components
import { SidebarHtml } from './src/components/Sidebar.js';
import { DashboardHtml } from './src/pages/Dashboard/Dashboard.js';
import { ConversationsHtml } from './src/pages/Conversations/Conversations.js';
import { PromptHtml } from './src/pages/Prompt/Prompt.js';
import { BlockedHtml } from './src/pages/Blocked/Blocked.js';
import { SettingsHtml } from './src/pages/Settings/Settings.js';
import { MonitorHtml } from './src/pages/Monitor/Monitor.js';
import { TestBotHtml } from './src/pages/TestBot/TestBot.js';

import { t, setLang, currentLang } from './src/i18n.js';

// Mount UI First (will be done inside init)
function renderUI() {
  document.getElementById('sidebar').innerHTML = SidebarHtml(t);
  document.getElementById('main-content').innerHTML = `
    ${DashboardHtml(t)}
    ${ConversationsHtml(t)}
    ${PromptHtml(t)}
    ${BlockedHtml(t)}
    ${SettingsHtml(t)}
    ${MonitorHtml(t)}
    ${TestBotHtml(t)}
  `;

  // Attach Event Listeners after DOM is ready
  document.getElementById('btn-start').addEventListener('click', () => window.startBot());
  document.getElementById('btn-stop').addEventListener('click', () => window.stopBot());
  document.getElementById('btn-save-prompt').addEventListener('click', () => window.savePrompt());
  document.getElementById('btn-save-settings').addEventListener('click', () => window.saveSettings());
  document.getElementById('cfg-provider').addEventListener('change', () => window.onProviderChange());
  document.getElementById('prompt-editor').addEventListener('input', updateCharCount);
  
  // WhatsApp Settings Buttons
  const btnWALogin = document.getElementById('btn-settings-wa-login');
  if (btnWALogin) btnWALogin.addEventListener('click', () => window.startBot());
  
  const btnWALogout = document.getElementById('btn-settings-wa-logout');
  if (btnWALogout) btnWALogout.addEventListener('click', () => window.logoutWhatsApp());

  if (document.getElementById('btn-prompt-test')) {
    document.getElementById('btn-prompt-test').addEventListener('click', () => {
      const msg = document.getElementById('prompt-test-input').value.trim();
      if (msg) {
        document.getElementById('prompt-test-input').value = '';
        navigateTo('testbot');
        sendTestMessage(msg);
      }
    });
    
    document.getElementById('prompt-test-input').addEventListener('keypress', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        document.getElementById('btn-prompt-test').click();
      }
    });
  }

  if (document.getElementById('btn-testbot-send')) {
    document.getElementById('btn-testbot-send').addEventListener('click', () => {
      const msg = document.getElementById('testbot-input').value.trim();
      if (msg) {
        document.getElementById('testbot-input').value = '';
        sendTestMessage(msg);
      }
    });

    document.getElementById('testbot-input').addEventListener('keypress', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        document.getElementById('btn-testbot-send').click();
      }
    });
  }

  const quickSelect = document.getElementById('quick-model-select');
  if (quickSelect) {
    quickSelect.addEventListener('change', async (e) => {
      const newModel = e.target.value;
      if (!newModel) return;
      try {
        const cfg = await BotService.GetConfig();
        cfg.OLLAMA_MODEL = newModel;
        await BotService.SaveConfig(cfg);
        
        const settingsModel = document.getElementById('cfg-model');
        if (settingsModel) {
          if (![...settingsModel.options].some(o => o.value === newModel)) {
            settingsModel.add(new Option(newModel, newModel));
          }
          settingsModel.value = newModel;
        }
        
        const status = await BotService.GetStatus();
        if (status) Events.Emit('status_change', status);
        
        showToast(t('js_settings_ok'));
      } catch (err) {
        showToast('❌ ' + err, true);
      }
    });
  }

  document.querySelectorAll('.nav-item').forEach(item => {
    item.addEventListener('click', e => {
      e.preventDefault();
      navigateTo(item.dataset.page);
    });
  });
}

// ── State ─────────────────────────────────────────────────────────────────────
let currentContact = null;
let currentPage = 'dashboard';
let allContacts = [];

// ── Toast ──────────────────────────────────────────────────────────────────────
let toastTimer = null;
function showToast(msg, isError = false) {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.classList.toggle('error', isError);
  el.classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => el.classList.remove('show'), 2800);
}

// ── Navigation ────────────────────────────────────────────────────────────────

// The event listeners for navigation are now inside renderUI(), we just keep the router logic
window.navigateTo = navigateTo;
function navigateTo(page) {
  if (currentPage === page) return;

  document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('active'));
  const navEl = document.querySelector(`[data-page="${page}"]`);
  if (navEl) navEl.classList.add('active');

  document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
  const pageEl = document.getElementById(`page-${page}`);
  if (pageEl) pageEl.classList.add('active');

  currentPage = page;

  if (page === 'conversations') loadContacts();
  if (page === 'prompt') loadPrompt();
  if (page === 'blocked') loadBlocked();
  if (page === 'settings') loadSettings();
  if (page === 'testbot') renderTestChat();
}

// ── Test Bot ──────────────────────────────────────────────────────────────────
let testBotHistory = [];

window.clearTestChat = function() {
  testBotHistory = [];
  renderTestChat();
};

async function sendTestMessage(msg) {
  testBotHistory.push({ role: 'user', content: msg, created_at: new Date().toISOString() });
  renderTestChat();
  
  // Show typing indicator
  const msgsEl = document.getElementById('testbot-msgs-container');
  if (msgsEl) {
    const row = document.createElement('div');
    row.className = 'msg-row typing-indicator';
    row.innerHTML = `<div class="msg-bubble assistant">Yazıyor...</div>`;
    msgsEl.appendChild(row);
    msgsEl.scrollTop = msgsEl.scrollHeight;
  }

  try {
    // We send testBotHistory mapped to what backend expects
    const mappedHistory = testBotHistory.slice(0, -1).map(m => ({ phone: m.phone || "", role: m.role, content: m.content, created_at: m.created_at || "" }));
    let result = "";
    if (BotService.TestPrompt) {
      result = await BotService.TestPrompt(mappedHistory, msg);
    } else {
      result = await $Call.ByName("main.BotService.TestPrompt", mappedHistory, msg);
    }
    
    // Remove typing indicator
    const typing = msgsEl?.querySelector('.typing-indicator');
    if (typing) typing.remove();

    testBotHistory.push({ role: 'assistant', content: result, created_at: new Date().toISOString() });
    renderTestChat();
  } catch (err) {
    const typing = msgsEl?.querySelector('.typing-indicator');
    if (typing) typing.remove();
    showToast('❌ Test error: ' + err, true);
  }
}

function renderTestChat() {
  const msgsEl = document.getElementById('testbot-msgs-container');
  const lastMsgEl = document.getElementById('testbot-contact-last');
  
  if (!msgsEl) return;

  if (testBotHistory.length === 0) {
    msgsEl.innerHTML = `
      <div class="conv-empty">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z" />
        </svg>
        <span>Test mesajı yazarak asistanı deneyin</span>
      </div>
    `;
    if (lastMsgEl) lastMsgEl.textContent = '...';
    return;
  }

  msgsEl.innerHTML = '';
  testBotHistory.forEach(m => {
    const row = document.createElement('div');
    row.className = 'msg-row';
    row.innerHTML = `
      <div class="msg-bubble ${m.role}">${escapeHtml(m.content)}</div>
      <div class="msg-time ${m.role === 'assistant' ? 'right' : ''}">${formatDate(m.created_at)}</div>
    `;
    msgsEl.appendChild(row);
  });
  
  msgsEl.scrollTop = msgsEl.scrollHeight;
  if (lastMsgEl) {
    lastMsgEl.innerHTML = escapeHtml(testBotHistory[testBotHistory.length - 1].content);
  }
}


// ── Bot Start / Stop ──────────────────────────────────────────────────────────
window.startBot = async function () {
  setStatus('waiting', t('js_starting'), t('js_connecting'));
  setBtnState(false, false);

  try {
    const result = await BotService.Start();
    if (result === 'waiting_qr') {
      setStatus('waiting', t('js_qr_wait'), t('js_qr_hint'));

      // Dashboard QR - spinner göster, eski resmi temizle
      const qrImg = document.getElementById('qr-img');
      const qrSpinner = document.getElementById('qr-spinner');
      if (qrImg) { qrImg.style.display = 'none'; qrImg.src = ''; }
      if (qrSpinner) qrSpinner.style.display = 'block';
      document.getElementById('qr-area').style.display = 'block';

      // Settings QR - spinner göster, eski resmi temizle
      const sQrArea = document.getElementById('settings-qr-area');
      const sQrImg = document.getElementById('settings-qr-img');
      const sQrSpinner = document.getElementById('settings-qr-spinner');
      if (sQrImg) { sQrImg.style.display = 'none'; sQrImg.src = ''; }
      if (sQrSpinner) sQrSpinner.style.display = 'block';
      if (sQrArea) sQrArea.style.display = 'block';

    } else if (result === 'connected' || result === 'already_running') {
      setStatus('running', t('js_running'), t('js_running_hint'));
      document.getElementById('qr-area').style.display = 'none';
      setBtnState(false, true);
      showToast(t('js_bot_started'));
    }
  } catch (err) {
    setStatus('stopped', t('js_error'), err.toString());
    setBtnState(true, false);
    showToast('❌ ' + err.toString(), true);
  }
};

window.stopBot = async function () {
  try {
    await BotService.Stop();
    setStatus('stopped', t('status_stopped'), t('js_stop_hint'));
    setBtnState(true, false);
    document.getElementById('qr-area').style.display = 'none';
    showToast(t('js_bot_stopped'));
  } catch (err) {
    showToast('❌ ' + err.toString(), true);
  }
};

function setBtnState(startEnabled, stopEnabled) {
  document.getElementById('btn-start').disabled = !startEnabled;
  document.getElementById('btn-stop').disabled = !stopEnabled;
}

let _logoutConfirmPending = false;
let _logoutConfirmTimer = null;

window.logoutWhatsApp = async function () {
  const btn = document.getElementById('btn-settings-wa-logout');

  if (!_logoutConfirmPending) {
    // İlk tıklama: onay bekleniyor
    _logoutConfirmPending = true;
    if (btn) {
      btn.textContent = t('wa_logout_confirm2') || '⚠️ Emin misiniz? Tekrar tıkla';
      btn.style.background = '#c0392b';
    }
    _logoutConfirmTimer = setTimeout(() => {
      _logoutConfirmPending = false;
      if (btn) {
        btn.textContent = t('wa_btn_logout');
        btn.style.background = '';
      }
    }, 3000);
    return;
  }

  // İkinci tıklama: onayla ve logout yap
  clearTimeout(_logoutConfirmTimer);
  _logoutConfirmPending = false;
  if (btn) {
    btn.textContent = t('wa_btn_logout');
    btn.style.background = '';
    btn.disabled = true;
  }

  try {
    await BotService.LogoutWhatsApp();
    showToast(t('wa_logout_success'));
    setTimeout(() => window.startBot(), 500);
  } catch (err) {
    showToast('❌ ' + err.toString(), true);
  } finally {
    if (btn) btn.disabled = false;
  }
};


function setStatus(state, label, sub) {
  document.getElementById('status-label').textContent = label;
  document.getElementById('status-sub').textContent = sub;

  // Icon wrap class
  const wrap = document.getElementById('status-icon-wrap');
  wrap.className = 'status-icon-wrap ' + state;

  // Sidebar mini dot
  const dot = document.getElementById('status-dot-mini');
  dot.className = 'pulse-dot ' + (state === 'running' ? '' : state === 'waiting' ? 'warning' : 'idle');

  // Icon inner SVG
  const svgEl = document.getElementById('status-svg');
  if (svgEl) {
    if (state === 'running') {
      svgEl.innerHTML = '<path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />';
    } else if (state === 'waiting') {
      svgEl.innerHTML = '<path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />';
    } else {
      // Stopped
      svgEl.innerHTML = '<path stroke-linecap="round" stroke-linejoin="round" d="M8.25 3v1.5M4.5 8.25H3m18 0h-1.5M4.5 12H3m18 0h-1.5m-15 3.75H3m18 0h-1.5M8.25 19.5V21M12 3v1.5m0 15V21m3.75-18v1.5m0 15V21m-9-1.5h10.5a2.25 2.25 0 002.25-2.25V6.75a2.25 2.25 0 00-2.25-2.25H6.75A2.25 2.25 0 004.5 6.75v10.5a2.25 2.25 0 002.25 2.25zm.75-12h9v9h-9v-9z" />';
    }
  }
}

// ── Events ───────────────────────────────────────────────────────────────────
// Wails v3: Events.On callback'i { data: <gerçek_değer> } objesi alır
Events.On('qr_code', (evt) => {
  const b64 = evt?.data ?? evt; // geriye dönük uyumluluk için fallback
  if (!b64) return;
  
  // Dashboard QR
  const img = document.getElementById('qr-img');
  const spinner = document.getElementById('qr-spinner');
  if (img) img.src = b64;
  if (img) img.style.display = 'block';
  if (spinner) spinner.style.display = 'none';
  const qrArea = document.getElementById('qr-area');
  if (qrArea) qrArea.style.display = 'block';

  // Settings QR
  const sImg = document.getElementById('settings-qr-img');
  const sSpinner = document.getElementById('settings-qr-spinner');
  if (sImg) sImg.src = b64;
  if (sImg) sImg.style.display = 'block';
  if (sSpinner) sSpinner.style.display = 'none';
  const sQrArea = document.getElementById('settings-qr-area');
  if (sQrArea) sQrArea.style.display = 'block';
});

Events.On('qr_timeout', async () => {
  console.log('QR code expired, generating a new one...');
  try {
    await BotService.Stop();
    window.startBot();
  } catch(e) {}
});

Events.On('wa_connected', () => {
  setStatus('running', t('js_running'), t('js_running_hint'));
  setBtnState(false, true);
  document.getElementById('qr-area').style.display = 'none';
  const sQrArea = document.getElementById('settings-qr-area');
  if (sQrArea) sQrArea.style.display = 'none';
  showToast(t('js_wa_connected'));
});

Events.On('status_change', (evt) => {
  // Wails v3 event verisi evt.data içinde geliyor
  const status = evt?.data ?? evt;
  if (!status) return;
  const { running, connected, model } = status;

  const quickSelect = document.getElementById('quick-model-select');
  if (quickSelect && model) {
    if (![...quickSelect.options].some(o => o.value === model)) {
       quickSelect.add(new Option(model, model));
    }
    quickSelect.value = model;
  }

  if (running && connected) {
    setStatus('running', t('js_running'), t('js_running_hint'));
    setBtnState(false, true);
  } else if (running && !connected) {
    setStatus('waiting', t('js_qr_wait'), t('js_qr_hint'));
    setBtnState(false, false);
  } else {
    setStatus('stopped', t('status_stopped'), t('js_stop_hint'));
    setBtnState(true, false);
  }

  // Update Settings Page Status
  const sStatus = document.getElementById('settings-wa-status');
  const sLoginBtn = document.getElementById('btn-settings-wa-login');
  const sLogoutBtn = document.getElementById('btn-settings-wa-logout');

  if (sStatus) {
    if (connected) {
      sStatus.textContent = t('wa_connected');
      sStatus.className = 'status-badge running';
      if (sLoginBtn) sLoginBtn.style.display = 'none';
      if (sLogoutBtn) sLogoutBtn.style.display = 'block';
    } else {
      sStatus.textContent = running ? t('js_qr_wait') : t('wa_disconnected');
      sStatus.className = 'status-badge ' + (running ? 'waiting' : 'stopped');
      if (sLoginBtn) sLoginBtn.style.display = running ? 'none' : 'block';
      if (sLogoutBtn) sLogoutBtn.style.display = 'none';
    }
  }
});

Events.On('sys_log', (evt) => {
  let line = evt?.data ?? evt;
  if (!line || typeof line !== 'string') return;
  const container = document.getElementById('sys-logs-container');
  if (container) {
    const div = document.createElement('div');
    div.className = 'log-row';
    
    // Extract time "[15:04:05]" if exists, though my Go backend appends it.
    let timeStr = new Date().toLocaleTimeString('tr-TR', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    const match = line.match(/^\[(.*?)\] (.*)/);
    if(match) {
        timeStr = match[1];
        line = match[2];
    }
    
    let type = 'SYS';
    let typeClass = 'system';
    
    if (line.includes('[GELEN]')) {
       type = 'IN'; typeClass = 'in'; line = line.replace('[GELEN] ', '');
    } else if (line.includes('[GÖNDERİLEN]')) {
       type = 'OUT'; typeClass = 'out'; line = line.replace('[GÖNDERİLEN] ', '');
    } else if (line.includes('[SİSTEM]')) {
       type = 'SYS'; typeClass = 'system'; line = line.replace('[SİSTEM] ', '');
    } else if (line.includes('error') || line.includes('hata')) {
       type = 'ERR'; typeClass = 'error';
    }
    
    if (typeClass === 'error') div.classList.add('error-row');

    div.innerHTML = `
      <div class="log-time">${timeStr}</div>
      <div class="log-badge-wrap"><span class="log-badge ${typeClass}">${type}</span></div>
      <div class="log-content">${line}</div>
    `;
    
    container.appendChild(div);
    if(container.childNodes.length > 300) container.removeChild(container.firstChild);
    container.scrollTop = container.scrollHeight;
  }
});

Events.On('new_msg', (evt) => {
  const m = evt?.data ?? evt;
  if (!m) return;

  if (currentContact === m.phone) {
    const msgsEl = document.getElementById('msgs-container');
    if (msgsEl) {
      if (msgsEl.innerHTML.includes('status-icon-wrap')) { // Quick check for empty
         // Instead of deleting, just safely append. If it's the empty UI, clear it first
         if (msgsEl.querySelector('svg')) msgsEl.innerHTML = '';
      }
      // Or just simply search string
      if (msgsEl.innerHTML.includes('js_no_msg_yet') || msgsEl.innerHTML.includes('conv-empty')) {
        msgsEl.innerHTML = '';
      }
      
      const row = document.createElement('div');
      row.className = 'msg-row';
      row.innerHTML = `
        <div class="msg-bubble ${m.role}">${escapeHtml(m.content)}</div>
        <div class="msg-time ${m.role === 'assistant' ? 'right' : ''}">${formatDate(m.created_at)}</div>
      `;
      msgsEl.appendChild(row);
      msgsEl.scrollTop = msgsEl.scrollHeight;
    }
  }

  BotService.GetContacts().then(contacts => {
    if (contacts) {
      allContacts = contacts;
      updateStats(allContacts);
      if (currentPage === 'conversations') {
        const item = document.querySelector(`.contact-item[data-phone="${m.phone}"]`);
        if (item) {
          const lastEl = item.querySelector('.contact-last');
          if (lastEl) lastEl.innerHTML = escapeHtml(m.content); // Use innerHTML to handle newlines from escapeHtml
        } else {
          renderContactList(allContacts);
        }
      }
    }
  });
});

Events.On('contact_updated', (phone) => {
  BotService.GetContacts().then(contacts => {
    if (contacts) {
      allContacts = contacts;
      if (currentPage === 'conversations') {
        renderContactList(allContacts);
      }
    }
  });
});

// ── Contacts / Conversations ──────────────────────────────────────────────────
async function loadContacts() {
  const listEl = document.getElementById('contact-list');
  listEl.innerHTML = `<div class="contact-list-header">${t('contacts')}</div><div class="list-loading">${t('loading')}</div>`;

  try {
    allContacts = await BotService.GetContacts() || [];
    renderContactList(allContacts);
    updateStats(allContacts);
  } catch (err) {
    listEl.innerHTML = `<div class="contact-list-header">${t('contacts')}</div><div class="list-loading">${t('js_err_prefix')} ${err}</div>`;
  }
}

function renderContactList(contacts) {
  const listEl = document.getElementById('contact-list');

  // Deduplicate before processing
  const uniqueContactsMap = new Map();
  if (contacts) {
    contacts.forEach(c => {
      const fPhone = formatPhone(c.phone);
      if (!uniqueContactsMap.has(fPhone) || c.last_msg) {
        if (!uniqueContactsMap.has(fPhone)) {
          uniqueContactsMap.set(fPhone, c);
        }
      }
    });
  }
  const cleanContacts = Array.from(uniqueContactsMap.values());

  if (!cleanContacts || cleanContacts.length === 0) {
    listEl.innerHTML = `
      <div class="contact-list-header">${t('contacts')}</div>
      <div class="list-empty">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z" /></svg>
        ${t('js_no_conv')}
      </div>`;
    updateStats([]);
    return;
  }

  listEl.innerHTML = `<div class="contact-list-header">${t('contacts')} · ${cleanContacts.length}</div>`;
  cleanContacts.forEach(c => {
    const div = document.createElement('div');
    div.className = 'contact-item' + (currentContact === c.phone ? ' active' : '');
    div.dataset.phone = c.phone;
    div.innerHTML = `
      <div style="display: flex; gap: 12px; align-items: center;">
        <div class="contact-avatar" style="width: 44px; height: 44px; background: var(--border-light); border-radius: 50%; display: flex; align-items: center; justify-content: center; color: var(--text-muted); flex-shrink: 0;">
          ${c.profile_pic ? 
            `<img src="${c.profile_pic}" style="width: 100%; height: 100%; object-fit: cover; border-radius: 50%;" />` : 
            `<svg style="width: 24px; height: 24px;" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" /></svg>`
          }
        </div>
        <div style="display: flex; flex-direction: column; flex: 1; overflow: hidden;">
          <div class="contact-top" style="margin-bottom: 2px;">
            <span class="contact-phone">${c.contact_name ? escapeHtml(c.contact_name) : formatPhone(c.phone)}</span>
            ${c.blocked ? `<span class="contact-blocked-badge">${t('js_blocked_badge')}</span>` : ''}
          </div>
          <div class="contact-last">${escapeHtml(c.last_msg || t('js_no_msg'))}</div>
        </div>
      </div>
    `;
    div.onclick = () => loadConversation(c.phone, div, c.blocked);
    listEl.appendChild(div);
  });

  updateStats(cleanContacts);
}

function updateStats(contacts) {
  const blocked = (contacts || []).filter(c => c.blocked).length;
  const total = (contacts || []).length;

  animateNum('stat-contacts', total);
  animateNum('stat-blocked', blocked);

  BotService.GetTotalMessages().then(count => {
    animateNum('stat-msgs', count);
  }).catch(() => {});

  // Badges
  const bBadge = document.getElementById('badge-blocked');
  if (bBadge) {
    bBadge.textContent = blocked;
    bBadge.style.display = blocked > 0 ? 'inline-block' : 'none';
  }
  const cBadge = document.getElementById('badge-conversations');
  if (cBadge) {
    cBadge.textContent = total;
    cBadge.style.display = total > 0 ? 'inline-block' : 'none';
  }
}

function animateNum(id, newVal) {
  const el = document.getElementById(id);
  if (!el) return;
  el.textContent = newVal;
}

async function loadConversation(phone, el, isBlocked) {
  currentContact = phone;
  document.querySelectorAll('.contact-item').forEach(i => i.classList.remove('active'));
  el.classList.add('active');

  const view = document.getElementById('conv-view');
  view.innerHTML = `<div class="list-loading">${t('loading')}</div>`;

  try {
    const msgs = await BotService.GetConversation(phone);

    const currentContactObj = (allContacts || []).find(c => c.phone === phone);
    const cleanPhone = formatPhone(phone);
    const displayName = (currentContactObj && currentContactObj.contact_name) ? currentContactObj.contact_name : cleanPhone;
    view.innerHTML = `
      <div class="conv-header">
        <span class="conv-header-name">${escapeHtml(displayName)}</span>
        <div>
          ${isBlocked
        ? `<button class="btn btn-success" style="font-size:12px" onclick="unblockContact('${phone}')">${t('js_unblock')}</button>`
        : `<button class="btn btn-danger" style="font-size:12px" onclick="blockContact('${phone}')">${t('js_block')}</button>`
      }
        </div>
      </div>
      <div class="conv-messages" id="msgs-container"></div>
    `;

    const msgsEl = document.getElementById('msgs-container');
    const msgList = msgs || [];

    if (msgList.length === 0) {
      msgsEl.innerHTML = `<div class="conv-empty"><svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.76c0 1.6 1.123 2.994 2.707 3.227 1.087.16 2.185.283 3.293.369V21l4.076-4.076a1.526 1.526 0 011.037-.443 48.282 48.282 0 005.68-.494c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0012 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018z" /></svg><span>${t('js_no_msg_yet')}</span></div>`;
      return;
    }

    // Toplam konuşma sayısını ve total mesajı koru (allContacts'tan)
    updateStats(allContacts);

    msgList.forEach(m => {
      const row = document.createElement('div');
      row.className = 'msg-row';
      row.innerHTML = `
        <div class="msg-bubble ${m.role}">${escapeHtml(m.content)}</div>
        <div class="msg-time ${m.role === 'assistant' ? 'right' : ''}">${formatDate(m.created_at)}</div>
      `;
      msgsEl.appendChild(row);
    });

    msgsEl.scrollTop = msgsEl.scrollHeight;
  } catch (err) {
    view.innerHTML = `<div class="list-loading">Hata: ${err}</div>`;
  }
}

// ── Block / Unblock ───────────────────────────────────────────────────────────
window.blockContact = async function (phone) {
  try {
    await BotService.BlockContact(phone);
    allContacts = allContacts.map(c => c.phone === phone ? { ...c, blocked: true } : c);
    renderContactList(allContacts);
    const el = document.querySelector(`.contact-item[data-phone="${phone}"]`);
    if (el) loadConversation(phone, el, true);
    showToast('🚫 ' + formatPhone(phone) + ' ' + t('js_blocked'));
  } catch (err) {
    showToast('❌ ' + err, true);
  }
};

window.unblockContact = async function (phone) {
  try {
    await BotService.UnblockContact(phone);
    allContacts = allContacts.map(c => c.phone === phone ? { ...c, blocked: false } : c);
    renderContactList(allContacts);
    const el = document.querySelector(`.contact-item[data-phone="${phone}"]`);
    if (el) loadConversation(phone, el, false);
    showToast('🔓 ' + formatPhone(phone) + ' ' + t('js_unblocked'));
  } catch (err) {
    showToast('❌ ' + err, true);
  }
};

// ── Prompt ────────────────────────────────────────────────────────────────────
async function loadPrompt() {
  try {
    const text = await BotService.GetPrompt();
    const editor = document.getElementById('prompt-editor');
    editor.value = text || '';
    updateCharCount();
  } catch (err) {
    showToast(t('js_prompt_err'), true);
  }
}

function updateCharCount() {
  const editor = document.getElementById('prompt-editor');
  const el = document.getElementById('char-count');
  if (editor && el) el.textContent = editor.value.length.toLocaleString();
}

window.savePrompt = async function () {
  const content = document.getElementById('prompt-editor').value;
  const btn = document.getElementById('btn-save-prompt');
  btn.disabled = true;
  try {
    await BotService.SavePrompt(content);
    const saved = document.getElementById('prompt-saved');
    saved.classList.add('show');
    setTimeout(() => saved.classList.remove('show'), 2500);
    showToast(t('js_prompt_ok'));
  } catch (err) {
    showToast(t('js_save_err') + ' ' + err, true);
  } finally {
    btn.disabled = false;
  }
};

// ── Blocked Page ──────────────────────────────────────────────────────────────
async function loadBlocked() {
  const listEl = document.getElementById('blocked-list');
  listEl.innerHTML = `<div class="list-loading">${t('loading')}</div>`;

  try {
    const contacts = await BotService.GetContacts() || [];
    allContacts = contacts;
    const blocked = contacts.filter(c => c.blocked);

    if (blocked.length === 0) {
      listEl.innerHTML = `
        <div class="list-empty" style="padding:40px">
          <svg style="color:var(--success)" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
          ${t('js_no_blocked')}
        </div>`;
      return;
    }

    listEl.innerHTML = '';
    blocked.forEach(c => {
      const div = document.createElement('div');
      div.className = 'blocked-item';
      div.dataset.phone = c.phone;
      div.innerHTML = `
        <div class="blocked-left">
          <div class="blocked-avatar">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" /></svg>
          </div>
          <div>
            <div class="blocked-phone">${formatPhone(c.phone)}</div>
            <div class="blocked-since">${t('js_blocked_since')}</div>
          </div>
        </div>
        <button class="btn btn-success" style="font-size:12px"
          onclick="unblockFromBlocked('${c.phone}', this.closest('.blocked-item'))">
          ${t('js_unblock')}
        </button>
      `;
      listEl.appendChild(div);
    });
  } catch (err) {
    listEl.innerHTML = `<div class="list-loading">${t('js_err_prefix')} ${err}</div>`;
  }
}

window.unblockFromBlocked = async function (phone, el) {
  try {
    await BotService.UnblockContact(phone);
    // Önce allContacts güncelle, sonra DOM'u güncelle (race condition önleme)
    allContacts = allContacts.map(c => c.phone === phone ? { ...c, blocked: false } : c);
    updateStats(allContacts);
    el.style.opacity = '0';
    el.style.transform = 'translateX(20px)';
    el.style.transition = 'all 0.2s ease';
    setTimeout(() => el.remove(), 200);
    showToast(t('js_unblock'));
  } catch (err) {
    showToast('❌ ' + err, true);
  }
};

function renderOpenRouterModels(selectEl, models, currentModel) {
  if (!selectEl) return;
  const freeIds = ['openrouter/free', 'openrouter/owl-alpha'];
  const free = [], paid = [];
  for (const m of models) {
    if (m.endsWith(':free') || freeIds.includes(m)) {
      free.push(m);
    } else {
      paid.push(m);
    }
  }
  free.sort((a, b) => a.localeCompare(b));
  paid.sort((a, b) => a.localeCompare(b));

  let html = '';
  if (free.length > 0) {
    html += '<optgroup label="Free">';
    for (const m of free) html += `<option value="${m}">${m}</option>`;
    html += '</optgroup>';
  }
  if (paid.length > 0) {
    html += '<optgroup label="Paid">';
    for (const m of paid) html += `<option value="${m}">${m}</option>`;
    html += '</optgroup>';
  }
  selectEl.innerHTML = html || '<option value="">No models found</option>';

  if (currentModel) {
    selectEl.value = currentModel;
    if (selectEl.selectedIndex === -1) {
      const opt = document.createElement('option');
      opt.value = currentModel;
      opt.textContent = currentModel + ' (Not Found)';
      selectEl.appendChild(opt);
      selectEl.value = currentModel;
    }
  }
}

async function loadSettings() {
  try {
    const cfg = await BotService.GetConfig();
    const provider = cfg.AI_PROVIDER || 'ollama';
    const providerSelect = document.getElementById('cfg-provider');
    if (providerSelect) providerSelect.value = provider;

    const apiKeyInput = document.getElementById('cfg-api-key');
    if (apiKeyInput) apiKeyInput.value = cfg.AI_API_KEY || '';

    const apiUrlInput = document.getElementById('cfg-api-url');
    if (apiUrlInput) apiUrlInput.value = cfg.AI_API_URL || 'https://openrouter.ai/api/v1';

    toggleProviderFields(provider);

    if (provider === 'openrouter') {
      const modelSelect = document.getElementById('cfg-model-or');
      try {
        const models = await BotService.GetModels();
        renderOpenRouterModels(modelSelect, models || [], cfg.AI_MODEL || 'qwen/qwen3-next-80b-a3b-instruct:free');
      } catch (err) {
        console.warn("Could not fetch OpenRouter models:", err);
        modelSelect.innerHTML = `<option value="${cfg.AI_MODEL || 'qwen/qwen3-next-80b-a3b-instruct:free'}">${cfg.AI_MODEL || 'qwen/qwen3-next-80b-a3b-instruct:free'} (Error)</option>`;
      }
    } else {
      const modelSelect = document.getElementById('cfg-model');
      if (modelSelect) {
        try {
          const models = await BotService.GetModels();
          if (models && models.length > 0) {
            modelSelect.innerHTML = models.map(m => `<option value="${m}">${m}</option>`).join('');
          } else {
            modelSelect.innerHTML = `<option value="${cfg.OLLAMA_MODEL || 'gemma4:e4b'}">${cfg.OLLAMA_MODEL || 'gemma4:e4b'}</option>`;
          }
        } catch (err) {
          console.warn("Could not fetch models:", err);
          modelSelect.innerHTML = `<option value="${cfg.OLLAMA_MODEL || 'gemma4:e4b'}">${cfg.OLLAMA_MODEL || 'gemma4:e4b'} (Error)</option>`;
        }
        modelSelect.value = cfg.OLLAMA_MODEL || 'gemma4:e4b';
        if (modelSelect.selectedIndex === -1 && cfg.OLLAMA_MODEL) {
          const opt = document.createElement('option');
          opt.value = cfg.OLLAMA_MODEL;
          opt.text = cfg.OLLAMA_MODEL + " (Not Found)";
          modelSelect.appendChild(opt);
          modelSelect.value = cfg.OLLAMA_MODEL;
        }
      }
    }

    document.getElementById('cfg-url').value = cfg.OLLAMA_URL || 'http://localhost:11434';
    
    const lang = cfg.LANGUAGE || 'en';
    const langInput = document.querySelector(`input[name="cfg-lang"][value="${lang}"]`);
    if (langInput) langInput.checked = true;
  } catch (err) {
    showToast(t('js_settings_err'), true);
  }
}

function toggleProviderFields(provider) {
  const openrouterIds = ['group-api-key', 'group-api-url', 'group-openrouter-model'];
  const ollamaIds = ['group-ollama-model', 'group-ollama-url'];
  openrouterIds.forEach(id => {
    const el = document.getElementById(id);
    if (el) el.style.display = provider === 'openrouter' ? '' : 'none';
  });
  ollamaIds.forEach(id => {
    const el = document.getElementById(id);
    if (el) el.style.display = provider === 'ollama' ? '' : 'none';
  });
}

window.onProviderChange = async function () {
  const provider = document.getElementById('cfg-provider').value;
  toggleProviderFields(provider);
  if (provider === 'ollama') {
    const modelSelect = document.getElementById('cfg-model');
    if (modelSelect) {
      modelSelect.innerHTML = '<option value="">Loading...</option>';
      try {
        const models = await BotService.GetModels();
        if (models && models.length > 0) {
          modelSelect.innerHTML = models.map(m => `<option value="${m}">${m}</option>`).join('');
        } else {
          modelSelect.innerHTML = '<option value="gemma4:e4b">gemma4:e4b</option>';
        }
      } catch (err) {
        modelSelect.innerHTML = '<option value="gemma4:e4b">gemma4:e4b (Error)</option>';
      }
    }
  } else {
    const modelSelect = document.getElementById('cfg-model-or');
    if (modelSelect) {
      modelSelect.innerHTML = '<option value="">Loading...</option>';
      try {
        const models = await BotService.GetModels();
        renderOpenRouterModels(modelSelect, models || [], 'qwen/qwen3-next-80b-a3b-instruct:free');
      } catch (err) {
        modelSelect.innerHTML = '<option value="qwen/qwen3-next-80b-a3b-instruct:free">qwen/qwen3-next-80b-a3b-instruct:free (Error)</option>';
      }
    }
  }
}

window.saveSettings = async function () {
  const btn = document.getElementById('btn-save-settings');
  btn.disabled = true;

  const provider = document.getElementById('cfg-provider').value.trim();
  const cfg = {
    AI_PROVIDER: provider,
    AI_API_KEY: document.getElementById('cfg-api-key')?.value.trim() || '',
    AI_API_URL: document.getElementById('cfg-api-url')?.value.trim() || '',
    AI_MODEL: provider === 'openrouter' ? (document.getElementById('cfg-model-or')?.value.trim() || '') : '',
    OLLAMA_MODEL: provider === 'ollama' ? document.getElementById('cfg-model').value.trim() : '',
    OLLAMA_URL: document.getElementById('cfg-url').value.trim(),
    DB_TYPE: 'sqlite',
    DB_URL: 'messages.db',
    LANGUAGE: document.querySelector('input[name="cfg-lang"]:checked')?.value || 'en',
  };

  try {
    await BotService.SaveConfig(cfg);
    setLang(cfg.LANGUAGE);
    renderUI(); 

    document.getElementById('page-dashboard').classList.remove('active');
    document.getElementById('page-settings').classList.add('active');
    document.querySelector('[data-page="dashboard"]').classList.remove('active');
    document.querySelector('[data-page="settings"]').classList.add('active');
    
    updateStats(allContacts);
    
    const status = await BotService.GetStatus();
    if (status) Events.Emit('status_change', status);

    const quickSelect = document.getElementById('quick-model-select');
    if (quickSelect) quickSelect.value = cfg.AI_PROVIDER === 'openrouter' ? cfg.AI_MODEL : cfg.OLLAMA_MODEL;
    
    loadSettings();

    const saved = document.getElementById('settings-saved');
    if (saved) {
      saved.classList.add('show');
      setTimeout(() => saved.classList.remove('show'), 3000);
    }
    showToast(t('js_settings_ok'));
  } catch (err) {
    showToast(t('js_save_err') + ' ' + err, true);
  } finally {
    const freshBtn = document.getElementById('btn-save-settings');
    if (freshBtn) freshBtn.disabled = false;
  }
};

function formatPhone(phone) {
  return phone.replace(/:\d+/, '').replace('@lid', '').replace('@s.whatsapp.net', '');
}

function formatDate(dt) {
  if (!dt) return '';
  try {
    return new Date(dt).toLocaleString('tr-TR', {
      hour: '2-digit', minute: '2-digit',
      day: '2-digit', month: '2-digit',
    });
  } catch { return dt; }
}

function escapeHtml(str) {
  return (str || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/\n/g, '<br>');
}

// ── Init ──────────────────────────────────────────────────────────────────────
async function init() {
  try {
    // 1. Dil ayarlari ve arayuz render
    const cfg = await BotService.GetConfig();
    if (cfg && cfg.LANGUAGE) {
      setLang(cfg.LANGUAGE);
    } else {
      setLang('en');
    }
    renderUI();

    const status = await BotService.GetStatus();
    if (status) {
      Events.Emit('status_change', status);
      // Auto-start bot connection if it's currently disconnected 
      if (!status.running && !status.connected) {
        window.startBot();
      }
    }
    
    // 2. Load Ollama Models for Sidebar
    try {
      const models = await BotService.GetModels();
      const sidebarSelect = document.getElementById('quick-model-select');
      if (sidebarSelect && models && models.length > 0) {
        sidebarSelect.innerHTML = models.map(m => `<option value="${m}">${m}</option>`).join('');
        if (cfg && cfg.OLLAMA_MODEL) {
          if (!models.includes(cfg.OLLAMA_MODEL)) {
            sidebarSelect.innerHTML += `<option value="${cfg.OLLAMA_MODEL}">${cfg.OLLAMA_MODEL}</option>`;
          }
          sidebarSelect.value = cfg.OLLAMA_MODEL;
        }
      }
    } catch(e) {}
    
    // Load contacts initially to populate stats
    const contacts = await BotService.GetContacts();
    if (contacts) {
      allContacts = contacts;
      updateStats(contacts);
    }
  } catch (e) {
    console.log('API boot error. Starting UI anyway:', e);
    renderUI();
  }
}

init();
