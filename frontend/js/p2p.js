const P2P = {
    panel: null,
    currentTab: 'discover',

    init() {
        this.panel = document.getElementById('p2pPanel');
        this.setupEventListeners();
        this.loadLocalAddresses();
    },

    setupEventListeners() {
        document.getElementById('p2pBtn').addEventListener('click', () => this.togglePanel());
        document.getElementById('closeP2P').addEventListener('click', () => this.togglePanel());

        document.querySelectorAll('.p2p-tab').forEach(tab => {
            tab.addEventListener('click', (e) => this.switchTab(e.target.dataset.tab));
        });

        document.getElementById('discoverBtn').addEventListener('click', () => this.discoverDevices());
        document.getElementById('generateCodeBtn').addEventListener('click', () => this.generatePairingCode());
        document.getElementById('joinCodeBtn').addEventListener('click', () => this.joinWithCode());

        this.refreshTransfers();
        setInterval(() => this.refreshTransfers(), 5000);
    },

    togglePanel() {
        const isVisible = this.panel.style.display !== 'none';
        this.panel.style.display = isVisible ? 'none' : 'block';
        if (!isVisible) {
            this.refresh();
        }
    },

    switchTab(tab) {
        this.currentTab = tab;

        document.querySelectorAll('.p2p-tab').forEach(t => t.classList.remove('active'));
        document.querySelector(`[data-tab="${tab}"]`).classList.add('active');

        document.getElementById('p2pDiscover').style.display = tab === 'discover' ? 'block' : 'none';
        document.getElementById('p2pPairing').style.display = tab === 'pairing' ? 'block' : 'none';
        document.getElementById('p2pTransfers').style.display = tab === 'transfers' ? 'block' : 'none';
    },

    async loadLocalAddresses() {
        try {
            const response = await fetch('/api/p2p/addresses');
            const data = await response.json();
            this.displayAddresses(data.addresses || []);
        } catch (error) {
            console.error('Failed to load addresses:', error);
        }
    },

    displayAddresses(addresses) {
        const container = document.getElementById('localAddresses');
        if (!addresses || addresses.length === 0) {
            container.innerHTML = '<p class="text-muted">无可用地址</p>';
            return;
        }

        container.innerHTML = addresses.map(addr => {
            const type = addr.startsWith('ipv4:') ? 'IPv4' :
                        addr.startsWith('ipv6:') ? 'IPv6' :
                        addr.startsWith('public:') ? '公网' : '其他';
            const displayAddr = addr.replace(/^(ipv4:|ipv6:|public:)/, '').replace(/^\[|\]$/g, '');
            return `<div class="address-item">
                <span class="address-type">${type}</span>
                <span class="address-value">${displayAddr}</span>
            </div>`;
        }).join('');
    },

    async discoverDevices() {
        const btn = document.getElementById('discoverBtn');
        btn.disabled = true;
        btn.textContent = '扫描中...';

        const deviceList = document.getElementById('deviceList');
        deviceList.innerHTML = '<p class="text-muted">正在扫描...</p>';

        try {
            const response = await fetch('/api/p2p/discover');
            const data = await response.json();

            if (!data.devices || data.devices.length === 0) {
                deviceList.innerHTML = '<p class="text-muted">未发现设备，请确保设备在同一网络</p>';
            } else {
                deviceList.innerHTML = data.devices.map(device => `
                    <div class="device-item">
                        <div class="device-info">
                            <span class="device-name">设备 ${device.id.substring(0, 8)}</span>
                            <span class="device-address">${device.ip}:${device.port}</span>
                        </div>
                        <button class="btn btn-sm btn-primary" onclick="P2P.sendFileTo('${device.id}')">发送文件</button>
                    </div>
                `).join('');
            }
        } catch (error) {
            deviceList.innerHTML = '<p class="text-error">扫描失败</p>';
            console.error('Discover error:', error);
        } finally {
            btn.disabled = false;
            btn.textContent = '🔍 扫描设备';
        }
    },

    async generatePairingCode() {
        const codeContainer = document.getElementById('pairingCode');
        codeContainer.innerHTML = '<p class="text-muted">生成中...</p>';

        try {
            const response = await fetch('/api/p2p/pairing', { method: 'POST' });
            const data = await response.json();

            if (data.code) {
                codeContainer.innerHTML = `
                    <div class="pairing-code-display">
                        <span class="code">${data.code}</span>
                        <small>有效期5分钟</small>
                    </div>
                    <p class="text-muted">将此码告知对方进行连接</p>
                `;
            } else {
                codeContainer.innerHTML = '<p class="text-error">生成失败</p>';
            }
        } catch (error) {
            codeContainer.innerHTML = '<p class="text-error">生成失败</p>';
            console.error('Generate code error:', error);
        }
    },

    async joinWithCode() {
        const input = document.getElementById('pairingInput');
        const code = input.value.trim();

        if (code.length !== 6) {
            alert('请输入6位配对码');
            return;
        }

        try {
            const response = await fetch('/api/p2p/pairing/join', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ code })
            });

            const data = await response.json();

            if (data.peerId) {
                alert(`成功连接到设备: ${data.peerId.substring(0, 8)}`);
                input.value = '';
            } else {
                alert(data.error || '连接失败');
            }
        } catch (error) {
            alert('连接失败');
            console.error('Join error:', error);
        }
    },

    async refreshTransfers() {
        try {
            const response = await fetch('/api/p2p/transfers');
            const data = await response.json();
            this.displayTransfers(data.transfers || []);
        } catch (error) {
            console.error('Failed to load transfers:', error);
        }
    },

    displayTransfers(transfers) {
        const container = document.getElementById('transferList');

        if (!transfers || transfers.length === 0) {
            container.innerHTML = '<p class="text-muted">暂无传输任务</p>';
            return;
        }

        container.innerHTML = transfers.map(transfer => `
            <div class="transfer-item">
                <div class="transfer-info">
                    <span class="transfer-name">${transfer.fileName}</span>
                    <span class="transfer-progress">${transfer.progress.toFixed(1)}%</span>
                </div>
                <div class="transfer-bar">
                    <div class="transfer-bar-fill" style="width: ${transfer.progress}%"></div>
                </div>
                <div class="transfer-stats">
                    <span>${this.formatSize(transfer.transferred)} / ${this.formatSize(transfer.totalSize)}</span>
                    <span>${this.formatSpeed(transfer.speed)}</span>
                </div>
            </div>
        `).join('');
    },

    sendFileTo(deviceId) {
        const input = document.createElement('input');
        input.type = 'file';
        input.onchange = async (e) => {
            const files = e.target.files;
            if (files.length > 0) {
                alert(`将发送 ${files.length} 个文件到设备 ${deviceId.substring(0, 8)}`);
            }
        };
        input.click();
    },

    formatSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    },

    formatSpeed(bytesPerSec) {
        if (bytesPerSec === 0) return '0 B/s';
        return this.formatSize(bytesPerSec) + '/s';
    },

    refresh() {
        this.loadLocalAddresses();
        if (this.currentTab === 'discover') {
            this.discoverDevices();
        } else if (this.currentTab === 'transfers') {
            this.refreshTransfers();
        }
    }
};

document.addEventListener('DOMContentLoaded', () => P2P.init());