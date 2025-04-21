package main

import (
	"fmt"
)

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
