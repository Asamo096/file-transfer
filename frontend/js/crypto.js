const Crypto = {
    key: null,

    parseKeyFromURL() {
        const hash = window.location.hash;
        if (hash && hash.startsWith('#key=')) {
            this.key = decodeURIComponent(hash.substring(5));
            return this.key;
        }
        return null;
    },

    getKey() {
        if (!this.key) {
            this.parseKeyFromURL();
        }
        return this.key;
    },

    isEncrypted() {
        return !!this.getKey();
    },

    base64ToArrayBuffer(base64) {
        const binaryString = atob(base64);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
        }
        return bytes.buffer;
    },

    arrayBufferToBase64(buffer) {
        const bytes = new Uint8Array(buffer);
        let binary = '';
        for (let i = 0; i < bytes.length; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return btoa(binary);
    },

    async getCryptoKey() {
        if (!this.key) {
            throw new Error('No encryption key available');
        }
        const keyData = this.base64ToArrayBuffer(this.key);
        return await crypto.subtle.importKey(
            'raw',
            keyData,
            { name: 'AES-GCM' },
            false,
            ['encrypt', 'decrypt']
        );
    },

    async encrypt(data) {
        if (!this.isEncrypted()) {
            return data;
        }

        const cryptoKey = await this.getCryptoKey();
        const iv = crypto.getRandomValues(new Uint8Array(12));
        const encrypted = await crypto.subtle.encrypt(
            { name: 'AES-GCM', iv },
            cryptoKey,
            typeof data === 'string' ? new TextEncoder().encode(data) : data
        );

        const combined = new Uint8Array(iv.length + encrypted.byteLength);
        combined.set(iv);
        combined.set(new Uint8Array(encrypted), iv.length);
        
        return combined;
    },

    async decrypt(data) {
        if (!this.isEncrypted()) {
            return data;
        }

        const cryptoKey = await this.getCryptoKey();
        const iv = data.slice(0, 12);
        const ciphertext = data.slice(12);
        
        const decrypted = await crypto.subtle.decrypt(
            { name: 'AES-GCM', iv },
            cryptoKey,
            ciphertext
        );

        return decrypted;
    }
};