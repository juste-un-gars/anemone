/* Anemone - Admin restore users page (legacy) */
var pageData = JSON.parse(document.getElementById('page-data').textContent || '{}');
var t = pageData.translations || {};
var allBackups = pageData.backups || [];

var selectedPeerID = null;
var selectedPeerName = '';

document.addEventListener('DOMContentLoaded', function() {
    var peerSelect = document.getElementById('peer-select');
    if (!peerSelect) return;
    var rows = document.querySelectorAll('#backups-table tr[data-peer-id]');

    var peers = {};
    var peerOrder = [];
    rows.forEach(function(row) {
        var peerID = row.getAttribute('data-peer-id');
        var peerName = row.getAttribute('data-peer-name');
        if (!peers[peerID]) {
            peers[peerID] = peerName;
            peerOrder.push(peerID);
        }
    });

    peerOrder.forEach(function(id) {
        var option = document.createElement('option');
        option.value = id;
        option.textContent = peers[id];
        peerSelect.appendChild(option);
    });

    if (peerOrder.length > 0) {
        peerSelect.value = peerOrder[0];
        filterByPeer();
    }

    peerSelect.addEventListener('change', filterByPeer);
});

function filterByPeer() {
    var peerSelect = document.getElementById('peer-select');
    selectedPeerID = peerSelect.value;
    selectedPeerName = peerSelect.options[peerSelect.selectedIndex].text;

    var rows = document.querySelectorAll('#backups-table tr[data-peer-id]');
    rows.forEach(function(row) {
        row.style.display = (row.getAttribute('data-peer-id') === selectedPeerID) ? '' : 'none';
    });

    var restoreBtn = document.getElementById('restore-all-btn');
    if (restoreBtn) {
        restoreBtn.textContent = t.restoreAllFrom + ' ' + selectedPeerName;
    }
}

function restoreUser(userID, peerID, shareName, sourceServer, username) {
    if (!confirm(t.confirmRestore + ' ' + username + ' ?')) return;

    var statusDiv = document.getElementById('status-message');
    statusDiv.className = 'mb-6 bg-blue-50 border border-blue-200 text-blue-700 px-4 py-3 rounded relative';
    statusDiv.textContent = t.restoring + ' ' + username + '...';
    statusDiv.classList.remove('hidden');

    var formData = new FormData();
    formData.append('user_id', userID);
    formData.append('peer_id', peerID);
    formData.append('share_name', shareName);
    formData.append('source_server', sourceServer);

    fetch('/admin/restore-users/restore', { method: 'POST', body: formData })
    .then(function(response) { return response.json(); })
    .then(function(data) {
        if (data.success) {
            statusDiv.className = 'mb-6 bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded relative';
            statusDiv.textContent = data.message;
        } else {
            statusDiv.className = 'mb-6 bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded relative';
            statusDiv.textContent = t.error + ': ' + data.error;
        }
    })
    .catch(function(error) {
        statusDiv.className = 'mb-6 bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded relative';
        statusDiv.textContent = t.errorConnection + ': ' + error;
    });
}

function restoreAll() {
    if (!confirm(t.confirmAll + ' ' + selectedPeerName + ' ?')) return;

    var backupsToRestore = allBackups.filter(function(b) { return String(b.peerID) === String(selectedPeerID); });
    var completed = 0;
    var total = backupsToRestore.length;

    var statusDiv = document.getElementById('status-message');
    statusDiv.className = 'mb-6 bg-blue-50 border border-blue-200 text-blue-700 px-4 py-3 rounded relative';
    statusDiv.textContent = t.restoringAll + ' 0/' + total;
    statusDiv.classList.remove('hidden');

    backupsToRestore.forEach(function(backup) {
        var formData = new FormData();
        formData.append('user_id', backup.userID);
        formData.append('peer_id', backup.peerID);
        formData.append('share_name', backup.shareName);
        formData.append('source_server', backup.sourceServer);

        fetch('/admin/restore-users/restore', { method: 'POST', body: formData })
        .then(function(response) { return response.json(); })
        .then(function(data) {
            completed++;
            statusDiv.textContent = t.restoringAll + ' ' + completed + '/' + total;
            if (completed === total) {
                statusDiv.className = 'mb-6 bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded relative';
                statusDiv.textContent = t.successAll;
            }
        })
        .catch(function(error) {
            completed++;
            console.error('Restore error:', error);
        });
    });
}

document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');
    switch (action) {
        case 'restoreUser':
            restoreUser(
                target.getAttribute('data-user-id'),
                target.getAttribute('data-peer-id'),
                target.getAttribute('data-share-name'),
                target.getAttribute('data-source-server'),
                target.getAttribute('data-username')
            );
            break;
        case 'restoreAll':
            restoreAll();
            break;
    }
});
