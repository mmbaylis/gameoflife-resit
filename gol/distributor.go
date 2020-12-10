package gol

import (
	"strconv"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioInput    chan uint8
}


func findAliveNeighbours(world [][]byte, col int, row int) int {
	aliveNeighbours := 0
	for _, i := range []int{-1,0,1} {
		for _, j := range []int{-1,0,1} {
			if i == 0 && j == 0 {
				continue
			}

			living := world[(col+i+len(world))%len(world)][(row+j+len(world[0]))%len(world[0])] !=0
			if living {
				aliveNeighbours++
			}
		}
	}
	return aliveNeighbours
}

func calculateNextState(p Params, world [][]byte) [][]byte {

	newWorld := make([][]byte, len(world))

	for i := range newWorld {
		newWorld[i] = make([]byte, len(world[i]))
		copy(newWorld[i], world[i])
	}

	for col := 0; col < len(world); col++ {
		for row := 0; row < len (world[0]); row++ {
			/*find number of alive neighbours */
			aliveNeighbours := findAliveNeighbours(world, col, row)

			if world[col][row] !=0 {
				/* if current cell is not dead */

				if aliveNeighbours < 2 {
					newWorld[col][row] = 0
				}

				if aliveNeighbours > 3 {
					newWorld[col][row] = 0
				}
			}
			if world[col][row] == 0 {
				/* if current cell is dead */

				if aliveNeighbours == 3 {
					newWorld[col][row] = 0xFF
				}
			}
		}
	}

	return newWorld
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {

	aliveCells := []util.Cell{}

	for x, col := range world {
		for y, v := range col {
			if v != 0 {
				aliveCells = append(aliveCells, util.Cell{y, x})
			}
		}
	}

	return aliveCells
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	height := strconv.Itoa(p.ImageHeight)
	width := strconv.Itoa(p.ImageWidth)

	c.ioCommand <- ioInput
	c.ioFilename <- height + "x" + width
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			world[i][j] = <- c.ioInput
			if world[i][j] == 255{
				c.events <- CellFlipped{0, util.Cell{i,j}}
				//send a cell flipped event
			}
		}
	}

	// TODO: For all initially alive cells send a CellFlipped Event.

	turn := p.Turns

	// TODO: Execute all turns of the Game of Life.
	for i := 1; i <= turn; i++ {
		world = calculateNextState(p, world)
		aliveCells := calculateAliveCells(p, world)
		for _, cell := range aliveCells{
			c.events <- CellFlipped{0, cell}
		}
		c.events <- TurnComplete{i}
	}

	c.events <- FinalTurnComplete{turn, calculateAliveCells(p, world)}

	// TODO: Send correct Events when required, e.g. CellFlipped, TurnComplete and FinalTurnComplete.
	// 	See event.go for a list of all events.

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
