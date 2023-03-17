package data

import (
	"fmt"
	"strings"
)

type PrintFormat struct {
	addSpace    int
	alignFormat []string
	minWidth    []int
	colWidth    []int
	rows        [][]string
}

// Added row. Use \t for column break.
func (fp *PrintFormat) AddRow(str string) {
	// split string by \t
	rowParts := strings.Split(str, "\t")

	// Prepare length for format
	for i, thisPart := range rowParts {
		// check if a minimum width has been set for this column
		if len(fp.minWidth)-1 >= i {
			// if this column's content is shorter than the minimum width, pad it with spaces
			if len(thisPart) < fp.minWidth[i] {
				for j := len(thisPart); j < fp.minWidth[i]; j++ {
					thisPart = " " + thisPart
				}
			}
		}

		// If this column has not been added yet to col width tracker, add it
		if len(fp.colWidth)-1 < i {
			fp.colWidth = append(fp.colWidth, len(thisPart))
			continue
		}

		// if the current part is longer than the previous content of this column, update the width
		if len(thisPart) > fp.colWidth[i] {
			fp.colWidth[i] = len(thisPart)
		}
	}

	// Append rowparts to fp rows
	fp.rows = append(fp.rows, rowParts)
}

// Print everything in memory
func (fp *PrintFormat) Flush() {
	// Iterate through each row in the fp.rows slice
	for _, thisRow := range fp.rows {
		// Initialize an empty string to hold the formatted row
		printRow := ""
		// Iterate through each column/part in thisRow
		for i, thisRowPart := range thisRow {
			// Calculate the number of spaces needed to pad the current column
			addSpace := fp.colWidth[i] - len(thisRowPart)

			// Generate the string of spaces to use for padding
			spaceing := ""
			for j := addSpace; j > 0; j-- {
				spaceing += " "
			}

			// Determine whether to align the current column to the left or right
			alignRight := true
			if len(fp.alignFormat)-1 >= i {
				if fp.alignFormat[i] == "l" {
					alignRight = false
				}
			}

			// Add the current column/part to the formatted row, padding with spaces as needed
			if alignRight {
				printRow += spaceing + thisRowPart
			} else {
				if i >= len(thisRow)-1 { // If last column, do not add space at end
					printRow += thisRowPart
				} else {
					printRow += thisRowPart + spaceing
				}
			}

			// Add additional spacing between columns if specified
			for j := fp.addSpace; j > 0; j-- {
				printRow += " "
			}
		}
		// Print the formatted row to the console, excluding the final newline character
		fmt.Println(printRow[:len(printRow)-1])
	}

	// Clear the rows and column width data from the PrintFormat struct
	fp.rows = nil
	fp.colWidth = nil
}

// add space determin minimum space between, align format as slice of string with l or r (left or right)
// where index represent the column
func FormatPrint(addSpace int, alignFormat []string, minWidth []int) PrintFormat {
	// Create a new PrintFormat object with the following properties:
	var retVal = &PrintFormat{
		addSpace:    addSpace,     // - addSpace is set to the value of the addSpace argument
		alignFormat: alignFormat,  // - alignFormat is set to the value of the alignFormat argument
		minWidth:    minWidth,     // - minWidth is set to the value of the minWidth argument
		colWidth:    []int{0},     // - colWidth is initialized with a single element of 0
		rows:        [][]string{}, // - rows is initialized as an empty 2D slice
	}

	// Return the value of the PrintFormat object created above
	return *retVal
}
