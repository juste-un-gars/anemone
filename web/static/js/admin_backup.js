/* Anemone - Admin backup page (legacy) */
var pageData = JSON.parse(document.getElementById('page-data').textContent || '{}');
var t = pageData.translations || {};

function downloadBackup(filename) {
    document.getElementById('downloadFilename').value = filename;
    document.getElementById('downloadModal').classList.remove('hidden');
    document.getElementById('passphrase').value = '';
    document.getElementById('passphrase_confirm').value = '';
}

function closeModal() {
    document.getElementById('downloadModal').classList.add('hidden');
}

function deleteBackup(filename) {
    if (!confirm(t.deleteConfirm + '\n\n' + filename)) {
        return;
    }

    fetch('/admin/backup/delete', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded'
        },
        body: 'filename=' + encodeURIComponent(filename)
    })
    .then(function(response) {
        if (response.ok) {
            alert(t.deleteSuccess);
            location.reload();
        } else {
            return response.text().then(function(text) {
                throw new Error(text || t.deleteError);
            });
        }
    })
    .catch(function(error) {
        alert(t.deleteError + ': ' + error.message);
    });
}

document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');
    switch (action) {
        case 'downloadBackup':
            downloadBackup(target.getAttribute('data-filename'));
            break;
        case 'deleteBackup':
            deleteBackup(target.getAttribute('data-filename'));
            break;
        case 'closeModal':
            closeModal();
            break;
    }
});

document.getElementById('downloadForm').addEventListener('submit', function(e) {
    var passphrase = document.getElementById('passphrase').value;
    var confirmVal = document.getElementById('passphrase_confirm').value;

    if (passphrase !== confirmVal) {
        e.preventDefault();
        alert(t.passwordMismatch);
        return;
    }

    if (passphrase.length < 12) {
        e.preventDefault();
        alert(t.passwordTooShort);
        return;
    }
});

// Close modal on escape key
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
        closeModal();
    }
});
