// Anemone - WireGuard VPN page JS (externalized from v2_wireguard.html)
var pageData = JSON.parse(document.getElementById('page-data')?.textContent || '{}');
var t = pageData.translations || {};

document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');
    switch (action) {
        case 'openModal':
            document.getElementById(target.getAttribute('data-target')).classList.remove('hidden');
            break;
        case 'closeModal':
            document.getElementById(target.getAttribute('data-target')).classList.add('hidden');
            break;
        case 'deleteWireguard':
            if (confirm(t.deleteConfirm)) document.getElementById('deleteForm').submit();
            break;
    }
});

// Auto-submit on auto_start checkbox change
var autoStart = document.querySelector('input[name="auto_start"]');
if (autoStart) {
    autoStart.addEventListener('change', function() { this.form.submit(); });
}
