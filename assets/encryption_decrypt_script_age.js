(function() {
  const form = document.getElementById('{{.FORM_ID}}');
  const passwordInput = document.getElementById('{{.PASSWORD_INPUT_ID}}');
  const messageDiv = document.getElementById('{{.CONTENT_ID}}');
  const noOp = () => {};
  const noOpStorage = { getItem: () => null, setItem: noOp, removeItem: noOp, clear: noOp, length: 0 };
  const storage = {{.STORAGE_MODE}};
  const passphraseStorageKey = 'age_passphrase_{{.ENCRYPTED_DATA_PREFIX}}';

  const toBytes = (b64) => Uint8Array.from(atob(b64), c => c.charCodeAt(0));
  const encryptedBytes = toBytes('{{.ENCRYPTED_DATA}}');

  async function decryptWithPassphrase(passphrase) {
    const age = window.ageEncryption;
    if (!age) throw new Error('age library not loaded');
    const decrypter = new age.Decrypter();
    decrypter.addPassphrase(passphrase);
    return await decrypter.decrypt(encryptedBytes, 'text');
  }

  function showDecrypted(html) {
    if (passwordInput) passwordInput.value = '';
    if (form) form.reset();

    document.open();
    document.write(html);
    document.close();
  }

  // Try to auto-decrypt from cached passphrase on load
  window.addEventListener('load', async function() {
    let cached = null;
    try {
      cached = storage.getItem(passphraseStorageKey);
    } catch (e) {}
    if (!cached) return;

    try {
      const html = await decryptWithPassphrase(cached);
      showDecrypted(html);
    } catch (e) {
      try { storage.removeItem(passphraseStorageKey); } catch (err) {}
    }
  });

  if (form) {
    form.addEventListener('submit', async function(e) {
      e.preventDefault();
      const passphrase = passwordInput ? passwordInput.value : '';

      try {
        const html = await decryptWithPassphrase(passphrase);
        try {
          storage.setItem(passphraseStorageKey, passphrase);
        } catch (err) {}
        showDecrypted(html);
      } catch (err) {
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
