export const SettingsHtml = (t) => `
<div id="page-settings" class="page">
  <div class="page-header">
    <h1 class="page-title">${t('settings_title')}</h1>
    <p class="page-subtitle">${t('settings_sub')}</p>
  </div>

  <div class="settings-section">
    <div class="settings-section-header">
      <div class="settings-section-icon">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 017.843 4.582M12 3a8.997 8.997 0 00-7.843 4.582m15.686 0A11.953 11.953 0 0112 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0121 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0112 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 013 12c0-1.605.42-3.113 1.157-4.418" /></svg>
      </div>
      ${t('lang_title')}
    </div>
    <div class="settings-body">
      <div class="form-group">
        <label class="form-label">${t('lang_label')}</label>
        <div class="lang-toggle">
          <input type="radio" id="lang-en" name="cfg-lang" value="en" class="visually-hidden" />
          <label for="lang-en" class="lang-btn">${t('lang_en')}</label>
          <input type="radio" id="lang-tr" name="cfg-lang" value="tr" class="visually-hidden" />
          <label for="lang-tr" class="lang-btn">${t('lang_tr')}</label>
        </div>
      </div>
    </div>
  </div>

  <div class="settings-section">
    <div class="settings-section-header">
      <div class="settings-section-icon">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M15.59 14.37a6 6 0 01-5.84 7.36 3 3 0 11-2.27-4.99l.91-.71m5.67-5.35c.18-.7.2-1.46.06-2.2-1.2-6.28-7.7-8.3-11.45-5.2A3 3 0 004 8.28m11.59 6.09L20 20v-3l-2.09-2.09c.8-1.07 1.2-2.4 1.09-3.83-1.2-6.28-7.7-8.3-11.45-5.2a3 3 0 00-1.8 7.37m0 0l-1.63 1.63M4 18l1.63-1.63m14.37-14.37L18.37 4m-4.45 4.45a3 3 0 11-4.24-4.24" /></svg>
      </div>
      ${t('model_title')}
    </div>
    <div class="settings-body">
      <div class="form-group">
        <label class="form-label" for="cfg-provider">Provider</label>
        <select id="cfg-provider" class="form-input">
          <option value="ollama">Ollama (Local)</option>
          <option value="openrouter">OpenRouter (Cloud)</option>
        </select>
      </div>
      <div class="form-group" id="group-api-key" style="display:none">
        <label class="form-label" for="cfg-api-key">API Key</label>
        <input type="password" id="cfg-api-key" class="form-input" placeholder="sk-or-v1-..." />
      </div>
      <div class="form-group" id="group-api-url" style="display:none">
        <label class="form-label" for="cfg-api-url">API URL</label>
        <input type="text" id="cfg-api-url" class="form-input" placeholder="https://openrouter.ai/api/v1" />
      </div>
      <div class="form-group" id="group-ollama-model">
        <label class="form-label" for="cfg-model">${t('model_name')}</label>
        <select id="cfg-model" class="form-input">
          <option value="">Loading...</option>
        </select>
        <span class="form-hint">${t('model_hint')}</span>
      </div>
      <div class="form-group" id="group-openrouter-model" style="display:none">
        <label class="form-label" for="cfg-model-or">Model</label>
        <select id="cfg-model-or" class="form-input">
          <option value="">Loading...</option>
        </select>
        <span class="form-hint">${t('model_hint')}</span>
      </div>
      <div class="form-group" id="group-ollama-url">
        <label class="form-label" for="cfg-url">${t('model_url')}</label>
        <input type="text" id="cfg-url" class="form-input" placeholder="http://localhost:11434" />
      </div>
    </div>
  </div>

  <div class="settings-section">
    <div class="settings-section-header">
      <div class="settings-section-icon">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.76c0 1.6 1.123 2.994 2.707 3.227 1.087.16 2.185.283 3.293.369V21l4.076-4.076a1.526 1.526 0 011.037-.443 48.282 48.282 0 005.68-.494c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0012 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018z" /></svg>
      </div>
      ${t('wa_title')}
    </div>
    <div class="settings-body">
      <div class="form-group">
        <label class="form-label">${t('wa_status')}</label>
        <div id="settings-wa-status" class="status-badge stopped">${t('wa_disconnected')}</div>
      </div>
      
      <div id="settings-wa-actions" style="margin-top:12px; display:flex; flex-direction:column; gap:12px;">
        <button id="btn-settings-wa-login" class="btn btn-primary" style="width:fit-content">
          ${t('wa_qr_btn')}
        </button>
        
        <button id="btn-settings-wa-logout" class="btn btn-danger" style="width:fit-content; display:none;">
          ${t('wa_btn_logout')}
        </button>

        <div id="settings-qr-area" style="display:none; text-align:center; padding:15px; border:1px solid var(--border); border-radius:12px; background:var(--bg-app);">
          <div style="font-weight:600; margin-bottom:10px;">${t('wa_qr_title')}</div>
          <div id="settings-qr-spinner" class="spinner" style="margin:20px auto;"></div>
          <img id="settings-qr-img" src="" style="display:none; margin:0 auto; border-radius:8px; width:200px; height:200px;" />
          <p style="font-size:12px; color:var(--text-muted); margin-top:10px;">${t('qr_steps')}</p>
        </div>

      </div>
    </div>
  </div>


  <div style="display:flex;align-items:center;gap:12px;">
    <button class="btn btn-primary" id="btn-save-settings">
      <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" /></svg>
      ${t('btn_save_settings')}
    </button>
    <span class="save-status" id="settings-saved">
      <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" style="width:16px;height:16px;display:inline;vertical-align:text-bottom"><path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" /></svg>
      ${t('settings_saved_hint')}
    </span>
  </div>
</div>
`;
