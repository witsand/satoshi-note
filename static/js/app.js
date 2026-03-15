/* ── Satoshi Note — app.js ── */
'use strict';

// ── Constants ────────────────────────────────────────────────────────────────
const LS_REFUND = 'sn_refund_code';
const LS_HISTORY = 'sn_history';
const LS_COUNTRY = 'sn_country_prefix';

const DIAL_CODES = [
  ['+93',  'Afghanistan (+93)'],
  ['+355', 'Albania (+355)'],
  ['+213', 'Algeria (+213)'],
  ['+376', 'Andorra (+376)'],
  ['+244', 'Angola (+244)'],
  ['+54',  'Argentina (+54)'],
  ['+374', 'Armenia (+374)'],
  ['+297', 'Aruba (+297)'],
  ['+247', 'Ascension Island (+247)'],
  ['+61',  'Australia (+61)'],
  ['+43',  'Austria (+43)'],
  ['+994', 'Azerbaijan (+994)'],
  ['+973', 'Bahrain (+973)'],
  ['+880', 'Bangladesh (+880)'],
  ['+375', 'Belarus (+375)'],
  ['+32',  'Belgium (+32)'],
  ['+501', 'Belize (+501)'],
  ['+229', 'Benin (+229)'],
  ['+975', 'Bhutan (+975)'],
  ['+591', 'Bolivia (+591)'],
  ['+387', 'Bosnia and Herzegovina (+387)'],
  ['+267', 'Botswana (+267)'],
  ['+55',  'Brazil (+55)'],
  ['+246', 'British Indian Ocean Territory (+246)'],
  ['+673', 'Brunei (+673)'],
  ['+359', 'Bulgaria (+359)'],
  ['+257', 'Burundi (+257)'],
  ['+855', 'Cambodia (+855)'],
  ['+237', 'Cameroon (+237)'],
  ['+1',   'Canada / United States (+1)'],
  ['+238', 'Cape Verde (+238)'],
  ['+236', 'Central African Republic (+236)'],
  ['+235', 'Chad (+235)'],
  ['+56',  'Chile (+56)'],
  ['+86',  'China (+86)'],
  ['+57',  'Colombia (+57)'],
  ['+269', 'Comoros (+269)'],
  ['+682', 'Cook Islands (+682)'],
  ['+506', 'Costa Rica (+506)'],
  ['+385', 'Croatia (+385)'],
  ['+53',  'Cuba (+53)'],
  ['+357', 'Cyprus (+357)'],
  ['+420', 'Czech Republic (+420)'],
  ['+45',  'Denmark (+45)'],
  ['+253', 'Djibouti (+253)'],
  ['+243', 'DR Congo (+243)'],
  ['+670', 'East Timor (+670)'],
  ['+593', 'Ecuador (+593)'],
  ['+20',  'Egypt (+20)'],
  ['+503', 'El Salvador (+503)'],
  ['+240', 'Equatorial Guinea (+240)'],
  ['+291', 'Eritrea (+291)'],
  ['+372', 'Estonia (+372)'],
  ['+268', 'Eswatini (+268)'],
  ['+251', 'Ethiopia (+251)'],
  ['+500', 'Falkland Islands (+500)'],
  ['+298', 'Faroe Islands (+298)'],
  ['+679', 'Fiji (+679)'],
  ['+358', 'Finland (+358)'],
  ['+33',  'France (+33)'],
  ['+594', 'French Guiana (+594)'],
  ['+689', 'French Polynesia (+689)'],
  ['+241', 'Gabon (+241)'],
  ['+220', 'Gambia (+220)'],
  ['+995', 'Georgia (+995)'],
  ['+49',  'Germany (+49)'],
  ['+233', 'Ghana (+233)'],
  ['+350', 'Gibraltar (+350)'],
  ['+30',  'Greece (+30)'],
  ['+299', 'Greenland (+299)'],
  ['+590', 'Guadeloupe (+590)'],
  ['+502', 'Guatemala (+502)'],
  ['+224', 'Guinea (+224)'],
  ['+245', 'Guinea-Bissau (+245)'],
  ['+592', 'Guyana (+592)'],
  ['+509', 'Haiti (+509)'],
  ['+504', 'Honduras (+504)'],
  ['+852', 'Hong Kong (+852)'],
  ['+36',  'Hungary (+36)'],
  ['+354', 'Iceland (+354)'],
  ['+91',  'India (+91)'],
  ['+62',  'Indonesia (+62)'],
  ['+98',  'Iran (+98)'],
  ['+964', 'Iraq (+964)'],
  ['+353', 'Ireland (+353)'],
  ['+972', 'Israel (+972)'],
  ['+39',  'Italy (+39)'],
  ['+225', 'Ivory Coast (+225)'],
  ['+81',  'Japan (+81)'],
  ['+962', 'Jordan (+962)'],
  ['+254', 'Kenya (+254)'],
  ['+686', 'Kiribati (+686)'],
  ['+965', 'Kuwait (+965)'],
  ['+996', 'Kyrgyzstan (+996)'],
  ['+856', 'Laos (+856)'],
  ['+371', 'Latvia (+371)'],
  ['+961', 'Lebanon (+961)'],
  ['+266', 'Lesotho (+266)'],
  ['+231', 'Liberia (+231)'],
  ['+218', 'Libya (+218)'],
  ['+423', 'Liechtenstein (+423)'],
  ['+370', 'Lithuania (+370)'],
  ['+352', 'Luxembourg (+352)'],
  ['+853', 'Macau (+853)'],
  ['+261', 'Madagascar (+261)'],
  ['+265', 'Malawi (+265)'],
  ['+60',  'Malaysia (+60)'],
  ['+960', 'Maldives (+960)'],
  ['+223', 'Mali (+223)'],
  ['+356', 'Malta (+356)'],
  ['+692', 'Marshall Islands (+692)'],
  ['+596', 'Martinique (+596)'],
  ['+230', 'Mauritius (+230)'],
  ['+52',  'Mexico (+52)'],
  ['+691', 'Micronesia (+691)'],
  ['+373', 'Moldova (+373)'],
  ['+377', 'Monaco (+377)'],
  ['+976', 'Mongolia (+976)'],
  ['+382', 'Montenegro (+382)'],
  ['+212', 'Morocco (+212)'],
  ['+258', 'Mozambique (+258)'],
  ['+95',  'Myanmar (+95)'],
  ['+264', 'Namibia (+264)'],
  ['+674', 'Nauru (+674)'],
  ['+977', 'Nepal (+977)'],
  ['+31',  'Netherlands (+31)'],
  ['+599', 'Netherlands Antilles (+599)'],
  ['+687', 'New Caledonia (+687)'],
  ['+64',  'New Zealand (+64)'],
  ['+505', 'Nicaragua (+505)'],
  ['+227', 'Niger (+227)'],
  ['+234', 'Nigeria (+234)'],
  ['+683', 'Niue (+683)'],
  ['+672', 'Norfolk Island (+672)'],
  ['+850', 'North Korea (+850)'],
  ['+389', 'North Macedonia (+389)'],
  ['+47',  'Norway (+47)'],
  ['+968', 'Oman (+968)'],
  ['+92',  'Pakistan (+92)'],
  ['+680', 'Palau (+680)'],
  ['+970', 'Palestine (+970)'],
  ['+507', 'Panama (+507)'],
  ['+675', 'Papua New Guinea (+675)'],
  ['+595', 'Paraguay (+595)'],
  ['+51',  'Peru (+51)'],
  ['+63',  'Philippines (+63)'],
  ['+48',  'Poland (+48)'],
  ['+351', 'Portugal (+351)'],
  ['+974', 'Qatar (+974)'],
  ['+242', 'Republic of the Congo (+242)'],
  ['+40',  'Romania (+40)'],
  ['+7',   'Russia (+7)'],
  ['+250', 'Rwanda (+250)'],
  ['+290', 'Saint Helena (+290)'],
  ['+508', 'Saint Pierre and Miquelon (+508)'],
  ['+685', 'Samoa (+685)'],
  ['+378', 'San Marino (+378)'],
  ['+239', 'São Tomé and Príncipe (+239)'],
  ['+966', 'Saudi Arabia (+966)'],
  ['+221', 'Senegal (+221)'],
  ['+381', 'Serbia (+381)'],
  ['+248', 'Seychelles (+248)'],
  ['+232', 'Sierra Leone (+232)'],
  ['+65',  'Singapore (+65)'],
  ['+421', 'Slovakia (+421)'],
  ['+386', 'Slovenia (+386)'],
  ['+677', 'Solomon Islands (+677)'],
  ['+252', 'Somalia (+252)'],
  ['+27',  'South Africa (+27)'],
  ['+82',  'South Korea (+82)'],
  ['+34',  'Spain (+34)'],
  ['+94',  'Sri Lanka (+94)'],
  ['+249', 'Sudan (+249)'],
  ['+597', 'Suriname (+597)'],
  ['+46',  'Sweden (+46)'],
  ['+41',  'Switzerland (+41)'],
  ['+963', 'Syria (+963)'],
  ['+886', 'Taiwan (+886)'],
  ['+992', 'Tajikistan (+992)'],
  ['+255', 'Tanzania (+255)'],
  ['+66',  'Thailand (+66)'],
  ['+228', 'Togo (+228)'],
  ['+690', 'Tokelau (+690)'],
  ['+676', 'Tonga (+676)'],
  ['+216', 'Tunisia (+216)'],
  ['+90',  'Turkey (+90)'],
  ['+993', 'Turkmenistan (+993)'],
  ['+688', 'Tuvalu (+688)'],
  ['+256', 'Uganda (+256)'],
  ['+380', 'Ukraine (+380)'],
  ['+971', 'United Arab Emirates (+971)'],
  ['+44',  'United Kingdom (+44)'],
  ['+598', 'Uruguay (+598)'],
  ['+998', 'Uzbekistan (+998)'],
  ['+678', 'Vanuatu (+678)'],
  ['+58',  'Venezuela (+58)'],
  ['+84',  'Vietnam (+84)'],
  ['+681', 'Wallis and Futuna (+681)'],
  ['+967', 'Yemen (+967)'],
  ['+260', 'Zambia (+260)'],
  ['+263', 'Zimbabwe (+263)'],
];

// ── State ─────────────────────────────────────────────────────────────────────
const state = {
  step: 1,               // single voucher wizard step
  vouchers: null,        // current voucher array from API
  batchStep: 'form',     // 'form' | 'results'
  activeTab: 'single',
};

let _fundPoller = null;
let _dialCode = '+1';
let _singleExpiry = 259200;
let _batchCount = 8;
let _batchExpiry = 7776000;

function buildDialDropdown(preferred) {
  _dialCode = preferred;
  const btn    = $('dial-code-btn');
  const panel  = $('dial-code-panel');
  const search = $('dial-code-search');
  const list   = $('dial-code-list');
  btn.textContent = preferred;

  function renderList(filter) {
    list.innerHTML = '';
    const lf = filter.toLowerCase();
    DIAL_CODES.forEach(([code, label]) => {
      const name = label.replace(/ \([^)]+\)$/, '');
      if (lf && !name.toLowerCase().includes(lf) && !code.includes(lf)) return;
      const li = document.createElement('li');
      li.textContent = `${name} (${code})`;
      li.dataset.code = code;
      if (code === _dialCode) li.classList.add('selected');
      li.onclick = () => {
        _dialCode = code;
        btn.textContent = code;
        btn.classList.remove('open');
        panel.classList.add('hidden');
        search.value = '';
        renderList('');
      };
      list.appendChild(li);
    });
  }

  renderList('');

  btn.onclick = e => {
    e.stopPropagation();
    const opening = panel.classList.contains('hidden');
    panel.classList.toggle('hidden', !opening);
    btn.classList.toggle('open', opening);
    if (opening) { search.value = ''; renderList(''); search.focus(); }
  };

  search.addEventListener('input', () => renderList(search.value));
  search.addEventListener('keydown', e => {
    if (e.key === 'Escape') { panel.classList.add('hidden'); btn.classList.remove('open'); }
  });

  document.addEventListener('click', e => {
    if (!$('dial-code-wrapper').contains(e.target)) {
      panel.classList.add('hidden');
      btn.classList.remove('open');
    }
  });
}

function startFundingPoll(secret) {
  stopFundingPoll();
  _fundPoller = setInterval(async () => {
    try {
      const r = await fetch('/voucher/status/' + secret);
      if (!r.ok) return;
      const s = await r.json();
      if (s.balance_msat > 0) {
        stopFundingPoll();
        renderShareStep(state.vouchers[0]);
        showStep(3);
      }
    } catch { /* network blip — retry next tick */ }
  }, 2500);
}

function stopFundingPoll() {
  if (_fundPoller) { clearInterval(_fundPoller); _fundPoller = null; }
}

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

// ── Dial code detect ──────────────────────────────────────────────────────────
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
  const link = `${window.location.origin}/redeem?lnurl=${encodeURIComponent(claimLnurl)}`;
  const dur = daysFromSeconds(refundAfterSeconds);
  return `⚡ You’ve been sent a *Bitcoin voucher.*\n\nSomeone sent you a small amount of Bitcoin to try for yourself.\n\nClaim it here: ${link}\n\nThe page will show you *how to get a wallet and redeem it step-by-step*. It only takes a few minutes.\n\nTip: If you don't have a wallet yet, *Blink* is a great place to start.\n\nThis voucher expires in ${dur}, so make sure to claim it before then.`;
  // return `You've received a Bitcoin voucher! ⚡\n\nTap this link to claim it:\n${link}\n\nYou'll need a Lightning wallet — try Blink (blink.sv) for beginners.\n\nThis voucher expires in ${dur}.`;
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
  const preferred = await detectDialCode();
  buildDialDropdown(preferred);

  document.querySelectorAll('#single-expiry-pills .pill-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('#single-expiry-pills .pill-btn')
        .forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      _singleExpiry = parseInt(btn.dataset.secs, 10);
    });
  });
}

async function handleCreateSingle() {
  const btn = $('btn-create-single');
  const errEl = $('single-step1-error');
  errEl.classList.remove('visible');

  const dialCode = _dialCode;
  const rawNumber = $('phone-number').value.trim();

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
      refund_after_seconds: _singleExpiry,
      single_use: true,
    });

    state.vouchers = vouchers;

    // Store phone for step 3
    const digits = rawNumber.replace(/\D/g, '').replace(/^0/, '');
    state.e164 = dialCode.replace('+', '') + digits;
    state.dialCode = dialCode;
    state.localNumber = rawNumber;
    state.hasPhone = rawNumber.length > 0;

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
  // Show phone from step 1
  const phoneLine = $('step2-phone-display');
  if (phoneLine) phoneLine.textContent = '+' + state.e164;
  const phoneRow = $('step2-phone-row');
  if (phoneRow) phoneRow.style.display = state.hasPhone ? '' : 'none';
  $('btn-back-step2').textContent = state.hasPhone ? '← Change phone number' : '← Add phone number';

  $('btn-next-step2').onclick = () => {
    stopFundingPoll();
    renderShareStep(voucher);
    showStep(3);
  };

  // QR — click to open wallet
  const container = $('single-qr-container');
  renderQR(container, voucher.fund_lnurl, 256);
  container.style.cursor = 'pointer';
  container.title = 'Tap to open in wallet';
  container.onclick = () => { window.location.href = 'lightning:' + voucher.fund_lnurl; };

  // Truncated LNURL text — click to copy
  const lnurlEl = $('single-lnurl-text');
  const truncated = voucher.fund_lnurl.slice(0, 20) + '…' + voucher.fund_lnurl.slice(-6);
  lnurlEl.textContent = truncated;
  lnurlEl.style.cursor = 'pointer';
  lnurlEl.title = 'Tap to copy';
  lnurlEl.onclick = () => copyToClipboard(voucher.fund_lnurl, lnurlEl);

  // Save to history now (balance is 0 but LNURLs are ready)
  const digits = state.localNumber.replace(/\D/g, '').replace(/^0/, '');
  const e164 = state.dialCode.replace('+', '') + digits;
  pushHistory({
    id: uuidv4(),
    type: 'single',
    createdAt: Math.floor(Date.now() / 1000),
    phone: '+' + e164,
    batchName: voucher.batch_name,
    refundAfterSeconds: voucher.refund_after_seconds,
    vouchers: state.vouchers,
  });

  // Start auto-detection
  startFundingPoll(voucher.secret);
}

function renderShareStep(voucher) {
  const digits = state.localNumber.replace(/\D/g, '').replace(/^0/, '');
  const e164 = state.dialCode.replace('+', '') + digits;
  $('share-phone-display').textContent = '+' + e164;

  // "Sending to" row and WhatsApp button — only when phone was supplied
  $('share-phone-row').style.display = state.hasPhone ? '' : 'none';
  $('btn-whatsapp').style.display = state.hasPhone ? '' : 'none';

  const msg = buildWAMessage(voucher.claim_lnurl, voucher.refund_after_seconds);

  $('btn-whatsapp').onclick = () => {
    const url = `https://wa.me/${e164}?text=${encodeURIComponent(msg)}`;
    window.open(url, '_blank');
  };

  const redeemLink = `${window.location.origin}/redeem?lnurl=${encodeURIComponent(voucher.claim_lnurl)}`;
  const shareBtn = $('btn-share');
  shareBtn.onclick = async () => {
    if (navigator.share) {
      try {
        await navigator.share({
          title: 'Your Bitcoin voucher',
          text: msg,
          // url: redeemLink,
        });
      } catch (e) {
        if (e.name !== 'AbortError') copyToClipboard(redeemLink, shareBtn);
      }
    } else {
      copyToClipboard(redeemLink, shareBtn);
    }
  };

  $('btn-done-single').onclick = () => {
    state.vouchers = null;
    resetSingleWizard();
    showStep(1);
  };

  // Add Funds toggle — shows fund QR inline without leaving step 3
  const addFundsSection = $('add-funds-section');
  $('btn-add-funds').onclick = () => {
    const visible = !addFundsSection.classList.contains('hidden');
    addFundsSection.classList.toggle('hidden', visible);
    $('btn-add-funds').textContent = visible ? 'Add Funds' : 'Hide funding QR';
    if (!visible) {
      const qrEl = $('step3-fund-qr-container');
      renderQR(qrEl, voucher.fund_lnurl, 256);
      qrEl.style.cursor = 'pointer';
      qrEl.onclick = () => { window.location.href = 'lightning:' + voucher.fund_lnurl; };
      const lnEl = $('step3-fund-lnurl-text');
      lnEl.textContent = voucher.fund_lnurl.slice(0, 20) + '…' + voucher.fund_lnurl.slice(-6);
      lnEl.style.cursor = 'pointer';
      lnEl.onclick = () => copyToClipboard(voucher.fund_lnurl, lnEl);
      // Restart polling so auto-advance still works from step 3
      startFundingPoll(voucher.secret);
    } else {
      stopFundingPoll();
    }
  };
}

function resetSingleWizard() {
  stopFundingPoll();
  $('phone-number').value = '';
  $('single-step1-error').classList.remove('visible');
  $('single-qr-container').innerHTML = '';
  $('single-lnurl-text').textContent = '';
}

// ── Batch vouchers ────────────────────────────────────────────────────────────
function showBatchStep(step) {
  state.batchStep = step;
  ['form', 'results'].forEach(s => {
    const el = $(`batch-${s}`);
    if (el) el.classList.toggle('hidden', s !== step);
  });
}

async function handleCreateBatch() {
  const btn = $('btn-create-batch');
  const errEl = $('batch-error');
  errEl.classList.remove('visible');

  const name = $('batch-name').value.trim() || `batch-${Date.now()}`;
  const singleUse = $('batch-single-use').checked;
  const refundCode = localStorage.getItem(LS_REFUND) || '';

  btn.disabled = true;
  const origHTML = btn.innerHTML;
  btn.innerHTML = '<span class="spinner"></span> Creating…';

  try {
    const vouchers = await createVouchers({
      batch_name: name,
      amount: _batchCount,
      refund_code: refundCode,
      refund_after_seconds: _batchExpiry,
      single_use: singleUse,
    });

    state.vouchers = vouchers;
    state.batchExpiry = _batchExpiry;
    state.batchName = name;

    pushHistory({
      id: uuidv4(),
      type: 'batch',
      createdAt: Math.floor(Date.now() / 1000),
      phone: null,
      batchName: name,
      refundAfterSeconds: _batchExpiry,
      vouchers,
    });

    renderBatchResults(vouchers);
    showBatchStep('results');
  } catch (err) {
    errEl.textContent = err.message || 'Failed to create vouchers. Try again.';
    errEl.classList.add('visible');
  } finally {
    btn.disabled = false;
    btn.innerHTML = origHTML;
  }
}

function renderBatchResults(vouchers) {
  // Mini QR grid
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

  // PDF button
  $('btn-download-pdf').onclick = () => downloadPDF(vouchers, state.batchName, state.batchExpiry);

  // Fund QR — same interaction pattern as single tab
  const batchFundLnurl = vouchers[0].batch_fund_lnurl;
  const container = $('batch-qr-container');
  renderQR(container, batchFundLnurl, 256);
  container.style.cursor = 'pointer';
  container.title = 'Tap to open in wallet';
  container.onclick = () => { window.location.href = 'lightning:' + batchFundLnurl; };
  const lnurlEl = $('batch-lnurl-text');
  const truncated = batchFundLnurl.slice(0, 20) + '…' + batchFundLnurl.slice(-6);
  lnurlEl.textContent = truncated;
  lnurlEl.style.cursor = 'pointer';
  lnurlEl.title = 'Tap to copy';
  lnurlEl.onclick = () => copyToClipboard(batchFundLnurl, lnurlEl);
  $('batch-fund-note').textContent = `This funds all ${vouchers.length} vouchers equally.`;

  // Done
  $('btn-done-batch').onclick = () => {
    state.vouchers = null;
    resetBatchForm();
    showBatchStep('form');
  };
}

function resetBatchForm() {
  $('batch-name').value = '';
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
function isAutoName(name) {
  return /^batch-\d+$/.test(name) || /^single-\d+$/.test(name);
}

async function renderHistory() {
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

  // Fetch active status for all vouchers in parallel
  const statusCache = new Map();
  const allSecrets = history.flatMap(e => e.vouchers.map(v => v.secret));
  await Promise.all(allSecrets.map(async secret => {
    try {
      const res = await fetch(`/voucher/status/${secret}`);
      if (res.ok) {
        const data = await res.json();
        statusCache.set(secret, data.active === true);
      } else {
        statusCache.set(secret, false);
      }
    } catch {
      statusCache.set(secret, false);
    }
  }));

  // Filter entries where at least one voucher is still active
  const activeEntries = history.filter(entry =>
    entry.vouchers.some(v => statusCache.get(v.secret))
  );

  if (!activeEntries.length) {
    container.innerHTML = `
      <div class="empty-state">
        <div class="empty-state-icon">🗒️</div>
        <p>No active vouchers.<br>All have been redeemed or expired.</p>
      </div>`;
    return;
  }

  container.innerHTML = '';
  activeEntries.forEach(entry => {
    const card = document.createElement('div');
    card.className = 'history-card';

    const date = new Date(entry.createdAt * 1000).toLocaleString();
    const activeCount = entry.vouchers.filter(v => statusCache.get(v.secret)).length;
    const typeLabel = entry.type === 'single' ? 'Single' : `Batch (${activeCount})`;
    const badge = entry.type === 'single' ? 'badge-single' : 'badge-batch';

    let metaLine;
    if (entry.type === 'single') {
      const phoneDigits = entry.phone ? entry.phone.replace(/\D/g, '') : '';
      metaLine = phoneDigits.length > 3
        ? `${entry.phone} &nbsp;·&nbsp; ${date}`
        : date;
    } else {
      const name = entry.batchName;
      metaLine = (!name || isAutoName(name)) ? date : `${name} &nbsp;·&nbsp; ${date}`;
    }

    card.innerHTML = `
      <div class="history-card-header">
        <span class="badge ${badge}">${typeLabel}</span>
      </div>
      <div class="history-card-meta">
        ${metaLine}
      </div>
      <div class="history-card-actions">
        <button class="btn btn-secondary btn-sm" data-action="reqr" data-id="${entry.id}">Re-show QR</button>
      </div>`;

    container.appendChild(card);
  });

  // Bind action buttons
  container.querySelectorAll('[data-action]').forEach(btn => {
    btn.addEventListener('click', () => {
      const entry = activeEntries.find(e => e.id === btn.dataset.id);
      if (!entry) return;
      if (btn.dataset.action === 'reqr') openQRModal(entry);
    });
  });
}

function openQRModal(entry) {
  const voucher = entry.vouchers[0];
  const lnurl = entry.type === 'single' ? voucher.fund_lnurl : voucher.batch_fund_lnurl;
  const container = $('modal-qr-container');
  renderQR(container, lnurl, 240);
  container.style.cursor = 'pointer';
  container.onclick = () => { window.location.href = 'lightning:' + lnurl; };
  $('modal-title').textContent = entry.type === 'single' ? 'Fund Voucher' : 'Fund Batch';

  const lnurlEl = $('modal-lnurl-text');
  const truncated = lnurl.slice(0, 20) + '…' + lnurl.slice(-6);
  lnurlEl.textContent = truncated;
  lnurlEl.title = 'Tap to copy';
  lnurlEl.onclick = () => copyToClipboard(lnurl, lnurlEl);

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

function initBatchPills() {
  document.querySelectorAll('#batch-count-pills .pill-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('#batch-count-pills .pill-btn')
        .forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      _batchCount = parseInt(btn.dataset.count, 10);
    });
  });
  document.querySelectorAll('#batch-expiry-pills .pill-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('#batch-expiry-pills .pill-btn')
        .forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      _batchExpiry = parseInt(btn.dataset.secs, 10);
    });
  });
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
  $('phone-number').addEventListener('keydown', e => {
    if (e.key === 'Enter') handleCreateSingle();
  });

  // Batch vouchers
  $('btn-create-batch').addEventListener('click', handleCreateBatch);
  $('batch-name').addEventListener('keydown', e => { if (e.key === 'Enter') handleCreateBatch(); });

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
  $('btn-back-step2').addEventListener('click', () => { stopFundingPoll(); showStep(1); });

  // Batch pill pickers
  initBatchPills();

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
