package main

import (
	"fmt"
	"time"
)

func main() {
	_, err := GenerateMaze(40, 40, 15)
	if err != nil {
		fmt.Printf("could not gen maze %v\n", err)
		return
	}
	return

	game, err := NewGame(7, 7)
	if err != nil {
		fmt.Printf("game creation failed %v\n", err)
		return
	}

	mazeMap := make([][]int, 7)
	for i := range mazeMap {
		mazeMap[i] = make([]int, 7)
		for j := range mazeMap {
			mazeMap[i][j] = 3
		}
	}

	fmt.Println("initial game state")
	PrintMaze(game.Maze, []Position{game.Player.Pos})

	for {
		time.Sleep(1 * time.Second)
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
		PrintMaze(mazeMap, []Position{game.Player.Pos})
		if (Position{}) == door {
			fmt.Println("You Escaped!")
			return
		}

		err = game.OpenDoor(door)
		if err != nil {
			fmt.Printf("could not open door %v\n", err)
			return

		}
	}

}

func PrintMaze(maze [][]int, players []Position) {
	// fmt.Printf("\033[2J\033[H")
	var playerRune rune = '😊'
	var door rune = '🚪'
	var wall rune = '🧱'
	var floor rune = '🟫'
	fog := "🌫️"
	for i := range players {
		safeSet(maze, players[i], 4)
	}
	for i := 0; i < len(maze)+2; i += 1 {
		fmt.Printf("%v", fog)
	}
	fmt.Println()
	for i := range maze {
		fmt.Printf("%v", fog)
		for j := range maze[i] {
			switch maze[i][j] {
			case 0:
				fmt.Printf("%c", floor)
			case 1:
				fmt.Printf("%c", wall)
			case 2:
				fmt.Printf("%c", door)
			case 3:
				fmt.Printf("%v", fog)
			case 4:
				fmt.Printf("%c", playerRune)
			}

		}
		fmt.Printf("%v\n", fog)
	}
	for i := 0; i < len(maze)+2; i += 1 {
		fmt.Printf("%v", fog)
	}
	fmt.Println()
}

const doorVal = 2

type Game struct {
	Player struct {
		Pos       Position
		Inventory map[string]bool
	}
	Maze    [][]int
	Visited [][]bool
}

type Position struct {
	X int
	Y int
}

func (p *Position) IsInBounds(xmax, ymax int) bool {
	return p.X < xmax && p.Y < ymax && p.X >= 0 && p.Y >= 0
}
