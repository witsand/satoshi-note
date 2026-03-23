/* ── Wallet Picker ───────────────────────────────────────────────────────────
 * Shared by index.html (fund flow) and redeem.html (claim flow).
 * Only shown on iOS — Android's lightning: scheme already triggers the
 * system app picker. Stores preference in localStorage (sn_wallet).
 */

const isIOS = /iPhone|iPad|iPod/i.test(navigator.userAgent);

const LS_WALLET = 'sn_wallet';

const WALLETS = [
  { id: 'blink',   name: 'Blink',             recommended: true },
  { id: 'phoenix', name: 'Phoenix' },
  { id: 'breez',   name: 'Misty Breez' },
  { id: 'fedi',    name: 'Fedi' },
  { id: 'wos',     name: 'Wallet of Satoshi' },
  { id: 'zeus',    name: 'Zeus' },
  { id: 'blitz',   name: 'Blitz Wallet' },
  { id: 'other',   name: 'Other wallet' },
];

function getPreferredWallet() {
  const id = localStorage.getItem(LS_WALLET);
  return id ? (WALLETS.find(w => w.id === id) || null) : null;
}

function setPreferredWallet(id) {
  localStorage.setItem(LS_WALLET, id);
}

// Lazily create the bottom-sheet overlay once, reuse it.
let _pickerEl = null;
function _getOrCreatePicker() {
  if (_pickerEl) return _pickerEl;

  const overlay = document.createElement('div');
  overlay.className = 'wallet-sheet-overlay hidden';
  overlay.innerHTML = `
    <div class="wallet-sheet">
      <p class="wallet-sheet-title">Which wallet do you use?</p>
      <div class="wallet-options"></div>
      <button class="wallet-sheet-cancel">Cancel</button>
    </div>`;

  overlay.querySelector('.wallet-sheet-cancel').addEventListener('click', () => {
    overlay.classList.add('hidden');
  });
  overlay.addEventListener('click', e => {
    if (e.target === overlay) overlay.classList.add('hidden');
  });

  document.body.appendChild(overlay);
  _pickerEl = overlay;
  return overlay;
}

function showWalletPicker(onSelect) {
  const overlay = _getOrCreatePicker();
  const optionsEl = overlay.querySelector('.wallet-options');
  optionsEl.innerHTML = '';

  WALLETS.forEach(w => {
    const btn = document.createElement('button');
    btn.className = 'wallet-option';
    btn.innerHTML = w.recommended
      ? `${w.name} <span style="background:var(--orange);color:#000;font-size:0.68rem;font-weight:700;padding:2px 8px;border-radius:10px;">Recommended</span>`
      : w.name;
    btn.addEventListener('click', () => {
      overlay.classList.add('hidden');
      onSelect(w);
    });
    optionsEl.appendChild(btn);
  });

  overlay.classList.remove('hidden');
}

/**
 * Attach a wallet-aware open button + "change wallet" link beneath a QR element.
 * Only inserts UI on iOS. On other platforms it is a no-op (QR onclick handles opening).
 * Safe to call multiple times on the same anchor — cleans up previous insertion first.
 *
 * @param {HTMLElement} anchorEl  - element to insert after (the copy-code <p>)
 * @param {string}      lnurl     - the LNURL string to open
 */
function attachWalletButton(anchorEl, lnurl) {
  // Remove any previously inserted wallet controls on this anchor
  let next = anchorEl.nextElementSibling;
  while (next && (next.classList.contains('wallet-open-btn') || next.classList.contains('wallet-change-link'))) {
    const toRemove = next;
    next = next.nextElementSibling;
    toRemove.remove();
  }

  // Wallet picker UI is only needed on iOS
  if (!isIOS) return;

  const pref = getPreferredWallet();

  const walletBtn = document.createElement('button');
  walletBtn.className = 'btn btn-secondary btn-sm wallet-open-btn';
  walletBtn.style.marginTop = '6px';
  walletBtn.textContent = pref ? `Open in ${pref.name}` : 'Open in wallet';

  const changeLink = document.createElement('a');
  changeLink.href = '#';
  changeLink.className = 'wallet-change-link';
  changeLink.textContent = pref ? 'change wallet' : '';

  walletBtn.addEventListener('click', () => {
    const current = getPreferredWallet();
    if (current) {
      window.location.href = 'lightning:' + lnurl;
    } else {
      showWalletPicker(w => {
        setPreferredWallet(w.id);
        walletBtn.textContent = `Open in ${w.name}`;
        changeLink.textContent = 'change wallet';
        window.location.href = 'lightning:' + lnurl;
      });
    }
  });

  changeLink.addEventListener('click', e => {
    e.preventDefault();
    showWalletPicker(w => {
      setPreferredWallet(w.id);
      walletBtn.textContent = `Open in ${w.name}`;
      changeLink.textContent = 'change wallet';
    });
  });

  anchorEl.insertAdjacentElement('afterend', changeLink);
  anchorEl.insertAdjacentElement('afterend', walletBtn);
}
