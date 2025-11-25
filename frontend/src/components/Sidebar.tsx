import { useState, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import axios from 'axios'

const API_URL = 'http://localhost:8080/api'

interface Document {
    id: string
    title: string
    created_at: string
}

interface SidebarProps {
    isOpen: boolean
}

export default function Sidebar({ isOpen }: SidebarProps) {
    const [documents, setDocuments] = useState<Document[]>([])
    const [loading, setLoading] = useState(true)
    const navigate = useNavigate()
    const location = useLocation()

    useEffect(() => {
        fetchDocuments()
    }, [])

    const fetchDocuments = async () => {
        try {
            const res = await axios.get(`${API_URL}/documents?limit=100`)
            setDocuments(res.data || [])
        } catch (error) {
            console.error('Failed to fetch documents:', error)
        } finally {
            setLoading(false)
        }
    }

    const createDocument = async () => {
        try {
            const res = await axios.post(`${API_URL}/documents`, {
                title: 'Untitled',
                content: '# Untitled\n\nStart writing...'
            })
            await fetchDocuments()
            navigate(`/document/${res.data.id}`)
        } catch (error) {
            console.error('Failed to create document:', error)
        }
    }

    const isActive = (docId: string) => {
        return location.pathname === `/document/${docId}`
    }

    return (
        <div className={`sidebar ${isOpen ? '' : 'collapsed'}`}>
            <div className="sidebar-header">
                <div className="sidebar-title">Documents</div>
            </div>

            <div className="document-list">
                {loading ? (
                    <div className="loading">
                        <div className="spinner"></div>
                    </div>
                ) : documents.length === 0 ? (
                    <div style={{ padding: '20px', textAlign: 'center', color: 'var(--text-muted)' }}>
                        No documents yet
                    </div>
                ) : (
                    documents.map(doc => (
                        <div
                            key={doc.id}
                            className={`document-item ${isActive(doc.id) ? 'active' : ''}`}
                            onClick={() => navigate(`/document/${doc.id}`)}
                        >
                            <span className="document-icon">📄</span>
                            <span className="document-title">{doc.title}</span>
                        </div>
                    ))
                )}
            </div>

            <div className="new-doc-btn" onClick={createDocument}>
                + New Document
            </div>
        </div>
    )
}
