package main

import (
	"fmt"
)

func main() {
	game, err := NewGame(7, 7)
	if err != nil {
		fmt.Printf("game creation failed %v\n", err)
		return
	}

	mazeMap := make([][]int, 7)
	for i := range mazeMap {
		mazeMap[i] = make([]int, 7)
	}

	fmt.Println("initial game state")
	PrintMaze(game.Maze)

	for {
		door := Position{}

		newPos, err := game.DescribeRoom()
		if err != nil {
			fmt.Printf("could not describe room %v\n", err)
			return
		}
		for pos, val := range newPos {
			if val == 2 {
				door = pos
			}
			mazeMap[pos.X][pos.Y] = val
		}
		fmt.Println("Current Map State")
		PrintMaze(mazeMap)
		if (Position{}) == door {
			fmt.Println("no more doors")
			return
		}

		fmt.Println("You Escaped!")
		err = game.OpenDoor(door)
		if err != nil {
			fmt.Printf("could not open door %v\n", err)
			return

		}
	}

}
