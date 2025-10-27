// ============================================================================
// UI UTILITIES
// ============================================================================
const UI = {
    showLoading(text = 'Caricamento...') {
        const overlay = document.getElementById('loading-overlay');
        const loadingText = document.getElementById('loading-text');
        if (overlay && loadingText) {
            loadingText.textContent = text;
            overlay.classList.add('show');
        }
    },

    hideLoading() {
        const overlay = document.getElementById('loading-overlay');
        if (overlay) {
            overlay.classList.remove('show');
        }
    },

    showToast(message, isSuccess) {
        const toast = document.getElementById('toast');
        if (toast) {
            toast.textContent = message;
            toast.className = 'toast' + (isSuccess ? ' success' : '');
            toast.style.display = 'block';
            setTimeout(() => toast.style.display = 'none', 4000);
        }
    },

    showNotification(message, type) {
        let panel = document.getElementById('notificationPanel');
        if (!panel) {
            panel = document.createElement('div');
            panel.id = 'notificationPanel';
            panel.style.cssText = 'position: fixed; bottom: 20px; left: 20px; z-index: 10000; max-width: 350px;';
            document.body.appendChild(panel);
        }

        const notification = document.createElement('div');
        notification.style.cssText = `
            background: white;
            border-left: 4px solid ${type === 'success' ? '#4caf50' : '#f44336'};
            box-shadow: 0 2px 8px rgba(0,0,0,0.2);
            padding: 12px;
            margin-bottom: 10px;
            border-radius: 4px;
        `;
        notification.textContent = message;
        panel.appendChild(notification);

        setTimeout(() => {
            notification.remove();
        }, 5000);
    },

    formatSlotDateTime(dateString) {
        const date = new Date(dateString);
        return date.toLocaleString('it-IT', {
            weekday: 'long',
            year: 'numeric',
            month: 'long',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    }
};
