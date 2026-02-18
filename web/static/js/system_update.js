/* Anemone system update */
(function() {
    var pageData = JSON.parse(document.getElementById('page-data').textContent || '{}');
    var t = pageData.translations || {};

    function checkForUpdates() {
        var button = document.getElementById('checkButton');
        var messageArea = document.getElementById('messageArea');
        button.disabled = true;
        button.textContent = t.checking + '...';

        fetch('/admin/system/update/check', { method: 'POST', headers: { 'Content-Type': 'application/json' } })
        .then(function(r) { return r.json(); })
        .then(function(data) {
            if (data.success) {
                messageArea.innerHTML = '<div class="v2-card" style="border-left:3px solid var(--success);margin-bottom:1rem;"><div style="font-size:0.875rem;color:var(--success);">' + data.updateMessage + '</div></div>';
                if (data.updateInfo) {
                    document.getElementById('latestVersion').textContent = data.updateInfo.latest_version;
                    document.getElementById('latestVersion').style.color = data.updateInfo.available ? 'var(--success)' : 'var(--text-secondary)';
                    var now = new Date();
                    document.getElementById('lastCheckText').textContent = t.lastCheck + ': ' + now.toLocaleString();
                    if (data.updateInfo.available) { setTimeout(function() { location.reload(); }, 2000); }
                }
            } else {
                messageArea.innerHTML = '<div class="v2-card" style="border-left:3px solid var(--error);margin-bottom:1rem;"><div style="font-size:0.875rem;color:var(--error);">' + data.error + '</div></div>';
            }
            button.disabled = false;
            button.textContent = t.checkNow;
        })
        .catch(function(error) {
            messageArea.innerHTML = '<div class="v2-card" style="border-left:3px solid var(--error);margin-bottom:1rem;"><div style="font-size:0.875rem;color:var(--error);">Error: ' + error.message + '</div></div>';
            button.disabled = false;
            button.textContent = t.checkNow;
        });
    }

    function confirmUpdate() {
        if (!confirm(t.confirmMessage + '\n\n' + t.confirmWarning)) return;
        var button = document.getElementById('installButton');
        var messageArea = document.getElementById('messageArea');
        button.disabled = true;
        button.textContent = t.installing;

        fetch('/admin/system/update/install', { method: 'POST', headers: { 'Content-Type': 'application/json' } })
        .then(function(r) { return r.json(); })
        .then(function(data) {
            if (data.success) {
                messageArea.innerHTML = '<div class="v2-card" style="border-left:3px solid var(--success);margin-bottom:1rem;"><div style="font-size:0.875rem;color:var(--success);">' + data.message + '<br><small style="color:var(--text-muted);">' + t.restarting + '</small></div></div>';
                messageArea.scrollIntoView({ behavior: 'smooth' });
                setTimeout(function() { window.location.href = '/login'; }, 30000);
            } else {
                messageArea.innerHTML = '<div class="v2-card" style="border-left:3px solid var(--error);margin-bottom:1rem;"><div style="font-size:0.875rem;color:var(--error);">' + data.error + '</div></div>';
                button.disabled = false;
                button.textContent = t.installButton;
            }
        })
        .catch(function(error) {
            messageArea.innerHTML = '<div class="v2-card" style="border-left:3px solid var(--error);margin-bottom:1rem;"><div style="font-size:0.875rem;color:var(--error);">Error: ' + error.message + '</div></div>';
            button.disabled = false;
            button.textContent = t.installButton;
        });
    }

    document.addEventListener('click', function(e) {
        var target = e.target.closest('[data-action]');
        if (!target) return;
        var action = target.getAttribute('data-action');
        switch (action) {
            case 'checkForUpdates': checkForUpdates(); break;
            case 'confirmUpdate': confirmUpdate(); break;
        }
    });
})();
