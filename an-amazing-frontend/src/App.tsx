import './App.css'
import { Stage, Layer, Rect } from 'react-konva';
import axios from 'axios';
import { useEffect, useState } from 'react';


function App() {

  const [newPositions, setNewPositions] = useState<position[]>([])
  const [maze, setMaze] = useState<mazeBlock[]>()
  const [player, setPlayer] = useState<position>()

  const inc = Math.min(window.innerWidth, window.innerHeight) / Math.sqrt(Math.max(1, (maze || []).length)) / 2
  const getMap = async () => {
    const result: startgameResponse = (await axios.get("http://localhost:3000/api/startgame")).data
    console.log("positions")
    console.log(result.positions)
    const mazeTemp = []
    for (let i = 0; i < result.positions.length; i++) {
      for (let j = 0; j < result.positions[0].length; j++) {
        mazeTemp.push({ x: i, y: j, key: i * result.positions.length + j, val: result.positions[i][j] })

      }
    }
    setMaze(mazeTemp)

    setPlayer(result.player)
  }


  useEffect(() => {
    getMap().then(() => {
      console.log("maze is set")
      console.log(maze)
    }
    )
  }, [])



  return (
    <Stage width={window.innerWidth / 1.1} height={window.innerHeight / 1.1}>
      <Layer draggable>
        {maze && maze.map((block: mazeBlock) => {
          console.log("writing")
          return (
            <Rect
              key={block.key}
              x={block.x * inc}
              y={block.y * inc}
              width={inc}
              height={inc}
              fill={extendedColors[block.val]}
            />
          )
        })
        }
      </Layer>
    </Stage>
  );
};

export default App

type position = {
  x: number;
  y: number;
}

type mazeBlock = {
  x: number;
  y: number;
  key: number;
  val: number;
}

type startgameResponse = {
  positions: number[][];
  player: position;
  cols: number;
  rows: number;
}

const extendedColors = [
  '#1f77b4', // Muted Blue
  '#ff7f0e', // Safety Orange
  '#2ca02c', // Cooked Asparagus Green
  '#d62728', // Brick Red
  '#9467bd', // Muted Purple
  '#8c564b', // Cast Iron Brown
  '#e377c2', // Raspberry Yogurt Pink
  '#7f7f7f', // Middle Gray
  '#bcbd22', // Curry Yellow-Green
  '#17becf', // Blue-Teal
  '#aec7e8', // Light Blue
  '#ffbb78', // Peach
  '#98df8a', // Light Green
  '#ff9896', // Light Red
  '#c5b0d5', // Light Purple
  '#c49c94', // Light Brown
  '#f7b6d2', // Light Pink
  '#c7c7c7', // Light Gray
  '#dbdb8d', // Light Yellow-Green
  '#9edae5'  // Light Blue-Teal
];
