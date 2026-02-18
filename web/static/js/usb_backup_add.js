// Anemone - USB backup add form JS (externalized from v2_usb_backup_add.html)

// Drive select -> fill mount_path input
var driveSelect = document.getElementById('driveSelect');
if (driveSelect) {
    driveSelect.addEventListener('change', function() {
        document.getElementById('mount_path').value = this.value;
    });
}

// Backup type radios -> toggle share selection visibility
var backupTypeRadios = document.querySelectorAll('input[name="backup_type"]');
for (var i = 0; i < backupTypeRadios.length; i++) {
    backupTypeRadios[i].addEventListener('change', function() {
        document.getElementById('shareSelection').style.display = this.value === 'config' ? 'none' : '';
    });
}
