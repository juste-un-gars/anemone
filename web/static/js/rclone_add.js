// Anemone - Rclone add form JS (externalized from v2_rclone_add.html)

// Switch visible provider fields based on selection
function switchProvider(pt) {
    document.getElementById('sftpFields').style.display = pt === 'sftp' ? '' : 'none';
    document.getElementById('s3Fields').style.display = pt === 's3' ? '' : 'none';
    document.getElementById('webdavFields').style.display = pt === 'webdav' ? '' : 'none';
    document.getElementById('remoteFields').style.display = pt === 'remote' ? '' : 'none';
}

// Provider type select change
var providerType = document.getElementById('providerType');
if (providerType) {
    providerType.addEventListener('change', function() { switchProvider(this.value); });
}

// Crypt enabled checkbox toggle
var cryptEnabled = document.getElementById('cryptEnabled');
if (cryptEnabled) {
    cryptEnabled.addEventListener('change', function() {
        document.getElementById('cryptFields').style.display = this.checked ? '' : 'none';
    });
}
