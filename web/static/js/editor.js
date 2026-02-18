/* Anemone - OnlyOffice editor page */
(function() {
    var pageData = JSON.parse(document.getElementById('page-data').textContent || '{}');

    function showCertError() {
        var ooUrl = pageData.ooURL || '';
        document.getElementById('editor-container').innerHTML =
            '<div class="editor-loading" style="flex-direction:column;gap:1rem;text-align:center;">' +
            '<p>Cannot load OnlyOffice editor.</p>' +
            '<p>You may need to accept the self-signed certificate:</p>' +
            '<a href="' + ooUrl + '/healthcheck" target="_blank" style="color:#6366f1;text-decoration:underline;">Open OnlyOffice to accept certificate</a>' +
            '<p style="font-size:0.8rem;color:#666;">After accepting, reload this page.</p>' +
            '</div>';
    }

    if (typeof DocsAPI !== 'undefined') {
        var editorConfig = JSON.parse(document.getElementById('editor-config').textContent || '{}');
        new DocsAPI.DocEditor("editor-container", editorConfig);
    } else {
        showCertError();
    }
})();
