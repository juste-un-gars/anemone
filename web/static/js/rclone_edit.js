// Anemone - Rclone edit form JS (externalized from v2_rclone_edit.html)
var pageData = JSON.parse(document.getElementById('page-data')?.textContent || '{}');
var t = pageData.translations || {};

// Update schedule field visibility based on frequency selection
function updateRcloneFields() {
    var f = document.getElementById('sync_frequency').value;
    document.getElementById('intervalField').style.display = f === 'interval' ? '' : 'none';
    document.getElementById('timeField').style.display = f !== 'interval' ? '' : 'none';
    document.getElementById('weekdayField').style.display = f === 'weekly' ? '' : 'none';
    document.getElementById('monthdayField').style.display = f === 'monthly' ? '' : 'none';
}

// Crypt enabled checkbox toggle
var cryptEnabled = document.getElementById('cryptEnabled');
if (cryptEnabled) {
    cryptEnabled.addEventListener('change', function() {
        document.getElementById('cryptFields').style.display = this.checked ? '' : 'none';
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
    syncFrequency.addEventListener('change', function() { updateRcloneFields(); });
}

// Delete button with confirmation
var deleteBtn = document.querySelector('[data-action="confirmDelete"]');
if (deleteBtn) {
    deleteBtn.addEventListener('click', function(e) {
        if (!confirm(t.confirmDelete)) {
            e.preventDefault();
        }
    });
}
