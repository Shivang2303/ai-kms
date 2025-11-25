interface TopBarProps {
    onToggleSidebar: () => void
    onToggleGraph: () => void
}

export default function TopBar({ onToggleSidebar, onToggleGraph }: TopBarProps) {
    return (
        <div className="top-bar">
            <button className="btn" onClick={onToggleSidebar} title="Toggle Sidebar">
                ☰
            </button>

            <div className="logo">
                <span>⚡</span>
                <span>AI-KMS</span>
            </div>

            <div className="search-bar">
                <input type="text" placeholder="Search documents..." />
            </div>

            <div className="top-actions">
                <button className="btn" onClick={onToggleGraph}>
                    🔗 Graph
                </button>
                <button className="btn-primary btn">
                    📤 Share
                </button>
            </div>
        </div>
    )
}
