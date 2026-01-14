// Global State
let currentTab = 'blocks';
let refreshInterval = null;

// Utility Functions
function formatHash(hash) {
    if (!hash) return 'N/A';
    return hash.length > 16 ? `${hash.slice(0, 8)}...${hash.slice(-8)}` : hash;
}

function formatTimestamp(timestamp) {
    if (!timestamp) return 'N/A';
    const date = new Date(timestamp * 1000);
    return date.toLocaleString();
}

function formatNumber(num) {
    if (num === undefined || num === null) return '0';
    return num.toLocaleString();
}

function formatHashrate(difficulty) {
    // Rough estimate: hashrate = difficulty / 10 (hashes per second)
    const hashrate = difficulty / 10;
    if (hashrate > 1000000) {
        return `${(hashrate / 1000000).toFixed(2)} MH/s`;
    } else if (hashrate > 1000) {
        return `${(hashrate / 1000).toFixed(2)} KH/s`;
    }
    return `${hashrate.toFixed(2)} H/s`;
}

// Connection Status
function updateConnectionStatus(isConnected, message = '') {
    const statusEl = document.getElementById('connectionStatus');
    const dotEl = statusEl.querySelector('.status-dot');
    const textEl = statusEl.querySelector('.status-text');

    if (isConnected) {
        dotEl.className = 'status-dot connected';
        textEl.textContent = 'Connected to node';
    } else {
        dotEl.className = 'status-dot error';
        textEl.textContent = message || 'Connection failed';
    }
}

// Hero Stats Update
async function updateHeroStats() {
    try {
        const infoResult = await api.getBlockchainInfo();
        if (infoResult.success) {
            const { height, difficulty, bestBlock } = infoResult.data;
            document.getElementById('blockHeight').textContent = formatNumber(height);
            document.getElementById('difficulty').textContent = formatNumber(difficulty);
            document.getElementById('hashrate').textContent = formatHashrate(difficulty);
        }

        const mempoolResult = await api.getMempool();
        if (mempoolResult.success) {
            document.getElementById('mempoolSize').textContent = 
                formatNumber(mempoolResult.data.transactions?.length || 0);
        }
    } catch (error) {
        console.error('Error updating hero stats:', error);
    }
}

// Blocks Tab
async function loadBlocks() {
    const container = document.getElementById('blocksList');
    container.innerHTML = '<div class="loading">Loading blocks...</div>';

    try {
        const result = await api.getRecentBlocks(10);
        
        if (!result.success) {
            container.innerHTML = `<div class="empty-state">
                <div class="empty-state-icon">❌</div>
                <p>Failed to load blocks: ${result.error}</p>
            </div>`;
            return;
        }

        const blocks = result.data.blocks || [];
        
        if (blocks.length === 0) {
            container.innerHTML = `<div class="empty-state">
                <div class="empty-state-icon">📦</div>
                <p>No blocks found</p>
            </div>`;
            return;
        }

        container.innerHTML = blocks.map(block => `
            <div class="block-card">
                <div class="block-header">
                    <div class="block-height">Block #${formatNumber(block.height)}</div>
                    <div class="block-hash">${formatHash(block.hash)}</div>
                </div>
                <div class="block-info">
                    <div class="block-info-item">
                        <div class="block-info-label">Transactions</div>
                        <div class="block-info-value">${block.transactionCount || 0}</div>
                    </div>
                    <div class="block-info-item">
                        <div class="block-info-label">Timestamp</div>
                        <div class="block-info-value">${formatTimestamp(block.timestamp)}</div>
                    </div>
                    <div class="block-info-item">
                        <div class="block-info-label">Nonce</div>
                        <div class="block-info-value">${formatNumber(block.nonce)}</div>
                    </div>
                    <div class="block-info-item">
                        <div class="block-info-label">Previous Hash</div>
                        <div class="block-info-value code">${formatHash(block.prevHash)}</div>
                    </div>
                </div>
            </div>
        `).join('');

    } catch (error) {
        container.innerHTML = `<div class="empty-state">
            <div class="empty-state-icon">❌</div>
            <p>Error loading blocks</p>
        </div>`;
    }
}

function refreshBlocks() {
    loadBlocks();
    updateHeroStats();
}

// Wallets Tab
async function loadWallets() {
    const container = document.getElementById('walletsList');
    container.innerHTML = '<div class="loading">Loading wallets...</div>';

    try {
        const result = await api.listWallets();
        
        if (!result.success) {
            container.innerHTML = `<div class="empty-state">
                <div class="empty-state-icon">❌</div>
                <p>Failed to load wallets: ${result.error}</p>
            </div>`;
            return;
        }

        const wallets = result.data.wallets || [];
        
        if (wallets.length === 0) {
            container.innerHTML = `<div class="empty-state">
                <div class="empty-state-icon">👛</div>
                <p>No wallets found. Create your first wallet!</p>
            </div>`;
            return;
        }

        container.innerHTML = wallets.map(wallet => `
            <div class="wallet-card">
                <div class="wallet-address">${wallet.address}</div>
                <div class="wallet-balance">${wallet.balance || 0} IC</div>
                <div class="wallet-balance-label">Balance</div>
            </div>
        `).join('');

    } catch (error) {
        container.innerHTML = `<div class="empty-state">
            <div class="empty-state-icon">❌</div>
            <p>Error loading wallets</p>
        </div>`;
    }
}

async function createNewWallet() {
    const container = document.getElementById('walletsList');
    const originalContent = container.innerHTML;
    container.innerHTML = '<div class="loading">Creating wallet...</div>';

    try {
        const result = await api.createWallet();
        
        if (!result.success) {
            alert(`Failed to create wallet: ${result.error}`);
            container.innerHTML = originalContent;
            return;
        }

        alert(`Wallet created successfully!\nAddress: ${result.data.address}`);
        await loadWallets();

    } catch (error) {
        alert('Error creating wallet');
        container.innerHTML = originalContent;
    }
}

// Mempool Tab
async function loadMempool() {
    const container = document.getElementById('mempoolList');
    container.innerHTML = '<div class="loading">Loading mempool...</div>';

    try {
        const result = await api.getMempool();
        
        if (!result.success) {
            container.innerHTML = `<div class="empty-state">
                <div class="empty-state-icon">❌</div>
                <p>Failed to load mempool: ${result.error}</p>
            </div>`;
            return;
        }

        const transactions = result.data.transactions || [];
        
        if (transactions.length === 0) {
            container.innerHTML = `<div class="empty-state">
                <div class="empty-state-icon">📭</div>
                <p>Mempool is empty</p>
            </div>`;
            return;
        }

        container.innerHTML = transactions.map(tx => `
            <div class="tx-card">
                <div class="tx-id">Transaction ID: ${formatHash(tx.id)}</div>
                <div class="tx-details">
                    <div class="block-info-item">
                        <div class="block-info-label">Inputs</div>
                        <div class="block-info-value">${tx.inputCount || 0}</div>
                    </div>
                    <div class="block-info-item">
                        <div class="block-info-label">Outputs</div>
                        <div class="block-info-value">${tx.outputCount || 0}</div>
                    </div>
                    <div class="block-info-item">
                        <div class="block-info-label">Total Value</div>
                        <div class="block-info-value">${tx.totalValue || 0} IC</div>
                    </div>
                </div>
            </div>
        `).join('');

    } catch (error) {
        container.innerHTML = `<div class="empty-state">
            <div class="empty-state-icon">❌</div>
            <p>Error loading mempool</p>
        </div>`;
    }
}

function refreshMempool() {
    loadMempool();
    updateHeroStats();
}

// Mining Tab
async function loadMiningInfo() {
    const container = document.getElementById('miningInfo');
    container.innerHTML = '<div class="loading">Loading mining info...</div>';

    try {
        const result = await api.getMiningInfo();
        
        if (!result.success) {
            container.innerHTML = `<div class="empty-state">
                <div class="empty-state-icon">❌</div>
                <p>Failed to load mining info: ${result.error}</p>
            </div>`;
            return;
        }

        const info = result.data;
        
        container.innerHTML = `
            <div class="mining-card">
                <div class="mining-label">Current Difficulty</div>
                <div class="mining-value">${formatNumber(info.difficulty)}</div>
            </div>
            <div class="mining-card">
                <div class="mining-label">Network Hashrate</div>
                <div class="mining-value">${formatHashrate(info.difficulty)}</div>
            </div>
            <div class="mining-card">
                <div class="mining-label">Mining Status</div>
                <div class="mining-value">${info.isMining ? '⛏️ Active' : '⏸️ Inactive'}</div>
            </div>
            <div class="mining-card">
                <div class="mining-label">Block Reward</div>
                <div class="mining-value">${info.blockReward || 50} IC</div>
            </div>
        `;

    } catch (error) {
        container.innerHTML = `<div class="empty-state">
            <div class="empty-state-icon">❌</div>
            <p>Error loading mining info</p>
        </div>`;
    }
}

function refreshMiningInfo() {
    loadMiningInfo();
}

// Tab Switching
function switchTab(tabName) {
    // Update tab buttons
    document.querySelectorAll('.tab-button').forEach(btn => {
        btn.classList.remove('active');
        if (btn.dataset.tab === tabName) {
            btn.classList.add('active');
        }
    });

    // Update tab content
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.remove('active');
    });
    document.getElementById(`${tabName}-tab`).classList.add('active');

    // Load content for the active tab
    currentTab = tabName;
    loadTabContent(tabName);

    // Clear and set new refresh interval
    if (refreshInterval) {
        clearInterval(refreshInterval);
    }
    
    // Auto-refresh every 10 seconds
    refreshInterval = setInterval(() => {
        loadTabContent(currentTab);
        updateHeroStats();
    }, 10000);
}

function loadTabContent(tabName) {
    switch (tabName) {
        case 'blocks':
            loadBlocks();
            break;
        case 'wallets':
            loadWallets();
            break;
        case 'mempool':
            loadMempool();
            break;
        case 'mining':
            loadMiningInfo();
            break;
    }
}

// Health Check
async function checkConnection() {
    const result = await api.healthCheck();
    updateConnectionStatus(result.success, result.error);
    return result.success;
}

// Navigation
function setupNavigation() {
    document.querySelectorAll('.nav-link').forEach(link => {
        link.addEventListener('click', (e) => {
            if (link.getAttribute('href')?.startsWith('#')) {
                e.preventDefault();
                const tabName = link.getAttribute('href').substring(1);
                if (document.querySelector(`[data-tab="${tabName}"]`)) {
                    switchTab(tabName);
                }
            }
        });
    });
}

// Initialize
async function init() {
    console.log('Initializing Integrity Coin Explorer...');
    
    // Setup tab switching
    document.querySelectorAll('.tab-button').forEach(btn => {
        btn.addEventListener('click', () => {
            switchTab(btn.dataset.tab);
        });
    });

    // Setup navigation
    setupNavigation();

    // Check connection
    const isConnected = await checkConnection();
    
    if (!isConnected) {
        updateConnectionStatus(false, 'Cannot connect to node. Make sure the backend server is running on port 8080.');
    }

    // Load initial content
    await updateHeroStats();
    await loadBlocks();

    // Set up auto-refresh
    refreshInterval = setInterval(() => {
        loadTabContent(currentTab);
        updateHeroStats();
    }, 10000);

    console.log('Initialization complete!');
}

// Start when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
