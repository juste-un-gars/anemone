/* Anemone shares management */
(function() {
    var pageData = JSON.parse(document.getElementById('page-data').textContent || '{}');
    var t = pageData.translations || {};

    function deleteShare(shareID, shareName) {
        if (!confirm(t.deleteConfirm + '\n\n' + shareName)) return;
        fetch('/shares/' + shareID + '/delete', { method: 'POST' })
        .then(function(r) { if (r.ok) window.location.reload(); else alert(t.error); });
    }

    function syncShare(shareID, shareName, btn) {
        if (!confirm('Sync "' + shareName + '" ?')) return;
        var orig = btn.innerHTML;
        btn.disabled = true;
        btn.textContent = 'Sync...';

        fetch('/sync/share/' + shareID, { method: 'POST' })
        .then(function(r) { return r.json(); })
        .then(function(data) {
            btn.disabled = false;
            btn.innerHTML = orig;
            alert(data.message);
        })
        .catch(function(err) {
            btn.disabled = false;
            btn.innerHTML = orig;
            alert('Error: ' + err);
        });
    }

    document.addEventListener('click', function(e) {
        var target = e.target.closest('[data-action]');
        if (!target) return;
        var action = target.getAttribute('data-action');
        switch (action) {
            case 'deleteShare':
                deleteShare(target.getAttribute('data-id'), target.getAttribute('data-name'));
                break;
            case 'syncShare':
                syncShare(target.getAttribute('data-id'), target.getAttribute('data-name'), target);
                break;
        }
    });
})();
