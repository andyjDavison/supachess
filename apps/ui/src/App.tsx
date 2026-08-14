import CpuChessGame from "./features/cpu-chess-game";
import "./index.css";

function App() {
  return (
    <main className="h-screen w-screen overflow-hidden flex items-center justify-center">
      <div className="w-1/3">
        <CpuChessGame />
      </div>
    </main>
  );
}

export default App;
