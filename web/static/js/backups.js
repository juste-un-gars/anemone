/* Anemone backups page JS - tabs, SSH keys, server backup, incoming */

var pageData = JSON.parse(document.getElementById('page-data')?.textContent || '{}');
var t = pageData.translations || {};

/* Tab switching */
function switchTab(tab) {
    var url = new URL(window.location);
    url.searchParams.set('tab', tab);
    window.history.replaceState({}, '', url);

    document.querySelectorAll('.v2-tab').forEach(function(btn) {
        btn.classList.toggle('active', btn.getAttribute('data-tab') === tab);
    });

    document.querySelectorAll('.v2-tab-panel').forEach(function(p) {
        p.classList.toggle('active', p.id === 'tab-' + tab);
    });
}

/* SSH key functions */
function copyPublicKey() {
    var textarea = document.getElementById('ssh-public-key');
    textarea.select();
    document.execCommand('copy');
    alert(t.sshKeyCopied);
}

function generateSSHKey() {
    if (!confirm(t.sshKeyGenerateConfirm)) return;
    fetch('/admin/rclone/generate-key', { method: 'POST' })
        .then(function(r) { if (!r.ok) throw new Error('HTTP ' + r.status); return r.json(); })
        .then(function(data) { if (data.exists) location.reload(); else alert(data.error || 'Error'); })
        .catch(function(err) { alert('Error: ' + err); });
}

function regenerateSSHKey() {
    if (!confirm(t.sshKeyRegenerateConfirm)) return;
    fetch('/admin/rclone/generate-key', { method: 'POST' })
        .then(function(r) { if (!r.ok) throw new Error('HTTP ' + r.status); return r.json(); })
        .then(function(data) { if (data.exists) location.reload(); else alert(data.error || 'Error'); })
        .catch(function(err) { alert('Error: ' + err); });
}

/* Server backup: download modal */
function openDownloadModal(filename) {
    document.getElementById('downloadFilename').value = filename;
    document.getElementById('dlPassphrase').value = '';
    document.getElementById('dlPassphraseConfirm').value = '';
    document.getElementById('downloadModal').classList.remove('hidden');
}

function closeDownloadModal() {
    document.getElementById('downloadModal').classList.add('hidden');
}

document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') closeDownloadModal();
});

var dlForm = document.getElementById('downloadForm');
if (dlForm) {
    dlForm.addEventListener('submit', function(e) {
        var p = document.getElementById('dlPassphrase').value;
        var c = document.getElementById('dlPassphraseConfirm').value;
        if (p !== c) { e.preventDefault(); alert(t.downloadErrorMismatch); return; }
        if (p.length < 12) { e.preventDefault(); alert(t.downloadErrorLength); return; }
    });
}

/* Server backup: delete */
function deleteServerBackup(filename) {
    if (!confirm(t.backupDeleteConfirm)) return;
    fetch('/admin/backup/delete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: 'filename=' + encodeURIComponent(filename)
    }).then(function(r) {
        if (r.ok) { alert(t.backupDeleteSuccess); location.reload(); }
        else { r.text().then(function(txt) { alert(t.backupDeleteError + ': ' + txt); }); }
    }).catch(function(err) { alert(t.backupDeleteError + ': ' + err.message); });
}

/* Event delegation */
document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');

    switch (action) {
        case 'switchTab':
            switchTab(target.getAttribute('data-tab'));
            break;
        case 'copyPublicKey':
            copyPublicKey();
            break;
        case 'generateSSHKey':
            generateSSHKey();
            break;
        case 'regenerateSSHKey':
            regenerateSSHKey();
            break;
        case 'openDownloadModal':
            openDownloadModal(target.getAttribute('data-filename'));
            break;
        case 'deleteServerBackup':
            deleteServerBackup(target.getAttribute('data-filename'));
            break;
        case 'closeDownloadModal':
            closeDownloadModal();
            break;
        case 'confirmDelete':
            if (!confirm(t.incomingConfirmDelete)) {
                e.preventDefault();
            }
            break;
    }
});

/* Restore tab from URL on load */
(function() {
    var params = new URLSearchParams(window.location.search);
    var tab = params.get('tab');
    if (tab && document.getElementById('tab-' + tab)) {
        switchTab(tab);
    }
})();
