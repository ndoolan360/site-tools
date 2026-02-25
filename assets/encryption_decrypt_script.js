(function() {
  const derivedKeyStorageKey = 'derived_key_{{.SALT}}_{{.ITERATIONS}}';
  const form = document.getElementById('{{.FORM_ID}}');
  const passwordInput = document.getElementById('{{.PASSWORD_INPUT_ID}}');
  const messageDiv = document.getElementById('{{.CONTENT_ID}}');
  const noOp = () => {};
  const noOpStorage = { getItem: () => null, setItem: noOp, removeItem: noOp, clear: noOp, length: 0 };
  const storage = {{.STORAGE_MODE}};

  const toBytes = (b64) => Uint8Array.from(atob(b64), c => c.charCodeAt(0));
  const toBase64 = (bytes) => btoa(String.fromCharCode(...bytes));

  async function deriveKeyBytes(password) {
    const passwordKey = await crypto.subtle.importKey(
      'raw',
      new TextEncoder().encode(password),
      'PBKDF2',
      false,
      ['deriveBits']
    );

    const keyBits = await crypto.subtle.deriveBits(
      { name: 'PBKDF2', salt: toBytes('{{.SALT}}'), iterations: {{.ITERATIONS}}, hash: 'SHA-256' },
      passwordKey,
      256
    );

    return new Uint8Array(keyBits);
  }

  async function importAesKey(rawKey) {
    return crypto.subtle.importKey('raw', rawKey, { name: 'AES-GCM' }, false, ['decrypt']);
  }

  async function decryptWithKey(key) {
    const encrypted = toBytes('{{.ENCRYPTED_DATA}}');
    const nonce = encrypted.slice(0, 12);
    const data = encrypted.slice(12);
    const decrypted = await crypto.subtle.decrypt({ name: 'AES-GCM', iv: nonce }, key, data);

    return new TextDecoder().decode(decrypted);
  }

  function getCachedKeyBytes() {
    try {
      const cached = storage.getItem(derivedKeyStorageKey);
      if (!cached) return null;
      return toBytes(cached);
    } catch (e) {
      storage.removeItem(derivedKeyStorageKey);
      return null;
    }
  }

  // Replace page content
  function showDecrypted(html) {
    if (passwordInput) passwordInput.value = '';
    if (form) form.reset();

    document.open();
    document.write(html);
    document.close();
  }

  // Check browser compatibility
  if (!window.crypto || !window.crypto.subtle) {
    if (messageDiv) {
      messageDiv.textContent = 'Your browser does not support the Web Crypto API. Please use a modern browser.';
      messageDiv.style.color = 'red';
    }
    return;
  }

  // Try to auto-decrypt from cached key on load
  window.addEventListener('load', async function() {
    const cachedKeyBytes = getCachedKeyBytes();
    if (!cachedKeyBytes) return;

    try {
      const key = await importAesKey(cachedKeyBytes);
      const html = await decryptWithKey(key);
      showDecrypted(html);
    } catch (e) {
      storage.removeItem(derivedKeyStorageKey);
    }
  });

  // Handle form submission
  if (form) {
    form.addEventListener('submit', async function(e) {
      e.preventDefault();

      try {
        const rawKey = await deriveKeyBytes(passwordInput.value);
        const key = await importAesKey(rawKey);
        const html = await decryptWithKey(key);

        try {
          storage.setItem(derivedKeyStorageKey, toBase64(rawKey));
        } catch (e) {}
        showDecrypted(html);
      } catch (e) {
        if (messageDiv) {
          messageDiv.textContent = 'Incorrect password. Please try again.';
          messageDiv.style.color = 'red';
        }
        if (passwordInput) {
          passwordInput.value = '';
        }
      }
    });
  }
})();
