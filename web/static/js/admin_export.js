/* Anemone - Admin backup export page (legacy) */
var pageData = JSON.parse(document.getElementById('page-data').textContent || '{}');
var t = pageData.translations || {};

document.getElementById('exportForm').addEventListener('submit', function(e) {
    var passphrase = document.getElementById('passphrase').value;
    var passphraseConfirm = document.getElementById('passphrase_confirm').value;
    var confirmCheckbox = document.getElementById('confirm_backup').checked;

    if (passphrase !== passphraseConfirm) {
        e.preventDefault();
        alert(t.mismatch);
        return;
    }
    if (passphrase.length < 12) {
        e.preventDefault();
        alert(t.tooShort);
        return;
    }
    if (!confirmCheckbox) {
        e.preventDefault();
        alert(t.confirm);
        return;
    }
    if (!window.confirm(t.ready)) {
        e.preventDefault();
    }
});
