import "./index.css";
import { Chessboard } from "react-chessboard";

function App() {
  const chessboardOptions = {};

  return (
    <main className="h-screen w-screen overflow-hidden flex items-center justify-center">
      <div className="w-1/3">
        <Chessboard options={chessboardOptions} />
      </div>
    </main>
  );
}

export default App;
