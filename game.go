package main

import (
	"fmt"
)

func NewGame(x_dim, y_dim int) (Game, error) {
	g := Game{}
	g.Player.Pos = Position{0, 0}
	g.Player.Inventory = map[string]bool{}
	//g.Maze = make([][]int, y_dim)
	//for i := range g.Maze {
	//	g.Maze[i] = make([]int, x_dim, 0)
	//}
	// 1 is wall, 0 is a room
	g.Visited = make([][]bool, x_dim)
	for i := range g.Visited {
		g.Visited[i] = make([]bool, y_dim)
	}
	g.Maze = [][]int{
		{0, 0, 2, 0, 0, 0, 0},
		{0, 0, 1, 1, 1, 1, 0},
		{1, 1, 1, 1, 1, 1, 0},
		{0, 0, 0, 1, 1, 1, 0},
		{0, 0, 0, 2, 0, 0, 0},
		{0, 0, 0, 1, 1, 1, 1},
		{0, 0, 0, 2, 0, 0, 0},
	}

	return g, nil
}

func PrintMaze(maze [][]int) {
	for i := range maze {
		fmt.Printf("%v\n", maze[i])
	}
}

func (g *Game) DescribeRoom() (map[Position]int, error) {
	if ok, _ := g.tryVisit(g.Player.Pos); !ok {
		return nil, nil
	}
	newInfo := map[Position]int{}
	mazeVal, err := g.getMazeValue(g.Player.Pos)
	if err != nil {
		return nil, err
	}
	newInfo[g.Player.Pos] = mazeVal
	queue := make([]Position, 0)
	queue = append(queue, g.Player.Pos)
	directions := [4][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
	fmt.Printf("queue: %v\n", queue)

	for len(queue) > 0 {
		current := queue[0]
		fmt.Printf("checking position: %v\n", current)
		queue = queue[1:]
		for _, dir := range directions {
			check := current
			check.X += dir[0]
			check.Y += dir[1]
			if ok, _ := g.tryVisit(check); !ok {
				continue
			}
			val, err := g.getMazeValue(check)
			if err != nil {
				return nil, err
			}
			newInfo[check] = val
			if val > 0 {
				continue
			}
			queue = append(queue, check)
		}
	}

	return newInfo, nil

}

func (g *Game) OpenDoor(door Position) error {
	val, err := g.getMazeValue(door)
	if err != nil {
		return err
	}
	if !g.Visited[door.X][door.Y] {
		return fmt.Errorf("You have not discovered this location")
	}
	if val != doorVal {
		return fmt.Errorf("This isn't a door")
	}

	g.Player.Pos = door
	err = g.setMazeValue(door, 0)
	if err != nil {
		return err
	}
	err = g.setVisit(door, false)
	if err != nil {
		return err
	}

	return nil

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

func (g *Game) getMazeValue(p Position) (int, error) {
	err := g.boundsCheckPos(p)
	if err != nil {
		return 0, err
	}
	return g.Maze[p.X][p.Y], nil
}

func (g *Game) setMazeValue(p Position, val int) error {
	err := g.boundsCheckPos(p)
	if err != nil {
		return err
	}
	g.Maze[p.X][p.Y] = val
	return nil
}

func (g *Game) tryVisit(p Position) (bool, error) {
	err := g.boundsCheckPos(p)
	if err != nil {
		return false, err
	}
	if g.Visited[p.X][p.Y] {
		return false, nil
	}
	g.Visited[p.X][p.Y] = true
	return true, nil
}

func (g *Game) setVisit(p Position, state bool) error {
	err := g.boundsCheckPos(p)
	if err != nil {
		return err
	}
	g.Visited[p.X][p.Y] = state
	return nil
}

func (g *Game) boundsCheckPos(p Position) error {
	if len(g.Maze) <= p.X || p.X < 0 {
		return fmt.Errorf("invalid position access X bounds %v", p)
	}
	if len(g.Maze[0]) <= p.Y || p.Y < 0 {
		return fmt.Errorf("invalid position access Y bounds %v", p)
	}
	return nil
}
