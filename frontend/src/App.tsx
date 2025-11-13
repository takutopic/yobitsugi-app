import { Routes, Route } from 'react-router-dom'
import { HomePage } from './HomePage'
import { JobDetailPage } from './JobDetailPage'
import './App.css'

function App() {
  // Define routing
  return (
    <div>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/job/:id" element={<JobDetailPage />} />
      </Routes>
    </div>
  )
}

export default App