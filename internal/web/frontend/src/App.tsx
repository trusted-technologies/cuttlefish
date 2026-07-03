import { BrowserRouter, Routes, Route } from "react-router-dom"
import { ThemeProvider } from "@/components/theme-provider"
import IndexPage from "@/pages/Index"
import SlavePage from "@/pages/Slave"

function App() {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="cuttlefish-theme">
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<IndexPage />} />
          <Route path="/slave/:id" element={<SlavePage />} />
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  )
}

export default App
