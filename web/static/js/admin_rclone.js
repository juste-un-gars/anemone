/* Anemone - Admin rclone page (legacy) */
var pageData = JSON.parse(document.getElementById('page-data').textContent || '{}');
var t = pageData.translations || {};

function copyPublicKey() {
    var textarea = document.getElementById('public-key');
    textarea.select();
    document.execCommand('copy');
    alert(t.copied);
}

function generateKey() {
    if (!confirm(t.generateConfirm)) return;

    fetch('/admin/rclone/generate-key', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
    })
    .then(function(response) { return response.json(); })
    .then(function(data) {
        if (data.error) {
            alert('Error: ' + data.error);
        } else {
            window.location.href = '/admin/rclone?key_generated=1';
        }
    })
    .catch(function(err) {
        alert('Error: ' + err);
    });
}

function confirmRegenerateKey() {
    if (!confirm(t.regenerateConfirm)) return;
    generateKey();
}

document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');
    switch (action) {
        case 'copyPublicKey':
            copyPublicKey();
            break;
        case 'regenerateKey':
            confirmRegenerateKey();
            break;
        case 'generateKey':
            generateKey();
            break;
    }
});
