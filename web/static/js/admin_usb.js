/* Anemone - Admin USB backup page (legacy) */
var pageData = JSON.parse(document.getElementById('page-data').textContent || '{}');
var t = pageData.translations || {};

function toggleShareSelection(index, show) {
    var el = document.getElementById('shareSelection' + index);
    if (!el) return;
    el.style.display = show ? 'block' : 'none';
    el.querySelectorAll('input[type="checkbox"]').forEach(function(cb) {
        cb.checked = show;
    });
}

function ejectDisk(mountPath) {
    if (!confirm(t.ejectConfirm)) return;

    fetch('/api/admin/storage/disk/unmount', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({mount_path: mountPath, eject: true})
    })
    .then(function(resp) { return resp.json(); })
    .then(function(data) {
        if (data.success) {
            alert(t.ejected);
            location.reload();
        } else {
            alert(t.error + ': ' + data.error);
        }
    })
    .catch(function(err) {
        alert(t.error + ': ' + err);
    });
}

document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');
    switch (action) {
        case 'ejectDisk':
            ejectDisk(target.getAttribute('data-mount-path'));
            break;
    }
});

document.addEventListener('change', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');
    if (action === 'toggleShareSelection') {
        var index = target.getAttribute('data-index');
        var show = target.getAttribute('data-show') === 'true';
        toggleShareSelection(index, show);
    }
});
