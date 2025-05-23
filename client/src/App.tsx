import './App.css'
import { Stage, Layer, Rect } from 'react-konva';
import axios from 'axios';
import { useEffect, useState } from 'react';
import TextField from '@mui/material/TextField'
import Konva from 'konva';
import Typography from '@mui/material/Typography';
import { Button } from '@mui/material';

const APP_URI = import.meta.env.VITE_APP_URI


function App() {

  //const [newPositions, setNewPositions] = useState<Map<position, number>>()
  const [maze, setMaze] = useState<mazeBlock[]>()
  const [player, setPlayer] = useState<position>()
  //const [command, setCommand] = useState("")
  const [cols, setCols] = useState(0)
  const [rows, setRows] = useState(0)
  const [chatResponse, setChatResponse] = useState("")
  const [chatRequest, setChatRequest] = useState("Where am I?")

  const inc = Math.min(window.innerWidth, window.innerHeight) / Math.sqrt(Math.max(1, (maze || []).length)) / 1.4
  const getMap = async () => {
    const result: startgameResponse = (await axios.post(`${APP_URI}startgame`,
      { columns: 25, rows: 25 })
    ).data
    console.log("positions")
    const mazeTemp = []
    setCols(result.cols)
    setRows(result.rows)
    for (let i = 0; i < result.rows; i++) {
      for (let j = 0; j < result.cols; j++) {
        mazeTemp.push({ x: i, y: j, key: i * 100 + j, val: 5 })
      }
    }
    setMaze(mazeTemp)

    setPlayer(result.player)
    await submitChat()
  }

  //type retVal struct {
  //	Positions map[Position]int `json:"positions"`
  //	Player    Position         `json:"player"`
  //}
  //

  type describeReq = {
    positions: position[],
    values: number[],
    player: position
  }

  const RequestDescribe = async () => {
    if (!maze) return
    const result = await axios.get<describeReq>(`${APP_URI}describe`)
    if (result.status != 200) {
      console.log("error happened\n", result)
      return
    }
    console.log(result)
    const newPositions = result.data.positions
    const newValues = result.data.values
    setPlayer(result.data.player)
    for (let i = 0; i < newPositions.length; i++) {
      const index = newPositions[i].X * rows + newPositions[i].Y
      maze[index].val = newValues[i]
    }


  }


  useEffect(() => {
    getMap()
  }, [])


  useEffect(() => {
    RequestDescribe()
  }, [maze])




  const RequestMove = async (move: position) => {
    axios.post(`${APP_URI}move`, {
      position: move
    }).then(function(res) {
      if (res.status != 200) {
        console.log("error non 200 response", res)
      }
      RequestDescribe()
    }).catch(function(err) {
      console.log(err)
    })
  }


  const handleClick = (event: Konva.KonvaEventObject<MouseEvent>) => {
    const move: position = {
      X: Math.floor(event.target.index / rows),
      Y: event.target.index % cols
    }
    if (maze && maze[event.target.index].val == 5) {
      alert("invalid move")
      return
    }
    RequestMove(move)


  }


  const submitChat = async () => {
    const result = await axios.post(`${APP_URI}chat`, { chat: chatRequest })
    if (result.status != 200) {
      console.log("error getting chat")
      alert(result)
    }
    console.log(result)
    setChatResponse(result.data.Response)
    await RequestDescribe()

  }


  return (
    <div>
      <Stage width={window.innerWidth} height={window.innerHeight / 1.2}>
        <Layer>
          {maze && player && maze.map((block: mazeBlock) => {
            const color = !(player.X == block.x && player.Y == block.y) ?
              extendedColors[block.val] : extendedColors[3]

            return (
              <Rect
                key={block.key}
                x={block.x * inc}
                y={block.y * inc}
                width={inc}
                height={inc}
                fill={color}
                onClick={handleClick}
              />
            )
          })
          }
        </Layer>
      </Stage>
      <Typography variant='body1'>
        {chatResponse}
      </Typography>
      <div>
        <TextField
          id="outlined-basic"
          label="Input"
          variant="outlined"
          multiline

          value={chatRequest}
          onChange={(event: React.ChangeEvent<HTMLInputElement>) => {
            setChatRequest(event.target.value);
          }}
        />
        <Button variant="contained" onClick={submitChat}>
          Submit
        </Button>
      </div>

    </div>
  );
};

export default App

type position = {
  X: number;
  Y: number;
}

type mazeBlock = {
  x: number;
  y: number;
  key: number;
  val: number;
}

type startgameResponse = {
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
