// Anemone - USB backup edit form JS (externalized from v2_usb_backup_edit.html)

// Update schedule field visibility based on frequency selection
function updateUSBFields() {
    var f = document.getElementById('sync_frequency').value;
    document.getElementById('intervalField').style.display = f === 'interval' ? '' : 'none';
    document.getElementById('timeField').style.display = f !== 'interval' ? '' : 'none';
    document.getElementById('weekdayField').style.display = f === 'weekly' ? '' : 'none';
    document.getElementById('monthdayField').style.display = f === 'monthly' ? '' : 'none';
}

// Backup type radios -> toggle share selection visibility
var backupTypeRadios = document.querySelectorAll('input[name="backup_type"]');
for (var i = 0; i < backupTypeRadios.length; i++) {
    backupTypeRadios[i].addEventListener('change', function() {
        document.getElementById('shareSelection').style.display = this.value === 'config' ? 'none' : '';
    });
}

// Sync enabled checkbox toggle
var syncEnabled = document.getElementById('sync_enabled');
if (syncEnabled) {
    syncEnabled.addEventListener('change', function() {
        document.getElementById('scheduleOpts').style.display = this.checked ? '' : 'none';
    });
}

// Sync frequency select change
var syncFrequency = document.getElementById('sync_frequency');
if (syncFrequency) {
    syncFrequency.addEventListener('change', function() { updateUSBFields(); });
}
