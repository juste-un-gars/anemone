/* Anemone common JS - sidebar, collapse, theme switcher, event delegation */

function toggleSidebar() {
    document.getElementById('sidebar').classList.toggle('open');
    document.getElementById('sidebarOverlay').classList.toggle('open');
}

function toggleCollapse(btn) {
    var expanded = btn.getAttribute('aria-expanded') === 'true';
    btn.setAttribute('aria-expanded', String(!expanded));
    var sub = btn.nextElementSibling;
    if (sub) sub.classList.toggle('open');
}

function setTheme(mode) {
    localStorage.setItem('anemone-theme', mode);
    if (mode === 'dark' || (mode === 'auto' && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
        document.documentElement.classList.add('dark');
    } else {
        document.documentElement.classList.remove('dark');
    }
    document.querySelectorAll('.v2-theme-btn').forEach(function(b) {
        b.classList.toggle('active', b.getAttribute('data-theme') === mode);
    });
}

/* Init theme button active state */
(function() {
    var t = localStorage.getItem('anemone-theme') || 'dark';
    document.querySelectorAll('.v2-theme-btn').forEach(function(b) {
        b.classList.toggle('active', b.getAttribute('data-theme') === t);
    });
})();

/* Global event delegation for data-action attributes */
document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');

    switch (action) {
        case 'toggleSidebar':
            toggleSidebar();
            break;
        case 'toggleCollapse':
            toggleCollapse(target);
            break;
        case 'setTheme':
            setTheme(target.getAttribute('data-theme'));
            break;
        case 'copyInput':
            var inputId = target.getAttribute('data-target');
            var input = document.getElementById(inputId);
            if (input) {
                input.select();
                document.execCommand('copy');
                var orig = target.textContent;
                target.textContent = '\u2713 OK';
                setTimeout(function() { target.textContent = orig; }, 2000);
            }
            break;
    }
});

/* Generic data-confirm on form submit buttons */
document.addEventListener('click', function(e) {
    var btn = e.target.closest('[data-confirm]');
    if (!btn) return;
    var msg = btn.getAttribute('data-confirm');
    if (!confirm(msg)) {
        e.preventDefault();
    }
});

/* data-submit-disable: disable button and change text on form submit */
document.addEventListener('submit', function(e) {
    var btn = e.target.querySelector('[data-submit-disable]');
    if (!btn) return;
    btn.disabled = true;
    btn.textContent = btn.getAttribute('data-submit-disable');
});

/* Sync frequency toggle (used in peers add/edit) */
function initSyncFrequencyToggle() {
    var sel = document.getElementById('sync_frequency');
    if (!sel) return;
    sel.addEventListener('change', function() {
        var f = this.value;
        var el = document.getElementById('interval_section');
        if (el) el.style.display = f === 'interval' ? '' : 'none';
        el = document.getElementById('sync_time_section');
        if (el) el.style.display = f === 'interval' ? 'none' : '';
        el = document.getElementById('day_of_week_section');
        if (el) el.style.display = f === 'weekly' ? '' : 'none';
        el = document.getElementById('day_of_month_section');
        if (el) el.style.display = f === 'monthly' ? '' : 'none';
    });
}
initSyncFrequencyToggle();
