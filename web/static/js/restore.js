// restore.js — Restore page logic (extracted from v2_restore.html)

var pageData = JSON.parse(document.getElementById('page-data')?.textContent || '{}');
var t = pageData.translations || {};
var lang = pageData.lang || 'en';

function tr(template, values) {
    var result = template;
    for (var key in values) {
        if (values.hasOwnProperty(key)) {
            result = result.replace('{' + key + '}', values[key]);
        }
    }
    return result;
}

var backups = [];
var currentBackup = null;
var currentPath = '/';
var fileTree = null;
var selectedItems = new Set();

document.addEventListener('DOMContentLoaded', async function() { await loadBackups(); });

document.getElementById('backup-select').addEventListener('change', async function(e) {
    var value = e.target.value;
    if (!value) {
        document.getElementById('backup-info').classList.add('hidden');
        document.getElementById('file-browser').classList.add('hidden');
        return;
    }
    var parts = value.split(':');
    var peerIdStr = parts[0];
    var sourceServer = parts[1];
    var shareName = parts.slice(2).join(':');
    var peerId = parseInt(peerIdStr);
    currentBackup = backups.find(function(b) { return b.peer_id === peerId && b.source_server === sourceServer && b.share_name === shareName; });
    if (!currentBackup) return;
    document.getElementById('backup-file-count').textContent = currentBackup.file_count;
    document.getElementById('backup-size').textContent = formatBytes(currentBackup.total_size);
    document.getElementById('backup-date').textContent = formatDate(currentBackup.last_modified);
    document.getElementById('backup-info').classList.remove('hidden');
    await loadFiles(peerId, shareName, sourceServer);
});

// Event delegation for data-action buttons
document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');
    if (action === 'selectAll') { selectAll(); }
    else if (action === 'clearSelection') { clearSelection(); }
    else if (action === 'downloadSelection') { downloadSelection(); }
});

// Event listener for select-all checkbox
document.getElementById('select-all-checkbox').addEventListener('change', function() {
    toggleSelectAll();
});

async function loadBackups() {
    try {
        var response = await fetch('/api/restore/backups');
        if (!response.ok) throw new Error('Failed to load backups');
        backups = await response.json() || [];
        var select = document.getElementById('backup-select');
        if (backups.length === 0) { document.getElementById('no-backups').classList.remove('hidden'); return; }
        backups.forEach(function(backup) {
            var option = document.createElement('option');
            option.value = backup.peer_id + ':' + backup.source_server + ':' + backup.share_name;
            option.textContent = backup.peer_name + ' - ' + backup.share_name + ' (from ' + backup.source_server + ')';
            select.appendChild(option);
        });
    } catch (error) { console.error('Error loading backups:', error); alert(t.errorLoadingBackups); }
}

async function loadFiles(peerId, shareName, sourceServer) {
    document.getElementById('loading').classList.remove('hidden');
    document.getElementById('file-browser').classList.add('hidden');
    try {
        var response = await fetch('/api/restore/files?peer_id=' + peerId + '&backup=' + encodeURIComponent(shareName) + '&source_server=' + encodeURIComponent(sourceServer));
        if (!response.ok) throw new Error('Failed to load files');
        fileTree = await response.json();
        currentPath = '/';
        renderFiles();
        document.getElementById('file-browser').classList.remove('hidden');
    } catch (error) { console.error('Error loading files:', error); alert(t.errorLoadingFiles); }
    finally { document.getElementById('loading').classList.add('hidden'); }
}

function renderFiles() {
    var fileList = document.getElementById('file-list');
    fileList.innerHTML = '';
    var currentNode = getNodeAtPath(currentPath);
    if (!currentNode || !currentNode.children) return;
    var items = Object.values(currentNode.children).sort(function(a, b) {
        if (a.is_dir && !b.is_dir) return -1;
        if (!a.is_dir && b.is_dir) return 1;
        return a.name.localeCompare(b.name);
    });
    items.forEach(function(item) {
        var row = document.createElement('tr');
        // Checkbox
        var checkboxCell = document.createElement('td');
        var checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.className = 'item-checkbox';
        checkbox.style.accentColor = 'var(--info)';
        checkbox.dataset.path = item.path;
        checkbox.dataset.isDir = item.is_dir;
        checkbox.addEventListener('change', function() { toggleItem(item.path, item.is_dir); });
        checkboxCell.appendChild(checkbox);
        row.appendChild(checkboxCell);
        // Name
        var nameCell = document.createElement('td');
        if (item.is_dir) {
            var link = document.createElement('a');
            link.href = '#';
            link.style.cssText = 'display:flex;align-items:center;gap:0.5rem;color:var(--info);text-decoration:none;font-weight:500;';
            link.innerHTML = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--warning)" stroke-width="2"><path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>' + item.name;
            link.addEventListener('click', function(e) { e.preventDefault(); navigateToDir(item.path); });
            nameCell.appendChild(link);
        } else {
            nameCell.innerHTML = '<div style="display:flex;align-items:center;gap:0.5rem;color:var(--text-primary);"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" stroke-width="2"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>' + item.name + '</div>';
        }
        row.appendChild(nameCell);
        // Size
        var sizeCell = document.createElement('td');
        sizeCell.style.color = 'var(--text-muted)';
        sizeCell.textContent = item.is_dir ? '-' : formatBytes(item.size);
        row.appendChild(sizeCell);
        // Modified
        var modCell = document.createElement('td');
        modCell.style.color = 'var(--text-muted)';
        modCell.textContent = item.is_dir ? '-' : formatDate(item.mod_time);
        row.appendChild(modCell);
        // Actions
        var actionsCell = document.createElement('td');
        actionsCell.style.textAlign = 'right';
        if (!item.is_dir) {
            var downloadLink = document.createElement('a');
            downloadLink.href = '#';
            downloadLink.style.cssText = 'font-size:0.75rem;color:var(--info);text-decoration:none;';
            downloadLink.textContent = t.downloadAction;
            downloadLink.addEventListener('click', function(e) { e.preventDefault(); downloadFile(item.path); });
            actionsCell.appendChild(downloadLink);
        }
        row.appendChild(actionsCell);
        fileList.appendChild(row);
    });
    updateBreadcrumb();
}

function getNodeAtPath(path) {
    if (path === '/') return fileTree;
    var parts = path.split('/').filter(function(p) { return p; });
    var node = fileTree;
    for (var i = 0; i < parts.length; i++) {
        if (!node.children || !node.children[parts[i]]) return null;
        node = node.children[parts[i]];
    }
    return node;
}

function navigateToDir(path) { currentPath = path; renderFiles(); }

function updateBreadcrumb() {
    var breadcrumb = document.getElementById('breadcrumb');
    breadcrumb.innerHTML = '';
    var rootLi = document.createElement('li');
    var rootLink = document.createElement('a');
    rootLink.href = '#';
    rootLink.style.cssText = 'color:var(--text-muted);display:flex;align-items:center;';
    rootLink.innerHTML = '<svg width="18" height="18" viewBox="0 0 20 20" fill="currentColor"><path d="M10.707 2.293a1 1 0 00-1.414 0l-7 7a1 1 0 001.414 1.414L4 10.414V17a1 1 0 001 1h2a1 1 0 001-1v-2a1 1 0 011-1h2a1 1 0 011 1v2a1 1 0 001 1h2a1 1 0 001-1v-6.586l.293.293a1 1 0 001.414-1.414l-7-7z"/></svg>';
    rootLink.addEventListener('click', function(e) { e.preventDefault(); navigateToDir('/'); });
    rootLi.appendChild(rootLink);
    breadcrumb.appendChild(rootLi);
    if (currentPath !== '/') {
        var parts = currentPath.split('/').filter(function(p) { return p; });
        var accPath = '';
        parts.forEach(function(part, index) {
            accPath += '/' + part;
            var li = document.createElement('li');
            li.style.display = 'flex';
            li.style.alignItems = 'center';
            var sep = document.createElement('span');
            sep.style.cssText = 'color:var(--text-muted);margin:0 0.25rem;';
            sep.textContent = '/';
            li.appendChild(sep);
            var partLink = document.createElement('a');
            partLink.href = '#';
            partLink.style.cssText = 'font-size:0.8125rem;font-weight:500;text-decoration:none;color:' + (index === parts.length - 1 ? 'var(--text-primary)' : 'var(--text-muted)') + ';';
            partLink.textContent = part;
            var pathToNavigate = accPath;
            partLink.addEventListener('click', function(e) { e.preventDefault(); navigateToDir(pathToNavigate); });
            li.appendChild(partLink);
            breadcrumb.appendChild(li);
        });
    }
}

async function downloadFile(filePath) {
    if (!currentBackup) return;
    var fileName = filePath.split('/').pop();
    var url = '/api/restore/download?peer_id=' + currentBackup.peer_id + '&backup=' + encodeURIComponent(currentBackup.share_name) + '&file=' + encodeURIComponent(filePath) + '&source_server=' + encodeURIComponent(currentBackup.source_server);
    try {
        var response = await fetch(url);
        if (!response.ok) throw new Error('HTTP error! status: ' + response.status);
        var blob = await response.blob();
        var blobUrl = window.URL.createObjectURL(blob);
        var a = document.createElement('a');
        a.href = blobUrl; a.download = fileName;
        document.body.appendChild(a); a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(blobUrl);
    } catch (error) { console.error('Download error:', error); alert(t.errorDownload); }
}

function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    var k = 1024;
    var sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    var i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function formatDate(dateString) {
    var date = new Date(dateString);
    var now = new Date();
    var diff = now - date;
    var days = Math.floor(diff / (1000 * 60 * 60 * 24));
    if (days === 0) {
        var hours = Math.floor(diff / (1000 * 60 * 60));
        if (hours === 0) { var mins = Math.floor(diff / (1000 * 60)); return tr(t.minutesAgo, {minutes: mins}); }
        return tr(t.hoursAgo, {hours: hours});
    }
    if (days < 30) return tr(t.daysAgo, {days: days});
    return date.toLocaleDateString(lang === 'fr' ? 'fr-FR' : 'en-US');
}

function toggleItem(path) {
    if (selectedItems.has(path)) { selectedItems.delete(path); } else { selectedItems.add(path); }
    updateSelectionUI();
}

function toggleSelectAll() {
    var checkbox = document.getElementById('select-all-checkbox');
    document.querySelectorAll('.item-checkbox').forEach(function(cb) { cb.checked = checkbox.checked; if (checkbox.checked) selectedItems.add(cb.dataset.path); else selectedItems.delete(cb.dataset.path); });
    updateSelectionUI();
}

function selectAll() {
    document.querySelectorAll('.item-checkbox').forEach(function(cb) { cb.checked = true; selectedItems.add(cb.dataset.path); });
    document.getElementById('select-all-checkbox').checked = true;
    updateSelectionUI();
}

function clearSelection() {
    selectedItems.clear();
    document.querySelectorAll('.item-checkbox').forEach(function(cb) { cb.checked = false; });
    document.getElementById('select-all-checkbox').checked = false;
    updateSelectionUI();
}

function updateSelectionUI() {
    var count = selectedItems.size;
    var toolbar = document.getElementById('selection-toolbar');
    var countEl = document.getElementById('selection-count');
    if (count > 0) { toolbar.classList.remove('hidden'); toolbar.style.display = 'flex'; countEl.textContent = count + ' ' + t.selectionCount; }
    else { toolbar.classList.add('hidden'); }
    var itemCheckboxes = document.querySelectorAll('.item-checkbox');
    document.getElementById('select-all-checkbox').checked = itemCheckboxes.length > 0 && Array.from(itemCheckboxes).every(function(cb) { return cb.checked; });
}

async function downloadSelection() {
    if (selectedItems.size === 0) { alert(t.errorSelection); return; }
    if (!currentBackup) return;
    var paths = Array.from(selectedItems);
    var url = '/api/restore/download-multiple?peer_id=' + currentBackup.peer_id + '&backup=' + encodeURIComponent(currentBackup.share_name) + '&source_server=' + encodeURIComponent(currentBackup.source_server);
    var form = document.createElement('form');
    form.method = 'POST'; form.action = url;
    paths.forEach(function(path) { var input = document.createElement('input'); input.type = 'hidden'; input.name = 'paths'; input.value = path; form.appendChild(input); });
    document.body.appendChild(form); form.submit(); document.body.removeChild(form);
}
