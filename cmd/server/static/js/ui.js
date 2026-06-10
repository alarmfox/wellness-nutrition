// ============================================================================
// UI UTILITIES
// ============================================================================
var BUSINESS_TIME_ZONE = 'Europe/Rome';

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
            panel.className = 'notification-panel';
            document.body.appendChild(panel);
        }

        const notification = document.createElement('div');
        notification.className = 'notification' + (type === 'success' ? ' success' : '');
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
            minute: '2-digit',
            timeZone: BUSINESS_TIME_ZONE
        });
    }
};

function handleLogout() {
    if (!confirm('Sicuro di voler uscire dall\'applicazione?')) {
        return;
    }

    fetch('/api/auth/logout', {
        method: 'DELETE',
        headers: {
            'X-CSRF-Token': getCookie('csrf_token')
        }
    }).then(() => {
        window.location.href = '/signin';
    });
}

document.addEventListener('click', function(event) {
    const trigger = event.target.closest('[data-action="logout"]');
    if (!trigger) {
        return;
    }

    event.preventDefault();
    handleLogout();
});

document.addEventListener('keydown', function(event) {
    if (event.key !== 'Escape') {
        return;
    }

    const visibleModals = Array.from(document.querySelectorAll('.modal'))
        .filter(modal => window.getComputedStyle(modal).display !== 'none');
    const modal = visibleModals[visibleModals.length - 1];
    if (!modal) {
        return;
    }

    event.preventDefault();

    const closeControl = modal.querySelector('.close, [onclick^="close"], [onclick*="closeModal"]');
    if (closeControl) {
        closeControl.click();
        return;
    }

    modal.style.display = 'none';
});
