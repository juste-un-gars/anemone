/* Anemone files page JS - upload, mkdir, rename, delete, modals */

var pageData = JSON.parse(document.getElementById('page-data')?.textContent || '{}');
var t = pageData.translations || {};
var currentShare = pageData.currentShare || '';
var currentPath = pageData.currentPath || '';

function switchShare(name) {
    window.location.href = '/files?share=' + encodeURIComponent(name);
}

function renameItem(share, path, oldName) {
    document.getElementById('renameShare').value = share;
    document.getElementById('renameOldPath').value = path;
    document.getElementById('renameNewName').value = oldName;
    document.getElementById('renameModal').classList.remove('hidden');
    document.getElementById('renameNewName').focus();
    document.getElementById('renameNewName').select();
}

function deleteItem(share, path, name) {
    if (!confirm(t.confirmDelete + '\n\n' + name)) return;
    fetch('/api/files/delete', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({share: share, path: path})
    }).then(function(r) { return r.json(); }).then(function(data) {
        if (data.success) { window.location.reload(); }
        else { alert(t.error + ' ' + (data.message || '')); }
    }).catch(function(err) { alert(t.error + ' ' + err); });
}

/* Mkdir form */
var mkdirForm = document.getElementById('mkdirForm');
if (mkdirForm) {
    mkdirForm.addEventListener('submit', function(e) {
        e.preventDefault();
        var name = document.getElementById('mkdirName').value.trim();
        if (!name) return;
        fetch('/api/files/mkdir', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({share: currentShare, path: currentPath, name: name})
        }).then(function(r) { return r.json(); }).then(function(data) {
            if (data.success) {
                document.getElementById('mkdirModal').classList.add('hidden');
                window.location.reload();
            } else { alert(t.error + ' ' + (data.message || '')); }
        }).catch(function(err) { alert(t.error + ' ' + err); });
    });
}

/* Rename form */
var renameForm = document.getElementById('renameForm');
if (renameForm) {
    renameForm.addEventListener('submit', function(e) {
        e.preventDefault();
        var newName = document.getElementById('renameNewName').value.trim();
        if (!newName) return;
        var share = document.getElementById('renameShare').value;
        var oldPath = document.getElementById('renameOldPath').value;
        fetch('/api/files/rename', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({share: share, path: oldPath, new_name: newName})
        }).then(function(r) { return r.json(); }).then(function(data) {
            if (data.success) {
                document.getElementById('renameModal').classList.add('hidden');
                window.location.reload();
            } else { alert(t.error + ' ' + (data.message || '')); }
        }).catch(function(err) { alert(t.error + ' ' + err); });
    });
}

/* Upload form */
var uploadForm = document.getElementById('uploadForm');
if (uploadForm) {
    uploadForm.addEventListener('submit', function(e) {
        e.preventDefault();
        var files = document.getElementById('uploadFiles').files;
        if (files.length === 0) return;
        var formData = new FormData();
        formData.append('share', currentShare);
        formData.append('path', currentPath);
        for (var i = 0; i < files.length; i++) {
            formData.append('files', files[i]);
        }
        var xhr = new XMLHttpRequest();
        var progressDiv = document.getElementById('uploadProgress');
        var bar = document.getElementById('uploadBar');
        var pct = document.getElementById('uploadPercent');
        progressDiv.classList.remove('hidden');
        xhr.upload.addEventListener('progress', function(ev) {
            if (ev.lengthComputable) {
                var percent = Math.round((ev.loaded / ev.total) * 100);
                bar.style.width = percent + '%';
                pct.textContent = percent + '%';
            }
        });
        xhr.addEventListener('load', function() {
            if (xhr.status === 200) {
                document.getElementById('uploadModal').classList.add('hidden');
                window.location.reload();
            } else {
                try {
                    var data = JSON.parse(xhr.responseText);
                    alert(t.error + ' ' + (data.message || ''));
                } catch(e) { alert(t.error); }
            }
            progressDiv.classList.add('hidden');
            bar.style.width = '0%';
        });
        xhr.addEventListener('error', function() {
            alert(t.error);
            progressDiv.classList.add('hidden');
        });
        xhr.open('POST', '/api/files/upload');
        xhr.send(formData);
    });
}

/* Share selector change */
var shareSelector = document.getElementById('shareSelector');
if (shareSelector) {
    shareSelector.addEventListener('change', function() {
        switchShare(this.value);
    });
}

/* Close modals on background click */
['uploadModal', 'mkdirModal', 'renameModal'].forEach(function(id) {
    var el = document.getElementById(id);
    if (el) {
        el.addEventListener('click', function(e) {
            if (e.target === this) this.classList.add('hidden');
        });
    }
});

/* Event delegation */
document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');

    switch (action) {
        case 'openModal':
            var modalId = target.getAttribute('data-target');
            if (modalId) document.getElementById(modalId).classList.remove('hidden');
            break;
        case 'closeModal':
            var closeId = target.getAttribute('data-target');
            if (closeId) document.getElementById(closeId).classList.add('hidden');
            break;
        case 'renameItem':
            renameItem(
                target.getAttribute('data-share'),
                target.getAttribute('data-path'),
                target.getAttribute('data-name')
            );
            break;
        case 'deleteItem':
            deleteItem(
                target.getAttribute('data-share'),
                target.getAttribute('data-path'),
                target.getAttribute('data-name')
            );
            break;
    }
});
