/* Anemone quota calculation */
(function() {
    var b = document.querySelector('[name="quota_backup_gb"]');
    var d = document.getElementById('quota_data_gb');
    var t = document.getElementById('quota_total_display');
    if (!b || !d || !t) return;
    function upd() { t.textContent = ((parseInt(b.value)||0) + (parseInt(d.value)||0)) + ' GB'; }
    b.addEventListener('input', upd);
    d.addEventListener('input', upd);
})();
