/* Anemone storage management */

var pageData = JSON.parse(document.getElementById('page-data').textContent || '{}');
var t = pageData.translations || {};
var poolNames = (pageData.poolNames || '').split(',').filter(function(p) { return p; });

var verificationToken = null;
var pendingAction = null;

/* Tab switching */
function showTab(tabName) {
    document.querySelectorAll('.v2-tab-panel').forEach(function(el) { el.classList.remove('active'); });
    document.querySelectorAll('.v2-tab').forEach(function(el) { el.classList.remove('active'); });
    var panel = document.getElementById('tab-' + tabName);
    if (panel) panel.classList.add('active');
    var tab = document.querySelector('.v2-tab[data-tab="' + tabName + '"]');
    if (tab) tab.classList.add('active');
    if (tabName === 'datasets') loadDatasets();
    if (tabName === 'snapshots') loadSnapshots();
}

/* Password verification */
function requirePassword(title, warning, callback) {
    pendingAction = callback;
    document.getElementById('passwordModalTitle').textContent = title;
    document.getElementById('passwordModalWarning').textContent = warning;
    document.getElementById('verifyPassword').value = '';
    document.getElementById('passwordModal').classList.remove('hidden');
}

function closePasswordModal() {
    document.getElementById('passwordModal').classList.add('hidden');
    pendingAction = null;
}

function submitPasswordVerification(e) {
    e.preventDefault();
    var password = document.getElementById('verifyPassword').value;
    fetch('/api/admin/verify-password', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({password: password})
    })
    .then(function(resp) { return resp.json(); })
    .then(function(data) {
        if (data.success && data.token) {
            verificationToken = data.token;
            var action = pendingAction;
            closePasswordModal();
            if (action) action();
        } else {
            alert(data.error || t.loginError);
        }
    })
    .catch(function(err) {
        alert(t.error + ': ' + err);
    });
}

/* SMART Modal */
function helpIcon(tip) {
    return '<span class="help-tooltip" data-tip="' + tip.replace(/"/g, '&quot;') + '">?</span>';
}

function formatBytes(units) {
    var bytes = units * 512 * 1000;
    if (bytes >= 1e15) return (bytes / 1e15).toFixed(1) + ' PB';
    if (bytes >= 1e12) return (bytes / 1e12).toFixed(1) + ' TB';
    if (bytes >= 1e9) return (bytes / 1e9).toFixed(1) + ' GB';
    return (bytes / 1e6).toFixed(1) + ' MB';
}

function formatHours(hours) {
    if (hours < 24) return hours + 'h';
    var days = Math.floor(hours / 24);
    if (days < 365) return days + 'j (' + hours.toLocaleString() + 'h)';
    var years = (days / 365).toFixed(1);
    return years + ' ans (' + hours.toLocaleString() + 'h)';
}

function getValueClass(value, warnThresh, critThresh, inverse) {
    if (inverse) {
        if (value <= critThresh) return 'smart-value-critical';
        if (value <= warnThresh) return 'smart-value-warning';
        return 'smart-value-good';
    } else {
        if (value >= critThresh) return 'smart-value-critical';
        if (value >= warnThresh) return 'smart-value-warning';
        return 'smart-value-good';
    }
}

function smartMetricRow(label, value, helpTip, valueClass) {
    return '<div style="display:flex;align-items:center;justify-content:space-between;padding:0.375rem 0;border-bottom:1px solid var(--border);">' +
        '<div style="display:flex;align-items:center;font-size:0.8125rem;color:var(--text-secondary);">' + label + helpIcon(helpTip) + '</div>' +
        '<div style="font-size:0.8125rem;font-weight:500;" class="' + (valueClass || '') + '">' + value + '</div></div>';
}

function showSMARTDetails(diskName) {
    document.getElementById('smartDiskName').textContent = diskName;
    document.getElementById('smartContent').innerHTML = '<div style="text-align:center;color:var(--text-muted);padding:2rem;">' + t.loading + '...</div>';
    document.getElementById('smartModal').classList.remove('hidden');

    fetch('/api/admin/storage/disk/' + diskName + '/smart')
    .then(function(r) { return r.json(); })
    .then(function(data) {
        if (!data.available) {
            document.getElementById('smartContent').innerHTML = '<div style="text-align:center;color:var(--text-muted);padding:2rem;">' + t.smartNotAvailable + '</div>';
            return;
        }
        var html = '';
        var healthClass = data.healthy ? 'background:rgba(16,185,129,0.1);border:1px solid var(--success);color:var(--success);' : 'background:rgba(239,68,68,0.1);border:1px solid var(--error);color:var(--error);';
        var healthText = data.healthy ? t.smartStatusGood : t.smartStatusCritical;
        html += '<div style="display:flex;align-items:center;gap:0.5rem;padding:0.75rem;border-radius:8px;margin-bottom:1rem;' + healthClass + '">';
        html += '<span style="font-weight:500;">' + t.smartHealth + ': ' + healthText + '</span>' + helpIcon(t.smartHelpHealth) + '</div>';

        html += '<div class="smart-card"><div class="smart-card-title">' + t.smartGeneralInfo + '</div>';
        var tempClass = data.temperature > 60 ? 'smart-value-critical' : (data.temperature > 50 ? 'smart-value-warning' : 'smart-value-good');
        html += smartMetricRow(t.smartTemp, data.temperature > 0 ? data.temperature + '°C' : '-', t.smartHelpTemp, tempClass);
        var powerHoursClass = data.power_on_hours > 50000 ? 'smart-value-warning' : '';
        html += smartMetricRow(t.smartPowerOn, data.power_on_hours > 0 ? formatHours(data.power_on_hours) : '-', t.smartHelpPowerOn, powerHoursClass);
        html += smartMetricRow(t.smartPowerCycles, data.power_cycle_count > 0 ? data.power_cycle_count.toLocaleString() : '-', t.smartHelpPowerCycles, '');
        html += '</div>';

        if (data.is_nvme) {
            html += '<div class="smart-card"><div class="smart-card-title">' + t.smartNvmeInfo + '</div>';
            html += smartMetricRow(t.smartMediaErrors, data.media_errors, t.smartHelpMediaErrors, data.media_errors > 0 ? 'smart-value-critical' : 'smart-value-good');
            html += smartMetricRow(t.smartUnsafeShutdowns, data.unsafe_shutdowns, t.smartHelpUnsafeShutdowns, data.unsafe_shutdowns > 100 ? 'smart-value-warning' : '');
            html += '</div>';
            html += '<div class="smart-card"><div class="smart-card-title">' + t.smartSsdWear + '</div>';
            html += smartMetricRow(t.smartAvailableSpare, data.available_spare + '%', t.smartHelpAvailableSpare, getValueClass(data.available_spare, 20, 10, true));
            html += smartMetricRow(t.smartPercentageUsed, data.percentage_used + '%', t.smartHelpPercentageUsed, getValueClass(data.percentage_used, 80, 100, false));
            html += '</div>';
            if (data.data_units_written > 0 || data.data_units_read > 0) {
                html += '<div class="smart-card"><div class="smart-card-title">' + t.smartDataVolume + '</div>';
                html += smartMetricRow(t.smartDataWritten, formatBytes(data.data_units_written), t.smartHelpDataWritten, '');
                html += smartMetricRow(t.smartDataRead, formatBytes(data.data_units_read), t.smartHelpDataRead, '');
                html += '</div>';
            }
        } else {
            var hasErrors = data.reallocated_sectors > 0 || data.pending_sectors > 0 || data.uncorrectable_sectors > 0;
            html += '<div class="smart-card" style="' + (hasErrors ? 'border:1px solid var(--error);' : '') + '"><div class="smart-card-title">' + t.smartDiskErrors + '</div>';
            html += smartMetricRow(t.smartReallocated, data.reallocated_sectors, t.smartHelpReallocated, getValueClass(data.reallocated_sectors, 10, 100, false));
            html += smartMetricRow(t.smartPending, data.pending_sectors, t.smartHelpPending, data.pending_sectors > 0 ? 'smart-value-critical' : 'smart-value-good');
            html += smartMetricRow(t.smartUncorrectable, data.uncorrectable_sectors, t.smartHelpUncorrectable, data.uncorrectable_sectors > 0 ? 'smart-value-critical' : 'smart-value-good');
            html += '</div>';
        }

        if (data.attributes && data.attributes.length > 0) {
            html += '<details class="smart-card"><summary class="smart-card-title" style="cursor:pointer;">' + t.smartAllAttributes + ' (' + data.attributes.length + ')</summary>';
            html += '<div style="overflow-x:auto;margin-top:0.5rem;max-height:12rem;overflow-y:auto;"><table class="v2-table" style="font-size:0.75rem;"><thead><tr>';
            html += '<th>ID</th><th>' + t.smartAttrName + '</th><th style="text-align:right;">' + t.smartAttrValue + '</th><th style="text-align:right;">' + t.smartAttrWorst + '</th><th>' + t.smartAttrRaw + '</th></tr></thead><tbody>';
            data.attributes.forEach(function(attr) {
                var rowStyle = attr.status === 'failing' ? 'background:rgba(239,68,68,0.1);' : (attr.status === 'warning' ? 'background:rgba(245,158,11,0.1);' : '');
                html += '<tr style="' + rowStyle + '"><td>' + attr.id + '</td><td>' + attr.name + '</td><td style="text-align:right;">' + attr.value + '</td><td style="text-align:right;">' + attr.worst + '</td><td style="font-family:monospace;">' + attr.raw_value + '</td></tr>';
            });
            html += '</tbody></table></div></details>';
        }
        document.getElementById('smartContent').innerHTML = html;
    }).catch(function(err) {
        document.getElementById('smartContent').innerHTML = '<div style="text-align:center;color:var(--error);padding:2rem;">' + t.error + ': ' + err + '</div>';
    });
}

function closeSMARTModal() { document.getElementById('smartModal').classList.add('hidden'); }

/* Pool operations */
function startScrub(poolName) {
    if (!confirm(t.confirmScrub)) return;
    fetch('/api/admin/storage/pool/' + poolName + '/scrub', {method: 'POST'})
    .then(function(r) { return r.json(); })
    .then(function(data) {
        if (data.success) { alert(t.scrubStarted); location.reload(); }
        else alert(t.scrubError + ': ' + (data.error || 'Unknown'));
    }).catch(function(err) { alert(t.scrubError + ': ' + err); });
}

function showCreatePoolModal() {
    loadAvailableDisks();
    document.getElementById('createPoolModal').classList.remove('hidden');
}

function closeCreatePoolModal() { document.getElementById('createPoolModal').classList.add('hidden'); }

function loadAvailableDisks() {
    fetch('/api/admin/storage/disks/available')
    .then(function(resp) { return resp.json(); })
    .then(function(disks) {
        var html = '';
        disks.forEach(function(disk) {
            var disabled = disk.in_use ? 'disabled' : '';
            var label = disk.in_use ? disk.path + ' (' + disk.size_human + ') - ' + disk.in_use_by : disk.path + ' (' + disk.size_human + ')';
            html += '<label style="display:flex;align-items:center;gap:0.5rem;padding:0.25rem 0;font-size:0.8125rem;color:' + (disk.in_use ? 'var(--text-muted)' : 'var(--text-primary)') + ';"><input type="checkbox" name="poolDisks" value="' + disk.path + '" ' + disabled + '>' + label + '</label>';
        });
        document.getElementById('poolDisksList').innerHTML = html || '<div style="text-align:center;color:var(--text-muted);padding:1rem;font-size:0.8125rem;">' + t.noAvailableDisks + '</div>';
    })
    .catch(function(err) {
        document.getElementById('poolDisksList').innerHTML = '<div style="text-align:center;color:var(--error);padding:1rem;font-size:0.8125rem;">' + t.error + '</div>';
    });
}

function createPool(e) {
    e.preventDefault();
    var name = document.getElementById('poolName').value;
    var vdevType = document.getElementById('poolVdevType').value;
    var compression = document.getElementById('poolCompression').value;
    var mountpoint = document.getElementById('poolMountpoint').value;
    var disks = Array.from(document.querySelectorAll('input[name="poolDisks"]:checked')).map(function(el) { return el.value; });
    if (disks.length === 0) { alert(t.selectAtLeastOneDisk); return; }
    closeCreatePoolModal();
    requirePassword(t.createPool, t.createPoolWarning, function() {
        fetch('/api/admin/storage/pool', {
            method: 'POST',
            headers: {'Content-Type': 'application/json', 'X-Verification-Token': verificationToken},
            body: JSON.stringify({name: name, vdev_type: vdevType, disks: disks, compression: compression, mountpoint: mountpoint, force: true})
        })
        .then(function(resp) { return resp.json(); })
        .then(function(data) {
            if (data.success) { alert(t.poolCreated); location.reload(); }
            else alert(t.error + ': ' + data.error);
        })
        .catch(function(err) { alert(t.error + ': ' + err); });
    });
}

function showDestroyPoolModal(poolName) {
    requirePassword(t.destroyPool + ': ' + poolName, t.destroyPoolWarning, function() {
        fetch('/api/admin/storage/pool-destroy/' + poolName + '?force=true', {
            method: 'DELETE',
            headers: {'X-Verification-Token': verificationToken}
        })
        .then(function(resp) { return resp.json(); })
        .then(function(data) {
            if (data.success) { alert(t.poolDestroyed); location.reload(); }
            else alert(t.error + ': ' + data.error);
        })
        .catch(function(err) { alert(t.error + ': ' + err); });
    });
}

function showExportPoolModal(poolName) {
    requirePassword(t.exportPool + ': ' + poolName, t.exportPoolWarning, function() {
        fetch('/api/admin/storage/pool-export/' + poolName, {
            method: 'POST',
            headers: {'X-Verification-Token': verificationToken}
        })
        .then(function(resp) { return resp.json(); })
        .then(function(data) {
            if (data.success) { alert(t.poolExported); location.reload(); }
            else alert(t.error + ': ' + data.error);
        })
        .catch(function(err) { alert(t.error + ': ' + err); });
    });
}

function showImportPoolModal() {
    fetch('/api/admin/storage/pools/importable')
    .then(function(r) { return r.json(); })
    .then(function(pools) {
        if (!pools || pools.length === 0) { alert(t.noImportablePools); return; }
        var poolName = prompt(t.selectPoolToImport + ':\n' + pools.map(function(p) { return p.name; }).join('\n'));
        if (!poolName) return;
        requirePassword(t.importPool + ': ' + poolName, t.importPoolWarning, function() {
            fetch('/api/admin/storage/pools/import', {
                method: 'POST',
                headers: {'Content-Type': 'application/json', 'X-Verification-Token': verificationToken},
                body: JSON.stringify({name: poolName, force: true})
            })
            .then(function(resp) { return resp.json(); })
            .then(function(data) {
                if (data.success) { alert(t.poolImported); location.reload(); }
                else alert(t.error + ': ' + data.error);
            })
            .catch(function(err) { alert(t.error + ': ' + err); });
        });
    }).catch(function(err) { alert(t.error + ': ' + err); });
}

/* Dataset operations */
function loadDatasets() {
    if (poolNames.length === 0) {
        document.getElementById('datasets-list').innerHTML = '<div style="text-align:center;color:var(--text-muted);padding:2rem;">' + t.noPools + '</div>';
        return;
    }
    var html = '';
    var loaded = 0;
    poolNames.forEach(function(pool) {
        fetch('/api/admin/storage/datasets?parent=' + pool)
        .then(function(resp) { return resp.json(); })
        .then(function(datasets) {
            if (datasets && datasets.length > 0) {
                html += '<div style="margin-bottom:1.5rem;"><div style="font-weight:500;color:var(--text-primary);margin-bottom:0.5rem;">' + pool + '</div>';
                html += '<div style="overflow-x:auto;"><table class="v2-table"><thead><tr>';
                html += '<th>' + t.datasetHeader + '</th><th>' + t.typeHeader + '</th><th style="text-align:right;">' + t.usedHeader + '</th><th style="text-align:right;">' + t.availableHeader + '</th><th>' + t.compressionHeader + '</th><th>' + t.actionsHeader + '</th>';
                html += '</tr></thead><tbody>';
                datasets.forEach(function(ds) {
                    html += '<tr><td>' + ds.name + '</td><td>' + ds.type + '</td><td style="text-align:right;">' + ds.used_human + '</td><td style="text-align:right;">' + ds.avail_human + '</td><td>' + ds.compression + '</td>';
                    html += '<td><button data-action="deleteDataset" data-name="' + ds.name + '" class="v2-btn v2-btn-danger v2-btn-sm">' + t.deleteBtn + '</button></td></tr>';
                });
                html += '</tbody></table></div></div>';
            }
            loaded++;
            if (loaded === poolNames.length) {
                document.getElementById('datasets-list').innerHTML = html || '<div style="text-align:center;color:var(--text-muted);padding:2rem;">' + t.noDatasets + '</div>';
            }
        })
        .catch(function(err) {
            console.error('Error loading datasets for', pool, err);
            loaded++;
            if (loaded === poolNames.length) {
                document.getElementById('datasets-list').innerHTML = html || '<div style="text-align:center;color:var(--text-muted);padding:2rem;">' + t.noDatasets + '</div>';
            }
        });
    });
}

function showCreateDatasetModal() { document.getElementById('createDatasetModal').classList.remove('hidden'); }
function closeCreateDatasetModal() { document.getElementById('createDatasetModal').classList.add('hidden'); }

function createDataset(e) {
    e.preventDefault();
    var name = document.getElementById('datasetName').value;
    var compression = document.getElementById('datasetCompression').value;
    var quota = document.getElementById('datasetQuota').value;
    closeCreateDatasetModal();
    requirePassword(t.createDataset, t.createDatasetWarning, function() {
        fetch('/api/admin/storage/dataset', {
            method: 'POST',
            headers: {'Content-Type': 'application/json', 'X-Verification-Token': verificationToken},
            body: JSON.stringify({name: name, compression: compression, quota: quota})
        })
        .then(function(resp) { return resp.json(); })
        .then(function(data) {
            if (data.success) { alert(t.datasetCreated); loadDatasets(); }
            else alert(t.error + ': ' + data.error);
        })
        .catch(function(err) { alert(t.error + ': ' + err); });
    });
}

function deleteDataset(name) {
    requirePassword(t.deleteDataset + ': ' + name, t.deleteDatasetWarning, function() {
        fetch('/api/admin/storage/dataset-delete?name=' + encodeURIComponent(name), {
            method: 'DELETE',
            headers: {'X-Verification-Token': verificationToken}
        })
        .then(function(resp) { return resp.json(); })
        .then(function(data) {
            if (data.success) { alert(t.datasetDeleted); loadDatasets(); }
            else alert(t.error + ': ' + data.error);
        })
        .catch(function(err) { alert(t.error + ': ' + err); });
    });
}

/* Snapshot operations */
function loadSnapshots() {
    fetch('/api/admin/storage/snapshots')
    .then(function(resp) { return resp.json(); })
    .then(function(snapshots) {
        var html = '';
        if (snapshots && snapshots.length > 0) {
            html = '<div style="overflow-x:auto;"><table class="v2-table"><thead><tr>';
            html += '<th>' + t.snapshotHeader + '</th><th>' + t.datasetHeader + '</th><th style="text-align:right;">' + t.usedHeader + '</th><th>' + t.createdHeader + '</th><th>' + t.actionsHeader + '</th>';
            html += '</tr></thead><tbody>';
            snapshots.forEach(function(snap) {
                html += '<tr><td>' + snap.snap_name + '</td><td>' + snap.dataset + '</td><td style="text-align:right;">' + snap.used_human + '</td><td>' + snap.creation_str + '</td>';
                html += '<td><button data-action="rollbackSnapshot" data-name="' + snap.name + '" class="v2-btn v2-btn-secondary v2-btn-sm" style="margin-right:0.25rem;">' + t.rollbackBtn + '</button>';
                html += '<button data-action="deleteSnapshot" data-name="' + snap.name + '" class="v2-btn v2-btn-danger v2-btn-sm">' + t.deleteBtn + '</button></td></tr>';
            });
            html += '</tbody></table></div>';
        } else {
            html = '<div style="text-align:center;color:var(--text-muted);padding:2rem;">' + t.noSnapshots + '</div>';
        }
        document.getElementById('snapshots-list').innerHTML = html;
        /* Populate snapshot dataset dropdown */
        var select = document.getElementById('snapshotDataset');
        if (select) {
            select.innerHTML = '';
            var loadedCount = 0;
            poolNames.forEach(function(pool) {
                fetch('/api/admin/storage/datasets?parent=' + pool)
                .then(function(dsResp) { return dsResp.json(); })
                .then(function(datasets) {
                    datasets.forEach(function(ds) {
                        var opt = document.createElement('option');
                        opt.value = ds.name;
                        opt.textContent = ds.name;
                        select.appendChild(opt);
                    });
                    loadedCount++;
                })
                .catch(function(err) {
                    console.error(err);
                    loadedCount++;
                });
            });
        }
    })
    .catch(function(err) {
        document.getElementById('snapshots-list').innerHTML = '<div style="text-align:center;color:var(--error);padding:2rem;">' + t.error + '</div>';
    });
}

function showCreateSnapshotModal() { document.getElementById('createSnapshotModal').classList.remove('hidden'); }
function closeCreateSnapshotModal() { document.getElementById('createSnapshotModal').classList.add('hidden'); }

function createSnapshot(e) {
    e.preventDefault();
    var dataset = document.getElementById('snapshotDataset').value;
    var name = document.getElementById('snapshotName').value;
    fetch('/api/admin/storage/snapshot', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({dataset: dataset, name: name})
    })
    .then(function(resp) { return resp.json(); })
    .then(function(data) {
        if (data.success) { alert(t.snapshotCreated); closeCreateSnapshotModal(); loadSnapshots(); }
        else alert(t.error + ': ' + data.error);
    })
    .catch(function(err) { alert(t.error + ': ' + err); });
}

function deleteSnapshot(name) {
    requirePassword(t.deleteSnapshot + ': ' + name, t.deleteSnapshotWarning, function() {
        fetch('/api/admin/storage/snapshot-delete?name=' + encodeURIComponent(name), {
            method: 'DELETE',
            headers: {'X-Verification-Token': verificationToken}
        })
        .then(function(resp) { return resp.json(); })
        .then(function(data) {
            if (data.success) { alert(t.snapshotDeleted); loadSnapshots(); }
            else alert(t.error + ': ' + data.error);
        })
        .catch(function(err) { alert(t.error + ': ' + err); });
    });
}

function rollbackSnapshot(name) {
    requirePassword(t.rollbackSnapshot + ': ' + name, t.rollbackWarning, function() {
        fetch('/api/admin/storage/snapshot-rollback', {
            method: 'POST',
            headers: {'Content-Type': 'application/json', 'X-Verification-Token': verificationToken},
            body: JSON.stringify({snapshot: name, destroy_recent: true})
        })
        .then(function(resp) { return resp.json(); })
        .then(function(data) {
            if (data.success) { alert(t.rollbackSuccess); location.reload(); }
            else alert(t.error + ': ' + data.error);
        })
        .catch(function(err) { alert(t.error + ': ' + err); });
    });
}

/* Disk format operations */
function showFormatDiskModal(device) {
    document.getElementById('formatDiskDevice').value = device;
    document.getElementById('formatDiskDeviceDisplay').textContent = device;
    var diskName = device.replace('/dev/', '');
    document.getElementById('formatMountPath').value = '/mnt/' + diskName;
    document.getElementById('formatMount').checked = true;
    document.getElementById('formatSharedAccess').checked = true;
    document.getElementById('formatPersistent').checked = true;
    document.getElementById('mountPathContainer').style.display = 'block';
    document.getElementById('formatSharedAccessContainer').style.display = 'block';
    document.getElementById('formatPersistentContainer').style.display = 'block';
    document.getElementById('formatDiskModal').classList.remove('hidden');
}

function closeFormatDiskModal() { document.getElementById('formatDiskModal').classList.add('hidden'); }

function toggleMountPath() {
    var checked = document.getElementById('formatMount').checked;
    document.getElementById('mountPathContainer').style.display = checked ? 'block' : 'none';
    document.getElementById('formatSharedAccessContainer').style.display = checked ? 'block' : 'none';
    document.getElementById('formatPersistentContainer').style.display = checked ? 'block' : 'none';
}

function formatDisk(e) {
    e.preventDefault();
    var device = document.getElementById('formatDiskDevice').value;
    var filesystem = document.getElementById('formatFilesystem').value;
    var label = document.getElementById('formatLabel').value;
    var mount = document.getElementById('formatMount').checked;
    var mountPath = mount ? document.getElementById('formatMountPath').value : '';
    var sharedAccess = mount ? document.getElementById('formatSharedAccess').checked : false;
    var persistent = mount ? document.getElementById('formatPersistent').checked : false;
    if (mount && mountPath) {
        if (!mountPath.startsWith('/mnt/') && !mountPath.startsWith('/media/')) {
            alert(t.mountPathError);
            return;
        }
    }
    closeFormatDiskModal();
    requirePassword(t.formatDisk + ': ' + device, t.formatDiskWarning, function() {
        fetch('/api/admin/storage/disk/format', {
            method: 'POST',
            headers: {'Content-Type': 'application/json', 'X-Verification-Token': verificationToken},
            body: JSON.stringify({device: device, filesystem: filesystem, label: label, force: true, mount: mount, mount_path: mountPath, shared_access: sharedAccess, persistent: persistent})
        })
        .then(function(resp) { return resp.json(); })
        .then(function(data) {
            if (data.success) { alert(t.diskFormatted); location.reload(); }
            else alert(t.error + ': ' + data.error);
        })
        .catch(function(err) { alert(t.error + ': ' + err); });
    });
}

/* Mount disk operations */
function showMountDiskModal(device, diskName) {
    document.getElementById('mountDiskDevice').value = device;
    document.getElementById('mountDiskDeviceDisplay').textContent = device;
    document.getElementById('mountDiskPath').value = '/mnt/' + diskName;
    document.getElementById('mountSharedAccess').checked = true;
    document.getElementById('mountPersistent').checked = false;
    document.getElementById('mountDiskModal').classList.remove('hidden');
}

function closeMountDiskModal() { document.getElementById('mountDiskModal').classList.add('hidden'); }

function mountDisk(e) {
    e.preventDefault();
    var device = document.getElementById('mountDiskDevice').value;
    var mountPath = document.getElementById('mountDiskPath').value;
    var sharedAccess = document.getElementById('mountSharedAccess').checked;
    var persistent = document.getElementById('mountPersistent').checked;
    if (!mountPath.startsWith('/mnt/') && !mountPath.startsWith('/media/')) {
        alert(t.mountPathError);
        return;
    }
    fetch('/api/admin/storage/disk/mount', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({device: device, mount_path: mountPath, shared_access: sharedAccess, persistent: persistent})
    })
    .then(function(resp) { return resp.json(); })
    .then(function(data) {
        if (data.success) { alert(t.diskMounted); closeMountDiskModal(); location.reload(); }
        else alert(t.error + ': ' + data.error);
    })
    .catch(function(err) { alert(t.error + ': ' + err); });
}

/* Unmount / Eject */
function unmountDisk(mountPath) {
    if (!confirm(t.unmountConfirm + ': ' + mountPath + '?')) return;
    fetch('/api/admin/storage/disk/unmount', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({mount_path: mountPath, eject: false})
    })
    .then(function(resp) { return resp.json(); })
    .then(function(data) {
        if (data.success) { alert(t.diskUnmounted); location.reload(); }
        else alert(t.error + ': ' + data.error);
    })
    .catch(function(err) { alert(t.error + ': ' + err); });
}

function ejectDisk(mountPath) {
    if (!confirm(t.ejectConfirm + ': ' + mountPath + '?')) return;
    fetch('/api/admin/storage/disk/unmount', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({mount_path: mountPath, eject: true})
    })
    .then(function(resp) { return resp.json(); })
    .then(function(data) {
        if (data.success) { alert(t.diskEjected); location.reload(); }
        else alert(t.error + ': ' + data.error);
    })
    .catch(function(err) { alert(t.error + ': ' + err); });
}

/* Event delegation */
document.addEventListener('click', function(e) {
    var target = e.target.closest('[data-action]');
    if (!target) return;
    var action = target.getAttribute('data-action');

    switch (action) {
        case 'showTab': showTab(target.getAttribute('data-tab')); break;
        case 'refresh': location.reload(); break;
        case 'showSMARTDetails': showSMARTDetails(target.getAttribute('data-disk')); break;
        case 'unmountDisk': unmountDisk(target.getAttribute('data-mountpoint')); break;
        case 'ejectDisk': ejectDisk(target.getAttribute('data-mountpoint')); break;
        case 'showMountDiskModal': showMountDiskModal(target.getAttribute('data-path'), target.getAttribute('data-disk')); break;
        case 'showFormatDiskModal': showFormatDiskModal(target.getAttribute('data-path')); break;
        case 'showImportPoolModal': showImportPoolModal(); break;
        case 'showCreatePoolModal': showCreatePoolModal(); break;
        case 'startScrub': startScrub(target.getAttribute('data-pool')); break;
        case 'showExportPoolModal': showExportPoolModal(target.getAttribute('data-pool')); break;
        case 'showDestroyPoolModal': showDestroyPoolModal(target.getAttribute('data-pool')); break;
        case 'showCreateDatasetModal': showCreateDatasetModal(); break;
        case 'showCreateSnapshotModal': showCreateSnapshotModal(); break;
        case 'closePasswordModal': closePasswordModal(); break;
        case 'closeSMARTModal': closeSMARTModal(); break;
        case 'closeCreatePoolModal': closeCreatePoolModal(); break;
        case 'closeCreateDatasetModal': closeCreateDatasetModal(); break;
        case 'closeCreateSnapshotModal': closeCreateSnapshotModal(); break;
        case 'closeFormatDiskModal': closeFormatDiskModal(); break;
        case 'closeMountDiskModal': closeMountDiskModal(); break;
        case 'deleteDataset': deleteDataset(target.getAttribute('data-name')); break;
        case 'rollbackSnapshot': rollbackSnapshot(target.getAttribute('data-name')); break;
        case 'deleteSnapshot': deleteSnapshot(target.getAttribute('data-name')); break;
    }
});

/* Form submissions */
document.getElementById('passwordForm').addEventListener('submit', function(e) { submitPasswordVerification(e); });
document.getElementById('createPoolForm').addEventListener('submit', function(e) { createPool(e); });
document.getElementById('createDatasetForm').addEventListener('submit', function(e) { createDataset(e); });
document.getElementById('createSnapshotForm').addEventListener('submit', function(e) { createSnapshot(e); });
document.getElementById('formatDiskForm').addEventListener('submit', function(e) { formatDisk(e); });
document.getElementById('mountDiskForm').addEventListener('submit', function(e) { mountDisk(e); });

/* Format mount checkbox */
document.getElementById('formatMount').addEventListener('change', function() { toggleMountPath(); });

/* Close modals on escape */
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
        closeSMARTModal();
        closePasswordModal();
        closeCreatePoolModal();
        closeCreateDatasetModal();
        closeCreateSnapshotModal();
        closeFormatDiskModal();
        closeMountDiskModal();
    }
});
