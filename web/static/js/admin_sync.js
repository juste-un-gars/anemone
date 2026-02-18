/* Anemone - Admin sync config page (legacy) */
(function() {
    var intervalSelect = document.getElementById('interval');
    if (intervalSelect) {
        intervalSelect.addEventListener('change', function() {
            var fixedHourDiv = document.getElementById('fixedHourDiv');
            if (fixedHourDiv) {
                fixedHourDiv.style.display = (this.value === 'fixed') ? 'block' : 'none';
            }
        });
    }
})();
