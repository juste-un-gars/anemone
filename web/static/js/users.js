/* Anemone users page JS - delete/unlock users */

var pageData = JSON.parse(document.getElementById('page-data')?.textContent || '{}');
var t = pageData.translations || {};

function unlockUser(userId, username) {
    if (confirm(t.unlockConfirm.replace('%s', username))) {
        fetch('/admin/users/' + userId + '/unlock', {
            method: 'POST',
        }).then(function(response) {
            if (response.ok) {
                alert(t.unlockSuccess);
                window.location.reload();
            } else {
                alert(t.unlockError);
            }
        }).catch(function(error) {
            alert(t.unlockError + error);
        });
    }
}

function deleteUser(userId, username) {
    var warningMessage = t.deleteWarningTitle + '\n\n' +
        t.deleteWarningMessage + '\n' + username + '\n\n' +
        t.deleteWarningPrompt;

    var confirmation = prompt(warningMessage);

    if (confirmation === username) {
        if (confirm(t.deleteConfirm)) {
            fetch('/admin/users/' + userId + '/delete', {
                method: 'POST',
            }).then(function(response) {
                if (response.ok) {
                    alert(t.deleteSuccess);
                    window.location.reload();
                } else {
                    response.text().then(function(text) { alert(t.deleteErrorPrefix + text); });
                }
            }).catch(function(error) {
                alert(t.deleteError + error);
            });
        }
    } else if (confirmation !== null) {
        alert(t.deleteCancelled);
    }
}

/* Event delegation */
document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');

    switch (action) {
        case 'deleteUser':
            deleteUser(target.getAttribute('data-id'), target.getAttribute('data-username'));
            break;
        case 'unlockUser':
            unlockUser(target.getAttribute('data-id'), target.getAttribute('data-username'));
            break;
    }
});
