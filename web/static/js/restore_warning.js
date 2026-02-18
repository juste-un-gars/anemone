// Anemone - Restore warning page JS (externalized from restore_warning.html)
// NOTE: This script references elements (peerSelect, bulkRestoreForm) that may not
// exist in the current template. The code is preserved for backward compatibility.

(function() {
    var peerSelect = document.getElementById('peerSelect');

    if (!peerSelect) {
        return;
    }

    // Update hidden share_name field when peer is selected
    peerSelect.addEventListener('change', function() {
        var selectedOption = this.options[this.selectedIndex];
        var shareName = selectedOption.getAttribute('data-share-name');
        var shareNameField = document.getElementById('shareName');
        if (shareNameField) shareNameField.value = shareName || '';
    });

    // Auto-select if there's only one backup available
    if (peerSelect.options.length === 2) {
        peerSelect.selectedIndex = 1;
        var selectedOption = peerSelect.options[1];
        var shareName = selectedOption.getAttribute('data-share-name');
        var shareNameField = document.getElementById('shareName');
        if (shareNameField) shareNameField.value = shareName || '';
    }
})();

// Handle bulk restore form submission
var bulkRestoreForm = document.getElementById('bulkRestoreForm');
if (bulkRestoreForm) {
    bulkRestoreForm.addEventListener('submit', function(e) {
        e.preventDefault();

        var peerSelect = document.getElementById('peerSelect');
        var shareName = document.getElementById('shareName');

        if (!peerSelect || !peerSelect.value || peerSelect.value === '') {
            alert('Veuillez s\u00e9lectionner un serveur source avant de lancer la restauration.');
            return;
        }

        if (!shareName || !shareName.value || shareName.value === '') {
            alert('Erreur: Le nom du partage n\'a pas \u00e9t\u00e9 d\u00e9tect\u00e9. Veuillez r\u00e9essayer.');
            return;
        }

        if (!confirm('\u00cates-vous s\u00fbr de vouloir lancer la restauration automatique ? Cette op\u00e9ration peut prendre du temps.')) {
            return;
        }

        var progressContainer = document.getElementById('progressContainer');
        if (progressContainer) progressContainer.classList.remove('hidden');

        var formData = new FormData(this);

        var submitBtn = this.querySelector('button[type="submit"]');
        var selectField = this.querySelector('select');
        if (submitBtn) submitBtn.disabled = true;
        if (selectField) selectField.disabled = true;

        fetch('/restore-warning/bulk', {
            method: 'POST',
            body: formData
        })
        .then(function(response) { return response.json(); })
        .then(function(data) {
            var progressText = document.getElementById('progressText');
            var progressBar = document.getElementById('progressBar');
            if (data.success) {
                if (progressText) progressText.textContent = '\u2713 Restauration termin\u00e9e avec succ\u00e8s !';
                if (progressBar) {
                    progressBar.style.width = '100%';
                    progressBar.classList.remove('bg-blue-600');
                    progressBar.classList.add('bg-green-600');
                }
                setTimeout(function() {
                    window.location.href = '/dashboard';
                }, 2000);
            } else {
                if (progressText) progressText.textContent = '\u274c Erreur : ' + (data.error || 'Erreur inconnue');
                if (progressBar) {
                    progressBar.classList.remove('bg-blue-600');
                    progressBar.classList.add('bg-red-600');
                }
            }
        })
        .catch(function(error) {
            var progressText = document.getElementById('progressText');
            var progressBar = document.getElementById('progressBar');
            if (progressText) progressText.textContent = '\u274c Erreur r\u00e9seau : ' + error.message;
            if (progressBar) {
                progressBar.classList.remove('bg-blue-600');
                progressBar.classList.add('bg-red-600');
            }
        });
    });
}
