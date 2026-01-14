// API Configuration
const API_BASE_URL = 'http://localhost:8080/api';

// API Client
class BlockchainAPI {
    constructor(baseUrl) {
        this.baseUrl = baseUrl;
    }

    async request(endpoint, method = 'GET', body = null) {
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json',
            },
        };

        if (body) {
            options.body = JSON.stringify(body);
        }

        try {
            const response = await fetch(`${this.baseUrl}${endpoint}`, options);
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const data = await response.json();
            return { success: true, data };
        } catch (error) {
            console.error(`API Error (${endpoint}):`, error);
            return { success: false, error: error.message };
        }
    }

    // Blockchain Info
    async getBlockchainInfo() {
        return this.request('/blockchain/info');
    }

    // Get Block by Height
    async getBlock(height) {
        return this.request(`/blockchain/block/${height}`);
    }

    // Get Recent Blocks
    async getRecentBlocks(count = 10) {
        return this.request(`/blockchain/blocks?count=${count}`);
    }

    // Get Block Height
    async getBlockHeight() {
        return this.request('/blockchain/height');
    }

    // Wallet Operations
    async listWallets() {
        return this.request('/wallet/list');
    }

    async createWallet() {
        return this.request('/wallet/create', 'POST');
    }

    async getWalletBalance(address) {
        return this.request(`/wallet/balance/${address}`);
    }

    // Mempool
    async getMempool() {
        return this.request('/mempool');
    }

    // Mining Info
    async getMiningInfo() {
        return this.request('/mining/info');
    }

    // Transaction
    async sendTransaction(from, to, amount) {
        return this.request('/transaction/send', 'POST', { from, to, amount });
    }

    // Health Check
    async healthCheck() {
        return this.request('/health');
    }
}

// Export API instance
const api = new BlockchainAPI(API_BASE_URL);
