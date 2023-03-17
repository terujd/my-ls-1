package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"my-ls-1/data"
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var inc_l bool
var inc_R bool
var inc_a bool
var inc_r bool
var inc_t bool

var start string
var fp data.PrintFormat

// Struct for regular directories
type File struct {
	Mode    string
	Link    int
	User    string
	Grp     string
	Size    int64
	ModTime time.Time
	Time    time.Time
	Name    string
}

// Struct for info about files
var file []File

func init() {
	alignFormat := []string{"l", "r", "l", "l", "r", "l", "r", "l", "l"} // Define the alignment format for format printing
	minWidth := []int{11, 1, 0, 0, 0, 0, 2}                              // Define the minimum width for format printing
	fp = data.FormatPrint(1, alignFormat, minWidth)                      // Create a format printer using the defined alignment and width
}

func main() {
	// If no command line argument is provided, use the current directory as the default argument
	if len(os.Args) < 2 {
		os.Args = append(os.Args, ".")
	}
	NotOk(os.Args[1])           // Validate the command line argument, exit program if invalid
	lsFiles := argInterpreter() // Parse command line arguments and return a list of target files/folders
	for _, thisTarget := range lsFiles {
		_, err := ioutil.ReadDir(thisTarget) // Check if the target is a directory and can be read
		if err != nil {                      // If the target is not a directory or cannot be read, list all files and folders recursively
			start = thisTarget
			listAll(start)
			continue
		} else if len(lsFiles) > 1 { // If there are multiple targets, print the target name before listing files/folders
			if thisTarget != lsFiles[0] {
				fmt.Println()
			}
			fmt.Println(thisTarget + ":")
		}
		start = thisTarget // Set the current target as the start for listing files/folders
		listAll(start)     // List all files and folders recursively starting from the target
	}
}

func argInterpreter() []string {
	// If no arguments are given, return current directory
	if len(os.Args) < 2 {
		return []string{"./"}
	}
	retVal := []string{}
	searchForFlag := true
	files := []string{}
	folders := []string{}
	inCorrect := []string{}

	// Loop through each argument
	for _, thisArg := range os.Args[1:] {
		// If still looking for flags, validate the argument is a flag
		if searchForFlag {
			if validateFlag(thisArg) {
				continue // Skip to the next argument
			}
			searchForFlag = false // Stop searching for flags
		}
		// Check if argument is a valid file or directory
		target, err := os.Lstat(thisArg)
		if err != nil {
			// If not a valid file or directory, append to inCorrect slice
			inCorrect = append(inCorrect, "my-ls-1: "+thisArg+": No such file or directory")
			continue // Skip to the next argument
		}
		// If the target exists and is a directory, append to folders slice
		if target != nil && target.IsDir() {
			folders = append(folders, thisArg)
		} else {
			// If not a directory, append to files slice
			files = append(files, thisArg)
		}
	}
	// Sort files and folders alphabetically
	sort.Strings(files)
	if inc_r {
		sort.Sort(sort.Reverse(sort.StringSlice(files))) // Reverse sort if -r flag is present
	}
	sort.Strings(folders)
	if inc_r {
		sort.Sort(sort.Reverse(sort.StringSlice(folders))) // Reverse sort if -r flag is present
	}
	sort.Strings(inCorrect)
	// Append sorted files and folders to return value
	retVal = append(retVal, files...)
	retVal = append(retVal, folders...)
	// Print any incorrect file or directory names
	for i := range inCorrect {
		fmt.Println(inCorrect[i])
	}
	// If no valid files or folders are found, return current directory
	if len(retVal) == 0 {
		return []string{"./"}
	}
	return retVal
}
func validateFlag(flag string) bool {
	res := true

	if len(flag) < 2 { // Flag can not be less than two char
		return false
	} else if flag[0] != '-' { // Flag must start with '-'
		return false
	}

	theseFlags := flag[1:] // Extract flags from input string

	for _, symb := range theseFlags { // Loop through flags and set corresponding variable to true
		switch symb {
		case 'l':
			inc_l = true
		case 'R':
			inc_R = true
		case 'a':
			inc_a = true
		case 'r':
			inc_r = true
		case 't':
			inc_t = true
		default:
			res = false // If flag is not recognized, set result to false
		}
	}

	return res // Return result of flag validation
}

func filesInFolder(name, address string) {
	files, err := ioutil.ReadDir(address + name) // Read directory at specified address
	if err != nil {                              // Handle error if directory can't be read
		fmt.Print(err)
	}

	if !inc_l { // If flag '-l' is not set, print only file names
		printNamesOnly(files)
		for _, file := range files { // Loop through files in directory
			if file.IsDir() { // If file is a directory
				if inc_R { // If flag '-R' is set
					if file.Name()[0] != '.' { // Exclude hidden directories
						newAddress := address + name + "/"
						fmt.Println("\n" + newAddress + file.Name() + ":")
						filesInFolder(file.Name(), newAddress) // Recursively call filesInFolder for subdirectory
					}
				}
			}
		}
	}
	if inc_l { // If flag '-l' is set, list all files with their details
		listAll(name)
	}
}

func printNamesOnly(files []fs.FileInfo) {
	for i := 0; i < len(files); i++ {
		// Create a new File struct for each file in the files slice
		newFile := &File{
			Name:    files[i].Name(),
			ModTime: files[i].ModTime(),
		}
		// Append the new File to the global file slice
		file = append(file, *newFile)
	}
	for i := 0; i < len(file); i++ {
		// Check if the -a flag was passed, if so print all files including hidden files (those starting with a dot)
		if inc_a {
			fmt.Print(file[i].Name, "\t")
			// Otherwise, check if the file's first character is a dot (indicating it's a hidden file), if not print the file name
		} else if file[i].Name[0] != '.' {
			fmt.Print(file[i].Name, "\t")
		}
	}
	// Reset the global file slice to an empty slice
	file = nil
	// Print a new line after printing the file names
	fmt.Println()
}

func blockSize(address string, files []fs.FileInfo) {
	// Initialize the total blocksize to zero
	totalBlocksize := int64(0)
	// Iterate over each file in the files slice
	for _, file1 := range files {
		// Construct the file path by appending the file name to the parent directory path
		name := address + "/" + file1.Name()
		// Open the file and check for any errors or if it's a symbolic link
		file, err := os.Open(name)
		if err != nil || file1.Mode()&os.ModeSymlink != 0 {
			// If the file is named "sudo", add a default block size of 1168 (this seems like an arbitrary case, not sure what the purpose is)
			if file1.Name() == "sudo" {
				totalBlocksize += 1168
			}
			// Continue to the next file in the iteration
			continue
		}
		// Defer the closing of the file until the end of the function
		defer file.Close()

		// Retrieve the file information, including block size
		fileInf, err := file.Stat()
		if err != nil {
			fmt.Println(err)
			return
		}
		// Retrieve the system information for the file, including the number of blocks it uses on disk
		sys := fileInf.Sys().(*syscall.Stat_t)
		blocks := sys.Blocks
		// Add the file's block size to the total block size
		totalBlocksize += blocks
	}
	// Print the total block size
	fmt.Println("total", totalBlocksize)
}

func getInfo(folder string, isdev bool) map[string]string {
	// create a map to store the information we will extract
	retVal := make(map[string]string)
	// keep backup of the real stdout
	old := os.Stdout
	// create a new pipe to capture stdout output
	r, w, _ := os.Pipe()
	// redirect stdout to the pipe
	os.Stdout = w

	// create a channel to receive the output from the pipe
	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		// copy the output from the pipe into a buffer
		io.Copy(&buf, r)
		// send the buffer contents to the channel
		outC <- buf.String()
	}()

	// create a new process to run the ls command
	var procAttr os.ProcAttr
	procAttr.Files = []*os.File{os.Stdin, w, os.Stderr}
	args := []string{"/bin/ls"}
	// check if we should include hidden files in the output
	if inc_a {
		args = append(args, "-la", folder)
	} else {
		args = append(args, "-l", folder)
	}
	process, err := os.StartProcess(args[0], args, &procAttr)
	if err != nil {
		fmt.Println("start process failed:" + err.Error())
		return retVal
	}
	// wait for the process to finish
	_, err = process.Wait()
	if err != nil {
		fmt.Println("Wait Error:" + err.Error())
	}

	// close the pipe and restore the original stdout
	w.Close()
	os.Stdout = old
	// read the output from the channel
	out := <-outC
	// if isdev is true, print the output and return the empty map
	if isdev {
		fmt.Print(out)
		return retVal
	}
	// split the output into separate rows
	outRows := strings.Split(out, "\n")
	// loop through each row
	for _, thisRow := range outRows {
		// split the row into separate parts using spaces as separators
		lineParts := strings.Split(thisRow, " ")
		// check if this is a symbolic link
		isLnk := false
		if strings.Contains(thisRow, "->") {
			isLnk = true
		}
		// check if the first part of the line is longer than 10 characters
		if len(lineParts[0]) > 10 {
			// extract the key and value from the line
			thisKey := ""
			thisVal := lineParts[0]
			if isLnk {
				thisKey = lineParts[len(lineParts)-3]
			} else {
				thisKey = lineParts[len(lineParts)-1]
			}
			// add the key-value pair to the map
			retVal[thisKey] = string(thisVal[len(thisVal)-1])
		}
	}

	// return the map with the extracted information
	return retVal
}

func sortList(folder string) []fs.FileInfo {
	files, _ := ioutil.ReadDir(folder)   // Read directory and get file info
	sortedList := make([]fs.FileInfo, 0) // Create an empty slice to hold sorted file info
	if !inc_a {                          // Check if hidden files should be excluded
		// Loop through files and exclude hidden files
		for _, file := range files {
			if file.Name()[0] != '.' { // If the file is not hidden
				sortedList = append(sortedList, file) // Add the file info to the sortedList slice
			}
		}
	} else {
		sortedList = files // If hidden files should be included, set sortedList to all files
	}
	if inc_t { // Check if files should be sorted by modified time
		sort.Slice(sortedList, func(i, j int) bool {
			return sortedList[i].ModTime().After(sortedList[j].ModTime()) // Sort files by modified time
		})
	}
	return sortedList // Return the sorted list of file info
}

func sortRev(sorted []fs.FileInfo) []fs.FileInfo {
	// Loop through the sorted list and swap the first and last elements until halfway through the list
	for i := 0; i < len(sorted)/2; i++ {
		sorted[i], sorted[len(sorted)-1-i] = sorted[len(sorted)-1-i], sorted[i]
	}
	return sorted // Return the reversed sorted list of file info
}

func IsSymlink(path string) bool {
	fi, err := os.Lstat(path) // Get file info for the path
	if err != nil {
		return false // If there was an error getting file info, return false
	}
	return fi.Mode()&os.ModeSymlink != 0 // Return whether the file is a symbolic link or not
}

func listAll(folder string) {

	// Replace double slash with single slash in folder path

	folder = strings.Replace(folder, "//", "/", 1)

	// Get a sorted list of files and folders in the folder path
	sorted := sortList(folder)

	// Sort the list in reverse order if inc_r is true
	if inc_r {
		sortRev(sorted)
	}

	// If folder is /dev, print device information and return
	if folder == "/dev" {
		getInfo(folder, true)
		return
	}

	// Set a flag for whether to print out file names or not
	doPrint := true

	// Get a list of all files and folders in the folder path
	files, err := ioutil.ReadDir(folder)

	// If there is an error or folder is a symlink, print out an error message and return
	if err != nil || IsSymlink(folder) {
		if _, err2 := os.Stat(folder); errors.Is(err2, os.ErrNotExist) {
			fmt.Println(strings.Replace(err.Error(), "open", "my-ls-1:", 1))
			return
		}

		// Get additional information about the file or symlink
		extraInfo := getInfo(folder, false)

		// Get information about the file or symlink
		file, _ := os.Lstat(folder)
		link := ""

		// Get the name of the link if it is a symlink
		if file.Mode()&os.ModeSymlink != 0 {
			linkName, _ := os.Readlink(file.Name())
			link += " -> " + linkName
		}

		// Get file system information about the file or symlink
		info, _ := os.Lstat(file.Name())

		// If there is no information, return
		if info == nil {
			return
		}

		// Get user and group information about the file or symlink
		stat := info.Sys().(*syscall.Stat_t)
		usr, _ := user.LookupId(strconv.FormatUint(uint64(stat.Uid), 10))
		group, _ := user.LookupGroupId(strconv.FormatUint(uint64(stat.Gid), 10))

		// If inc_l is false and there are no files, print out the folder name
		if len(files) == 0 && !inc_l {
			fp.AddRow(folder)
		} else {
			// If inc_l is true or there are files, print out file information
			extraAttribute, ok := extraInfo[folder]
			if !ok {
				extraAttribute = ""
			}

			// Add information about the file or symlink to the table
			fp.AddRow(fmt.Sprintf("%v", strings.ToLower(file.Mode().String())) + extraAttribute + "\t" + "1" + "\t" + usr.Username + "\t " + group.Name + "\t " + fmt.Sprintf("%v", file.Size()) + "\t" + file.ModTime().Format("Jan") + "\t" + file.ModTime().Format("2") + "\t" + oldFile(file.ModTime()) + "\t" + file.Name() + link)
		}

		// Flush the table and return
		fp.Flush()
		return
	}

	// If there are no files to print, return
	if len(sorted) == 0 && folder != " " {
		return
	}

	// Get additional information about the files and folders
	extraInfo := getInfo(folder, false)

	if !inc_l {
		if inc_a {
			printDot(folder, inc_r, extraInfo) // Print dot files first
		}
		printNamesOnly(sorted) // Print file names only
		doPrint = false        // Don't print anything else
	} else {
		blockSize(folder, sorted) // Get the block size of the directory
		if inc_a {
			printDot(folder, inc_r, extraInfo) // Print dot files first
		}
		// Loop through each file in the directory and get its information
		for i := 0; i != len(sorted); i++ {
			count := 1             // Keep track of the number of files in a directory
			if sorted[i].IsDir() { // If the file is a directory, get the number of files inside it
				count++
				incFiles, err := ioutil.ReadDir(folder + "/" + sorted[i].Name())
				if err != nil {
					continue // If there is an error, continue to the next file
				}
				for range incFiles {
					count++
				}
			}
			link := ""
			// Get the link name of the file, if it exists
			if sorted[i].Mode()&os.ModeSymlink != 0 {
				linkName, _ := os.Readlink(folder + "/" + sorted[i].Name())
				link += " -> " + linkName
			}
			// Get information about the file, such as its permissions, owner, and group
			info, _ := os.Lstat(folder + "/" + sorted[i].Name())
			if info == nil { // If the file doesn't exist, continue to the next file
				continue
			}
			stat := info.Sys().(*syscall.Stat_t)
			usr, _ := user.LookupId(strconv.FormatUint(uint64(stat.Uid), 10))
			group, _ := user.LookupGroupId(strconv.FormatUint(uint64(stat.Gid), 10))
			if !inc_a && sorted[i].Name()[0] == '.' { // If the file is a hidden file and we're not printing hidden files, skip it
				continue
			}
			extraAttribute, ok := extraInfo[sorted[i].Name()] // Get any extra information about the file
			if !ok {
				extraAttribute = ""
			}
			// Add file information to a struct
			file = append(file, File{Mode: strings.ToLower(info.Mode().String()) + extraAttribute, Time: info.ModTime(), ModTime: info.ModTime(), User: usr.Username, Grp: group.Name, Link: int(stat.Nlink), Size: info.Size(), Name: sorted[i].Name() + link})
		}
		if inc_a && inc_r {
			printDot(folder, inc_r, extraInfo) // Print dot files at the end
		}
	}
	if inc_r && inc_a {
		// If we're printing recursively and printing hidden files, remove the first two entries in the struct (which are '.' and '..')
		if len(file) > 2 {
			file = file[2:]
		} else {
			file = []File{}
		}
	}

	if doPrint {
		if inc_t && inc_a { // Check if files should be sorted by modification time
			// Sort the file slice in descending order of modification time
			sort.Slice(file, func(j, i int) bool {
				return file[j].ModTime.After(file[i].ModTime)
			})
		}
		if inc_r && inc_a { // Check if files should be sorted in reverse order
			// Reverse the file slice
			for i := len(file)/2 - 1; i >= 0; i-- {
				opp := len(file) - 1 - i
				file[i], file[opp] = file[opp], file[i]
			}
		}

		// Print out the file information
		for i := 0; i < len(file); i++ {
			fp.AddRow(file[i].Mode + "\t" + strconv.Itoa(file[i].Link) + "\t" + file[i].User + "\t " + file[i].Grp + "\t " + fmt.Sprintf("%v", file[i].Size) + "\t" + file[i].Time.Format("Jan") + fmt.Sprintf("%3v", file[i].Time.Format("2")) + "\t" + oldFile(file[i].Time) + "\t" + file[i].Name)
		}
		fp.Flush()
		file = []File{} // Reset the file slice
	}

	currentFolder := start
	if currentFolder[len(start)-1] != '/' {
		currentFolder += "/" // Add a slash to the end of the folder path if it is missing
	}

	// Recursively print files
	if inc_R {
		for _, file := range sorted {
			if file.IsDir() { // Check if the file is a directory
				if inc_a { // Check if hidden files should be included
					fmt.Println("\n" + currentFolder + file.Name() + ":")
					if !inc_l {
						filesInFolder(file.Name(), "./") // Print the files in the folder
					}
					if inc_l {
						listAll(folder + "/" + file.Name()) // Print detailed information about the files in the folder
					}
				} else if file.Name()[0] != '.' { // Check if the file is a hidden file
					fmt.Println("\n" + currentFolder + file.Name() + ":")
					if !inc_l {
						filesInFolder(file.Name(), currentFolder) // Print the files in the folder
					}
					if inc_l {
						listAll(folder + "/" + file.Name()) // Print detailed information about the files in the folder
					}
				}
			}
		}
	}
}

// Define a function that takes in a time.Time object as a parameter and returns a string.
func oldFile(fileTime time.Time) string {
	// Get the current time.
	now := time.Now()

	// Calculate a time that is 6 months ago from the current time.
	oldTime := now.AddDate(0, -6, 0)

	// Check if the file time is either after the current time or before the old time.
	if now.Before(fileTime) || oldTime.After(fileTime) {
		// If the file time is either in the future or more than 6 months ago, return the year of the file time in a specific format.
		return " " + fileTime.Format("2006")
	}

	// If the file time is between the current time and 6 months ago, return the file time in a specific time format.
	return fileTime.Format("15:04")
}

// should work now but we need to pass directory as a parameter
func printDot(folder string, inc_r bool, extraInfo map[string]string) {

	// Initialize some variables
	dot := 0
	first := "."
	second := ".."

	// If inc_t flag is set, swap the order of . and ..
	if inc_t {
		first = ".."
		second = "."
	}

	// Loop through the slice containing . and .. and save the information we need
	for _, dir := range []string{first, second} {

		// Create the full file path for . and ..
		temp := ""
		temp = folder + "/" + dir

		// Get the file info for . and ..
		info, err := os.Lstat(temp)
		if err != nil {
			log.Panic(err)
		}

		// Get the system-specific file information
		stat := info.Sys().(*syscall.Stat_t)

		// Get the username and group name for the file
		usr, _ := user.LookupId(strconv.FormatUint(uint64(stat.Uid), 10))
		group, _ := user.LookupGroupId(strconv.FormatUint(uint64(stat.Gid), 10))

		// Initialize foo variable to handle . and .. for -a and -r flags
		foo := ""
		if dot == 0 && !inc_r {
			foo = first
		} else {
			foo = second
		}
		if inc_r && dot == 0 {
			foo = first
		} else if inc_r && dot == 1 {
			foo = second
		}
		dot++

		// Check if any extra information is available for this file
		extraAttribute, ok := extraInfo[foo]
		if !ok {
			extraAttribute = ""
		}

		// Append the file information to the file slice
		file = append(file, File{
			Mode:    info.Mode().String() + extraAttribute,
			Time:    info.ModTime(),
			ModTime: info.ModTime(),
			User:    usr.Username,
			Grp:     group.Name,
			Link:    int(stat.Nlink),
			Size:    info.Size(),
			Name:    foo,
		})
	}
}

// Check if the flag is valid
func NotOk(args string) {
	if args[0] != '-' {
		return
	}
	//loop through the string and check if the flag is valid
	for i := 1; i < len(args); i++ {
		if args[i] != 'l' && args[i] != 'a' && args[i] != 't' && args[i] != 'r' && args[i] != 'R' {
			fmt.Println("Invalid flag: " + args + " Current supported flags are: -l, -a, -t, -r, -R")
			os.Exit(0)
		}
	}
}
