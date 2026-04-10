import { BrowserRouter, Routes, Route } from "react-router-dom";
import { GamePage } from "./pages/GamePage";
import { EditorPage } from "./pages/EditorPage";
import { StoryEditorPage } from "./pages/StoryEditorPage";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<GamePage />} />
        <Route path="/editor" element={<EditorPage />} />
        <Route path="/editor/:storyId" element={<StoryEditorPage />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
