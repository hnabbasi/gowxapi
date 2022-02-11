package main

import (
	"errors"
	"fmt"
	"log"
)

func main() {
	random()
}

var Prefix string = "logger:"

func debug(message string) {
	msg := fmt.Sprintf("%v%v", Prefix, message)
	log.Println(msg)
}

func error(message string) {
	log.Fatal(errors.New(message))
}

func random() {
	input := [][]int{[]int{1, 1, 1}, []int{1, 0, 1}, []int{1, 1, 1}}
	fmt.Printf("before:\t%v\n", input)
	setZeroes(input)
	fmt.Printf("after:\t%v\n", input)
}

func setZeroes(matrix [][]int) {
	m := len(matrix)
	n := len(matrix[0])

	visited := map[string]bool{}

	for r := 0; r < m; r++ {
		for c := 0; c < n; c++ {
			if !visited[fmt.Sprintf("%v,%v", r, c)] && matrix[r][c] == 0 {
				processRowAndColumn(matrix, visited, r, c)
			}
		}
	}
}

func processRowAndColumn(matrix [][]int, visited map[string]bool, row int, col int) {
	for r := 0; r < len(matrix); r++ {
		visit := fmt.Sprintf("%v,%v", r, col)
		if !visited[visit] && matrix[r][col] != 0 {
			matrix[r][col] = 0
			visited[visit] = true
		}
	}
	for c := 0; c < len(matrix[0]); c++ {
		visit := fmt.Sprintf("%v,%v", row, c)
		if !visited[visit] && matrix[row][c] != 0 {
			matrix[row][c] = 0
			visited[visit] = true
		}
	}
}
