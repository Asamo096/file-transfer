const App = {
    elements: {},

    init() {
        this.cacheElements();
        this.bindEvents();
        this.loadFiles();
    },

    cacheElements() {
        this.elements = {
            fileList: document.getElementById('fileList'),
            breadcrumb: document.getElementById('breadcrumb'),
            modal: document.getElementById('modal'),
            modalTitle: document.getElementById('modalTitle'),
            modalBody: document.getElementById('modalBody'),
            modalFooter: document.getElementById('modalFooter'),
            closeModal: document.getElementById('closeModal'),
            fileInput: document.getElementById('fileInput'),
            uploadZone: document.getElementById('uploadZone'),
            viewToggle: document.getElementById('viewToggle'),
            refreshBtn: document.getElementById('refreshBtn'),
            newFolderBtn: document.getElementById('newFolderBtn'),
            uploadBtn: document.getElementById('uploadBtn'),
            toast: document.getElementById('toast')
        };
    },

    bindEvents() {
        this.elements.closeModal.addEventListener('click', () => this.closeModal());
        this.elements.modal.addEventListener('click', (e) => {
            if (e.target === this.elements.modal) this.closeModal();
        });
        this.elements.viewToggle.addEventListener('click', () => this.toggleView());
        this.elements.refreshBtn.addEventListener('click', () => this.loadFiles());
        this.elements.newFolderBtn.addEventListener('click', () => this.showNewFolderModal());
        this.elements.uploadBtn.addEventListener('click', () => this.elements.fileInput.click());
        this.elements.fileInput.addEventListener('change', (e) => this.handleFileSelect(e));

        this.elements.uploadZone.addEventListener('click', () => this.elements.fileInput.click());
        this.elements.uploadZone.addEventListener('dragover', (e) => {
            e.preventDefault();
            this.elements.uploadZone.classList.add('dragover');
        });
        this.elements.uploadZone.addEventListener('dragleave', () => {
            this.elements.uploadZone.classList.remove('dragover');
        });
        this.elements.uploadZone.addEventListener('drop', (e) => {
            e.preventDefault();
            this.elements.uploadZone.classList.remove('dragover');
            const files = Array.from(e.dataTransfer.files);
            this.uploadFiles(files);
        });

        this.elements.fileList.addEventListener('click', (e) => {
            const fileItem = e.target.closest('.file-item');
            if (!fileItem) return;
            if (e.target.closest('.btn')) return;

            const fileData = JSON.parse(fileItem.dataset.file);

            if (fileData.isDir) {
                FileManager.currentPath = fileData.path;
                this.loadFiles();
            } else if (FileManager.isEditable(fileData)) {
                this.editFile(fileData.path);
            } else {
                this.downloadFile(fileData.path);
            }
        });
    },

    async loadFiles() {
        try {
            this.elements.fileList.innerHTML = '<div class="loading">加载中...</div>';
            const data = await API.listFiles(FileManager.currentPath);
            FileManager.files = data.files || [];
            this.renderFileList();
            this.renderBreadcrumb();
        } catch (error) {
            this.showToast(error.message, 'error');
            this.elements.fileList.innerHTML = '<div class="empty-state"><span class="empty-icon">❌</span><p>加载失败</p></div>';
        }
    },

    renderFileList() {
        if (FileManager.files.length === 0) {
            this.elements.fileList.innerHTML = '<div class="empty-state"><span class="empty-icon">📂</span><p>目录为空</p></div>';
            this.elements.fileList.className = 'file-list ' + FileManager.viewMode + '-view';
            return;
        }

        this.elements.fileList.className = 'file-list ' + FileManager.viewMode + '-view';
        this.elements.fileList.innerHTML = FileManager.files.map(file => `
            <div class="file-item" data-file='${JSON.stringify(file).replace(/'/g, "&#39;")}'>
                <div class="file-icon">${FileManager.getFileIcon(file)}</div>
                <div class="file-info">
                    <div class="file-name">${this.escapeHtml(file.name)}</div>
                    <div class="file-meta">
                        ${file.isDir ? '' : FileManager.formatFileSize(file.size)}
                        ${file.isDir ? '' : ' • ' + FileManager.formatDate(file.modTime)}
                    </div>
                </div>
                <div class="file-actions">
                    ${!file.isDir ? `<button class="btn btn-icon" onclick="App.downloadFile('${this.escapeHtml(file.path)}')" title="下载">⬇️</button>` : ''}
                    ${FileManager.isEditable(file) ? `<button class="btn btn-icon" onclick="App.editFile('${this.escapeHtml(file.path)}')" title="编辑">✏️</button>` : ''}
                    <button class="btn btn-icon" onclick="App.renameFile('${this.escapeHtml(file.path)}', '${this.escapeHtml(file.name)}')" title="重命名">📝</button>
                    <button class="btn btn-icon" onclick="App.deleteFile('${this.escapeHtml(file.path)}')" title="删除">🗑️</button>
                </div>
            </div>
        `).join('');
    },

    renderBreadcrumb() {
        const parts = FileManager.currentPath.split('/').filter(Boolean);
        let html = '<span class="breadcrumb-item" data-path="/">🏠 根目录</span>';

        let path = '';
        parts.forEach(part => {
            path += '/' + part;
            html += `<span class="breadcrumb-item" data-path="${this.escapeHtml(path)}">${this.escapeHtml(part)}</span>`;
        });

        this.elements.breadcrumb.innerHTML = html;

        this.elements.breadcrumb.querySelectorAll('.breadcrumb-item').forEach(item => {
            item.addEventListener('click', () => {
                FileManager.currentPath = item.dataset.path;
                this.loadFiles();
            });
        });
    },

    toggleView() {
        FileManager.viewMode = FileManager.viewMode === 'list' ? 'grid' : 'list';
        this.elements.viewToggle.innerHTML = FileManager.viewMode === 'list' ? '<span class="grid-view">▦</span>' : '<span class="list-view">☰</span>';
        this.renderFileList();
    },

    showModal(title, body, footer) {
        this.elements.modalTitle.textContent = title;
        this.elements.modalBody.innerHTML = body;
        this.elements.modalFooter.innerHTML = footer;
        this.elements.modal.classList.add('active');
    },

    closeModal() {
        this.elements.modal.classList.remove('active');
    },

    showToast(message, type = 'info') {
        this.elements.toast.textContent = message;
        this.elements.toast.className = 'toast ' + type + ' active';
        setTimeout(() => {
            this.elements.toast.classList.remove('active');
        }, 3000);
    },

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    },

    showNewFolderModal() {
        this.showModal(
            '📁 新建文件夹',
            '<div class="form-group"><label>文件夹名称</label><input type="text" id="newFolderName" placeholder="请输入文件夹名称"></div>',
            '<button class="btn btn-secondary" onclick="App.closeModal()">取消</button><button class="btn btn-primary" onclick="App.createNewFolder()">创建</button>'
        );
        setTimeout(() => document.getElementById('newFolderName').focus(), 100);
    },

    async createNewFolder() {
        const name = document.getElementById('newFolderName').value.trim();
        if (!name) {
            this.showToast('请输入文件夹名称', 'error');
            return;
        }
        try {
            await API.createDir(FileManager.currentPath, name);
            this.showToast('文件夹创建成功', 'success');
            this.closeModal();
            this.loadFiles();
        } catch (error) {
            this.showToast(error.message, 'error');
        }
    },

    handleFileSelect(e) {
        const files = Array.from(e.target.files);
        this.uploadFiles(files);
        e.target.value = '';
    },

    async uploadFiles(files) {
        if (files.length === 0) return;

        let uploaded = 0;
        for (const file of files) {
            try {
                this.showModal(
                    '⬆️ 上传中',
                    `<div><p>正在上传: ${this.escapeHtml(file.name)}</p><div class="progress-bar"><div class="progress-fill" id="progressFill" style="width: 0%"></div></div><p>${uploaded + 1} / ${files.length}</p></div>`,
                    ''
                );

                await API.uploadFile(file, FileManager.currentPath, (percent) => {
                    const progressFill = document.getElementById('progressFill');
                    if (progressFill) progressFill.style.width = percent + '%';
                });

                uploaded++;
            } catch (error) {
                this.showToast(`${file.name}: 上传失败`, 'error');
            }
        }

        this.closeModal();
        this.showToast(`上传完成: ${uploaded} 个文件`, 'success');
        this.loadFiles();
    },

    downloadFile(path) {
        API.downloadFile(path);
    },

    async deleteFile(path) {
        try {
            await API.deleteFile(path);
            this.showToast('删除成功', 'success');
            this.loadFiles();
        } catch (error) {
            this.showToast(error.message, 'error');
        }
    },

    renameFile(path, oldName) {
        this.showModal(
            '📝 重命名',
            `<div class="form-group"><label>新名称</label><input type="text" id="newFileName" value="${this.escapeHtml(oldName)}"></div>`,
            '<button class="btn btn-secondary" onclick="App.closeModal()">取消</button><button class="btn btn-primary" id="renameSaveBtn">保存</button>'
        );
        setTimeout(() => {
            const input = document.getElementById('newFileName');
            input.focus();
            input.select();
            document.getElementById('renameSaveBtn').onclick = () => this.doRename(path);
        }, 100);
    },

    async doRename(oldPath) {
        const newName = document.getElementById('newFileName').value.trim();
        if (!newName) {
            this.showToast('请输入文件名', 'error');
            return;
        }
        try {
            const dir = oldPath.substring(0, oldPath.lastIndexOf('/'));
            const newPath = dir ? dir + '/' + newName : '/' + newName;
            await API.renameFile(oldPath, newPath);
            this.showToast('重命名成功', 'success');
            this.closeModal();
            this.loadFiles();
        } catch (error) {
            this.showToast(error.message, 'error');
        }
    },

    async editFile(path) {
        try {
            this.showModal('✏️ 加载中...', '<div class="loading">加载文件内容...</div>', '');
            const data = await API.readTextFile(path);
            this.showModal(
                '✏️ 编辑文件',
                `<div class="form-group"><label>文件内容</label><textarea id="fileEditor">${this.escapeHtml(data.content)}</textarea></div>`,
                '<button class="btn btn-secondary" onclick="App.closeModal()">取消</button><button class="btn btn-primary" id="saveEditBtn">保存</button>'
            );
            setTimeout(() => {
                document.getElementById('saveEditBtn').onclick = () => this.saveFileEdit(path);
            }, 100);
        } catch (error) {
            this.showToast(error.message, 'error');
            this.closeModal();
        }
    },

    async saveFileEdit(path) {
        const content = document.getElementById('fileEditor').value;
        try {
            await API.saveTextFile(path, content);
            this.showToast('保存成功', 'success');
            this.closeModal();
        } catch (error) {
            this.showToast(error.message, 'error');
        }
    }
};

document.addEventListener('DOMContentLoaded', () => App.init());