/* Anemone peers page JS - test/delete peers */

var pageData = JSON.parse(document.getElementById('page-data')?.textContent || '{}');
var t = pageData.translations || {};

function testPeer(peerId, peerName) {
    if (!confirm(t.testAction + ' "' + peerName + '" ?')) return;
    fetch('/admin/peers/' + peerId + '/test', { method: 'POST' })
    .then(function(r) { return r.json(); })
    .then(function(data) {
        alert(data.status === 'online' ? t.statusOnline : t.statusOffline);
        window.location.reload();
    })
    .catch(function() { alert('Error'); });
}

function deletePeer(peerId, peerName) {
    if (!confirm(t.deleteConfirm + '\n\n' + peerName)) return;
    fetch('/admin/peers/' + peerId + '/delete', { method: 'POST' })
    .then(function(r) { if (r.ok) window.location.reload(); else alert('Error'); });
}

/* Event delegation */
document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');

    switch (action) {
        case 'testPeer':
            testPeer(target.getAttribute('data-id'), target.getAttribute('data-name'));
            break;
        case 'deletePeer':
            deletePeer(target.getAttribute('data-id'), target.getAttribute('data-name'));
            break;
    }
});
