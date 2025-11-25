// Editor JavaScript with WebSocket support
const API_BASE = '/api';
const WS_BASE = window.location.origin.replace('http', 'ws');

let documentId = null;
let ws = null;

// Initialize editor
document.addEventListener('DOMContentLoaded', () => {
    const params = new URLSearchParams(window.location.search);
    documentId = params.get('id');
    
    if (documentId) {
        loadDocument(documentId);
        connectWebSocket(documentId);
    }
});

async function loadDocument(id) {
    try {
        const response = await fetch(`${API_BASE}/documents/${id}`);
        const doc = await response.json();
        
        document.getElementById('doc-title').value = doc.title;
        document.getElementById('editor').value = doc.content;
    } catch (error) {
        console.error('Error loading document:', error);
    }
}

function connectWebSocket(docId) {
    ws = new WebSocket(`${WS_BASE}/ws/document/${docId}`);
    
    ws.onopen = () => {
        console.log('WebSocket connected');
    };
    
    ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        handleWebSocketMessage(message);
    };
    
    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
    
    ws.onclose = () => {
        console.log('WebSocket disconnected');
    };
}

function handleWebSocketMessage(message) {
    switch (message.type) {
        case 'update':
            // Handle document update from other users
            const update = JSON.parse(message.payload);
            document.getElementById('editor').value = update.content;
            break;
        case 'pong':
            // Heartbeat response
            break;
    }
}

// Auto-save on content change
let saveTimeout;
document.getElementById('editor').addEventListener('input', () => {
    clearTimeout(saveTimeout);
    saveTimeout = setTimeout(() => {
        saveDocument();
    }, 2000);
});

async function saveDocument() {
    if (!documentId) return;
    
    const title = document.getElementById('doc-title').value;
    const content = document.getElementById('editor').value;
    
    try {
        await fetch(`${API_BASE}/documents/${documentId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ title, content })
        });
    } catch (error) {
        console.error('Error saving document:', error);
    }
}

document.getElementById('save-btn').addEventListener('click', saveDocument);

document.getElementById('summarize-btn').addEventListener('click', async () => {
    if (!documentId) return;
    
    try {
        const response = await fetch(`${API_BASE}/documents/${documentId}/summarize`, {
            method: 'POST'
        });
        const result = await response.json();
        
        const panel = document.getElementById('ai-panel');
        panel.classList.remove('hidden');
        document.getElementById('ai-content').innerHTML = `<p>${result.summary}</p>`;
    } catch (error) {
        console.error('Error summarizing:', error);
    }
});

document.getElementById('generate-graph-btn').addEventListener('click', async () => {
    // TODO: Implement graph generation
    console.log('Generate graph clicked');
});

