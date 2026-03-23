/* ── Satoshi Note — app.js ── */
'use strict';

// ── Constants ────────────────────────────────────────────────────────────────
const LS_REFUND = 'sn_refund_code';
const LS_HISTORY = 'sn_history';

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

// ── Config + crypto helpers ───────────────────────────────────────────────────
let _randomBytesLength = 16; // safe fallback within server's accepted 16–32 range
const _configReady = (async () => {
  try {
    const res = await fetch('/config');
    if (res.ok) {
      const d = await res.json();
      if (typeof d.random_bytes_length === 'number') _randomBytesLength = d.random_bytes_length;
    }
  } catch (_) {}
})();

function generateSecretHex(length) {
  const bytes = new Uint8Array(length);
  crypto.getRandomValues(bytes);
  return Array.from(bytes).map(b => b.toString(16).padStart(2, '0')).join('');
}

async function secretToPubKey(secretHex) {
  const n = secretHex.length / 2;
  const bytes = new Uint8Array(n);
  for (let i = 0; i < n; i++) bytes[i] = parseInt(secretHex.slice(i * 2, i * 2 + 2), 16);
  const hashBuf = await crypto.subtle.digest('SHA-256', bytes);
  // Take first n bytes of 32-byte hash (matches Go's h[:len(b)])
  const hashBytes = new Uint8Array(hashBuf, 0, n);
  return Array.from(hashBytes).map(b => b.toString(16).padStart(2, '0')).join('');
}

// ── Bech32 LNURL encoder (port of lnurl.go) ───────────────────────────────────
const _BECH32 = 'qpzry9x8gf2tvdw0s3jn54khce6mua7l';
function _b32poly(pre) {
  const b = (pre >>> 25) & 0x1f, c = ((pre & 0x1FFFFFF) * 32) >>> 0;
  return [0x3b6a57b2,0x26508e6d,0x1ea119fa,0x3d4233dd,0x2a1462b3]
    .reduce((acc, g, i) => (b >> i) & 1 ? (acc ^ g) >>> 0 : acc, c);
}
function _b32expand(hrp) {
  const r = [...hrp].map(c => c.charCodeAt(0) >> 5);
  r.push(0);
  return r.concat([...hrp].map(c => c.charCodeAt(0) & 31));
}
function _b32checksum(hrp, data) {
  let p = 1;
  for (const v of [..._b32expand(hrp), ...data, 0,0,0,0,0,0]) p = (_b32poly(p) ^ v) >>> 0;
  p = (p ^ 1) >>> 0;
  return [5,4,3,2,1,0].map(i => (p >>> (5*i)) & 31);
}
function _to5bit(data) {
  let acc = 0, bits = 0; const out = [];
  for (const v of data) {
    acc = ((acc << 8) | v) >>> 0; bits += 8;
    while (bits >= 5) { bits -= 5; out.push((acc >>> bits) & 31); }
  }
  if (bits > 0) out.push((acc << (5 - bits)) & 31);
  return out;
}
function lnurlEncode(url) {
  const data = _to5bit(new TextEncoder().encode(url));
  return ('lnurl1' + [...data, ..._b32checksum('lnurl', data)].map(b => _BECH32[b]).join('')).toUpperCase();
}

let _fundPoller = null;
let _dialCode = '+1';
let _singleExpiry = 259200;
let _batchCount = 8;
let _batchExpiry = 1209600;
let _selectedTemplate = 'classic';

function normalizeToE164(raw, dialCode) {
  const dialDigits = dialCode.replace('+', '');
  const trimmed = raw.trim();
  if (trimmed.startsWith('+')) {
    return trimmed.replace(/\D/g, '');
  }
  const digits = trimmed.replace(/\D/g, '').replace(/^0+/, '');
  if (digits.startsWith(dialDigits)) {
    return digits;
  }
  return dialDigits + digits;
}

const TEMPLATES = [
  { id: 'classic',  name: 'Classic',       desc: 'Orange header, single QR'         },
  { id: 'dual',     name: 'Dual Panel',     desc: 'Two QRs — claim + fund'           },
  { id: 'giftcard', name: 'Gift Card',      desc: 'Dark premium, Bitcoin feel'       },
  { id: 'minimal',  name: 'Minimal',        desc: 'Clean white, side-by-side QRs'   },
  { id: 'darkmode', name: 'Dark Mode',      desc: 'Black background, orange accents' },
  { id: 'fold',     name: 'Fold-in-Half',   desc: '2 per page, hides redeem QR'     },
  { id: 'bizcard',  name: 'Business Card',  desc: '5 per page, wallet-sized'         },
];

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

function startFundingPoll(pubkey) {
  stopFundingPoll();
  _fundPoller = setInterval(async () => {
    try {
      const r = await fetch('/voucher/status/' + pubkey);
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
function defaultDialCode() {
  return (window.DEFAULT_DIAL_CODE && window.DEFAULT_DIAL_CODE.trim()) || '+27';
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

function expiryAfterFundingLabel(refundAfterSeconds) {
  const days = Math.round(refundAfterSeconds / 86400);
  if (days < 14) return `Valid for ${days} days after funding`;
  if (days < 60) return `Valid for ${Math.round(days / 7)} weeks after funding`;
  const months = Math.round(days / 30);
  return `Valid for ${months} month${months > 1 ? 's' : ''} after funding`;
}

// ── WhatsApp message ──────────────────────────────────────────────────────────
function buildWAMessage(claimLnurl, refundAfterSeconds) {
  const link = `${window.location.origin}/redeem?lightning=${encodeURIComponent(claimLnurl)}`;
  const dur = daysFromSeconds(refundAfterSeconds);
  return `I sent you a small amount of Bitcoin to try.\n\nYou can claim it here: ${link}\n\nThis page explains how to get a wallet and redeem it step by step.\n\nThe voucher expires soon, so try claim it when you get a chance.`
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

function initSingleStep1() {
  buildDialDropdown(defaultDialCode());

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

  if (rawNumber.length > 0) {
    // Strip the acceptable "good mistakes": spaces, leading +, parentheses, dashes
    const stripped = rawNumber.replace(/[\s+()\-]/g, '');
    // After removing spaces/+, anything non-digit is invalid
    if (/\D/.test(stripped)) {
      errEl.textContent = 'Please enter a valid phone number — digits and spaces only.';
      errEl.classList.add('visible');
      return;
    }
    // Remove leading zeros (common local format), then check minimum length
    const digits = stripped.replace(/^0+/, '');
    if (digits.length < 6) {
      errEl.textContent = 'That phone number looks too short. Please double-check it.';
      errEl.classList.add('visible');
      return;
    }
  }

  const refundCode = localStorage.getItem(LS_REFUND) || '';
  const ts = Date.now();

  await _configReady;
  const secret = generateSecretHex(_randomBytesLength);
  const pubKey = await secretToPubKey(secret);

  btn.disabled = true;
  const origHTML = btn.innerHTML;
  btn.innerHTML = '<span class="spinner"></span> Creating…';

  try {
    const vouchers = await createVouchers({
      batch_name: `single-${ts}`,
      pub_keys: [pubKey],
      refund_code: refundCode,
      refund_after_seconds: _singleExpiry,
      single_use: true,
    });

    vouchers[0].secret = secret;
    vouchers[0].claim_lnurl = lnurlEncode(vouchers[0].withdraw_url_prefix + secret);
    vouchers[0].fund_lnurl = lnurlEncode(vouchers[0].fund_url_prefix + vouchers[0].pubkey);

    state.vouchers = vouchers;

    // Store phone for step 3
    state.e164 = normalizeToE164(rawNumber, dialCode);
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
  const voucherForRow = $('step2-voucher-for');
  if (voucherForRow) voucherForRow.style.display = state.hasPhone ? '' : 'none';
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
  const e164 = normalizeToE164(state.localNumber, state.dialCode);
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
  startFundingPoll(voucher.pubkey);
}

function renderShareStep(voucher) {
  const e164 = normalizeToE164(state.localNumber, state.dialCode);
  $('share-phone-display').textContent = '+' + e164;

  // "Sending to" row and WhatsApp button — only when phone was supplied
  $('share-phone-row').style.display = state.hasPhone ? '' : 'none';
  $('btn-whatsapp').style.display = state.hasPhone ? '' : 'none';

  const msg = buildWAMessage(voucher.claim_lnurl, voucher.refund_after_seconds);

  $('btn-whatsapp').onclick = () => {
    const url = `https://wa.me/${e164}?text=${encodeURIComponent(msg)}`;
    window.open(url, '_blank');
  };

  const redeemLink = `${window.location.origin}/redeem?lightning=${encodeURIComponent(voucher.claim_lnurl)}`;
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
      startFundingPoll(voucher.pubkey);
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

  await _configReady;
  const secrets = Array.from({ length: _batchCount }, () => generateSecretHex(_randomBytesLength));
  const pubKeys = await Promise.all(secrets.map(s => secretToPubKey(s)));
  const secretByPubKey = Object.fromEntries(pubKeys.map((pk, i) => [pk, secrets[i]]));

  btn.disabled = true;
  const origHTML = btn.innerHTML;
  btn.innerHTML = '<span class="spinner"></span> Creating…';

  try {
    const vouchers = await createVouchers({
      batch_name: name,
      pub_keys: pubKeys,
      refund_code: refundCode,
      refund_after_seconds: _batchExpiry,
      single_use: singleUse,
    });

    for (const v of vouchers) {
      const s = secretByPubKey[v.pubkey];
      v.secret = s;
      v.claim_lnurl = lnurlEncode(v.withdraw_url_prefix + s);
      v.fund_lnurl = lnurlEncode(v.fund_url_prefix + v.pubkey);
      v.batch_fund_lnurl = lnurlEncode(v.batch_fund_url_prefix + v.batch_id);
    }

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
  // Template picker (renders tiles asynchronously)
  renderTemplatePicker(vouchers);

  // Print button
  $('btn-print-vouchers').onclick = () => printVouchers(vouchers, state.batchName, state.batchExpiry);

  // Fund QR
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
  $('template-grid').innerHTML = '';
}

// ── Print / PDF system ────────────────────────────────────────────────────────

async function printVouchers(vouchers, batchName, refundAfterSeconds) {
  const btn = $('btn-print-vouchers');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner"></span> Preparing…';
  try {
    await loadjsPDF();
    const generators = {
      classic:  generatePDFClassic,
      dual:     generatePDFDualPanel,
      giftcard: generatePDFGiftCard,
      minimal:  generatePDFMinimal,
      darkmode: generatePDFDarkMode,
      fold:     generatePDFFold,
      bizcard:  generatePDFBizCard,
    };
    const doc = await generators[_selectedTemplate](vouchers, batchName, refundAfterSeconds);
    await appendBatchFundPage(doc, vouchers, batchName, refundAfterSeconds);
    doc.autoPrint();
    window.open(doc.output('bloburl'), '_blank');
  } catch (err) {
    alert('Print preparation failed: ' + err.message);
  } finally {
    btn.disabled = false;
    btn.textContent = 'Print Vouchers';
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

// ── Canvas helpers for QR / rotation ─────────────────────────────────────────

async function drawQROnCanvas(ctx, dataURL, x, y, size) {
  if (!dataURL) return;
  const img = new Image();
  await new Promise(r => { img.onload = r; img.src = dataURL; });
  ctx.drawImage(img, x, y, size, size);
}

function rotateCanvas180(src) {
  const dst = document.createElement('canvas');
  dst.width = src.width;
  dst.height = src.height;
  const ctx = dst.getContext('2d');
  ctx.translate(dst.width / 2, dst.height / 2);
  ctx.rotate(Math.PI);
  ctx.drawImage(src, -src.width / 2, -src.height / 2);
  return dst;
}

// ── PDF generators ────────────────────────────────────────────────────────────

async function generatePDFClassic(vouchers, batchName, refundAfterSeconds) {
  const { jsPDF } = window.jspdf;
  const doc = new jsPDF({ orientation: 'landscape', unit: 'mm', format: 'a4' });
  const W = 297, H = 210;
  const displayName = batchName && !isAutoName(batchName) ? batchName : 'Lightning Voucher';

  for (let i = 0; i < vouchers.length; i++) {
    if (i > 0) doc.addPage();
    const v = vouchers[i];

    doc.setFillColor(255, 255, 255);
    doc.rect(0, 0, W, H, 'F');

    doc.setFillColor(247, 147, 26);
    doc.rect(0, 0, W, 22, 'F');
    doc.setTextColor(0, 0, 0);
    doc.setFontSize(13);
    doc.setFont('helvetica', 'bold');
    doc.text((window.SITE_NAME||'Satoshi Note'), 8, 14);
    doc.setFont('helvetica', 'normal');
    doc.setFontSize(9);
    doc.text('Lightning Voucher', 8, 20);

    const claimWebURL = `${window.location.origin}/redeem?lightning=${encodeURIComponent(v.claim_lnurl)}`;
    const qrDataURL = await qrToDataURL(claimWebURL, 220);
    if (qrDataURL) doc.addImage(qrDataURL, 'PNG', 10, 28, 78, 78);

    doc.setTextColor(15, 15, 15);
    doc.setFont('helvetica', 'bold');
    doc.setFontSize(15);
    doc.text(displayName, 100, 42);
    doc.setFont('helvetica', 'normal');
    doc.setFontSize(10);
    doc.setTextColor(80, 80, 80);
    doc.text(expiryAfterFundingLabel(refundAfterSeconds), 100, 54);
    doc.setFontSize(9);
    doc.text('How to redeem:', 100, 70);
    doc.text('1. Install a Lightning wallet (try blink.sv)', 100, 79);
    doc.text('2. Open the app and tap Receive / Scan', 100, 87);
    doc.text('3. Scan the QR code on the left', 100, 95);
    doc.text('4. Confirm to receive your Bitcoin', 100, 103);

    doc.setFillColor(247, 147, 26);
    doc.rect(0, H - 12, W, 12, 'F');
    doc.setTextColor(0, 0, 0);
    doc.setFontSize(7);
    doc.text('Scan with a Lightning wallet · blink.sv for beginners', W / 2, H - 4, { align: 'center' });
  }
  return doc;
}

async function generatePDFDualPanel(vouchers, batchName, refundAfterSeconds) {
  const { jsPDF } = window.jspdf;
  const doc = new jsPDF({ orientation: 'landscape', unit: 'mm', format: 'a4' });
  const W = 297, H = 210;

  for (let i = 0; i < vouchers.length; i++) {
    if (i > 0) doc.addPage();
    const v = vouchers[i];

    doc.setFillColor(20, 20, 35);
    doc.rect(0, 0, W, H, 'F');
    doc.setFillColor(30, 30, 55);
    doc.rect(0, 0, W, 24, 'F');
    doc.setTextColor(247, 147, 26);
    doc.setFontSize(14);
    doc.setFont('helvetica', 'bold');
    doc.text('Lightning Voucher', 10, 16);
    doc.setTextColor(180, 180, 200);
    doc.setFont('helvetica', 'normal');
    doc.setFontSize(8);
    doc.text((window.SITE_NAME||'Satoshi Note'), W - 10, 16, { align: 'right' });

    const claimWebURL = `${window.location.origin}/redeem?lightning=${encodeURIComponent(v.claim_lnurl)}`;
    const claimQR = await qrToDataURL(claimWebURL, 220);
    if (claimQR) doc.addImage(claimQR, 'PNG', 10, 30, 100, 100);
    doc.setTextColor(255, 255, 255);
    doc.setFont('helvetica', 'bold');
    doc.setFontSize(11);
    doc.text('SCAN TO CLAIM', 10, 140);
    doc.setFont('helvetica', 'normal');
    doc.setFontSize(8);
    doc.setTextColor(180, 180, 200);
    doc.text('Open Lightning wallet and scan', 10, 148);

    doc.setDrawColor(60, 60, 90);
    doc.setLineWidth(0.5);
    doc.line(W / 2, 24, W / 2, H - 12);

    const fundURL = `lightning:${v.fund_lnurl}`;
    const fundQR = await qrToDataURL(fundURL, 160);
    if (fundQR) doc.addImage(fundQR, 'PNG', W / 2 + 10, 34, 75, 75);
    doc.setTextColor(247, 147, 26);
    doc.setFont('helvetica', 'bold');
    doc.setFontSize(10);
    doc.text('SCAN TO ADD FUNDS', W / 2 + 10, 118);
    doc.setFont('helvetica', 'normal');
    doc.setFontSize(8);
    doc.setTextColor(180, 180, 200);
    doc.text(expiryAfterFundingLabel(refundAfterSeconds), W / 2 + 10, 128);

    doc.setFillColor(247, 147, 26);
    doc.rect(0, H - 12, W, 12, 'F');
    doc.setTextColor(0, 0, 0);
    doc.setFontSize(7);
    doc.text((window.SITE_NAME||'Satoshi Note') + ' · Bitcoin Lightning Voucher', W / 2, H - 4, { align: 'center' });
  }
  return doc;
}

async function generatePDFGiftCard(vouchers, batchName, refundAfterSeconds) {
  const { jsPDF } = window.jspdf;
  const doc = new jsPDF({ orientation: 'landscape', unit: 'mm', format: 'a4' });
  const W = 297, H = 210;

  for (let i = 0; i < vouchers.length; i++) {
    if (i > 0) doc.addPage();
    const v = vouchers[i];

    doc.setFillColor(18, 12, 6);
    doc.rect(0, 0, W, H, 'F');
    doc.setFillColor(247, 147, 26);
    doc.rect(0, 0, 24, H, 'F');
    doc.setTextColor(0, 0, 0);
    doc.setFont('helvetica', 'bold');
    doc.setFontSize(11);
    doc.text('SATOSHI NOTE', 15, H / 2 + 28, { angle: 90 });

    doc.setTextColor(38, 25, 10);
    doc.setFontSize(140);
    doc.text('\u20BF', W * 0.62, H * 0.72, { align: 'center' });

    const claimWebURL = `${window.location.origin}/redeem?lightning=${encodeURIComponent(v.claim_lnurl)}`;
    const claimQR = await qrToDataURL(claimWebURL, 260);
    if (claimQR) {
      doc.setFillColor(255, 255, 255);
      doc.rect(W / 2 - 50, 22, 100, 100, 'F');
      doc.addImage(claimQR, 'PNG', W / 2 - 48, 24, 96, 96);
    }
    doc.setTextColor(247, 147, 26);
    doc.setFont('helvetica', 'bold');
    doc.setFontSize(14);
    doc.text('SCAN TO CLAIM', W / 2, 132, { align: 'center' });
    doc.setFont('helvetica', 'normal');
    doc.setFontSize(9);
    doc.setTextColor(200, 180, 160);
    doc.text(expiryAfterFundingLabel(refundAfterSeconds), W / 2, 142, { align: 'center' });

    const fundURL = `lightning:${v.fund_lnurl}`;
    const fundQR = await qrToDataURL(fundURL, 120);
    if (fundQR) {
      doc.setFillColor(255, 255, 255);
      doc.rect(W - 54, H - 56, 44, 44, 'F');
      doc.addImage(fundQR, 'PNG', W - 53, H - 55, 42, 42);
    }
    doc.setTextColor(140, 130, 110);
    doc.setFontSize(7);
    doc.text('Add funds', W - 32, H - 8, { align: 'center' });
  }
  return doc;
}

async function generatePDFMinimal(vouchers, batchName, refundAfterSeconds) {
  const { jsPDF } = window.jspdf;
  const doc = new jsPDF({ orientation: 'landscape', unit: 'mm', format: 'a4' });
  const W = 297, H = 210;
  const displayName = batchName && !isAutoName(batchName) ? batchName : 'Lightning Voucher';

  for (let i = 0; i < vouchers.length; i++) {
    if (i > 0) doc.addPage();
    const v = vouchers[i];

    doc.setFillColor(250, 250, 250);
    doc.rect(0, 0, W, H, 'F');
    doc.setFillColor(247, 147, 26);
    doc.rect(0, 0, W, 3, 'F');
    doc.rect(0, H - 3, W, 3, 'F');

    doc.setTextColor(20, 20, 20);
    doc.setFont('helvetica', 'bold');
    doc.setFontSize(16);
    doc.text(displayName, W / 2, 20, { align: 'center' });

    const qrSize = 88;
    const claimWebURL = `${window.location.origin}/redeem?lightning=${encodeURIComponent(v.claim_lnurl)}`;
    const claimQR = await qrToDataURL(claimWebURL, 220);
    const fundURL = `lightning:${v.fund_lnurl}`;
    const fundQR = await qrToDataURL(fundURL, 220);

    const leftX = W / 2 - qrSize - 12;
    const rightX = W / 2 + 12;
    const qrY = 28;

    if (claimQR) doc.addImage(claimQR, 'PNG', leftX, qrY, qrSize, qrSize);
    if (fundQR) doc.addImage(fundQR, 'PNG', rightX, qrY, qrSize, qrSize);

    doc.setTextColor(40, 40, 40);
    doc.setFont('helvetica', 'bold');
    doc.setFontSize(10);
    doc.text('CLAIM', leftX + qrSize / 2, qrY + qrSize + 10, { align: 'center' });
    doc.text('TOP UP', rightX + qrSize / 2, qrY + qrSize + 10, { align: 'center' });

    doc.setFont('helvetica', 'normal');
    doc.setFontSize(9);
    doc.setTextColor(100, 100, 100);
    doc.text(expiryAfterFundingLabel(refundAfterSeconds), W / 2, H - 10, { align: 'center' });
  }
  return doc;
}

async function generatePDFDarkMode(vouchers, batchName, refundAfterSeconds) {
  const { jsPDF } = window.jspdf;
  const doc = new jsPDF({ orientation: 'landscape', unit: 'mm', format: 'a4' });
  const W = 297, H = 210;

  for (let i = 0; i < vouchers.length; i++) {
    if (i > 0) doc.addPage();
    const v = vouchers[i];

    doc.setFillColor(13, 13, 13);
    doc.rect(0, 0, W, H, 'F');
    doc.setFillColor(247, 147, 26);
    doc.rect(0, 0, W, 2, 'F');
    doc.rect(0, H - 2, W, 2, 'F');

    doc.setTextColor(247, 147, 26);
    doc.setFont('helvetica', 'bold');
    doc.setFontSize(18);
    doc.text('Lightning Voucher', W / 2, 18, { align: 'center' });

    doc.setTextColor(28, 28, 28);
    doc.setFontSize(110);
    doc.text('\u20BF', W * 0.73, H * 0.70, { align: 'center' });

    const claimWebURL = `${window.location.origin}/redeem?lightning=${encodeURIComponent(v.claim_lnurl)}`;
    const claimQR = await qrToDataURL(claimWebURL, 260);
    if (claimQR) {
      doc.setFillColor(255, 255, 255);
      doc.rect(10, 24, 96, 96, 'F');
      doc.addImage(claimQR, 'PNG', 12, 26, 92, 92);
    }
    doc.setTextColor(255, 255, 255);
    doc.setFont('helvetica', 'bold');
    doc.setFontSize(10);
    doc.text('SCAN TO CLAIM', 10, 130);
    doc.setFont('helvetica', 'normal');
    doc.setFontSize(8);
    doc.setTextColor(140, 140, 140);
    doc.text(expiryAfterFundingLabel(refundAfterSeconds), 10, 140);

    const fundURL = `lightning:${v.fund_lnurl}`;
    const fundQR = await qrToDataURL(fundURL, 160);
    if (fundQR) {
      doc.setFillColor(255, 255, 255);
      doc.rect(W * 0.42, 30, 70, 70, 'F');
      doc.addImage(fundQR, 'PNG', W * 0.42 + 1, 31, 68, 68);
    }
    doc.setTextColor(247, 147, 26);
    doc.setFont('helvetica', 'bold');
    doc.setFontSize(9);
    doc.text('ADD FUNDS', W * 0.42 + 35, 110, { align: 'center' });
  }
  return doc;
}

// ── Canvas strip renderers for Fold + BizCard ─────────────────────────────────

async function renderFoldOutside(v, Wpx, Hpx, batchName, refundAfterSeconds) {
  const canvas = document.createElement('canvas');
  canvas.width = Wpx; canvas.height = Hpx;
  const ctx = canvas.getContext('2d');

  ctx.fillStyle = '#fffaf5';
  ctx.fillRect(0, 0, Wpx, Hpx);
  const hdr = Math.round(Hpx * 0.14);
  ctx.fillStyle = '#F7931A';
  ctx.fillRect(0, 0, Wpx, hdr);
  ctx.fillStyle = '#000';
  ctx.font = `bold ${Math.round(hdr * 0.6)}px sans-serif`;
  ctx.fillText('SATOSHI NOTE', Math.round(Wpx * 0.04), Math.round(hdr * 0.75));

  const fundURL = `lightning:${v.fund_lnurl}`;
  const fundQR = await qrToDataURL(fundURL, 200);
  const qrSize = Math.round(Math.min(Wpx * 0.55, Hpx * 0.52));
  const qrX = Math.round((Wpx - qrSize) / 2);
  const qrY = hdr + Math.round((Hpx - hdr - Hpx * 0.18 - qrSize) / 2);
  if (fundQR) {
    ctx.fillStyle = '#fff';
    ctx.fillRect(qrX - 3, qrY - 3, qrSize + 6, qrSize + 6);
    await drawQROnCanvas(ctx, fundQR, qrX, qrY, qrSize);
  }

  ctx.textAlign = 'center';
  ctx.fillStyle = '#333';
  ctx.font = `bold ${Math.round(Hpx * 0.07)}px sans-serif`;
  ctx.fillText('Scan to add funds', Wpx / 2, Hpx - Math.round(Hpx * 0.09));
  ctx.fillStyle = '#666';
  ctx.font = `${Math.round(Hpx * 0.055)}px sans-serif`;
  const name = batchName && !isAutoName(batchName) ? batchName : '';
  if (name) ctx.fillText(name, Wpx / 2, Hpx - Math.round(Hpx * 0.02));
  ctx.textAlign = 'left';
  return canvas;
}

async function renderFoldInside(v, Wpx, Hpx, refundAfterSeconds) {
  const canvas = document.createElement('canvas');
  canvas.width = Wpx; canvas.height = Hpx;
  const ctx = canvas.getContext('2d');

  ctx.fillStyle = '#111';
  ctx.fillRect(0, 0, Wpx, Hpx);

  const claimWebURL = `${window.location.origin}/redeem?lightning=${encodeURIComponent(v.claim_lnurl)}`;
  const claimQR = await qrToDataURL(claimWebURL, 200);
  const qrSize = Math.round(Math.min(Wpx * 0.55, Hpx * 0.52));
  const qrX = Math.round((Wpx - qrSize) / 2);
  const qrY = Math.round(Hpx * 0.1);
  if (claimQR) {
    ctx.fillStyle = '#fff';
    ctx.fillRect(qrX - 3, qrY - 3, qrSize + 6, qrSize + 6);
    await drawQROnCanvas(ctx, claimQR, qrX, qrY, qrSize);
  }

  ctx.textAlign = 'center';
  ctx.fillStyle = '#F7931A';
  ctx.font = `bold ${Math.round(Hpx * 0.09)}px sans-serif`;
  ctx.fillText('SCAN TO CLAIM', Wpx / 2, Hpx - Math.round(Hpx * 0.12));
  ctx.fillStyle = '#aaa';
  ctx.font = `${Math.round(Hpx * 0.065)}px sans-serif`;
  ctx.fillText(expiryAfterFundingLabel(refundAfterSeconds), Wpx / 2, Hpx - Math.round(Hpx * 0.04));
  ctx.textAlign = 'left';
  return canvas;
}

async function generatePDFFold(vouchers, batchName, refundAfterSeconds) {
  const { jsPDF } = window.jspdf;
  const doc = new jsPDF({ orientation: 'landscape', unit: 'mm', format: 'a4' });
  const W = 297, H = 210, stripH = 105, midX = W / 2;
  const px = 3.78; // mm to px at ~96dpi
  const halfWpx = Math.round(midX * px);
  const stripHpx = Math.round(stripH * px);

  for (let p = 0; p < Math.ceil(vouchers.length / 2); p++) {
    if (p > 0) doc.addPage();
    const v1 = vouchers[p * 2];
    const v2 = vouchers[p * 2 + 1] || null;

    doc.setFillColor(255, 255, 255);
    doc.rect(0, 0, W, H, 'F');

    // Voucher 1 — top strip
    const out1 = await renderFoldOutside(v1, halfWpx, stripHpx, batchName, refundAfterSeconds);
    const in1  = await renderFoldInside(v1, halfWpx, stripHpx, refundAfterSeconds);
    doc.addImage(out1.toDataURL('image/png'), 'PNG', 0,    0, midX,  stripH);
    doc.addImage(in1.toDataURL('image/png'),  'PNG', midX, 0, midX,  stripH);

    // Voucher 2 — bottom strip, rotated 180°
    if (v2) {
      const out2 = await renderFoldOutside(v2, halfWpx, stripHpx, batchName, refundAfterSeconds);
      const in2  = await renderFoldInside(v2, halfWpx, stripHpx, refundAfterSeconds);
      // Rotated: inside→left side, outside→right side
      doc.addImage(rotateCanvas180(in2).toDataURL('image/png'),  'PNG', 0,    stripH, midX, stripH);
      doc.addImage(rotateCanvas180(out2).toDataURL('image/png'), 'PNG', midX, stripH, midX, stripH);
    }

    // Dashed horizontal cut line at y=105
    doc.setDrawColor(120, 120, 120);
    doc.setLineDashPattern([3, 3], 0);
    doc.setLineWidth(0.4);
    doc.line(0, stripH, W, stripH);
    doc.setLineDashPattern([], 0);
    doc.setTextColor(100, 100, 100);
    doc.setFontSize(6);
    doc.setFont('helvetica', 'normal');
    doc.text('\u2702 CUT HERE', W / 2 - 8, stripH - 0.5);

    // Dashed vertical fold lines
    doc.setLineDashPattern([3, 3], 0);
    doc.line(midX, 0, midX, v2 ? H : stripH);
    doc.setLineDashPattern([], 0);
    doc.text('FOLD', midX + 1, stripH / 2, { angle: 270 });
    if (v2) doc.text('FOLD', midX + 1, stripH + stripH / 2, { angle: 270 });
  }
  return doc;
}

async function renderBizCardOutside(v, Wpx, Hpx, batchName) {
  const canvas = document.createElement('canvas');
  canvas.width = Wpx; canvas.height = Hpx;
  const ctx = canvas.getContext('2d');

  ctx.fillStyle = '#fff';
  ctx.fillRect(0, 0, Wpx, Hpx);
  const hdr = Math.round(Hpx * 0.20);
  ctx.fillStyle = '#F7931A';
  ctx.fillRect(0, 0, Wpx, hdr);
  ctx.fillStyle = '#000';
  ctx.font = `bold ${Math.round(hdr * 0.55)}px sans-serif`;
  ctx.fillText((window.SITE_NAME||'Satoshi Note'), Math.round(Wpx * 0.04), Math.round(hdr * 0.72));

  const fundURL = `lightning:${v.fund_lnurl}`;
  const fundQR = await qrToDataURL(fundURL, 140);
  const qrSize = Math.round(Hpx * 0.62);
  const qrX = Math.round(Wpx * 0.05);
  const qrY = hdr + Math.round((Hpx - hdr - qrSize) / 2);
  if (fundQR) {
    ctx.fillStyle = '#fff';
    ctx.fillRect(qrX - 2, qrY - 2, qrSize + 4, qrSize + 4);
    await drawQROnCanvas(ctx, fundQR, qrX, qrY, qrSize);
  }

  ctx.fillStyle = '#333';
  ctx.font = `${Math.round(Hpx * 0.11)}px sans-serif`;
  ctx.fillText('Scan to add funds', qrX + qrSize + Math.round(Wpx * 0.05), qrY + Math.round(qrSize * 0.45));
  ctx.fillStyle = '#888';
  ctx.font = `${Math.round(Hpx * 0.09)}px sans-serif`;
  ctx.fillText('Lightning voucher', qrX + qrSize + Math.round(Wpx * 0.05), qrY + Math.round(qrSize * 0.65));

  return canvas;
}

async function renderBizCardInside(v, Wpx, Hpx) {
  const canvas = document.createElement('canvas');
  canvas.width = Wpx; canvas.height = Hpx;
  const ctx = canvas.getContext('2d');

  ctx.fillStyle = '#111';
  ctx.fillRect(0, 0, Wpx, Hpx);

  const claimWebURL = `${window.location.origin}/redeem?lightning=${encodeURIComponent(v.claim_lnurl)}`;
  const claimQR = await qrToDataURL(claimWebURL, 140);
  const qrSize = Math.round(Hpx * 0.62);
  const qrX = Math.round(Wpx * 0.05);
  const qrY = Math.round((Hpx - qrSize) / 2);
  if (claimQR) {
    ctx.fillStyle = '#fff';
    ctx.fillRect(qrX - 2, qrY - 2, qrSize + 4, qrSize + 4);
    await drawQROnCanvas(ctx, claimQR, qrX, qrY, qrSize);
  }

  ctx.textAlign = 'right';
  ctx.fillStyle = '#F7931A';
  ctx.font = `bold ${Math.round(Hpx * 0.13)}px sans-serif`;
  ctx.fillText('SCAN TO CLAIM', Wpx - Math.round(Wpx * 0.04), qrY + Math.round(qrSize * 0.40));
  ctx.fillStyle = '#aaa';
  ctx.font = `${Math.round(Hpx * 0.10)}px sans-serif`;
  ctx.fillText('Open Lightning wallet', Wpx - Math.round(Wpx * 0.04), qrY + Math.round(qrSize * 0.62));
  ctx.fillText('& scan to receive BTC', Wpx - Math.round(Wpx * 0.04), qrY + Math.round(qrSize * 0.80));
  ctx.textAlign = 'left';
  return canvas;
}

async function generatePDFBizCard(vouchers, batchName, refundAfterSeconds) {
  const { jsPDF } = window.jspdf;
  const doc = new jsPDF({ orientation: 'portrait', unit: 'mm', format: 'a4' });
  const W = 210, H = 297;
  const cardW = 185, cardH = 55, halfW = 92.5;
  const perPage = 5, gap = 4;
  const topMargin = (H - perPage * cardH - (perPage - 1) * gap) / 2;
  const leftMargin = (W - cardW) / 2;
  const px = 3.78;
  const halfWpx = Math.round(halfW * px);
  const cardHpx = Math.round(cardH * px);

  for (let p = 0; p < Math.ceil(vouchers.length / perPage); p++) {
    if (p > 0) doc.addPage('a4', 'portrait');
    doc.setFillColor(255, 255, 255);
    doc.rect(0, 0, W, H, 'F');

    for (let c = 0; c < perPage; c++) {
      const vIdx = p * perPage + c;
      if (vIdx >= vouchers.length) break;
      const v = vouchers[vIdx];
      const cardY = topMargin + c * (cardH + gap);

      const outside = await renderBizCardOutside(v, halfWpx, cardHpx, batchName);
      doc.addImage(outside.toDataURL('image/png'), 'PNG', leftMargin, cardY, halfW, cardH);

      const inside = await renderBizCardInside(v, halfWpx, cardHpx);
      doc.addImage(rotateCanvas180(inside).toDataURL('image/png'), 'PNG', leftMargin + halfW, cardY, halfW, cardH);

      // Fold line
      doc.setDrawColor(140, 140, 140);
      doc.setLineDashPattern([2, 2], 0);
      doc.setLineWidth(0.3);
      doc.line(leftMargin + halfW, cardY, leftMargin + halfW, cardY + cardH);
      doc.setLineDashPattern([], 0);

      // Card border
      doc.setDrawColor(210, 210, 210);
      doc.setLineWidth(0.2);
      doc.rect(leftMargin, cardY, cardW, cardH);
    }
  }
  return doc;
}

async function appendBatchFundPage(doc, vouchers, batchName, refundAfterSeconds) {
  doc.addPage([297, 210], 'l');
  const W = 297, H = 210;

  doc.setFillColor(255, 255, 255);
  doc.rect(0, 0, W, H, 'F');

  doc.setFillColor(247, 147, 26);
  doc.rect(0, 0, W, 26, 'F');
  doc.setTextColor(0, 0, 0);
  doc.setFont('helvetica', 'bold');
  doc.setFontSize(15);
  doc.text('FUND ALL VOUCHERS', W / 2, 16, { align: 'center' });
  doc.setFont('helvetica', 'normal');
  doc.setFontSize(9);
  doc.text(`Funds all ${vouchers.length} vouchers equally`, W / 2, 23, { align: 'center' });

  if (batchName && !isAutoName(batchName)) {
    doc.setTextColor(20, 20, 20);
    doc.setFont('helvetica', 'bold');
    doc.setFontSize(18);
    doc.text(batchName, W / 2, 42, { align: 'center' });
  }

  const batchFundLNURL = vouchers[0].batch_fund_lnurl;
  const qrDataURL = await qrToDataURL(batchFundLNURL, 300);
  if (qrDataURL) {
    doc.setFillColor(255, 255, 255);
    doc.rect(W / 2 - 52, 48, 104, 104, 'F');
    doc.addImage(qrDataURL, 'PNG', W / 2 - 50, 50, 100, 100);
  }

  const textY = 160;
  doc.setTextColor(50, 50, 50);
  doc.setFont('helvetica', 'normal');
  doc.setFontSize(9);
  doc.text(`Created: ${new Date().toLocaleDateString()}`, W / 2, textY, { align: 'center' });
  doc.text(expiryAfterFundingLabel(refundAfterSeconds), W / 2, textY + 8, { align: 'center' });
  doc.text('Open your Lightning wallet and scan this code', W / 2, textY + 18, { align: 'center' });

  doc.setFillColor(247, 147, 26);
  doc.rect(0, H - 12, W, 12, 'F');
  doc.setTextColor(0, 0, 0);
  doc.setFontSize(7);
  doc.text((window.SITE_NAME||'Satoshi Note') + ' · Bitcoin Lightning Vouchers', W / 2, H - 4, { align: 'center' });
}

// ── Template picker UI ────────────────────────────────────────────────────────

async function renderTemplatePicker(vouchers) {
  const grid = $('template-grid');
  grid.innerHTML = '<p style="font-size:0.8rem;color:var(--text-muted);">Generating previews…</p>';

  const v = vouchers[0];
  const claimWebURL = `${window.location.origin}/redeem?lightning=${encodeURIComponent(v.claim_lnurl)}`;
  const fundLightningURL = `lightning:${v.fund_lnurl}`;

  const qrs = {
    claim:    await qrToDataURL(claimWebURL, 200),
    fund:     await qrToDataURL(fundLightningURL, 200),
    batchFund: await qrToDataURL(v.batch_fund_lnurl, 200),
    rawClaim: await qrToDataURL(v.claim_lnurl, 200),
  };

  grid.innerHTML = '';

  for (const tpl of TEMPLATES) {
    const tile = document.createElement('div');
    tile.className = 'template-tile' + (tpl.id === _selectedTemplate ? ' selected' : '');
    tile.dataset.id = tpl.id;

    const isPortrait = tpl.id === 'bizcard';
    const canvas = document.createElement('canvas');
    canvas.width  = isPortrait ? 254 : 360;
    canvas.height = isPortrait ? 360 : 254;

    const nameEl = document.createElement('div');
    nameEl.className = 'template-tile-name';
    nameEl.textContent = tpl.name;

    const descEl = document.createElement('div');
    descEl.className = 'template-tile-desc';
    descEl.textContent = tpl.desc;

    tile.appendChild(canvas);
    tile.appendChild(nameEl);
    tile.appendChild(descEl);

    tile.addEventListener('click', () => {
      _selectedTemplate = tpl.id;
      document.querySelectorAll('.template-tile').forEach(t => t.classList.remove('selected'));
      tile.classList.add('selected');
      openTemplatePreview(tpl.id, qrs, state.batchName, state.batchExpiry);
    });

    grid.appendChild(tile);

    // Draw preview asynchronously per tile
    drawTemplatePreview(canvas, tpl.id, qrs, state.batchName, state.batchExpiry);
  }
}

async function drawTemplatePreview(canvas, templateId, qrs, batchName, refundAfterSeconds) {
  const ctx = canvas.getContext('2d');
  const w = canvas.width, h = canvas.height;
  ctx.clearRect(0, 0, w, h);
  const fns = {
    classic:  drawClassicPreview,
    dual:     drawDualPreview,
    giftcard: drawGiftCardPreview,
    minimal:  drawMinimalPreview,
    darkmode: drawDarkModePreview,
    fold:     drawFoldPreview,
    bizcard:  drawBizCardPreview,
  };
  if (fns[templateId]) await fns[templateId](ctx, w, h, qrs, batchName, refundAfterSeconds);
}

async function openTemplatePreview(templateId, qrs, batchName, refundAfterSeconds) {
  const modal = $('template-preview-modal');
  const tpl = TEMPLATES.find(t => t.id === templateId);
  $('template-preview-title').textContent = tpl ? tpl.name : templateId;
  const isPortrait = templateId === 'bizcard';
  const canvas = $('template-preview-canvas');
  canvas.width  = isPortrait ? 565 : 800;
  canvas.height = isPortrait ? 800 : 565;
  modal.classList.remove('hidden');
  await drawTemplatePreview(canvas, templateId, qrs, batchName, refundAfterSeconds);
}

// ── Preview draw functions ────────────────────────────────────────────────────

async function drawClassicPreview(ctx, w, h, qrs, batchName, refundAfterSeconds) {
  const ORANGE = '#F7931A';
  ctx.fillStyle = '#fff';
  ctx.fillRect(0, 0, w, h);

  const hdr = Math.round(h * 0.14);
  ctx.fillStyle = ORANGE;
  ctx.fillRect(0, 0, w, hdr);
  ctx.fillStyle = '#000';
  ctx.font = `bold ${Math.round(hdr * 0.58)}px sans-serif`;
  ctx.fillText((window.SITE_NAME||'Satoshi Note'), Math.round(w * 0.02), Math.round(hdr * 0.72));
  ctx.font = `${Math.round(hdr * 0.42)}px sans-serif`;
  ctx.fillText('Lightning Voucher', Math.round(w * 0.02), Math.round(hdr * 0.95));

  const qrSize = Math.round(h * 0.54);
  const qrX = Math.round(w * 0.03);
  const qrY = hdr + Math.round((h - hdr - h * 0.10 - qrSize) / 2);
  if (qrs.rawClaim) await drawQROnCanvas(ctx, qrs.rawClaim, qrX, qrY, qrSize);

  const rx = Math.round(w * 0.42);
  ctx.fillStyle = '#111';
  ctx.font = `bold ${Math.round(h * 0.06)}px sans-serif`;
  const name = batchName && !isAutoName(batchName) ? batchName : 'Lightning Voucher';
  ctx.fillText(name.slice(0, 20), rx, qrY + Math.round(h * 0.08));
  ctx.fillStyle = '#555';
  ctx.font = `${Math.round(h * 0.048)}px sans-serif`;
  ctx.fillText(expiryAfterFundingLabel(refundAfterSeconds), rx, qrY + Math.round(h * 0.20));
  ctx.fillText('How to redeem:', rx, qrY + Math.round(h * 0.34));
  ctx.font = `${Math.round(h * 0.042)}px sans-serif`;
  ctx.fillText('1. Install Lightning wallet', rx, qrY + Math.round(h * 0.44));
  ctx.fillText('2. Tap Receive / Scan', rx, qrY + Math.round(h * 0.53));
  ctx.fillText('3. Scan the QR code', rx, qrY + Math.round(h * 0.62));
  ctx.fillText('4. Confirm to receive', rx, qrY + Math.round(h * 0.71));

  ctx.fillStyle = ORANGE;
  ctx.fillRect(0, h - Math.round(h * 0.10), w, Math.round(h * 0.10));
  ctx.fillStyle = '#000';
  ctx.font = `${Math.round(h * 0.042)}px sans-serif`;
  ctx.textAlign = 'center';
  ctx.fillText('Scan with a Lightning wallet', w / 2, h - Math.round(h * 0.03));
  ctx.textAlign = 'left';
}

async function drawDualPreview(ctx, w, h, qrs, batchName, refundAfterSeconds) {
  ctx.fillStyle = '#14141f';
  ctx.fillRect(0, 0, w, h);
  const hdr = Math.round(h * 0.15);
  ctx.fillStyle = '#1e1e37';
  ctx.fillRect(0, 0, w, hdr);
  ctx.fillStyle = '#F7931A';
  ctx.font = `bold ${Math.round(hdr * 0.52)}px sans-serif`;
  ctx.fillText('Lightning Voucher', Math.round(w * 0.02), Math.round(hdr * 0.70));

  const qrSize = Math.round(h * 0.52);
  const qrY = hdr + Math.round(h * 0.06);
  if (qrs.claim) await drawQROnCanvas(ctx, qrs.claim, Math.round(w * 0.03), qrY, qrSize);
  ctx.fillStyle = '#fff';
  ctx.font = `bold ${Math.round(h * 0.056)}px sans-serif`;
  ctx.fillText('SCAN TO CLAIM', Math.round(w * 0.03), qrY + qrSize + Math.round(h * 0.08));

  ctx.strokeStyle = '#333';
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.moveTo(w / 2, hdr);
  ctx.lineTo(w / 2, h - Math.round(h * 0.08));
  ctx.stroke();

  const rqrSize = Math.round(qrSize * 0.65);
  if (qrs.fund) await drawQROnCanvas(ctx, qrs.fund, Math.round(w * 0.53), qrY + Math.round(h * 0.04), rqrSize);
  ctx.fillStyle = '#F7931A';
  ctx.font = `bold ${Math.round(h * 0.046)}px sans-serif`;
  ctx.fillText('ADD FUNDS', Math.round(w * 0.53), qrY + rqrSize + Math.round(h * 0.12));
  ctx.fillStyle = '#aaa';
  ctx.font = `${Math.round(h * 0.038)}px sans-serif`;
  ctx.fillText(expiryAfterFundingLabel(refundAfterSeconds).slice(0, 24), Math.round(w * 0.53), qrY + rqrSize + Math.round(h * 0.20));

  ctx.fillStyle = '#F7931A';
  ctx.fillRect(0, h - Math.round(h * 0.08), w, Math.round(h * 0.08));
  ctx.fillStyle = '#000';
  ctx.font = `${Math.round(h * 0.04)}px sans-serif`;
  ctx.textAlign = 'center';
  ctx.fillText((window.SITE_NAME||'Satoshi Note'), w / 2, h - Math.round(h * 0.025));
  ctx.textAlign = 'left';
}

async function drawGiftCardPreview(ctx, w, h, qrs, batchName, refundAfterSeconds) {
  ctx.fillStyle = '#12080a';
  ctx.fillRect(0, 0, w, h);
  const stripe = Math.round(w * 0.07);
  ctx.fillStyle = '#F7931A';
  ctx.fillRect(0, 0, stripe, h);

  ctx.fillStyle = 'rgba(247,147,26,0.06)';
  ctx.font = `bold ${Math.round(h * 0.85)}px sans-serif`;
  ctx.textAlign = 'center';
  ctx.fillText('\u20BF', w * 0.62, h * 0.78);
  ctx.textAlign = 'left';

  const qrSize = Math.round(h * 0.55);
  const qrX = Math.round((w - qrSize) / 2 + stripe * 0.3);
  const qrY = Math.round(h * 0.10);
  if (qrs.claim) {
    ctx.fillStyle = '#fff';
    ctx.fillRect(qrX - 3, qrY - 3, qrSize + 6, qrSize + 6);
    await drawQROnCanvas(ctx, qrs.claim, qrX, qrY, qrSize);
  }
  ctx.fillStyle = '#F7931A';
  ctx.font = `bold ${Math.round(h * 0.062)}px sans-serif`;
  ctx.textAlign = 'center';
  ctx.fillText('SCAN TO CLAIM', w * 0.58, qrY + qrSize + Math.round(h * 0.09));
  ctx.fillStyle = '#c8b090';
  ctx.font = `${Math.round(h * 0.046)}px sans-serif`;
  ctx.fillText(expiryAfterFundingLabel(refundAfterSeconds), w * 0.58, qrY + qrSize + Math.round(h * 0.18));

  const fqrSize = Math.round(h * 0.22);
  if (qrs.fund) {
    ctx.fillStyle = '#fff';
    ctx.fillRect(w - fqrSize - Math.round(w * 0.04) - 2, h - fqrSize - Math.round(h * 0.06) - 2, fqrSize + 4, fqrSize + 4);
    await drawQROnCanvas(ctx, qrs.fund, w - fqrSize - Math.round(w * 0.04), h - fqrSize - Math.round(h * 0.06), fqrSize);
  }
  ctx.textAlign = 'left';
}

async function drawMinimalPreview(ctx, w, h, qrs, batchName, refundAfterSeconds) {
  ctx.fillStyle = '#fafafa';
  ctx.fillRect(0, 0, w, h);
  ctx.fillStyle = '#F7931A';
  ctx.fillRect(0, 0, w, Math.round(h * 0.018));
  ctx.fillRect(0, h - Math.round(h * 0.018), w, Math.round(h * 0.018));

  const name = batchName && !isAutoName(batchName) ? batchName : 'Lightning Voucher';
  ctx.fillStyle = '#111';
  ctx.font = `bold ${Math.round(h * 0.065)}px sans-serif`;
  ctx.textAlign = 'center';
  ctx.fillText(name.slice(0, 22), w / 2, Math.round(h * 0.12));
  ctx.textAlign = 'left';

  const qrSize = Math.round(h * 0.52);
  const leftX = Math.round(w / 2 - qrSize - w * 0.04);
  const rightX = Math.round(w / 2 + w * 0.04);
  const qrY = Math.round(h * 0.17);

  if (qrs.claim) await drawQROnCanvas(ctx, qrs.claim, leftX, qrY, qrSize);
  if (qrs.fund)  await drawQROnCanvas(ctx, qrs.fund,  rightX, qrY, qrSize);

  ctx.fillStyle = '#222';
  ctx.font = `bold ${Math.round(h * 0.050)}px sans-serif`;
  ctx.textAlign = 'center';
  ctx.fillText('CLAIM', leftX + qrSize / 2, qrY + qrSize + Math.round(h * 0.07));
  ctx.fillText('TOP UP', rightX + qrSize / 2, qrY + qrSize + Math.round(h * 0.07));
  ctx.fillStyle = '#777';
  ctx.font = `${Math.round(h * 0.040)}px sans-serif`;
  ctx.fillText(expiryAfterFundingLabel(refundAfterSeconds), w / 2, h - Math.round(h * 0.04));
  ctx.textAlign = 'left';
}

async function drawDarkModePreview(ctx, w, h, qrs, batchName, refundAfterSeconds) {
  ctx.fillStyle = '#0d0d0d';
  ctx.fillRect(0, 0, w, h);
  ctx.fillStyle = '#F7931A';
  ctx.fillRect(0, 0, w, Math.round(h * 0.015));
  ctx.fillRect(0, h - Math.round(h * 0.015), w, Math.round(h * 0.015));

  ctx.fillStyle = 'rgba(247,147,26,0.06)';
  ctx.font = `bold ${Math.round(h * 0.75)}px sans-serif`;
  ctx.textAlign = 'center';
  ctx.fillText('\u20BF', w * 0.73, h * 0.72);
  ctx.textAlign = 'left';

  ctx.fillStyle = '#F7931A';
  ctx.font = `bold ${Math.round(h * 0.072)}px sans-serif`;
  ctx.textAlign = 'center';
  ctx.fillText('Lightning Voucher', w / 2, Math.round(h * 0.12));
  ctx.textAlign = 'left';

  const qrSize = Math.round(h * 0.52);
  const qrX = Math.round(w * 0.03);
  const qrY = Math.round(h * 0.15);
  if (qrs.claim) {
    ctx.fillStyle = '#fff';
    ctx.fillRect(qrX - 2, qrY - 2, qrSize + 4, qrSize + 4);
    await drawQROnCanvas(ctx, qrs.claim, qrX, qrY, qrSize);
  }
  ctx.fillStyle = '#fff';
  ctx.font = `bold ${Math.round(h * 0.046)}px sans-serif`;
  ctx.fillText('SCAN TO CLAIM', qrX, qrY + qrSize + Math.round(h * 0.07));
  ctx.fillStyle = '#888';
  ctx.font = `${Math.round(h * 0.038)}px sans-serif`;
  ctx.fillText(expiryAfterFundingLabel(refundAfterSeconds), qrX, qrY + qrSize + Math.round(h * 0.14));

  const rqrSize = Math.round(qrSize * 0.55);
  if (qrs.fund) {
    ctx.fillStyle = '#fff';
    ctx.fillRect(Math.round(w * 0.44) - 2, qrY + Math.round(h * 0.04) - 2, rqrSize + 4, rqrSize + 4);
    await drawQROnCanvas(ctx, qrs.fund, Math.round(w * 0.44), qrY + Math.round(h * 0.04), rqrSize);
  }
  ctx.fillStyle = '#F7931A';
  ctx.font = `bold ${Math.round(h * 0.042)}px sans-serif`;
  ctx.fillText('ADD FUNDS', Math.round(w * 0.44), qrY + rqrSize + Math.round(h * 0.14));
}

async function drawFoldPreview(ctx, w, h, qrs, batchName, refundAfterSeconds) {
  // Two strips — top: voucher 1, bottom: voucher 2 (rotated)
  const mid = Math.round(h / 2);
  const foldX = Math.round(w / 2);
  const qrSize = Math.round(Math.min(w * 0.22, mid * 0.55));

  // Top strip — left: outside (fund), right: inside (claim)
  ctx.fillStyle = '#fffaf5';
  ctx.fillRect(0, 0, foldX, mid);
  ctx.fillStyle = '#F7931A';
  ctx.fillRect(0, 0, foldX, Math.round(mid * 0.14));
  ctx.fillStyle = '#000';
  ctx.font = `bold ${Math.round(mid * 0.09)}px sans-serif`;
  ctx.fillText('SATOSHI NOTE', Math.round(w * 0.02), Math.round(mid * 0.11));
  if (qrs.fund) await drawQROnCanvas(ctx, qrs.fund, Math.round(foldX / 2 - qrSize / 2), Math.round(mid * 0.20), qrSize);
  ctx.fillStyle = '#444';
  ctx.font = `${Math.round(mid * 0.075)}px sans-serif`;
  ctx.textAlign = 'center';
  ctx.fillText('Add funds', foldX / 2, Math.round(mid * 0.86));
  ctx.textAlign = 'left';

  ctx.fillStyle = '#111';
  ctx.fillRect(foldX, 0, w - foldX, mid);
  if (qrs.claim) await drawQROnCanvas(ctx, qrs.claim, Math.round(foldX + (w - foldX) / 2 - qrSize / 2), Math.round(mid * 0.20), qrSize);
  ctx.fillStyle = '#F7931A';
  ctx.font = `bold ${Math.round(mid * 0.075)}px sans-serif`;
  ctx.textAlign = 'center';
  ctx.fillText('CLAIM', foldX + (w - foldX) / 2, Math.round(mid * 0.88));
  ctx.textAlign = 'left';

  // Dashed cut line
  ctx.strokeStyle = '#999';
  ctx.setLineDash([4, 4]);
  ctx.lineWidth = 1;
  ctx.beginPath(); ctx.moveTo(0, mid); ctx.lineTo(w, mid); ctx.stroke();
  ctx.setLineDash([]);

  // Dashed fold line
  ctx.strokeStyle = '#aaa';
  ctx.setLineDash([3, 3]);
  ctx.beginPath(); ctx.moveTo(foldX, 0); ctx.lineTo(foldX, h); ctx.stroke();
  ctx.setLineDash([]);

  // Bottom strip — rotated representation (swap sides, darker)
  ctx.fillStyle = '#111';
  ctx.fillRect(0, mid, foldX, h - mid);
  if (qrs.claim) await drawQROnCanvas(ctx, qrs.claim, Math.round(foldX / 2 - qrSize / 2), mid + Math.round((h - mid) * 0.20), qrSize);

  ctx.fillStyle = '#fffaf5';
  ctx.fillRect(foldX, mid, w - foldX, h - mid);
  ctx.fillStyle = '#F7931A';
  ctx.fillRect(foldX, mid, w - foldX, Math.round((h - mid) * 0.14));
  ctx.fillStyle = '#000';
  ctx.font = `${Math.round((h - mid) * 0.075)}px sans-serif`;
  ctx.textAlign = 'center';
  ctx.fillText('Add funds', foldX + (w - foldX) / 2, mid + Math.round((h - mid) * 0.86));
  ctx.textAlign = 'left';
}

async function drawBizCardPreview(ctx, w, h, qrs, batchName, refundAfterSeconds) {
  ctx.fillStyle = '#f5f5f5';
  ctx.fillRect(0, 0, w, h);

  const cards = 5, gap = Math.round(h * 0.015);
  const totalGap = (cards - 1) * gap;
  const cardH = Math.round((h - totalGap) / cards);
  const foldX = Math.round(w / 2);
  const qrSize = Math.round(cardH * 0.60);

  for (let i = 0; i < cards; i++) {
    const y = i * (cardH + gap);

    // Outside (left)
    ctx.fillStyle = '#fff';
    ctx.fillRect(0, y, foldX, cardH);
    ctx.fillStyle = '#F7931A';
    ctx.fillRect(0, y, foldX, Math.round(cardH * 0.20));
    ctx.fillStyle = '#000';
    ctx.font = `bold ${Math.round(cardH * 0.14)}px sans-serif`;
    ctx.fillText('Satoshi', Math.round(w * 0.02), y + Math.round(cardH * 0.16));
    if (qrs.fund) await drawQROnCanvas(ctx, qrs.fund, Math.round(w * 0.02), y + Math.round(cardH * 0.24), qrSize);

    // Inside (right, dark)
    ctx.fillStyle = '#111';
    ctx.fillRect(foldX, y, w - foldX, cardH);
    if (qrs.claim) await drawQROnCanvas(ctx, qrs.claim, Math.round(foldX + (w - foldX) / 2 - qrSize / 2), y + Math.round(cardH * 0.20), qrSize);
    ctx.fillStyle = '#F7931A';
    ctx.font = `bold ${Math.round(cardH * 0.13)}px sans-serif`;
    ctx.textAlign = 'right';
    ctx.fillText('CLAIM', w - Math.round(w * 0.03), y + Math.round(cardH * 0.88));
    ctx.textAlign = 'left';

    // Fold line
    ctx.strokeStyle = '#aaa';
    ctx.setLineDash([2, 2]);
    ctx.lineWidth = 0.8;
    ctx.beginPath(); ctx.moveTo(foldX, y); ctx.lineTo(foldX, y + cardH); ctx.stroke();
    ctx.setLineDash([]);

    // Card border
    ctx.strokeStyle = '#ddd';
    ctx.lineWidth = 0.5;
    ctx.strokeRect(0, y, w, cardH);
  }
}

// ── History screen ────────────────────────────────────────────────────────────
function isAutoName(name) {
  return /^batch-\d+$/.test(name) || /^single-\d+$/.test(name);
}

// In-memory cache for the current session. Pre-populated from localStorage for terminal states.
const historyStatusCache = new Map();

function isTerminalStatus(s) {
  if (!s) return false;
  if (s.refunded === true) return true;
  // active=false, not refunded, no pending refund → redeemed
  if (s.active === false && s.refunded === false && s.refund_pending === false) return true;
  return false;
}

// Pre-populates historyStatusCache from vouchers that already have a _cachedStatus in localStorage.
function preloadTerminalStatuses(history) {
  history.forEach(entry => {
    entry.vouchers.forEach(v => {
      if (v._cachedStatus && !historyStatusCache.has(v.pubkey)) {
        historyStatusCache.set(v.pubkey, v._cachedStatus);
      }
    });
  });
}

// Fetches statuses for the given pubkeys via the batch endpoint, updates the in-memory cache,
// and persists any newly-terminal statuses back to localStorage.
async function fetchAndCacheStatuses(pubkeys, history) {
  try {
    const res = await fetch('/voucher/status/batch', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pubkeys }),
    });
    if (!res.ok) throw new Error();
    const data = await res.json();
    for (const pk of pubkeys) {
      historyStatusCache.set(pk, data[pk] ?? null);
    }
  } catch {
    for (const pk of pubkeys) {
      if (!historyStatusCache.has(pk)) historyStatusCache.set(pk, null);
    }
  }

  // Persist newly-terminal statuses to localStorage so they're never fetched again.
  let changed = false;
  history.forEach(entry => {
    entry.vouchers.forEach(v => {
      if (!v._cachedStatus) {
        const s = historyStatusCache.get(v.pubkey);
        if (isTerminalStatus(s)) {
          v._cachedStatus = s;
          changed = true;
        }
      }
    });
  });
  if (changed) saveHistory(history);
}

function classifyVoucher(s, now) {
  if (!s) return 'unfunded';
  if (s.active && s.expires_at === 0) return 'unfunded';
  if (s.active) {
    if (s.expires_at > 0 && now > s.expires_at) return 'expired';
    return 'funded';
  }
  if (s.refunded || s.refund_pending) return 'expired';
  return 'redeemed';
}

function getExpiredSubTag(expiredStatuses) {
  if (expiredStatuses.some(s => s && s.refund_pending)) return 'pending';
  if (expiredStatuses.some(s => s && !s.refunded)) return 'processing';
  return 'refunded';
}

function buildSectionCards(history) {
  const now = Math.floor(Date.now() / 1000);
  const sections = { unfunded: [], funded: [], redeemed: [], expired: [] };

  history.forEach(entry => {
    const total = entry.vouchers.length;
    const byState = { unfunded: [], funded: [], redeemed: [], expired: [] };

    entry.vouchers.forEach(v => {
      const s = historyStatusCache.get(v.pubkey);
      byState[classifyVoucher(s, now)].push(v);
    });

    ['unfunded', 'funded', 'redeemed', 'expired'].forEach(state => {
      const inState = byState[state];
      if (!inState.length) return;
      const expiredTag = state === 'expired'
        ? getExpiredSubTag(inState.map(v => historyStatusCache.get(v.pubkey)))
        : null;
      sections[state].push({ entry, count: inState.length, total, expiredTag });
    });
  });

  return sections;
}

function renderSectionBody(body, cards, showQR, history) {
  body.innerHTML = '';
  if (!cards.length) {
    body.innerHTML = `<p style="font-size:0.8rem;color:var(--text-muted);padding:8px 0">Nothing here.</p>`;
    return;
  }

  cards.forEach(({ entry, count, total, expiredTag }) => {
    const card = document.createElement('div');
    card.className = 'history-card';

    const date = new Date(entry.createdAt * 1000).toLocaleString();
    let typeLabel, typeBadge;
    if (entry.type === 'single') {
      typeLabel = 'Single';
      typeBadge = 'badge-single';
    } else {
      typeLabel = total > 1 ? `Batch (${count}/${total})` : 'Batch';
      typeBadge = 'badge-batch';
    }

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

    const subTagHTML = expiredTag
      ? ` <span class="badge badge-${expiredTag}">${expiredTag}</span>`
      : '';

    const actionsHTML = showQR
      ? `<div class="history-card-actions">
           <button class="btn btn-secondary btn-sm" data-action="reqr" data-id="${entry.id}">Re-show QR</button>
         </div>`
      : '';

    card.innerHTML = `
      <div class="history-card-header">
        <span class="badge ${typeBadge}">${typeLabel}</span>${subTagHTML}
      </div>
      <div class="history-card-meta">${metaLine}</div>
      ${actionsHTML}`;

    body.appendChild(card);
  });

  body.querySelectorAll('[data-action="reqr"]').forEach(btn => {
    btn.addEventListener('click', () => {
      const entry = history.find(e => e.id === btn.dataset.id);
      if (entry) openQRModal(entry);
    });
  });
}

async function expandSection(sectionEl, key, showQR, history) {
  const body = sectionEl.querySelector('.history-section-body');
  const isOpen = sectionEl.classList.contains('expanded');

  // Collapse
  if (isOpen) {
    sectionEl.classList.remove('expanded');
    body.style.display = 'none';
    return;
  }

  // Expand: load statuses — terminal states come from localStorage, rest are batch-fetched.
  sectionEl.classList.add('expanded');
  body.style.display = '';

  preloadTerminalStatuses(history);
  const missing = history.flatMap(e => e.vouchers.map(v => v.pubkey))
    .filter(pk => !historyStatusCache.has(pk));

  if (missing.length > 0) {
    body.innerHTML = `<p style="font-size:0.8rem;color:var(--text-muted);padding:8px 0">Loading…</p>`;
    await fetchAndCacheStatuses(missing, history);
  }

  updateSectionCounts(history, $('history-list'));
  const sections = buildSectionCards(history);
  renderSectionBody(body, sections[key], showQR, history);
}

function countByState(history) {
  const now = Math.floor(Date.now() / 1000);
  const counts = { unfunded: 0, funded: 0, redeemed: 0, expired: 0 };
  history.forEach(entry =>
    entry.vouchers.forEach(v => counts[classifyVoucher(historyStatusCache.get(v.pubkey), now)]++)
  );
  return counts;
}

function updateSectionCounts(history, container) {
  const counts = countByState(history);
  container.querySelectorAll('.history-section').forEach(sec => {
    const key = sec.dataset.sectionKey;
    const badge = sec.querySelector('.section-count');
    if (badge && counts[key] !== undefined) {
      badge.textContent = counts[key] > 0 ? counts[key] : '';
    }
  });
}

function computeLeaderboardCounts(history, cache) {
  const now = Math.floor(Date.now() / 1000);
  const d = new Date();
  const monthStart = Math.floor(new Date(Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), 1)).getTime() / 1000);

  let fundedMonth = 0, fundedAllTime = 0, redeemedMonth = 0, redeemedAllTime = 0;

  history.forEach(entry => {
    entry.vouchers.forEach(v => {
      const s = cache.get(v.pubkey);
      const state = classifyVoucher(s, now);
      const isFunded = state === 'funded' || state === 'redeemed' || state === 'expired';
      const isRedeemed = state === 'redeemed';

      if (isFunded) {
        fundedAllTime++;
        if (entry.createdAt >= monthStart) fundedMonth++;
      }
      if (isRedeemed) {
        redeemedAllTime++;
        if (entry.createdAt >= monthStart) redeemedMonth++;
      }
    });
  });

  return { funded_month: fundedMonth, funded_all_time: fundedAllTime, redeemed_month: redeemedMonth, redeemed_all_time: redeemedAllTime };
}

function renderLeaderboardContent(container, data, counts) {
  const monthLabel = new Date().toLocaleString('default', { month: 'long', year: 'numeric' });

  function medalFor(rank, total) {
    if (total === 0) return '–';
    if (rank === 1) return '🥇';
    if (rank === 2) return '🥈';
    if (rank === 3) return '🥉';
    if (total >= 10 && rank <= Math.ceil(total * 0.1)) return '🔥';
    return '⚡';
  }

  function ordinal(n) {
    const s = ['th', 'st', 'nd', 'rd'];
    const v = n % 100;
    return n + (s[(v - 20) % 10] || s[v] || s[0]);
  }

  function highScoresTable(scores, counts) {
    if (!scores) return '';
    const cols = [
      { key: 'funded_month',      label: 'Sats Shared',       scope: 'This Month', userCount: counts.funded_month },
      { key: 'redeemed_month',    label: 'Bitcoiners Minted', scope: 'This Month', userCount: counts.redeemed_month },
      { key: 'funded_all_time',   label: 'Sats Shared',       scope: 'All Time',   userCount: counts.funded_all_time },
      { key: 'redeemed_all_time', label: 'Bitcoiners Minted', scope: 'All Time',   userCount: counts.redeemed_all_time },
    ];
    const medals = [
      { cls: 'hs-gold',   icon: '🥇', status: 'LEGENDARY'  },
      { cls: 'hs-silver', icon: '🥈', status: 'ELITE'       },
      { cls: 'hs-bronze', icon: '🥉', status: 'RISING STAR' },
    ];
    const rows = medals.map((m, i) => `
      <tr class="hs-row ${m.cls}">
        <td class="hs-medal-cell">
          <span class="hs-medal-badge">${m.icon}</span>
          <span class="hs-status ${m.cls}">${m.status}</span>
        </td>
        ${cols.map(c => {
          const val = scores[c.key] != null && scores[c.key][i] != null ? scores[c.key][i] : null;
          const isYou = val != null && c.userCount > 0 && c.userCount === val;
          return `<td class="hs-score-cell ${m.cls}${isYou ? ' hs-you' : ''}">
            ${val != null ? val : '<span class="hs-empty">–</span>'}
            ${isYou ? '<span class="hs-you-badge">YOU</span>' : ''}
          </td>`;
        }).join('')}
      </tr>`).join('');

    return `
      <div class="hs-section">
        <h2 class="hs-title">⚡ HIGH SCORES ⚡</h2>
        <div class="hs-table-wrap">
          <table class="hs-table">
            <thead>
              <tr>
                <th class="hs-rank-head"></th>
                ${cols.map(c => `<th class="hs-col-head">${c.label}<br><span class="hs-scope">${c.scope}</span></th>`).join('')}
              </tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
        </div>
      </div>`;
  }

  function redeemedTeaserCard(label, isMonth) {
    const tips = isMonth
      ? [
          'Show someone the QR code and say "scan this for free bitcoin." Watch their face. That first reaction? That\'s the orange pill. ⚡',
          'The trick? Tell them it\'s real sats they can actually keep or spend. Curiosity does the rest. Try it today.',
          'Birthday card, coffee bet, random act of orange-pilling — the best redemptions come from the least expected moments. 🌱',
        ]
      : [
          'No redemptions yet — but one changed mind is worth a thousand funded vouchers. Keep planting seeds. 🌱',
        ];
    const tip = tips[Math.floor(Date.now() / 60000) % tips.length];
    return `
      <div class="lb-card lb-card-teaser">
        <div class="lb-card-medal">🌱</div>
        <div class="lb-card-label">${label}</div>
        <div class="lb-card-rank">–</div>
        <div class="lb-card-of">${isMonth ? 'no redemptions yet' : 'none yet'}</div>
        <div class="lb-card-desc">${tip}</div>
      </div>`;
  }

  function cardHTML(key, label, count) {
    const isRedeemed = key === 'redeemed_month' || key === 'redeemed_all_time';
    if (count === 0 && isRedeemed) return redeemedTeaserCard(label, key === 'redeemed_month');

    const { rank, total } = data[key];
    // If count is 0 for a funded card (e.g. no funding this month) show no rank
    if (count === 0) return `
      <div class="lb-card">
        <div class="lb-card-medal">–</div>
        <div class="lb-card-label">${label}</div>
        <div class="lb-card-rank">–</div>
        <div class="lb-card-of">no activity this month</div>
      </div>`;

    const v = count !== 1 ? 's' : '';
    const descriptions = {
      funded_month:      `This month you've funded ${count} voucher${v} — putting bitcoin directly into ${count} new set${v} of hands. That's the ${ordinal(rank)} highest score this month. Keep going, the revolution needs you.`,
      redeemed_month:    `${count} person${v} claimed their sats this month thanks to you — ${count} new bitcoiner${v} minted. You rank ${ordinal(rank)} this month. Every redemption is a mind opened.`,
      funded_all_time:   `You've funded ${count} voucher${v} in total, spreading the orange pill one sat at a time. All-time rank: ${ordinal(rank)} of ${total}. Legendary.`,
      redeemed_all_time: `${count} bitcoiner${v} minted by you, for life. You're ${ordinal(rank)} of all time. That's a legacy worth building.`,
    };
    return `
      <div class="lb-card">
        <div class="lb-card-medal">${medalFor(rank, total)}</div>
        <div class="lb-card-label">${label}</div>
        <div class="lb-card-rank">#${rank}</div>
        <div class="lb-card-of">of ${total}</div>
        <div class="lb-card-desc">${descriptions[key]}</div>
      </div>`;
  }

  const heroRankLine = counts.redeemed_month > 0
    ? `<p class="lb-subtitle">You're #${data.redeemed_month.rank} for Bitcoiners Minted this month — keep going! ⚡</p>`
    : counts.redeemed_all_time > 0
      ? `<p class="lb-subtitle">You're #${data.redeemed_all_time.rank} all time for Bitcoiners Minted. ⚡</p>`
      : `<p class="lb-subtitle">Your vouchers are out there — show someone the QR code and say "it's free bitcoin." That first claim changes everything. ⚡</p>`;

  container.innerHTML = `
    <div class="lb-hero">
      <div class="lb-trophy">🏆</div>
      <h1 class="lb-title">LEADERBOARD</h1>
      ${heroRankLine}
    </div>

    <div class="lb-category">
      <div class="lb-category-header">
        <span class="lb-category-cup">🏆</span>
        <span class="lb-category-title">${monthLabel}</span>
      </div>
      <div class="lb-grid">
        ${cardHTML('funded_month', 'Sats Shared', counts.funded_month)}
        ${cardHTML('redeemed_month', 'Bitcoiners Minted', counts.redeemed_month)}
      </div>
    </div>

    <div class="lb-category">
      <div class="lb-category-header">
        <span class="lb-category-cup">🏆</span>
        <span class="lb-category-title">Hall of Legends</span>
      </div>
      <div class="lb-grid">
        ${cardHTML('funded_all_time', 'Sats Shared', counts.funded_all_time)}
        ${cardHTML('redeemed_all_time', 'Bitcoiners Minted', counts.redeemed_all_time)}
      </div>
    </div>

    <p class="lb-motivate">Every redemption mints a new bitcoiner. Be the one who started it all. ⚡</p>
    ${highScoresTable(data.top_scores, counts)}`;
}

async function renderLeaderboardScreen(container) {
  container.innerHTML = `<div class="lb-loading">🏆 Loading your rank…</div>`;

  const history = getHistory();

  if (!history.length) {
    container.innerHTML = `<div class="lb-empty"><div style="font-size:2.5rem;margin-bottom:12px;">🏆</div><p>No vouchers yet.<br>Create and fund your first voucher<br>to appear on the leaderboard.</p></div>`;
    return;
  }

  preloadTerminalStatuses(history);
  const missing = history.flatMap(e => e.vouchers.map(v => v.pubkey))
    .filter(pk => !historyStatusCache.has(pk));
  if (missing.length > 0) await fetchAndCacheStatuses(missing, history);

  const counts = computeLeaderboardCounts(history, historyStatusCache);

  if (counts.funded_all_time === 0) {
    container.innerHTML = `<div class="lb-empty"><div style="font-size:2.5rem;margin-bottom:12px;">🏆</div><p>No funded vouchers yet.<br>Fund your first voucher<br>to appear on the leaderboard.</p></div>`;
    return;
  }
  try {
    const res = await fetch('/leaderboard', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(counts),
    });
    if (!res.ok) throw new Error();
    renderLeaderboardContent(container, await res.json(), counts);
  } catch {
    container.innerHTML = `<p class="lb-error">Could not load leaderboard. Try again later.</p>`;
  }
}

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

  const sectionDefs = [
    { key: 'unfunded', label: 'Active — Unfunded', showQR: true },
    { key: 'funded',   label: 'Active — Funded',   showQR: true },
    { key: 'redeemed', label: 'Redeemed',           showQR: false },
    { key: 'expired',  label: 'Expired',            showQR: false },
  ];

  sectionDefs.forEach(({ key, label, showQR }) => {
    const section = document.createElement('div');
    section.className = 'history-section';
    section.dataset.sectionKey = key;
    section.innerHTML = `
      <div class="history-section-title" role="button">
        <span class="section-title-label">${label}</span>
        <span class="section-count"></span>
        <span class="section-chevron">›</span>
      </div>
      <div class="history-section-body" style="display:none"></div>`;

    section.querySelector('.history-section-title').addEventListener('click', () => {
      expandSection(section, key, showQR, history);
    });

    container.appendChild(section);
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
function decodeLNURL(str) {
  const s = str.toLowerCase();
  const sep = s.lastIndexOf('1');
  if (sep < 1) throw new Error('invalid bech32');
  const CHARSET = 'qpzry9x8gf2tvdw0s3jn54khce6mua7l';
  const data5 = [];
  for (let i = sep + 1; i < s.length - 6; i++) {
    const idx = CHARSET.indexOf(s[i]);
    if (idx < 0) throw new Error('invalid bech32 char');
    data5.push(idx);
  }
  let acc = 0, bits = 0;
  const bytes = [];
  for (const v of data5) {
    acc = (acc << 5) | v;
    bits += 5;
    if (bits >= 8) { bits -= 8; bytes.push((acc >> bits) & 0xff); }
  }
  return new TextDecoder().decode(new Uint8Array(bytes));
}

async function verifyRefundCode(val) {
  // Returns { ok: true } | { ok: false, error: string } | { ok: null, warn: string }
  // ok=null means unverifiable (network/CORS) — caller shows a warning but may proceed

  if (val.includes('@')) {
    const [user, domain] = val.split('@');
    try {
      const resp = await fetch(`https://${domain}/.well-known/lnurlp/${encodeURIComponent(user)}`);
      if (!resp.ok) return { ok: false, error: 'Lightning address not found (HTTP ' + resp.status + ')' };
      const data = await resp.json();
      if (data.tag !== 'payRequest') return { ok: false, error: 'Not a valid Lightning address endpoint' };
      return { ok: true };
    } catch (e) {
      return { ok: null, warn: 'Could not reach this Lightning address to verify. Double-check it is correct.' };
    }
  }

  // LNURL bech32
  try {
    const url = decodeLNURL(val);
    try {
      const resp = await fetch(url);
      if (!resp.ok) return { ok: false, error: 'LNURL endpoint returned error (HTTP ' + resp.status + ')' };
      const data = await resp.json();
      if (data.tag !== 'payRequest') return { ok: false, error: 'LNURL is not a pay endpoint' };
      return { ok: true };
    } catch (e) {
      return { ok: null, warn: 'Could not reach the LNURL endpoint to verify. Double-check it is correct.' };
    }
  } catch (e) {
    return { ok: false, error: 'Could not decode LNURL: ' + e.message };
  }
}

function validateRefundCode(val) {
  if (/^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(val)) return true;
  if (/^lnurl1[ac-hj-np-z02-9]{20,}$/i.test(val)) return true;
  return false;
}

async function handleOnboardingSubmit() {
  const val = $('refund-code-input').value.trim();
  const errEl = $('onboarding-error');
  const btn = $('btn-onboarding-submit');

  // Second tap after unverifiable warning — proceed anyway
  if (btn._warnDismissed === val) {
    btn._warnDismissed = null;
    errEl.classList.remove('visible');
    localStorage.setItem(LS_REFUND, val);
    startApp();
    return;
  }

  if (!validateRefundCode(val)) {
    errEl.textContent = 'Enter a Lightning address (user@wallet.com) or LNURL1… string.';
    errEl.classList.add('visible');
    return;
  }
  errEl.classList.remove('visible');

  const origText = btn.textContent;
  btn.textContent = 'Verifying…';
  btn.disabled = true;

  let result;
  try {
    result = await verifyRefundCode(val);
  } finally {
    btn.textContent = origText;
    btn.disabled = false;
  }

  if (result.ok === false) {
    errEl.textContent = result.error;
    errEl.classList.add('visible');
    return;
  }
  if (result.ok === null) {
    errEl.textContent = result.warn + ' Tap "Get Started" again to proceed anyway.';
    errEl.classList.add('visible');
    btn._warnDismissed = val;
    return;
  }

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
  $('nav-history').addEventListener('click', async () => {
    historyStatusCache.clear(); // fresh status on each visit
    renderHistory();
    showScreen('screen-history');
    const history = getHistory();
    if (!history.length) return;
    preloadTerminalStatuses(history);
    const missing = history.flatMap(e => e.vouchers.map(v => v.pubkey))
      .filter(pk => !historyStatusCache.has(pk));
    if (missing.length) await fetchAndCacheStatuses(missing, history);
    updateSectionCounts(history, $('history-list'));
  });

  $('nav-back-from-history').addEventListener('click', () => {
    if (localStorage.getItem(LS_REFUND)) {
      showScreen('screen-app');
    } else {
      showScreen('screen-onboarding');
    }
  });

  // Nav: leaderboard
  $('nav-leaderboard').addEventListener('click', () => {
    showScreen('screen-leaderboard');
    renderLeaderboardScreen($('leaderboard-content'));
  });

  $('nav-back-from-leaderboard').addEventListener('click', () => {
    showScreen(localStorage.getItem(LS_REFUND) ? 'screen-app' : 'screen-onboarding');
  });

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

  // Template preview modal close
  $('template-preview-close').addEventListener('click', () => $('template-preview-modal').classList.add('hidden'));
  $('template-preview-modal').addEventListener('click', e => {
    if (e.target === $('template-preview-modal')) $('template-preview-modal').classList.add('hidden');
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
