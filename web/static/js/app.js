// Main application JavaScript
const API_BASE = '/api';

async function loadDocuments() {
    try {
        const response = await fetch(`${API_BASE}/documents`);
        const documents = await response.json();
        
        const list = document.getElementById('documents-list');
        list.innerHTML = '';
        
        documents.forEach(doc => {
            const card = document.createElement('div');
            card.className = 'document-card';
            card.innerHTML = `
                <h3>${doc.title}</h3>
                <p>${doc.format} • ${new Date(doc.created_at).toLocaleDateString()}</p>
            `;
            card.addEventListener('click', () => {
                window.location.href = `/editor.html?id=${doc.id}`;
            });
            list.appendChild(card);
        });
    } catch (error) {
        console.error('Error loading documents:', error);
    }
}

document.getElementById('create-doc-btn').addEventListener('click', async () => {
    // TODO: Implement document creation
    console.log('Create document clicked');
});

// Load documents on page load
loadDocuments();

