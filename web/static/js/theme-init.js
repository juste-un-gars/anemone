/* Anemone theme init - must load in <head> to prevent flash */
(function() {
    var stored = localStorage.getItem('anemone-theme');
    function apply(mode) {
        if (mode === 'dark' || (mode === 'auto' && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }
    }
    apply(stored || 'dark');
    if (!stored || stored === 'auto') {
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function() {
            if ((localStorage.getItem('anemone-theme') || 'auto') === 'auto') apply('auto');
        });
    }
})();
