package main

import (
	"fmt"
	"math/rand"
	"time"
)

func GenerateMaze(x, y, n int) ([][]int, error) {
	if x < 20 || y < 20 {
		return nil, fmt.Errorf("Invalid Maze size requested")
	}
	maze := make([][]int, x)
	visited := make([][]bool, x)
	for i := range maze {
		maze[i] = make([]int, y)
		visited[i] = make([]bool, y)
	}
	queue := []Position{{0, 0}, {x, y}}
	for range n {
		queue = append(queue, Position{rand.Intn(x + 1), rand.Intn(y + 1)})
	}
	directions := [4][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		border := true
		PrintMaze(maze, Position{})
		time.Sleep(5 * time.Millisecond)
		for _, dir := range directions {
			check := current
			check.X += dir[0]
			check.Y += dir[1]
			if vis, ok := safeGet(visited, check); vis == ok {
				continue
			}
			safeSet(visited, check, true)
			queue = append(queue, check)
			border = false
		}
		if border {
			safeSet(maze, current, 1)
		}
	}

	return maze, nil
}

func safeGet[T any](slice [][]T, index Position) (T, bool) {
	var zero T
	if !index.IsInBounds(len(slice), len(slice[0])) {
		return zero, false
	}
	return slice[index.X][index.Y], true
}

func safeSet[T any](slice [][]T, index Position, val T) {
	if !index.IsInBounds(len(slice), len(slice[0])) {
		return
	}
	slice[index.X][index.Y] = val
}
