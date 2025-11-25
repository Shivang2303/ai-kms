// Knowledge graph visualization JavaScript
const API_BASE = '/api';
const WS_BASE = window.location.origin.replace('http', 'ws');

let ws = null;

document.addEventListener('DOMContentLoaded', () => {
    loadGraph();
    connectUpdatesWebSocket();
});

async function loadGraph() {
    try {
        const response = await fetch(`${API_BASE}/graph`);
        const graph = await response.json();
        
        renderGraph(graph);
    } catch (error) {
        console.error('Error loading graph:', error);
    }
}

function renderGraph(graph) {
    const container = document.getElementById('graph-container');
    // TODO: Implement graph visualization using D3.js or vis.js
    container.innerHTML = '<p>Graph visualization will be implemented here</p>';
}

function connectUpdatesWebSocket() {
    ws = new WebSocket(`${WS_BASE}/ws/updates`);
    
    ws.onopen = () => {
        console.log('Updates WebSocket connected');
    };
    
    ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        if (message.type === 'graph_update') {
            loadGraph(); // Reload graph on update
        }
    };
    
    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
}

document.getElementById('refresh-graph-btn').addEventListener('click', loadGraph);

document.getElementById('generate-graph-btn').addEventListener('click', async () => {
    try {
        await fetch(`${API_BASE}/graph/generate`, {
            method: 'POST'
        });
        loadGraph();
    } catch (error) {
        console.error('Error generating graph:', error);
    }
});

