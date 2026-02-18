/* Anemone trash page JS - restore/delete trash items */

var pageData = JSON.parse(document.getElementById('page-data')?.textContent || '{}');
var t = pageData.translations || {};

function replacePlaceholders(text, params) {
    var result = text;
    for (var key in params) {
        if (params.hasOwnProperty(key)) {
            result = result.replace(new RegExp('\\{' + key + '\\}', 'g'), params[key]);
        }
    }
    return result;
}

function updateBulkActions() {
    var checkboxes = document.querySelectorAll('.file-checkbox:checked');
    var bulkActions = document.getElementById('bulkActions');
    var selectedCount = document.getElementById('selectedCount');
    var selectAll = document.getElementById('selectAll');

    if (checkboxes.length > 0) {
        bulkActions.classList.remove('hidden');
        selectedCount.textContent = checkboxes.length + ' ' + t.selected_count;
    } else {
        bulkActions.classList.add('hidden');
    }

    var allCheckboxes = document.querySelectorAll('.file-checkbox');
    selectAll.checked = allCheckboxes.length > 0 && checkboxes.length === allCheckboxes.length;
}

function toggleSelectAll() {
    var selectAll = document.getElementById('selectAll');
    var checkboxes = document.querySelectorAll('.file-checkbox');
    checkboxes.forEach(function(cb) { cb.checked = selectAll.checked; });
    updateBulkActions();
}

function deselectAll() {
    document.querySelectorAll('.file-checkbox').forEach(function(cb) { cb.checked = false; });
    document.getElementById('selectAll').checked = false;
    updateBulkActions();
}

function getSelectedFiles() {
    var selected = [];
    document.querySelectorAll('.file-checkbox:checked').forEach(function(cb) {
        selected.push({ share: cb.dataset.share, path: cb.dataset.path });
    });
    return selected;
}

function bulkRestore() {
    var files = getSelectedFiles();
    if (files.length === 0) return;
    var msg = replacePlaceholders(t.confirm_restore_bulk, {count: files.length});
    if (!confirm(msg)) return;
    var success = 0, failed = 0, done = 0;
    files.forEach(function(file) {
        fetch('/trash/restore?share=' + encodeURIComponent(file.share) + '&path=' + encodeURIComponent(file.path), { method: 'POST' })
        .then(function(response) {
            if (response.ok) { success++; } else { failed++; }
        })
        .catch(function() { failed++; })
        .finally(function() {
            done++;
            if (done === files.length) {
                var resultMsg = replacePlaceholders(t.restored_bulk, {success: success});
                if (failed > 0) { resultMsg += replacePlaceholders(t.failed_bulk, {failed: failed}); }
                alert(resultMsg);
                window.location.reload();
            }
        });
    });
}

function bulkDelete() {
    var files = getSelectedFiles();
    if (files.length === 0) return;
    var msg = replacePlaceholders(t.confirm_delete_bulk, {count: files.length});
    if (!confirm(msg)) return;
    var success = 0, failed = 0, done = 0;
    files.forEach(function(file) {
        fetch('/trash/delete?share=' + encodeURIComponent(file.share) + '&path=' + encodeURIComponent(file.path), { method: 'POST' })
        .then(function(response) {
            if (response.ok) { success++; } else { failed++; }
        })
        .catch(function() { failed++; })
        .finally(function() {
            done++;
            if (done === files.length) {
                var resultMsg = replacePlaceholders(t.deleted_bulk, {success: success});
                if (failed > 0) { resultMsg += replacePlaceholders(t.failed_bulk, {failed: failed}); }
                alert(resultMsg);
                window.location.reload();
            }
        });
    });
}

function restoreFile(shareName, relPath) {
    if (!confirm(t.confirm_restore + '\n\n' + relPath)) return;
    var button = event.target.closest('button');
    var originalContent = button.innerHTML;
    button.disabled = true;
    button.innerHTML = t.restoring;
    fetch('/trash/restore?share=' + encodeURIComponent(shareName) + '&path=' + encodeURIComponent(relPath), { method: 'POST' })
    .then(function(response) {
        if (response.ok) { alert(t.restored_success); window.location.reload(); }
        else { response.text().then(function(text) { alert(t.error + ' ' + text); button.disabled = false; button.innerHTML = originalContent; }); }
    })
    .catch(function(err) { alert(t.error + ' ' + err); button.disabled = false; button.innerHTML = originalContent; });
}

function deleteFile(shareName, relPath) {
    if (!confirm(t.confirm_delete + '\n\n' + relPath)) return;
    fetch('/trash/delete?share=' + encodeURIComponent(shareName) + '&path=' + encodeURIComponent(relPath), { method: 'POST' })
    .then(function(response) { if (response.ok) { window.location.reload(); } else { response.text().then(function(text) { alert(t.error + ' ' + text); }); } });
}

/* Event delegation for click actions */
document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');

    switch (action) {
        case 'bulkRestore':
            bulkRestore();
            break;
        case 'bulkDelete':
            bulkDelete();
            break;
        case 'deselectAll':
            deselectAll();
            break;
        case 'restoreFile':
            restoreFile(target.getAttribute('data-share'), target.getAttribute('data-path'));
            break;
        case 'deleteFile':
            deleteFile(target.getAttribute('data-share'), target.getAttribute('data-path'));
            break;
    }
});

/* Event delegation for checkbox changes */
document.addEventListener('change', function(e) {
    if (e.target.id === 'selectAll') {
        toggleSelectAll();
    } else if (e.target.classList.contains('file-checkbox')) {
        updateBulkActions();
    }
});
