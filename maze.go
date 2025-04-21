package main

import (
	"fmt"
	"math"
	"math/rand"
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
	type QT struct {
		val int
		pos Position
	}
	queue := []QT{{2, Position{0, 0}}, {3, Position{x - 1, y - 1}}}
	original := []Position{queue[0].pos, queue[1].pos}
	for j := range n {
		point := Position{rand.Intn(x), rand.Intn(y)}
		valid := true
		for i := range queue {
			if calcDistance(queue[i].pos, point) < 6 {
				valid = false
				break
			}
		}
		if valid {
			queue = append(queue, QT{j + 4, point})
			original = append(original, point)
		}
	}
	directions := [4][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, dir := range directions {
			check := current.pos.Add(Position{dir[0], dir[1]})
			vis, ok := safeGet(visited, check)
			if !ok { // out of bounds continue
				continue
			}
			val := get(maze, check)
			if vis && val == current.val { // rexploring own room, continue
				continue
			}
			if !vis {
				safeSet(maze, check, current.val)
				safeSet(visited, check, true)
				queue = append(queue, QT{current.val, check})
				continue
			}
			safeSet(maze, current.pos, 1)
			break
		}
	}
	thickness := 4
	for i := range x {
		wall := false
		count := 0
		for j := range y {
			if maze[i][j] > 1 {
				maze[i][j] = 0
			}
			if maze[i][j] == 1 { // in a wall
				wall = true
				count += 1
				continue
			}
			if !wall { // in a room
				continue
			}
			if count >= thickness { // left a wall that's big enough
				count = 0
				wall = false
				continue
			}
			maze[i][j] = 1
			count += 1
		}
	}
	for j := range y {
		wall := false
		count := 0
		for i := range x {
			if maze[i][j] == 1 { // in a wall
				wall = true
				count += 1
				continue
			}
			if !wall { // in a room
				continue
			}
			if count >= thickness { // left a wall that's big enough
				count = 0
				wall = false
				continue
			}
			maze[i][j] = 1
			count += 1
		}
	}

	for i := range x {
		for j := range y {
			visited[i][j] = false
		}
	}
	for pos := range original {
		q := []Position{original[pos]}
		for len(q) > 0 {
			current := q[0]
			q = q[1:]
			door := false
			for _, dir := range directions {
				check := current.Add(Position{dir[0], dir[1]})
				vis, ok := safeGet(visited, check)
				if !ok || vis { // out of bounds continue
					continue
				}
				val := get(maze, check)
				if val == 1 {
					safeSet(maze, check, 2)
					door = true
					break
				}
				safeSet(visited, check, true)
				q = append(q, check)
			}
			if door {
				break
			}
		}
	}

	return maze, nil
}

func safeGet[T any](slice [][]T, index Position) (T, bool) {
	var zero T
	if len(slice) == 0 {
		return zero, false
	}
	if !index.IsInBounds(len(slice), len(slice[0])) {
		return zero, false
	}
	return get(slice, index), true
}

func get[T any](slice [][]T, index Position) T {
	return slice[index.X][index.Y]
}

func safeSet[T any](slice [][]T, index Position, val T) {
	if len(slice) == 0 {
		return
	}
	if !index.IsInBounds(len(slice), len(slice[0])) {
		return
	}
	slice[index.X][index.Y] = val
}

func calcDistance(p, q Position) float64 {
	return math.Sqrt(math.Pow(float64(q.X-p.X), 2) + math.Pow(float64(q.Y-p.Y), 2))
}
