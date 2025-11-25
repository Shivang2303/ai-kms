import { useEffect, useRef, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Collaboration from '@tiptap/extension-collaboration'
import CollaborationCursor from '@tiptap/extension-collaboration-cursor'
import * as Y from 'yjs'
import { WebsocketProvider } from 'y-websocket'

const colors = ['#ff8935', '#ffa05a', '#ff7420', '#ffb67a', '#ff9f4a']

export default function DocumentEditor() {
    const { id } = useParams<{ id: string }>()
    const [connected, setConnected] = useState(false)
    const [userCount, setUserCount] = useState(1)

    // Create Yjs document (persistent across renders)
    const ydoc = useMemo(() => new Y.Doc(), [])
    const providerRef = useRef<WebsocketProvider>()

    const userName = useMemo(() =>
        'User ' + Math.floor(Math.random() * 1000), []
    )

    const userColor = useMemo(() =>
        colors[Math.floor(Math.random() * colors.length)], []
    )

    useEffect(() => {
        // Connect to WebSocket for real-time sync
        const provider = new WebsocketProvider(
            'ws://localhost:8080/ws/document/' + id,
            id!,
            ydoc
        )

        provider.on('status', (event: { status: string }) => {
            setConnected(event.status === 'connected')
        })

        provider.on('sync', () => {
            const awareness = provider.awareness
            setUserCount(awareness.getStates().size)
        })

        providerRef.current = provider

        return () => {
            provider.destroy()
        }
    }, [id, ydoc])

    const editor = useEditor({
        extensions: [
            StarterKit.configure({
                history: false  // Disable history, Yjs handles it
            }),
            Collaboration.configure({
                document: ydoc,
            }),
            CollaborationCursor.configure({
                provider: providerRef.current,
                user: {
                    name: userName,
                    color: userColor,
                },
            }),
        ],
        editorProps: {
            attributes: {
                class: 'ProseMirror',
            },
        },
    }, [ydoc])

    if (!editor) {
        return (
            <div className="editor-container">
                <div className="loading">
                    <div className="spinner"></div>
                    <span style={{ marginLeft: '12px' }}>Loading editor...</span>
                </div>
            </div>
        )
    }

    return (
        <div className="editor-container">
            <div className="editor-header">
                <input
                    type="text"
                    className="editor-title-input"
                    placeholder="Untitled"
                    defaultValue="Untitled"
                />

                <div className="collaboration-bar">
                    <div className="user-avatars">
                        {[...Array(Math.min(userCount, 5))].map((_, i) => (
                            <div
                                key={i}
                                className={`avatar ${i === 0 ? 'typing' : ''}`}
                                style={{ background: colors[i % colors.length] }}
                            >
                                {String.fromCharCode(65 + i)}
                            </div>
                        ))}
                        {userCount > 5 && (
                            <div className="avatar" style={{ background: 'var(--bg-tertiary)' }}>
                                +{userCount - 5}
                            </div>
                        )}
                    </div>

                    <button
                        className="btn-primary btn"
                        onClick={() => {
                            const url = window.location.href
                            navigator.clipboard.writeText(url)
                            alert('Link copied! Share it with others to collaborate.')
                        }}
                    >
                        📤 Share
                    </button>

                    <div style={{
                        fontSize: '12px',
                        color: connected ? 'var(--accent-primary)' : 'var(--text-muted)'
                    }}>
                        {connected ? '● Connected' : '○ Connecting...'}
                    </div>
                </div>
            </div>

            <div className="infinite-canvas">
                <div className="editor-content">
                    <EditorContent editor={editor} />
                </div>
            </div>
        </div>
    )
}
