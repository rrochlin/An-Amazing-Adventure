package main

import (
	"fmt"
	"time"
)

func main() {
	x_dim, y_dim := 40, 40
	game, err := NewGame(x_dim, y_dim)
	if err != nil {
		fmt.Printf("game creation failed %v\n", err)
		return
	}

	mazeMap := make([][]int, x_dim)
	for i := range mazeMap {
		mazeMap[i] = make([]int, y_dim)
		for j := range mazeMap {
			mazeMap[i][j] = 3
		}
	}

	fmt.Println("initial game state")
	PrintMaze(game.Maze, game.Player.Pos)

	for {
		time.Sleep(1 * time.Second)
		doors := []Position{}

		newPos, err := game.Describe()
		if err != nil {
			fmt.Printf("could not describe room %v\n", err)
			return
		}
		fmt.Printf("newPos: %v\n", newPos)
		for pos, val := range newPos {
			if val == 2 {
				doors = append(doors, pos)
			}
			mazeMap[pos.X][pos.Y] = val
		}
		PrintMaze(mazeMap, game.Player.Pos)
		fmt.Printf("game.Player.Pos: %v\n", game.Player.Pos)
		fmt.Printf("move somewhere: %v\n", doors)
		var x, y int
		fmt.Scan(&x, &y)
		err = game.Move(Position{x, y})
		if err != nil {
			fmt.Printf("invalid move: %v\n", err)
			return
		}
		fmt.Printf("game.Player.Pos: %v\n", game.Player.Pos)

	}

}

func PrintMaze(maze [][]int, player Position) {
	fmt.Printf("\033[2J\033[H")
	var playerRune rune = '😊'
	var door rune = '🚪'
	var wall rune = '🧱'
	var floor rune = '🟫'
	fog := "🌫️"
	for i := 0; i < len(maze)+2; i += 1 {
		fmt.Printf("%v", fog)
	}
	fmt.Println()
	for i := range maze {
		fmt.Printf("%v", fog)
		for j := range maze[i] {
			if player.X == i && player.Y == j {
				fmt.Printf("%c", playerRune)
			} else {
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
	Maze [][]int
	M    int
	N    int
}

type Position struct {
	X int
	Y int
}

func (p *Position) Add(o Position) Position {
	return Position{p.X + o.X, p.Y + o.Y}
}

func (p *Position) Diff(o Position) Position {
	return Position{p.X - o.X, p.Y - o.Y}
}

func (p *Position) Equal(o Position) bool {
	return p.X == o.X && p.Y == o.Y
}

func (p *Position) IsInBounds(xmax, ymax int) bool {
	return p.X < xmax && p.Y < ymax && p.X >= 0 && p.Y >= 0
}

func (g *Game) BoolMatrix() [][]bool {
	v := make([][]bool, g.M)
	for i := range g.N {
		v[i] = make([]bool, g.N)
	}
	return v
}
