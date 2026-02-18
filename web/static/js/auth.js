/* Anemone auth pages JS - login, activate, setup, reset-password */

var pageData = JSON.parse(document.getElementById('page-data')?.textContent || '{}');
var t = pageData.translations || {};

/* Language selector redirect */
function changeLanguage() {
    var lang = document.getElementById('language-selector').value;
    var url = new URL(window.location.href);
    url.searchParams.set('lang', lang);
    window.location.href = url.toString();
}

/* Password form validation (activate + setup) */
function validatePasswordForm(formId) {
    var form = document.getElementById(formId);
    if (!form) return;
    form.addEventListener('submit', function(e) {
        var password = document.getElementById('password').value;
        var confirm = document.getElementById('password_confirm') || document.getElementById('confirm_password');
        if (!confirm) return;
        if (password !== confirm.value) {
            e.preventDefault();
            alert(t.passwordMismatch || 'Passwords do not match');
            return false;
        }
        if (password.length < 8) {
            e.preventDefault();
            alert(t.passwordLength || 'Password must be at least 8 characters');
            return false;
        }
    });
}

/* Checkbox-gated continue button */
function initConfirmCheckboxes() {
    var confirm1 = document.getElementById('confirm1');
    var confirm2 = document.getElementById('confirm2');
    var continueBtn = document.getElementById('continue-btn');
    if (!confirm1 || !confirm2 || !continueBtn) return;

    function update() {
        if (confirm1.checked && confirm2.checked) {
            continueBtn.disabled = false;
            continueBtn.classList.remove('bg-gray-400', 'cursor-not-allowed');
            continueBtn.classList.add('anemone-gradient', 'hover:opacity-90');
        } else {
            continueBtn.disabled = true;
            continueBtn.classList.add('bg-gray-400', 'cursor-not-allowed');
            continueBtn.classList.remove('anemone-gradient', 'hover:opacity-90');
        }
    }
    confirm1.addEventListener('change', update);
    confirm2.addEventListener('change', update);
}

/* Copy text from an input to clipboard */
function copyFromInput(inputId, btn) {
    var input = document.getElementById(inputId);
    if (!input) return;
    input.select();
    document.execCommand('copy');
    var original = btn.textContent;
    btn.textContent = t.copied || 'Copied!';
    btn.classList.add('bg-green-600');
    setTimeout(function() {
        btn.textContent = original;
        btn.classList.remove('bg-green-600');
    }, 2000);
}

/* Download encryption key as file */
function downloadKey() {
    var input = document.getElementById('encryption-key');
    if (!input) return;
    var blob = new Blob([input.value], { type: 'text/plain' });
    var url = window.URL.createObjectURL(blob);
    var a = document.createElement('a');
    a.href = url;
    a.download = pageData.keyFilename || 'anemone-encryption-key.txt';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(url);
}

/* Event delegation for auth pages */
document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');

    switch (action) {
        case 'copyKey':
            copyFromInput('encryption-key', target);
            break;
        case 'copySyncPassword':
            copyFromInput('sync-password', target);
            break;
        case 'downloadKey':
            downloadKey();
            break;
    }
});

/* Change event delegation (for select elements) */
document.addEventListener('change', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');

    switch (action) {
        case 'changeLanguage':
            changeLanguage();
            break;
    }
});

/* Auto-init based on page elements */
validatePasswordForm('activation-form');
validatePasswordForm('setup-form');
initConfirmCheckboxes();
