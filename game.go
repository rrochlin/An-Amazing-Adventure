package main

import (
	"fmt"
	"math"
)

func NewGame(x_dim, y_dim int) (Game, error) {
	g := Game{}
	g.Player.Pos = Position{0, 0}
	g.Player.Inventory = map[string]bool{}
	maze, err := GenerateMaze(x_dim, y_dim, int(math.Min(float64(x_dim), float64(y_dim))/2))
	if err != nil {
		return Game{}, err
	}
	g.Maze = maze
	g.M = x_dim
	g.N = y_dim
	return g, nil
}

func (g *Game) Describe() (map[Position]int, error) {
	val, _ := safeGet(g.Maze, g.Player.Pos)
	if val == 0 {
		return g.describeRoom()
	}
	return g.describeHall()

}

func (g *Game) describeHall() (map[Position]int, error) {
	newInfo := map[Position]int{}
	queue := make([]Position, 0)
	queue = append(queue, g.Player.Pos)
	directions := [4][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
	visited := g.BoolMatrix()

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, dir := range directions {
			check := current.Add(Position{dir[0], dir[1]})
			vis, ok := safeGet(visited, check)
			if !ok {
				continue
			}
			val := get(g.Maze, check)
			safeSet(visited, check, true)
			if vis || val == 0 { // not in a room or already seen
				fmt.Printf("room or visit: %v\n", check)
				continue
			}
			canSee := g.checkView(check)

			if !canSee { // can't see the block
				fmt.Printf("can't see: %v\n", check)
				continue
			}

			newInfo[check] = val
			queue = append(queue, check)
		}
	}

	return newInfo, nil
}

func (g *Game) describeRoom() (map[Position]int, error) {
	newInfo := map[Position]int{}
	queue := make([]Position, 0)
	queue = append(queue, g.Player.Pos)
	directions := [4][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
	visited := g.BoolMatrix()

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, dir := range directions {
			check := current.Add(Position{dir[0], dir[1]})
			vis, ok := safeGet(visited, check)
			if !ok {
				continue
			}
			val := get(g.Maze, check)
			if vis || val == 1 { // not in a room or already seen
				continue
			}

			safeSet(visited, check, true)
			newInfo[check] = val
			if val > 0 { // is the door
				continue
			}
			queue = append(queue, check)
		}
	}

	return newInfo, nil

}

func (g *Game) OpenDoor(door Position) error {
	val, ok := safeGet(g.Maze, door)
	if !ok {
		return fmt.Errorf("Not a valid door location %v\n", door)
	}
	if val != doorVal {
		return fmt.Errorf("This isn't a door")
	}
	tileType := get(g.Maze, g.Player.Pos)
	directions := [4][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
	for _, dir := range directions {
		check := door.Add(Position{dir[0], dir[1]})
		if val, ok := safeGet(g.Maze, check); ok && val != tileType {
			g.Player.Pos = check
			return nil
		}
	}

	return fmt.Errorf("Could not find a suitable tile to move player to\n")
}

func (g *Game) Move(to Position) error {
	val, ok := safeGet(g.Maze, to)
	if !ok {
		return fmt.Errorf("invalid move")
	}
	if val == 2 {
		g.OpenDoor(to)
	} else {
		g.Player.Pos = to
	}

	return nil
}

func (g *Game) checkView(check Position) bool {
	diff := check.Diff(g.Player.Pos)
	if diff.X == 0 && diff.Y == 0 {
		return true
	}
	slope := float64(diff.Y) / float64(diff.X)
	x_inc := math.Copysign(1, float64(diff.X))
	y_inc := math.Copysign(1, float64(diff.Y))
	fmt.Printf("slope: %v\n", slope)
	fmt.Printf("x_inc: %v\n", x_inc)
	fmt.Printf("y_inc: %v\n", y_inc)

	rise, run := 0, 0
	inPath := []Position{}
	current := check
	for !current.Equal(g.Player.Pos) {
		if math.Abs(float64(run)*slope) <= math.Abs(float64(rise)) {
			current.X -= int(x_inc)
			run += 1
		} else {
			current.Y -= int(y_inc)
			rise += 1
		}
		fmt.Printf("current: %v\n", current)
		inPath = append(inPath, current)
		if get(g.Maze, current) == 0 {
			return false
		}
	}
	return true

}
