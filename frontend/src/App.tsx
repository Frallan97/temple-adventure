import { BrowserRouter, Routes, Route } from "react-router-dom";
import { GamePage } from "./pages/GamePage";
import { EditorPage } from "./pages/EditorPage";
import { StoryEditorPage } from "./pages/StoryEditorPage";
import { GameLogList, GameLogDetail } from "./pages/GameLogPage";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<GamePage />} />
        <Route path="/logs" element={<GameLogList />} />
        <Route path="/logs/:id" element={<GameLogDetail />} />
        <Route path="/editor" element={<EditorPage />} />
        <Route path="/editor/:storyId" element={<StoryEditorPage />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
