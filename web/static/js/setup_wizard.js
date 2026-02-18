/* Anemone setup wizard */
(function() {
    const pageData = JSON.parse(document.getElementById('page-data').textContent || '{}');
    const t = pageData.translations || {};
    const finalized = pageData.finalized || false;

    // State
    let currentStep = 0;
    let selectedMode = null;
    let selectedStorage = null;
    let selectedIncoming = 'same';
    let storageOptions = [];
    let availableDisks = [];
    let setupResult = null;
    let zfsConfigConfirmed = false;
    let restoreFile = null;
    let restoreResult = null;

    // Initialize
    document.addEventListener('DOMContentLoaded', function() {
        if (finalized) {
            currentStep = 6;
        } else {
            loadStorageOptions();
        }
    });

    // =============================================
    // MODE SELECTION
    // =============================================

    function selectMode(mode) {
        selectedMode = mode;
        document.querySelectorAll('.mode-radio').forEach(el => {
            el.classList.remove('selected');
            el.querySelector('div').classList.add('hidden');
            el.closest('.option-card').classList.remove('selected');
        });
        const radio = document.querySelector(`.mode-radio[data-mode="${mode}"]`);
        radio.classList.add('selected');
        radio.querySelector('div').classList.remove('hidden');
        radio.closest('.option-card').classList.add('selected');
        document.getElementById('btn-next-0').disabled = false;
    }

    // =============================================
    // STORAGE SELECTION
    // =============================================

    function selectStorage(type, path) {
        selectedStorage = { type, path };
        document.querySelectorAll('.storage-option').forEach(el => {
            el.classList.remove('selected');
        });
        document.querySelector(`.storage-option[data-type="${type}"]`)?.classList.add('selected');

        // Show/hide additional options
        document.getElementById('zfs-step-disks').classList.add('hidden');
        document.getElementById('zfs-step-raid').classList.add('hidden');
        document.getElementById('custom-path-options').classList.add('hidden');

        if (type === 'zfs_new') {
            document.getElementById('zfs-step-disks').classList.remove('hidden');
            loadAvailableDisks();
        } else if (type === 'custom') {
            document.getElementById('custom-path-options').classList.remove('hidden');
        }

        updateStorageNextButton();
    }

    // =============================================
    // ZFS SUB-STEP NAVIGATION
    // =============================================

    function showZfsRaidConfig() {
        // Get selected disks
        const selectedDisks = Array.from(document.querySelectorAll('.disk-card.selected:not(.disabled)'));
        if (selectedDisks.length === 0) return;

        // Update selected disks summary
        const summary = document.getElementById('selected-disks-summary');
        summary.innerHTML = selectedDisks.map(el => {
            const path = el.dataset.path;
            const info = el.querySelector('.text-sm.text-gray-500')?.textContent || '';
            return `<div class="flex justify-between"><span>${path}</span><span class="text-gray-400">${info}</span></div>`;
        }).join('');

        // Update RAID options based on disk count
        updateRaidOptions(selectedDisks.length);

        // Switch views
        document.getElementById('zfs-step-disks').classList.add('hidden');
        document.getElementById('zfs-step-raid').classList.remove('hidden');
    }

    function showZfsDiskSelection() {
        document.getElementById('zfs-step-raid').classList.add('hidden');
        document.getElementById('zfs-step-disks').classList.remove('hidden');
        zfsConfigConfirmed = false;
        updateStorageNextButton();
    }

    function confirmZfsConfig() {
        zfsConfigConfirmed = true;
        updateStorageNextButton();
        // Auto-advance to next step
        nextStep();
    }

    // =============================================
    // INCOMING SELECTION
    // =============================================

    function selectIncoming(type) {
        selectedIncoming = type;
        document.querySelectorAll('.incoming-radio').forEach(el => {
            el.classList.remove('selected');
            el.querySelector('div').classList.add('hidden');
            el.closest('.option-card').classList.remove('selected');
        });
        const radio = document.querySelector(`.incoming-radio[data-incoming="${type}"]`);
        radio.classList.add('selected');
        radio.querySelector('div').classList.remove('hidden');
        radio.closest('.option-card').classList.add('selected');

        if (type === 'separate') {
            document.getElementById('incoming-separate-options').classList.remove('hidden');
        } else {
            document.getElementById('incoming-separate-options').classList.add('hidden');
        }
    }

    // =============================================
    // DISK SELECTION FOR ZFS
    // =============================================

    function toggleDisk(path) {
        const diskCard = document.querySelector(`.disk-card[data-path="${path}"]`);
        if (diskCard.classList.contains('disabled')) return;

        diskCard.classList.toggle('selected');
        // Sync checkbox state with card selection
        const checkbox = diskCard.querySelector('input[type="checkbox"]');
        if (checkbox) {
            checkbox.checked = diskCard.classList.contains('selected');
        }
        updateStorageNextButton();
    }

    function updateStorageNextButton() {
        let enabled = false;
        let zfsDiskButtonEnabled = false;

        if (selectedStorage) {
            if (selectedStorage.type === 'zfs_new') {
                const selectedDisks = document.querySelectorAll('.disk-card.selected:not(.disabled)');
                const hasPoolName = document.getElementById('zfs-pool-name').value.trim() !== '';
                const mountpoint = document.getElementById('zfs-mountpoint').value.trim();
                const mountpointValid = mountpoint === '' || /^\/[a-zA-Z0-9/_\-]+$/.test(mountpoint);

                // Enable "Configure RAID" button when disks are selected, pool name is set, and mountpoint is valid
                zfsDiskButtonEnabled = selectedDisks.length > 0 && hasPoolName && mountpointValid;

                // Enable main "Next" button only when ZFS config is confirmed
                enabled = zfsConfigConfirmed;

                // Update disk count display
                const diskCountEl = document.getElementById('disk-count');
                if (selectedDisks.length > 0) {
                    diskCountEl.textContent = selectedDisks.length + ' ' + t.disks_selected;
                } else {
                    diskCountEl.textContent = '';
                }
            } else if (selectedStorage.type === 'custom') {
                enabled = document.getElementById('custom-data-dir').value.trim() !== '';
            } else {
                enabled = true;
            }
        }

        document.getElementById('btn-next-1').disabled = !enabled;

        // Update ZFS disk selection button
        const zfsBtn = document.getElementById('btn-zfs-to-raid');
        if (zfsBtn) {
            zfsBtn.disabled = !zfsDiskButtonEnabled;
        }
    }

    // Update RAID level options based on number of selected disks
    function updateRaidOptions(diskCount) {
        const select = document.getElementById('zfs-raid-level');
        const diskCountEl = document.getElementById('disk-count');
        const currentValue = select.value;

        // Update disk count display
        if (diskCount > 0) {
            diskCountEl.textContent = diskCount + ' ' + t.disks_selected;
        } else {
            diskCountEl.textContent = '';
        }

        // Define RAID options with their minimum disk requirements
        const raidOptions = [
            { value: 'single', minDisks: 1, label: t.raid_single },
            { value: 'mirror', minDisks: 2, label: t.raid_mirror },
            { value: 'raidz', minDisks: 3, label: t.raid_raidz1 },
            { value: 'raidz2', minDisks: 4, label: t.raid_raidz2 }
        ];

        // Clear and rebuild options
        select.innerHTML = '';
        let firstValidValue = null;
        let currentValueStillValid = false;

        raidOptions.forEach(opt => {
            if (diskCount >= opt.minDisks) {
                const option = document.createElement('option');
                option.value = opt.value;
                option.textContent = opt.label;
                select.appendChild(option);

                if (!firstValidValue) firstValidValue = opt.value;
                if (opt.value === currentValue) currentValueStillValid = true;
            }
        });

        // Restore previous selection if still valid, otherwise use first valid option
        if (currentValueStillValid) {
            select.value = currentValue;
        } else if (firstValidValue) {
            select.value = firstValidValue;
        }
    }

    // =============================================
    // LOAD STORAGE OPTIONS FROM API
    // =============================================

    async function loadStorageOptions() {
        try {
            const resp = await fetch('/setup/wizard/storage/options');
            storageOptions = await resp.json();
            renderStorageOptions();
        } catch (err) {
            console.error('Error loading storage options:', err);
            document.getElementById('storage-options').innerHTML = `<p class="text-red-500 text-center py-4">${t.error}</p>`;
        }
    }

    function renderStorageOptions() {
        const container = document.getElementById('storage-options');
        let html = '';

        storageOptions.forEach(opt => {
            const icon = getStorageIcon(opt.type);
            html += `
                <div class="option-card storage-option border-2 border-gray-200 rounded-lg p-6 cursor-pointer" data-type="${opt.type}" data-action="selectStorage" data-storage-type="${opt.type}" data-storage-path="${opt.path || ''}">
                    <div class="flex items-start space-x-4">
                        <div class="flex-shrink-0 w-12 h-12 ${icon.bg} rounded-lg flex items-center justify-center">
                            ${icon.svg}
                        </div>
                        <div class="flex-1">
                            <h3 class="text-lg font-semibold text-gray-900">${opt.name}</h3>
                            <p class="text-gray-600 mt-1">${opt.description}</p>
                            ${opt.path ? `<p class="text-sm text-indigo-600 mt-1 font-mono">${opt.path}</p>` : ''}
                        </div>
                    </div>
                </div>
            `;
        });

        container.innerHTML = html;
    }

    function getStorageIcon(type) {
        switch(type) {
            case 'default':
                return { bg: 'bg-green-100', svg: '<svg class="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>' };
            case 'zfs_new':
                return { bg: 'bg-purple-100', svg: '<svg class="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/></svg>' };
            case 'custom':
                return { bg: 'bg-gray-100', svg: '<svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>' };
            default:
                return { bg: 'bg-gray-100', svg: '<svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7"/></svg>' };
        }
    }

    // =============================================
    // LOAD AVAILABLE DISKS FOR ZFS
    // =============================================

    async function loadAvailableDisks() {
        try {
            const resp = await fetch('/setup/wizard/storage/disks');
            availableDisks = await resp.json();
            renderAvailableDisks();
        } catch (err) {
            console.error('Error loading disks:', err);
            document.getElementById('available-disks').innerHTML = `<p class="text-red-500">${t.error}</p>`;
        }
    }

    function renderAvailableDisks() {
        const container = document.getElementById('available-disks');
        if (!availableDisks || availableDisks.length === 0) {
            container.innerHTML = '<p class="text-gray-500 text-center py-2">' + t.no_available_disks + '</p>';
            return;
        }

        let html = '';
        availableDisks.forEach(disk => {
            const disabled = disk.in_use ? 'disabled' : '';
            const disabledClass = disk.in_use ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer';
            html += `
                <div class="disk-card border rounded-lg p-3 ${disabledClass} ${disabled}" data-path="${disk.path}" data-action="toggleDisk" data-disk-path="${disk.path}">
                    <div class="flex items-center justify-between">
                        <div class="flex items-center space-x-3">
                            <input type="checkbox" class="w-4 h-4 text-indigo-600 rounded" ${disabled}>
                            <div>
                                <div class="font-medium text-gray-900">${disk.path}</div>
                                <div class="text-sm text-gray-500">${disk.model || disk.name} - ${disk.size_human}</div>
                            </div>
                        </div>
                        <span class="px-2 py-1 text-xs rounded ${disk.type === 'nvme' ? 'bg-purple-100 text-purple-800' : disk.type === 'ssd' ? 'bg-blue-100 text-blue-800' : 'bg-gray-100 text-gray-800'}">${disk.type.toUpperCase()}</span>
                    </div>
                    ${disk.in_use ? '<p class="text-xs text-amber-600 mt-1">' + t.disk_in_use + '</p>' : ''}
                </div>
            `;
        });
        container.innerHTML = html;
    }

    // =============================================
    // STEP NAVIGATION
    // =============================================

    function nextStep() {
        // Handle restore mode from step 0
        if (currentStep === 0 && selectedMode === 'restore') {
            // Go to restore upload step
            hideAllSteps();
            document.getElementById('step-restore-upload').classList.remove('hidden');
            return;
        }

        if (currentStep === 0) {
            // Mode selected, go to storage
            currentStep = 1;
        } else if (currentStep === 1) {
            // Storage configured, go to incoming
            currentStep = 2;
        } else if (currentStep === 2) {
            // Incoming configured
            if (selectedMode === 'restore') {
                // For restore mode, go to restore confirm step
                hideAllSteps();
                document.getElementById('step-restore-confirm').classList.remove('hidden');
                return;
            }
            // Go to admin step
            currentStep = 3;
        } else if (currentStep === 3) {
            // Admin form - validate and go to summary
            if (!validateAdminForm()) return;
            updateSummary();
            currentStep = 4;
        }

        showStep(currentStep);
    }

    function prevStep() {
        if (currentStep > 0) {
            currentStep--;
            showStep(currentStep);
        }
    }

    function showStep(step) {
        // Hide all steps
        document.querySelectorAll('.step-content').forEach(el => el.classList.add('hidden'));
        // Show current step
        document.getElementById(`step-${step}`).classList.remove('hidden');

        // Restore sub-step visibility when returning to storage step
        if (step === 1 && selectedStorage) {
            if (selectedStorage.type === 'zfs_new') {
                if (zfsConfigConfirmed) {
                    document.getElementById('zfs-step-disks').classList.add('hidden');
                    document.getElementById('zfs-step-raid').classList.remove('hidden');
                } else {
                    document.getElementById('zfs-step-disks').classList.remove('hidden');
                    document.getElementById('zfs-step-raid').classList.add('hidden');
                }
            } else if (selectedStorage.type === 'custom') {
                document.getElementById('custom-path-options').classList.remove('hidden');
            }
        }

        // Update progress indicators
        for (let i = 0; i < 5; i++) {
            const indicator = document.getElementById(`step-indicator-${i}`);
            indicator.classList.remove('step-active', 'step-completed', 'step-pending');

            if (i < step) {
                indicator.classList.add('step-completed');
            } else if (i === step) {
                indicator.classList.add('step-active');
            } else {
                indicator.classList.add('step-pending');
            }

            // Update progress bars
            if (i < 4) {
                const progress = document.getElementById(`progress-${i}`);
                progress.style.width = i < step ? '100%' : '0%';
            }
        }
    }

    // =============================================
    // ADMIN FORM VALIDATION
    // =============================================

    function validateAdminForm() {
        const serverName = document.getElementById('server-name').value.trim();
        const username = document.getElementById('admin-username').value.trim();
        const password = document.getElementById('admin-password').value;
        const passwordConfirm = document.getElementById('admin-password-confirm').value;

        if (!serverName) {
            alert(t.server_name_required);
            return false;
        }

        if (!username || username.length < 3) {
            alert(t.username_required);
            return false;
        }

        if (password.length < 8) {
            alert(t.password_too_short);
            return false;
        }

        if (password !== passwordConfirm) {
            alert(t.passwords_mismatch);
            return false;
        }

        return true;
    }

    function updateSummary() {
        // Storage summary
        let storageSummary = '';
        if (selectedStorage) {
            switch (selectedStorage.type) {
                case 'default':
                    storageSummary = `${t.default_storage}: /srv/anemone`;
                    break;
                case 'zfs_new':
                    const poolName = document.getElementById('zfs-pool-name').value;
                    const raidLevel = document.getElementById('zfs-raid-level').value;
                    const selectedDisks = Array.from(document.querySelectorAll('.disk-card.selected:not(.disabled)')).map(el => el.dataset.path);
                    storageSummary = `${t.zfs_new}: ${poolName} (${raidLevel}) - ${selectedDisks.length} disque(s)`;
                    break;
                case 'custom':
                    storageSummary = `${t.custom_path}: ${document.getElementById('custom-data-dir').value}`;
                    break;
            }
        }
        document.getElementById('summary-storage').textContent = storageSummary;

        // Incoming summary
        let incomingSummary = selectedIncoming === 'same'
            ? t.same_storage
            : `${t.separate_path}: ${document.getElementById('incoming-path').value || '-'}`;
        document.getElementById('summary-incoming').textContent = incomingSummary;

        // Server name and admin summary
        const serverName = document.getElementById('server-name').value || '-';
        document.getElementById('summary-server-name').textContent = serverName;

        const username = document.getElementById('admin-username').value;
        const email = document.getElementById('admin-email').value;
        document.getElementById('summary-admin').textContent = `${t.admin_user}: ${username}${email ? ` (${email})` : ''}`;
    }

    // =============================================
    // ZFS CONFIRMATION MODAL
    // =============================================

    function showZfsConfirm() {
        const disks = Array.from(document.querySelectorAll('.disk-card.selected:not(.disabled)')).map(el => el.dataset.path);
        document.getElementById('zfs-confirm-disks').innerHTML = disks.map(d => `<div>${d}</div>`).join('');
        document.getElementById('zfs-confirm-modal').classList.remove('hidden');
    }

    function hideZfsConfirm() {
        document.getElementById('zfs-confirm-modal').classList.add('hidden');
    }

    function confirmZfsCreation() {
        hideZfsConfirm();
        doFinalizeSetup();
    }

    // =============================================
    // FINALIZE SETUP
    // =============================================

    async function finalizeSetup() {
        // If creating new ZFS pool, show confirmation first
        if (selectedStorage?.type === 'zfs_new') {
            showZfsConfirm();
            return;
        }
        doFinalizeSetup();
    }

    async function doFinalizeSetup() {
        showLoading(t.configuring_storage);

        try {
            // 1. Configure storage
            const storageConfig = buildStorageConfig();
            const storageResp = await fetch('/setup/wizard/storage/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(storageConfig)
            });

            if (!storageResp.ok) {
                const error = await storageResp.text();
                throw new Error(error);
            }

            const storageResult = await storageResp.json();

            // 2. Create admin account (skip if importing existing installation)
            if (!storageResult.skip_admin) {
                showLoading(t.creating_admin);
                const adminConfig = {
                    server_name: document.getElementById('server-name').value.trim(),
                    username: document.getElementById('admin-username').value.trim(),
                    password: document.getElementById('admin-password').value,
                    email: document.getElementById('admin-email').value.trim()
                };

                const adminResp = await fetch('/setup/wizard/admin', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(adminConfig)
                });

                if (!adminResp.ok) {
                    const error = await adminResp.text();
                    throw new Error(error);
                }

                const adminResult = await adminResp.json();
                setupResult = adminResult;
            } else {
                // Importing existing installation - no new credentials to show
                setupResult = { imported: true };
            }

            // 3. Finalize
            showLoading(t.finalizing);
            const finalResp = await fetch('/setup/wizard/finalize', {
                method: 'POST'
            });

            if (!finalResp.ok) {
                const error = await finalResp.text();
                throw new Error(error);
            }

            // Show success
            hideLoading();
            document.getElementById('encryption-key').textContent = setupResult.encryption_key || '-';
            document.getElementById('sync-password').textContent = setupResult.sync_password || '-';
            currentStep = 5;
            showStep(5);

        } catch (err) {
            hideLoading();
            alert(`${t.error}: ${err.message}`);
        }
    }

    function buildStorageConfig() {
        const config = {
            storage_type: selectedStorage?.type || 'default'
        };

        if (selectedStorage?.type === 'zfs_new') {
            config.zfs_pool_name = document.getElementById('zfs-pool-name').value;
            config.zfs_raid_level = document.getElementById('zfs-raid-level').value;
            config.zfs_mountpoint = document.getElementById('zfs-mountpoint').value;
            config.zfs_devices = Array.from(document.querySelectorAll('.disk-card.selected:not(.disabled)')).map(el => el.dataset.path);
        } else if (selectedStorage?.type === 'custom') {
            config.data_dir = document.getElementById('custom-data-dir').value;
        }

        // Incoming storage
        config.separate_incoming = selectedIncoming === 'separate';
        if (selectedIncoming === 'separate') {
            config.incoming_dir = document.getElementById('incoming-path').value;
        }

        return config;
    }

    // =============================================
    // LOADING OVERLAY
    // =============================================

    function showLoading(message) {
        document.getElementById('loading-message').textContent = message || t.loading;
        document.getElementById('loading-overlay').classList.remove('hidden');
    }

    function hideLoading() {
        document.getElementById('loading-overlay').classList.add('hidden');
    }

    // =============================================
    // SUCCESS PAGE HELPERS
    // =============================================

    function copyEncryptionKey() {
        const key = document.getElementById('encryption-key').textContent;
        navigator.clipboard.writeText(key);
    }

    function copySyncPassword() {
        const password = document.getElementById('sync-password').textContent;
        navigator.clipboard.writeText(password);
    }

    function downloadEncryptionKey() {
        const key = document.getElementById('encryption-key').textContent;
        const username = document.getElementById('admin-username').value;
        const blob = new Blob([`Anemone Encryption Key\n\nUser: ${username}\nKey: ${key}\n\nKeep this file safe!`], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `anemone-key-${username}.txt`;
        a.click();
        URL.revokeObjectURL(url);
    }

    function checkConfirmations() {
        const keySaved = document.getElementById('confirm-key-saved').checked;
        const understand = document.getElementById('confirm-understand').checked;
        const btn = document.getElementById('btn-goto-login');

        if (keySaved && understand) {
            btn.classList.remove('opacity-50', 'pointer-events-none');
            btn.disabled = false;
        } else {
            btn.classList.add('opacity-50', 'pointer-events-none');
            btn.disabled = true;
        }
    }

    function showRestartStep() {
        // Show step 6 with restart instructions (no polling - user clicks after restart)
        currentStep = 6;
        showStep(6);
    }

    function checkServerRestart() {
        fetch('/login', { method: 'HEAD' })
            .then(response => {
                if (response.ok || response.status === 200) {
                    window.location.href = '/login';
                } else {
                    setTimeout(checkServerRestart, 3000);
                }
            })
            .catch(() => {
                setTimeout(checkServerRestart, 3000);
            });
    }

    // =============================================
    // RESTORE FLOW FUNCTIONS
    // =============================================

    // Handle file selection
    function handleFileSelect(event) {
        const file = event.target.files[0];
        if (file) {
            restoreFile = file;
            document.getElementById('selected-file-name').textContent = file.name;
            document.getElementById('selected-file-name').classList.remove('hidden');
            updateRestoreValidateButton();
        }
    }

    // Enable/disable validate button
    function updateRestoreValidateButton() {
        const hasFile = restoreFile !== null;
        const hasPassphrase = document.getElementById('restore-passphrase').value.trim() !== '';
        document.getElementById('btn-validate-restore').disabled = !(hasFile && hasPassphrase);
    }

    // Validate and decrypt backup
    async function validateRestore() {
        const passphrase = document.getElementById('restore-passphrase').value.trim();
        if (!restoreFile || !passphrase) return;

        showLoading(t.validating_backup || 'Validating backup...');
        document.getElementById('restore-error').classList.add('hidden');

        try {
            const formData = new FormData();
            formData.append('backup', restoreFile);
            formData.append('passphrase', passphrase);

            const resp = await fetch('/setup/wizard/restore/validate', {
                method: 'POST',
                body: formData
            });

            const result = await resp.json();
            hideLoading();

            if (!result.valid) {
                // Show error
                document.getElementById('restore-error-text').textContent =
                    result.error === 'invalid_passphrase'
                        ? (t.invalid_passphrase || 'Invalid passphrase. Please check your passphrase and try again.')
                        : (result.error || 'Failed to validate backup');
                document.getElementById('restore-error').classList.remove('hidden');
                return;
            }

            // Store result and show confirm step
            restoreResult = result;
            showRestoreConfirm(result);

        } catch (err) {
            hideLoading();
            document.getElementById('restore-error-text').textContent = err.message;
            document.getElementById('restore-error').classList.remove('hidden');
        }
    }

    // Show restore confirm step with backup info
    function showRestoreConfirm(result) {
        // Populate backup info for later display
        document.getElementById('restore-server-name').textContent = result.server_name || '-';
        document.getElementById('restore-exported-at').textContent = result.exported_at || '-';
        document.getElementById('restore-users-count').textContent = result.users_count || 0;
        document.getElementById('restore-peers-count').textContent = result.peers_count || 0;

        // Populate users list
        const usersContainer = document.getElementById('restore-users-list');
        if (result.users && result.users.length > 0) {
            usersContainer.innerHTML = result.users.map(u => `
                <div class="flex items-center justify-between py-2 border-b border-gray-200 last:border-0">
                    <span class="font-medium">${u.username}</span>
                    <span class="text-sm text-gray-500">${u.is_admin ? 'Admin' : ''} ${u.email || ''}</span>
                </div>
            `).join('');
        } else {
            usersContainer.innerHTML = '<p class="text-gray-500">No users</p>';
        }

        // Populate peers list
        const peersContainer = document.getElementById('restore-peers-list');
        if (result.peers && result.peers.length > 0) {
            peersContainer.innerHTML = result.peers.map(p => `
                <div class="flex items-center justify-between py-2 border-b border-gray-200 last:border-0">
                    <span class="font-medium">${p.name}</span>
                    <span class="text-sm text-gray-500">${p.address}:${p.port}</span>
                </div>
            `).join('');
        } else {
            peersContainer.innerHTML = '<p class="text-gray-500">No peers</p>';
        }

        // Go to storage configuration step (same as new installation)
        // The user needs to choose where to store data before restoring
        currentStep = 1;
        showStep(currentStep);
        loadStorageOptions();
    }

    // Execute the restore
    async function executeRestore() {
        showLoading(t.configuring_storage || 'Configuring storage...');

        try {
            // Step 1: Configure storage (same as new installation)
            const storageConfig = buildStorageConfig();
            const storageResp = await fetch('/setup/wizard/storage/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(storageConfig)
            });

            if (!storageResp.ok) {
                const error = await storageResp.text();
                throw new Error('Storage configuration failed: ' + error);
            }

            const storageResult = await storageResp.json();

            // Step 2: Execute restore with configured paths
            showLoading(t.restoring || 'Restoring server configuration...');

            const restoreResp = await fetch('/setup/wizard/restore/execute', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    data_dir: storageResult.config?.data_dir || '/srv/anemone',
                    shares_dir: storageResult.config?.shares_dir || '/srv/anemone/shares',
                    incoming_dir: storageResult.config?.incoming_dir || '/srv/anemone/backups/incoming'
                })
            });

            if (!restoreResp.ok) {
                const error = await restoreResp.text();
                throw new Error(error);
            }

            const result = await restoreResp.json();
            hideLoading();

            // Show success
            document.getElementById('restore-success-users').textContent =
                `${result.users_count} ${t.users_restored || 'users restored'}`;
            document.getElementById('restore-success-peers').textContent =
                `${result.peers_count} ${t.peers_restored || 'peers restored'}`;

            hideAllSteps();
            document.getElementById('step-restore-success').classList.remove('hidden');

        } catch (err) {
            hideLoading();
            alert((t.restore_failed || 'Restore failed') + ': ' + err.message);
        }
    }

    // =============================================
    // NAVIGATION HELPERS FOR RESTORE FLOW
    // =============================================

    function showRestoreUpload() {
        hideAllSteps();
        document.getElementById('step-restore-upload').classList.remove('hidden');
    }

    function backToIncomingStep() {
        // Go back to step 2 (incoming storage configuration)
        currentStep = 2;
        showStep(currentStep);
    }

    function backToModeSelection() {
        selectedMode = null;
        currentStep = 0;
        restoreFile = null;
        restoreResult = null;
        document.getElementById('restore-file').value = '';
        document.getElementById('restore-passphrase').value = '';
        document.getElementById('selected-file-name').classList.add('hidden');
        document.getElementById('restore-error').classList.add('hidden');
        document.querySelectorAll('.mode-radio').forEach(el => {
            el.classList.remove('selected');
            el.querySelector('div').classList.add('hidden');
            el.closest('.option-card').classList.remove('selected');
        });
        document.getElementById('btn-next-0').disabled = true;
        showStep(0);
    }

    function hideAllSteps() {
        document.querySelectorAll('.step-content').forEach(el => el.classList.add('hidden'));
    }

    // =============================================
    // EVENT DELEGATION
    // =============================================

    // Click event delegation for all data-action elements
    document.addEventListener('click', function(e) {
        var target = e.target.closest('[data-action]');
        if (!target) return;
        var action = target.getAttribute('data-action');

        switch (action) {
            case 'selectMode':
                selectMode(target.getAttribute('data-mode'));
                break;
            case 'nextStep':
                nextStep();
                break;
            case 'prevStep':
                prevStep();
                break;
            case 'showZfsRaidConfig':
                showZfsRaidConfig();
                break;
            case 'showZfsDiskSelection':
                showZfsDiskSelection();
                break;
            case 'confirmZfsConfig':
                confirmZfsConfig();
                break;
            case 'selectIncoming':
                selectIncoming(target.getAttribute('data-incoming'));
                break;
            case 'selectStorage':
                selectStorage(target.getAttribute('data-storage-type'), target.getAttribute('data-storage-path'));
                break;
            case 'toggleDisk':
                e.preventDefault();
                toggleDisk(target.getAttribute('data-disk-path'));
                break;
            case 'copyEncryptionKey':
                copyEncryptionKey();
                break;
            case 'copySyncPassword':
                copySyncPassword();
                break;
            case 'downloadEncryptionKey':
                downloadEncryptionKey();
                break;
            case 'showRestartStep':
                showRestartStep();
                break;
            case 'finalizeSetup':
                finalizeSetup();
                break;
            case 'browseFile':
                document.getElementById('restore-file').click();
                break;
            case 'validateRestore':
                validateRestore();
                break;
            case 'executeRestore':
                executeRestore();
                break;
            case 'backToModeSelection':
                backToModeSelection();
                break;
            case 'backToIncomingStep':
                backToIncomingStep();
                break;
            case 'hideZfsConfirm':
                hideZfsConfirm();
                break;
            case 'confirmZfsCreation':
                confirmZfsCreation();
                break;
        }
    });

    // Change event delegation for checkboxes
    document.addEventListener('change', function(e) {
        if (e.target.id === 'confirm-key-saved' || e.target.id === 'confirm-understand') {
            checkConfirmations();
        }
        if (e.target.id === 'restore-file') {
            handleFileSelect(e);
        }
    });

    // Input event listeners for form fields
    const poolNameInput = document.getElementById('zfs-pool-name');
    if (poolNameInput) poolNameInput.addEventListener('input', updateStorageNextButton);

    const mountpointInput = document.getElementById('zfs-mountpoint');
    if (mountpointInput) mountpointInput.addEventListener('input', updateStorageNextButton);

    const customDataDirInput = document.getElementById('custom-data-dir');
    if (customDataDirInput) customDataDirInput.addEventListener('input', updateStorageNextButton);

    const restorePassphraseInput = document.getElementById('restore-passphrase');
    if (restorePassphraseInput) restorePassphraseInput.addEventListener('input', updateRestoreValidateButton);

    // Drag and drop support for file upload
    const dropZone = document.getElementById('drop-zone');
    if (dropZone) {
        dropZone.addEventListener('dragover', (e) => {
            e.preventDefault();
            dropZone.classList.add('border-indigo-400', 'bg-indigo-50');
        });
        dropZone.addEventListener('dragleave', (e) => {
            e.preventDefault();
            dropZone.classList.remove('border-indigo-400', 'bg-indigo-50');
        });
        dropZone.addEventListener('drop', (e) => {
            e.preventDefault();
            dropZone.classList.remove('border-indigo-400', 'bg-indigo-50');
            const files = e.dataTransfer.files;
            if (files.length > 0) {
                document.getElementById('restore-file').files = files;
                handleFileSelect({ target: { files: files } });
            }
        });
    }
})();
