import { useNavigate } from 'react-router-dom'
import { useState } from 'react'
import axios from 'axios'

const API_URL = 'http://localhost:8080/api'

export default function DocumentList() {
    const navigate = useNavigate()
    const [creating, setCreating] = useState(false)

    const createNewDocument = async () => {
        setCreating(true)
        try {
            const res = await axios.post(`${API_URL}/documents`, {
                title: 'Untitled',
                content: '# Untitled\n\nStart writing...'
            })
            navigate(`/document/${res.data.id}`)
        } catch (error) {
            console.error('Failed to create document:', error)
            alert('Failed to create document. Make sure the backend is running.')
        } finally {
            setCreating(false)
        }
    }

    return (
        <div className="editor-container">
            <div style={{
                padding: '60px 40px',
                maxWidth: '700px',
                margin: '0 auto',
                textAlign: 'center'
            }}>
                <div style={{ fontSize: '64px', marginBottom: '24px' }}>⚡</div>

                <h1 style={{
                    fontSize: '48px',
                    marginBottom: '16px',
                    color: 'var(--accent-primary)',
                    fontWeight: 700
                }}>
                    AI Knowledge System
                </h1>

                <p style={{
                    fontSize: '18px',
                    color: 'var(--text-secondary)',
                    marginBottom: '48px',
                    lineHeight: 1.8
                }}>
                    Obsidian-inspired editor with real-time collaboration, AI-powered search,
                    and beautiful graph visualization.
                </p>

                <div style={{
                    display: 'flex',
                    gap: '16px',
                    justifyContent: 'center',
                    flexWrap: 'wrap',
                    marginBottom: '60px'
                }}>
                    <button
                        className="btn-primary btn"
                        style={{ padding: '16px 32px', fontSize: '16px' }}
                        onClick={createNewDocument}
                        disabled={creating}
                    >
                        {creating ? '⏳ Creating...' : '📝 Create First Document'}
                    </button>

                    <button
                        className="btn"
                        style={{ padding: '16px 32px', fontSize: '16px' }}
                        onClick={() => window.open('https://github.com', '_blank')}
                    >
                        📖 Documentation
                    </button>
                </div>

                <div style={{
                    marginTop: '40px',
                    padding: '32px',
                    background: 'var(--bg-secondary)',
                    borderRadius: '12px',
                    border: '1px solid var(--border)',
                    textAlign: 'left'
                }}>
                    <h3 style={{
                        color: 'var(--accent-primary)',
                        marginBottom: '24px',
                        fontSize: '20px'
                    }}>
                        ✨ Features
                    </h3>
                    <div style={{
                        display: 'grid',
                        gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
                        gap: '24px'
                    }}>
                        <div>
                            <div style={{ fontSize: '32px', marginBottom: '8px' }}>👥</div>
                            <div style={{
                                color: 'var(--text-primary)',
                                fontWeight: 600,
                                marginBottom: '4px'
                            }}>
                                Real-time Collaboration
                            </div>
                            <div style={{ fontSize: '14px', color: 'var(--text-secondary)' }}>
                                Edit together with live cursors
                            </div>
                        </div>

                        <div>
                            <div style={{ fontSize: '32px', marginBottom: '8px' }}>🔗</div>
                            <div style={{
                                color: 'var(--text-primary)',
                                fontWeight: 600,
                                marginBottom: '4px'
                            }}>
                                Knowledge Graph
                            </div>
                            <div style={{ fontSize: '14px', color: 'var(--text-secondary)' }}>
                                Visualize document connections
                            </div>
                        </div>

                        <div>
                            <div style={{ fontSize: '32px', marginBottom: '8px' }}>🤖</div>
                            <div style={{
                                color: 'var(--text-primary)',
                                fontWeight: 600,
                                marginBottom: '4px'
                            }}>
                                AI-Powered
                            </div>
                            <div style={{ fontSize: '14px', color: 'var(--text-secondary)' }}>
                                Semantic search & RAG Q&A
                            </div>
                        </div>

                        <div>
                            <div style={{ fontSize: '32px', marginBottom: '8px' }}>📜</div>
                            <div style={{
                                color: 'var(--text-primary)',
                                fontWeight: 600,
                                marginBottom: '4px'
                            }}>
                                Infinite Canvas
                            </div>
                            <div style={{ fontSize: '14px', color: 'var(--text-secondary)' }}>
                                Distraction-free writing
                            </div>
                        </div>
                    </div>
                </div>

                <div style={{
                    marginTop: '40px',
                    padding: '24px',
                    background: 'rgba(255, 137, 53, 0.1)',
                    border: '1px solid var(--accent-primary)',
                    borderRadius: '8px',
                    fontSize: '14px',
                    color: 'var(--text-secondary)'
                }}>
                    💡 <strong style={{ color: 'var(--accent-primary)' }}>Tip:</strong> Use the sidebar to browse documents, or click "Create First Document" to get started!
                </div>
            </div>
        </div>
    )
}
