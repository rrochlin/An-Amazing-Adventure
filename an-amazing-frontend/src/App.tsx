import './App.css'
import { Stage, Layer, Rect, Circle, Text } from 'react-konva';
import { Axios } from 'axios';

function App() {

  return (
    <Stage width={window.innerWidth} height={window.innerHeight}>
      <Layer
        draggable
      >
        <Text text="Try to drag shapes" fontSize={15} />
        <Rect
          x={20}
          y={50}
          width={100}
          height={100}
          fill="red"
          shadowBlur={10}
        />
        <Circle
          x={200}
          y={100}
          radius={50}
          fill="green"
        />
      </Layer>
    </Stage>
  );
};

export default App
