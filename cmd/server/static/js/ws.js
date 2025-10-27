// ============================================================================
// WEBSOCKET HANDLER
// ============================================================================
const WebSocketHandler = {
    ws: null,

    connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;

        try {
            this.ws = new WebSocket(wsUrl);

            this.ws.onopen = () => {
                UI.showNotification('Connesso al server', 'success');
            };

            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    UI.showNotification(data.message, data.type === 'booking_created' ? 'success' : 'error');
                    Calendar.load();
                } catch (e) {
                    console.error('Error parsing WebSocket message:', e);
                }
            };

            this.ws.onclose = () => {
                UI.showNotification('Disconnesso dal server', 'error');
                setTimeout(() => this.connect(), 5000);
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
            };
        } catch (error) {
            console.error('Failed to create WebSocket:', error);
        }
    }
};

document.addEventListener('DOMContentLoaded', function() {
    WebSocketHandler.connect();
});
