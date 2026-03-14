/* ── Satoshi Note — app.js ── */
'use strict';

// ── Constants ────────────────────────────────────────────────────────────────
const LS_REFUND = 'sn_refund_code';
const LS_HISTORY = 'sn_history';
const LS_COUNTRY = 'sn_country_prefix';

const DIAL_CODES = [
  ['+1','US/CA (+1)'],  ['+7','RU (+7)'],    ['+20','EG (+20)'],
  ['+27','ZA (+27)'],   ['+30','GR (+30)'],   ['+31','NL (+31)'],
  ['+32','BE (+32)'],   ['+33','FR (+33)'],   ['+34','ES (+34)'],
  ['+36','HU (+36)'],   ['+39','IT (+39)'],   ['+40','RO (+40)'],
  ['+41','CH (+41)'],   ['+43','AT (+43)'],   ['+44','GB (+44)'],
  ['+45','DK (+45)'],   ['+46','SE (+46)'],   ['+47','NO (+47)'],
  ['+48','PL (+48)'],   ['+49','DE (+49)'],   ['+51','PE (+51)'],
  ['+52','MX (+52)'],   ['+53','CU (+53)'],   ['+54','AR (+54)'],
  ['+55','BR (+55)'],   ['+56','CL (+56)'],   ['+57','CO (+57)'],
  ['+58','VE (+58)'],   ['+60','MY (+60)'],   ['+61','AU (+61)'],
  ['+62','ID (+62)'],   ['+63','PH (+63)'],   ['+64','NZ (+64)'],
  ['+65','SG (+65)'],   ['+66','TH (+66)'],   ['+81','JP (+81)'],
  ['+82','KR (+82)'],   ['+84','VN (+84)'],   ['+86','CN (+86)'],
  ['+90','TR (+90)'],   ['+91','IN (+91)'],   ['+92','PK (+92)'],
  ['+93','AF (+93)'],   ['+94','LK (+94)'],   ['+95','MM (+95)'],
  ['+98','IR (+98)'],   ['+212','MA (+212)'], ['+213','DZ (+213)'],
  ['+216','TN (+216)'], ['+218','LY (+218)'], ['+220','GM (+220)'],
  ['+221','SN (+221)'], ['+223','ML (+223)'], ['+224','GN (+224)'],
  ['+225','CI (+225)'], ['+227','NE (+227)'], ['+228','TG (+228)'],
  ['+229','BJ (+229)'], ['+230','MU (+230)'], ['+231','LR (+231)'],
  ['+232','SL (+232)'], ['+233','GH (+233)'], ['+234','NG (+234)'],
  ['+235','TD (+235)'], ['+236','CF (+236)'], ['+237','CM (+237)'],
  ['+238','CV (+238)'], ['+239','ST (+239)'], ['+240','GQ (+240)'],
  ['+241','GA (+241)'], ['+242','CG (+242)'], ['+243','CD (+243)'],
  ['+244','AO (+244)'], ['+245','GW (+245)'], ['+246','IO (+246)'],
  ['+247','AC (+247)'], ['+248','SC (+248)'], ['+249','SD (+249)'],
  ['+250','RW (+250)'], ['+251','ET (+251)'], ['+252','SO (+252)'],
  ['+253','DJ (+253)'], ['+254','KE (+254)'], ['+255','TZ (+255)'],
  ['+256','UG (+256)'], ['+257','BI (+257)'], ['+258','MZ (+258)'],
  ['+260','ZM (+260)'], ['+261','MG (+261)'], ['+263','ZW (+263)'],
  ['+264','NA (+264)'], ['+265','MW (+265)'], ['+266','LS (+266)'],
  ['+267','BW (+267)'], ['+268','SZ (+268)'], ['+269','KM (+269)'],
  ['+290','SH (+290)'], ['+291','ER (+291)'], ['+297','AW (+297)'],
  ['+298','FO (+298)'], ['+299','GL (+299)'], ['+350','GI (+350)'],
  ['+351','PT (+351)'], ['+352','LU (+352)'], ['+353','IE (+353)'],
  ['+354','IS (+354)'], ['+355','AL (+355)'], ['+356','MT (+356)'],
  ['+357','CY (+357)'], ['+358','FI (+358)'], ['+359','BG (+359)'],
  ['+370','LT (+370)'], ['+371','LV (+371)'], ['+372','EE (+372)'],
  ['+373','MD (+373)'], ['+374','AM (+374)'], ['+375','BY (+375)'],
  ['+376','AD (+376)'], ['+377','MC (+377)'], ['+378','SM (+378)'],
  ['+380','UA (+380)'], ['+381','RS (+381)'], ['+382','ME (+382)'],
  ['+385','HR (+385)'], ['+386','SI (+386)'], ['+387','BA (+387)'],
  ['+389','MK (+389)'], ['+420','CZ (+420)'], ['+421','SK (+421)'],
  ['+423','LI (+423)'], ['+500','FK (+500)'], ['+501','BZ (+501)'],
  ['+502','GT (+502)'], ['+503','SV (+503)'], ['+504','HN (+504)'],
  ['+505','NI (+505)'], ['+506','CR (+506)'], ['+507','PA (+507)'],
  ['+508','PM (+508)'], ['+509','HT (+509)'], ['+590','GP (+590)'],
  ['+591','BO (+591)'], ['+592','GY (+592)'], ['+593','EC (+593)'],
  ['+594','GF (+594)'], ['+595','PY (+595)'], ['+596','MQ (+596)'],
  ['+597','SR (+597)'], ['+598','UY (+598)'], ['+599','AN (+599)'],
  ['+670','TL (+670)'], ['+672','NF (+672)'], ['+673','BN (+673)'],
  ['+674','NR (+674)'], ['+675','PG (+675)'], ['+676','TO (+676)'],
  ['+677','SB (+677)'], ['+678','VU (+678)'], ['+679','FJ (+679)'],
  ['+680','PW (+680)'], ['+681','WF (+681)'], ['+682','CK (+682)'],
  ['+683','NU (+683)'], ['+685','WS (+685)'], ['+686','KI (+686)'],
  ['+687','NC (+687)'], ['+688','TV (+688)'], ['+689','PF (+689)'],
  ['+690','TK (+690)'], ['+691','FM (+691)'], ['+692','MH (+692)'],
  ['+850','KP (+850)'], ['+852','HK (+852)'], ['+853','MO (+853)'],
  ['+855','KH (+855)'], ['+856','LA (+856)'], ['+880','BD (+880)'],
  ['+886','TW (+886)'], ['+960','MV (+960)'], ['+961','LB (+961)'],
  ['+962','JO (+962)'], ['+963','SY (+963)'], ['+964','IQ (+964)'],
  ['+965','KW (+965)'], ['+966','SA (+966)'], ['+967','YE (+967)'],
  ['+968','OM (+968)'], ['+970','PS (+970)'], ['+971','AE (+971)'],
  ['+972','IL (+972)'], ['+973','BH (+973)'], ['+974','QA (+974)'],
  ['+975','BT (+975)'], ['+976','MN (+976)'], ['+977','NP (+977)'],
  ['+992','TJ (+992)'], ['+993','TM (+993)'], ['+994','AZ (+994)'],
  ['+995','GE (+995)'], ['+996','KG (+996)'], ['+998','UZ (+998)'],
];

// ── State ─────────────────────────────────────────────────────────────────────
const state = {
  step: 1,               // single voucher wizard step
  vouchers: null,        // current voucher array from API
  batchStep: 'form',     // 'form' | 'fund' | 'results'
  activeTab: 'single',
};

// ── DOM refs ──────────────────────────────────────────────────────────────────
const $ = id => document.getElementById(id);

// ── UUID v4 ───────────────────────────────────────────────────────────────────
function uuidv4() {
  return ([1e7]+-1e3+-4e3+-8e3+-1e11).replace(/[018]/g, c =>
    (c ^ crypto.getRandomValues(new Uint8Array(1))[0] & 15 >> c / 4).toString(16));
}

// ── localStorage helpers ──────────────────────────────────────────────────────
function getHistory() {
  try { return JSON.parse(localStorage.getItem(LS_HISTORY) || '[]'); }
  catch { return []; }
}

function saveHistory(arr) {
  localStorage.setItem(LS_HISTORY, JSON.stringify(arr));
}

function pushHistory(entry) {
  const arr = getHistory();
  arr.unshift(entry);
  saveHistory(arr);
}

// ── QR helpers ────────────────────────────────────────────────────────────────
function renderQR(container, text, size) {
  container.innerHTML = '';
  new QRCode(container, { text, width: size, height: size, correctLevel: QRCode.CorrectLevel.M });
}

function qrToDataURL(text, size) {
  return new Promise(resolve => {
    const div = document.createElement('div');
    div.style.cssText = 'position:absolute;left:-9999px;top:-9999px';
    document.body.appendChild(div);
    new QRCode(div, { text, width: size, height: size, correctLevel: QRCode.CorrectLevel.M });
    // QRCode renders canvas or img; canvas has toDataURL directly
    setTimeout(() => {
      const canvas = div.querySelector('canvas');
      const dataURL = canvas ? canvas.toDataURL('image/png') : null;
      document.body.removeChild(div);
      resolve(dataURL);
    }, 80);
  });
}

// ── Clipboard ─────────────────────────────────────────────────────────────────
async function copyToClipboard(text, btn) {
  try {
    await navigator.clipboard.writeText(text);
    const orig = btn.textContent;
    btn.textContent = 'Copied!';
    setTimeout(() => { btn.textContent = orig; }, 1500);
  } catch {
    prompt('Copy this:', text);
  }
}

// ── API call ─────────────────────────────────────────────────────────────────
async function createVouchers(payload) {
  const res = await fetch('/voucher/create', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const msg = await res.text().catch(() => 'Request failed');
    throw new Error(msg || `HTTP ${res.status}`);
  }
  return res.json();
}

// ── Dial code select ──────────────────────────────────────────────────────────
function populateDialCodes(selectEl, preferred) {
  selectEl.innerHTML = '';
  DIAL_CODES.forEach(([code, label]) => {
    const opt = document.createElement('option');
    opt.value = code;
    opt.textContent = label;
    if (code === preferred) opt.selected = true;
    selectEl.appendChild(opt);
  });
}

async function detectDialCode() {
  const cached = localStorage.getItem(LS_COUNTRY);
  if (cached) return cached;
  try {
    const r = await fetch('https://ipapi.co/json/');
    const d = await r.json();
    const code = d.country_calling_code || '+1';
    localStorage.setItem(LS_COUNTRY, code);
    return code;
  } catch {
    return '+1';
  }
}

// ── Expiry text ───────────────────────────────────────────────────────────────
function expiryText(createdAt, refundAfterSeconds) {
  const secsLeft = (createdAt + refundAfterSeconds) - Math.floor(Date.now() / 1000);
  if (secsLeft <= 0) return { text: 'Expired', cls: 'expiry-expired' };
  const hrs = Math.floor(secsLeft / 3600);
  if (hrs < 24) return { text: `${hrs}h left`, cls: hrs < 6 ? 'expiry-warn' : 'expiry-ok' };
  const days = Math.floor(hrs / 24);
  return { text: `${days}d left`, cls: days < 2 ? 'expiry-warn' : 'expiry-ok' };
}

function daysFromSeconds(secs) {
  const d = Math.round(secs / 86400);
  return d === 1 ? '1 day' : `${d} days`;
}

// ── WhatsApp message ──────────────────────────────────────────────────────────
function buildWAMessage(claimLnurl, refundAfterSeconds) {
  const link = `${window.location.origin}/redeem.html?lnurl=${encodeURIComponent(claimLnurl)}`;
  const dur = daysFromSeconds(refundAfterSeconds);
  return `You've received a Bitcoin voucher! ⚡\n\nTap this link to claim it:\n${link}\n\nYou'll need a Lightning wallet — try Blink (blink.sv) for beginners.\n\nThis voucher expires in ${dur}.`;
}

// ── Single voucher wizard ─────────────────────────────────────────────────────
function showStep(n) {
  state.step = n;
  [1, 2, 3].forEach(i => {
    const el = $(`single-step-${i}`);
    if (el) el.classList.toggle('hidden', i !== n);
  });
  // update step dots
  const dots = document.querySelectorAll('#tab-single .step-dot');
  dots.forEach((d, i) => {
    d.classList.toggle('done', i < n - 1);
    d.classList.toggle('active', i === n - 1);
  });
  const lbl = $('step-label');
  if (lbl) lbl.textContent = `Step ${n} of 3`;
}

async function initSingleStep1() {
  const dialSel = $('dial-code');
  const preferred = await detectDialCode();
  populateDialCodes(dialSel, preferred);
}

async function handleCreateSingle() {
  const btn = $('btn-create-single');
  const errEl = $('single-step1-error');
  errEl.classList.remove('visible');

  const dialCode = $('dial-code').value;
  const rawNumber = $('phone-number').value.trim();
  if (!rawNumber) {
    errEl.textContent = 'Please enter a phone number.';
    errEl.classList.add('visible');
    return;
  }

  const refundCode = localStorage.getItem(LS_REFUND) || '';
  const ts = Date.now();

  btn.disabled = true;
  const origHTML = btn.innerHTML;
  btn.innerHTML = '<span class="spinner"></span> Creating…';

  try {
    const vouchers = await createVouchers({
      batch_name: `single-${ts}`,
      amount: 1,
      refund_code: refundCode,
      refund_after_seconds: 259200,
      single_use: true,
    });

    state.vouchers = vouchers;

    // Store phone for step 3
    const digits = rawNumber.replace(/\D/g, '').replace(/^0/, '');
    state.e164 = dialCode.replace('+', '') + digits;
    state.dialCode = dialCode;
    state.localNumber = rawNumber;

    renderFundStep(vouchers[0]);
    showStep(2);
  } catch (err) {
    errEl.textContent = err.message || 'Failed to create voucher. Try again.';
    errEl.classList.add('visible');
  } finally {
    btn.disabled = false;
    btn.innerHTML = origHTML;
  }
}

function renderFundStep(voucher) {
  const container = $('single-qr-container');
  renderQR(container, voucher.fund_lnurl, 256);
  $('single-lnurl-text').textContent = voucher.fund_lnurl;

  $('btn-copy-fund-lnurl').onclick = () => copyToClipboard(voucher.fund_lnurl, $('btn-copy-fund-lnurl'));
  $('btn-open-wallet-single').onclick = () => { window.location.href = 'lightning:' + voucher.fund_lnurl; };
  $('btn-funded-single').onclick = () => { renderShareStep(voucher); showStep(3); };
}

function renderShareStep(voucher) {
  const digits = state.localNumber.replace(/\D/g, '').replace(/^0/, '');
  const e164 = state.dialCode.replace('+', '') + digits;
  $('share-phone-display').textContent = '+' + e164;

  const msg = buildWAMessage(voucher.claim_lnurl, voucher.refund_after_seconds);

  $('btn-whatsapp').onclick = () => {
    const url = `https://wa.me/${e164}?text=${encodeURIComponent(msg)}`;
    window.open(url, '_blank');
  };

  const redeemLink = `${window.location.origin}/redeem.html?lnurl=${encodeURIComponent(voucher.claim_lnurl)}`;
  $('btn-copy-link').onclick = () => copyToClipboard(redeemLink, $('btn-copy-link'));

  $('btn-done-single').onclick = () => {
    const entry = {
      id: uuidv4(),
      type: 'single',
      createdAt: Math.floor(Date.now() / 1000),
      phone: '+' + e164,
      batchName: voucher.batch_name,
      refundAfterSeconds: voucher.refund_after_seconds,
      vouchers: state.vouchers,
    };
    pushHistory(entry);
    state.vouchers = null;
    resetSingleWizard();
    showStep(1);
  };
}

function resetSingleWizard() {
  $('phone-number').value = '';
  $('single-step1-error').classList.remove('visible');
  $('single-qr-container').innerHTML = '';
  $('single-lnurl-text').textContent = '';
}

// ── Batch vouchers ────────────────────────────────────────────────────────────
function showBatchStep(step) {
  state.batchStep = step;
  ['form', 'fund', 'results'].forEach(s => {
    const el = $(`batch-${s}`);
    if (el) el.classList.toggle('hidden', s !== step);
  });
}

async function handleCreateBatch() {
  const btn = $('btn-create-batch');
  const errEl = $('batch-error');
  errEl.classList.remove('visible');

  const name = $('batch-name').value.trim();
  const count = parseInt($('batch-count').value, 10) || 10;
  const expiry = parseInt($('batch-expiry').value, 10);
  const singleUse = $('batch-single-use').checked;

  if (!name) {
    errEl.textContent = 'Please enter a batch name.';
    errEl.classList.add('visible');
    return;
  }
  if (count < 1 || count > 100) {
    errEl.textContent = 'Voucher count must be between 1 and 100.';
    errEl.classList.add('visible');
    return;
  }

  const refundCode = localStorage.getItem(LS_REFUND) || '';

  btn.disabled = true;
  const origHTML = btn.innerHTML;
  btn.innerHTML = '<span class="spinner"></span> Creating…';

  try {
    const vouchers = await createVouchers({
      batch_name: name,
      amount: count,
      refund_code: refundCode,
      refund_after_seconds: expiry,
      single_use: singleUse,
    });

    state.vouchers = vouchers;
    state.batchExpiry = expiry;
    state.batchName = name;

    renderBatchFund(vouchers, count);
    showBatchStep('fund');
  } catch (err) {
    errEl.textContent = err.message || 'Failed to create vouchers. Try again.';
    errEl.classList.add('visible');
  } finally {
    btn.disabled = false;
    btn.innerHTML = origHTML;
  }
}

function renderBatchFund(vouchers, count) {
  const batchFundLnurl = vouchers[0].batch_fund_lnurl;
  const container = $('batch-qr-container');
  renderQR(container, batchFundLnurl, 256);
  $('batch-lnurl-text').textContent = batchFundLnurl;
  $('batch-fund-note').textContent = `This funds all ${count} vouchers equally.`;

  $('btn-copy-batch-lnurl').onclick = () => copyToClipboard(batchFundLnurl, $('btn-copy-batch-lnurl'));
  $('btn-open-wallet-batch').onclick = () => { window.location.href = 'lightning:' + batchFundLnurl; };
  $('btn-funded-batch').onclick = () => { renderBatchResults(vouchers); showBatchStep('results'); };
}

function renderBatchResults(vouchers) {
  const grid = $('batch-results-grid');
  grid.innerHTML = '';
  vouchers.forEach((v, i) => {
    const card = document.createElement('div');
    card.className = 'qr-mini-card';
    const qrDiv = document.createElement('div');
    qrDiv.className = 'qr-mini-container';
    new QRCode(qrDiv, { text: v.claim_lnurl, width: 72, height: 72, correctLevel: QRCode.CorrectLevel.M });
    const label = document.createElement('span');
    label.textContent = `${i + 1} of ${vouchers.length}`;
    card.appendChild(qrDiv);
    card.appendChild(label);
    grid.appendChild(card);
  });

  $('btn-download-pdf').onclick = () => downloadPDF(vouchers, state.batchName, state.batchExpiry);
  $('btn-done-batch').onclick = () => {
    const entry = {
      id: uuidv4(),
      type: 'batch',
      createdAt: Math.floor(Date.now() / 1000),
      phone: null,
      batchName: state.batchName,
      refundAfterSeconds: state.batchExpiry,
      vouchers: state.vouchers,
    };
    pushHistory(entry);
    state.vouchers = null;
    resetBatchForm();
    showBatchStep('form');
  };
}

function resetBatchForm() {
  $('batch-name').value = '';
  $('batch-count').value = '10';
  $('batch-error').classList.remove('visible');
  $('batch-qr-container').innerHTML = '';
  $('batch-results-grid').innerHTML = '';
}

// ── PDF generation ────────────────────────────────────────────────────────────
async function downloadPDF(vouchers, batchName, refundAfterSeconds) {
  const btn = $('btn-download-pdf');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner"></span> Generating…';

  try {
    await loadjsPDF();
    await generatePDF(vouchers, batchName, refundAfterSeconds);
  } catch (err) {
    alert('PDF generation failed: ' + err.message);
  } finally {
    btn.disabled = false;
    btn.textContent = 'Download PDF';
  }
}

function loadjsPDF() {
  return new Promise((resolve, reject) => {
    if (window.jspdf) { resolve(); return; }
    const s = document.createElement('script');
    s.src = 'https://cdn.jsdelivr.net/npm/jspdf@2.5.1/dist/jspdf.umd.min.js';
    s.onload = resolve;
    s.onerror = () => reject(new Error('Failed to load jsPDF'));
    document.head.appendChild(s);
  });
}

async function generatePDF(vouchers, batchName, refundAfterSeconds) {
  const { jsPDF } = window.jspdf;
  const doc = new jsPDF({ orientation: 'landscape', unit: 'mm', format: 'a5' });
  // A5 landscape: 210 × 148 mm ... actually jsPDF 'a5' in landscape = 210w × 148h
  const W = 210, H = 148;

  for (let i = 0; i < vouchers.length; i++) {
    if (i > 0) doc.addPage();

    const v = vouchers[i];

    // Orange header strip
    doc.setFillColor(247, 147, 26);
    doc.rect(0, 0, W, 18, 'F');
    doc.setTextColor(0, 0, 0);
    doc.setFontSize(10);
    doc.setFont('helvetica', 'bold');
    doc.text('Satoshi Note', 8, 12);
    doc.setFont('helvetica', 'normal');
    doc.setFontSize(8);
    doc.text('Lightning Voucher', 8, 17);

    // QR code (left panel, ~60mm wide)
    const qrDataURL = await qrToDataURL(v.claim_lnurl, 200);
    if (qrDataURL) {
      doc.addImage(qrDataURL, 'PNG', 8, 24, 52, 52);
    }

    // Right panel text
    doc.setTextColor(15, 15, 15);
    doc.setFont('helvetica', 'bold');
    doc.setFontSize(11);
    doc.text(batchName, 70, 32);

    doc.setFont('helvetica', 'normal');
    doc.setFontSize(9);
    doc.setTextColor(80, 80, 80);
    doc.text(`Voucher ${i + 1} of ${vouchers.length}`, 70, 40);
    doc.text(`Expires in: ${daysFromSeconds(refundAfterSeconds)}`, 70, 48);

    doc.setFontSize(8);
    doc.text('How to redeem:', 70, 60);
    doc.text('1. Install a Lightning wallet (try blink.sv)', 70, 67);
    doc.text('2. Open the app and tap Receive / Scan', 70, 73);
    doc.text('3. Scan the QR code on the left', 70, 79);
    doc.text('4. Confirm to receive your Bitcoin', 70, 85);

    // Footer strip
    doc.setFillColor(247, 147, 26);
    doc.rect(0, H - 10, W, 10, 'F');
    doc.setTextColor(0, 0, 0);
    doc.setFontSize(7);
    doc.text('Scan with a Lightning wallet · blink.sv for beginners', W / 2, H - 3, { align: 'center' });
  }

  doc.save(`${batchName.replace(/\s+/g, '-')}-vouchers.pdf`);
}

// ── History screen ────────────────────────────────────────────────────────────
function renderHistory() {
  const container = $('history-list');
  const history = getHistory();

  if (!history.length) {
    container.innerHTML = `
      <div class="empty-state">
        <div class="empty-state-icon">🗒️</div>
        <p>No vouchers yet.<br>Create one to see it here.</p>
      </div>`;
    return;
  }

  container.innerHTML = '';
  history.forEach(entry => {
    const card = document.createElement('div');
    card.className = 'history-card';

    const exp = expiryText(entry.createdAt, entry.refundAfterSeconds);
    const date = new Date(entry.createdAt * 1000).toLocaleDateString();
    const typeLabel = entry.type === 'single' ? 'Single' : `Batch (${entry.vouchers.length})`;
    const badge = entry.type === 'single' ? 'badge-single' : 'badge-batch';
    const label = entry.type === 'single' ? entry.phone : entry.batchName;

    card.innerHTML = `
      <div class="history-card-header">
        <span class="badge ${badge}">${typeLabel}</span>
        <span class="expiry-text ${exp.cls}">${exp.text}</span>
      </div>
      <div class="history-card-meta">
        ${label} &nbsp;·&nbsp; ${date}
      </div>
      <div class="history-card-actions">
        <button class="btn btn-secondary btn-sm" data-action="reqr" data-id="${entry.id}">Re-show QR</button>
        ${entry.type === 'batch' ? `<button class="btn btn-ghost btn-sm" data-action="repdf" data-id="${entry.id}">Re-download PDF</button>` : ''}
      </div>`;

    container.appendChild(card);
  });

  // Bind action buttons
  container.querySelectorAll('[data-action]').forEach(btn => {
    btn.addEventListener('click', () => {
      const entry = history.find(e => e.id === btn.dataset.id);
      if (!entry) return;
      if (btn.dataset.action === 'reqr') openQRModal(entry);
      if (btn.dataset.action === 'repdf') {
        downloadPDF(entry.vouchers, entry.batchName, entry.refundAfterSeconds);
      }
    });
  });
}

function openQRModal(entry) {
  const voucher = entry.vouchers[0];
  const lnurl = entry.type === 'single' ? voucher.claim_lnurl : voucher.batch_fund_lnurl;
  const container = $('modal-qr-container');
  renderQR(container, lnurl, 240);
  $('modal-title').textContent = entry.type === 'single' ? 'Claim QR' : 'Fund QR';
  $('qr-modal').classList.remove('hidden');
}

// ── Screen routing ────────────────────────────────────────────────────────────
function showScreen(id) {
  document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
  const el = $(id);
  if (el) el.classList.add('active');
}

function showTab(tab) {
  state.activeTab = tab;
  document.querySelectorAll('.tab-btn').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
  document.querySelectorAll('.tab-pane').forEach(p => p.classList.toggle('active', p.id === `tab-${tab}`));
}

// ── Onboarding ────────────────────────────────────────────────────────────────
function validateRefundCode(val) {
  return val.includes('@') || val.toLowerCase().startsWith('lnurl1');
}

function handleOnboardingSubmit() {
  const val = $('refund-code-input').value.trim();
  const errEl = $('onboarding-error');
  if (!validateRefundCode(val)) {
    errEl.textContent = 'Enter a Lightning address (user@wallet.com) or LNURL1… string.';
    errEl.classList.add('visible');
    return;
  }
  errEl.classList.remove('visible');
  localStorage.setItem(LS_REFUND, val);
  startApp();
}

function startApp() {
  showScreen('screen-app');
  renderRefundCodeDisplay();
}

function renderRefundCodeDisplay() {
  const el = $('refund-code-display');
  if (el) el.textContent = localStorage.getItem(LS_REFUND) || '—';
}

// ── Init ──────────────────────────────────────────────────────────────────────
async function init() {
  // Tab switching
  document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => showTab(btn.dataset.tab));
  });

  // Onboarding
  $('btn-onboarding-submit').addEventListener('click', handleOnboardingSubmit);
  $('refund-code-input').addEventListener('keydown', e => { if (e.key === 'Enter') handleOnboardingSubmit(); });

  // Single voucher
  $('btn-create-single').addEventListener('click', handleCreateSingle);

  // Batch vouchers
  $('btn-create-batch').addEventListener('click', handleCreateBatch);

  // Nav: history
  $('nav-history').addEventListener('click', () => {
    renderHistory();
    showScreen('screen-history');
  });

  $('nav-back-from-history').addEventListener('click', () => showScreen('screen-app'));

  // Change refund code
  $('btn-change-refund').addEventListener('click', () => {
    localStorage.removeItem(LS_REFUND);
    $('refund-code-input').value = localStorage.getItem(LS_REFUND) || '';
    showScreen('screen-onboarding');
  });

  // QR modal close
  $('modal-close-btn').addEventListener('click', () => $('qr-modal').classList.add('hidden'));
  $('qr-modal').addEventListener('click', e => {
    if (e.target === $('qr-modal')) $('qr-modal').classList.add('hidden');
  });

  // Back buttons in single wizard
  $('btn-back-step2').addEventListener('click', () => showStep(1));
  $('btn-back-step3').addEventListener('click', () => showStep(2));

  // Back buttons in batch
  $('btn-back-batch-fund').addEventListener('click', () => showBatchStep('form'));
  $('btn-back-batch-results').addEventListener('click', () => showBatchStep('fund'));

  // Detect dial code and populate
  initSingleStep1();

  // Show correct initial screen
  if (localStorage.getItem(LS_REFUND)) {
    startApp();
  } else {
    showScreen('screen-onboarding');
  }
}

document.addEventListener('DOMContentLoaded', init);
