const FileManager = {
    currentPath: '/',
    files: [],
    viewMode: 'list',

    getFileIcon(file) {
        if (file.isDir) return '📁';
        const ext = file.name.split('.').pop().toLowerCase();
        const icons = {
            'pdf': '📄',
            'doc': '📝',
            'docx': '📝',
            'xls': '📊',
            'xlsx': '📊',
            'jpg': '🖼️',
            'jpeg': '🖼️',
            'png': '🖼️',
            'gif': '🖼️',
            'svg': '🖼️',
            'zip': '📦',
            'rar': '📦',
            '7z': '📦',
            'mp3': '🎵',
            'mp4': '🎬',
            'html': '🌐',
            'css': '🎨',
            'js': '📜',
            'json': '📋',
            'txt': '📃'
        };
        return icons[ext] || '📄';
    },

    formatFileSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
    },

    formatDate(dateStr) {
        const date = new Date(dateStr);
        return date.toLocaleString('zh-CN');
    },

    isEditable(file) {
        if (file.isDir) return false;
        const textExts = ['txt', 'html', 'css', 'js', 'json', 'md', 'xml', 'csv'];
        const ext = file.name.split('.').pop().toLowerCase();
        return textExts.includes(ext);
    }
};