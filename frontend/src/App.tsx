import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { useState } from 'react'
import DocumentList from './components/DocumentList'
import DocumentEditor from './components/DocumentEditor'
import GraphView from './components/GraphView'
import TopBar from './components/TopBar'
import Sidebar from './components/Sidebar'
import './App.css'

function App() {
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [graphOpen, setGraphOpen] = useState(false)

  return (
    <BrowserRouter>
      <div className="app">
        <TopBar
          onToggleSidebar={() => setSidebarOpen(!sidebarOpen)}
          onToggleGraph={() => setGraphOpen(!graphOpen)}
        />

        <div className="main-content">
          <Sidebar isOpen={sidebarOpen} />

          <Routes>
            <Route path="/" element={<DocumentList />} />
            <Route path="/document/:id" element={<DocumentEditor />} />
          </Routes>
        </div>

        {graphOpen && <GraphView onClose={() => setGraphOpen(false)} />}
      </div>
    </BrowserRouter>
  )
}

export default App
