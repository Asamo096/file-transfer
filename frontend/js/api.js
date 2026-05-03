const API = {
    baseURL: '/api',

    async request(endpoint, options = {}) {
        const url = `${this.baseURL}${endpoint}`;
        try {
            const response = await fetch(url, {
                ...options,
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                }
            });
            const data = await response.json();
            if (!response.ok) {
                throw new Error(data.error || '请求失败');
            }
            return data;
        } catch (error) {
            console.error('API Error:', error);
            throw error;
        }
    },

    async listFiles(path = '/') {
        return this.request(`/files?path=${encodeURIComponent(path)}`);
    },

    async downloadFile(path) {
        const url = `${this.baseURL}/download?path=${encodeURIComponent(path)}`;
        window.location.href = url;
    },

    async uploadFile(file, path = '/', onProgress) {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('path', path);

        return new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest();
            
            xhr.upload.addEventListener('progress', (e) => {
                if (onProgress && e.lengthComputable) {
                    const percent = Math.round((e.loaded / e.total) * 100);
                    onProgress(percent);
                }
            });

            xhr.addEventListener('load', () => {
                if (xhr.status >= 200 && xhr.status < 300) {
                    resolve(JSON.parse(xhr.responseText));
                } else {
                    reject(new Error('上传失败'));
                }
            });

            xhr.addEventListener('error', () => reject(new Error('上传失败')));
            xhr.addEventListener('abort', () => reject(new Error('上传取消')));

            xhr.open('POST', `${this.baseURL}/upload`);
            xhr.send(formData);
        });
    },

    async deleteFile(path) {
        return this.request(`/files?path=${encodeURIComponent(path)}`, { method: 'DELETE' });
    },

    async renameFile(oldPath, newPath) {
        return this.request('/files', {
            method: 'PATCH',
            body: JSON.stringify({ oldPath, newPath })
        });
    },

    async readTextFile(path) {
        return this.request(`/read?path=${encodeURIComponent(path)}`);
    },

    async saveTextFile(path, content) {
        return this.request('/save', {
            method: 'POST',
            body: JSON.stringify({ path, content })
        });
    },

    async getInfo() {
        return this.request('/info');
    },

    async createDir(path, name) {
        return this.request('/mkdir', {
            method: 'POST',
            body: JSON.stringify({ path, name })
        });
    }
};