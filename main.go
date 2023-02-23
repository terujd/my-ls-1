package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// global variables to indicate the command line flags
var longListing = false
var showHidden = false
var recursive = false
var reverse = false
var sortByTime = false

type Group struct {
	Gid  string // group ID
	Name string // group name
}

var fileModeMap = map[os.FileMode]string{
	os.ModeDir:        "d",
	os.ModeSymlink:    "l",
	os.ModeNamedPipe:  "p",
	os.ModeSocket:     "s",
	os.ModeSetuid:     "s",
	os.ModeSetgid:     "s",
	os.ModeCharDevice: "c",
}

var filePermMap = map[os.FileMode]string{
	0400: "r",
	0200: "w",
	0100: "x",
	0040: "r",
	0020: "w",
	0010: "x",
	0004: "r",
	0002: "w",
	0001: "x",
}

func main() {
	// Set offset to 1 to skip program name.
	offset := 1

	// Check for command line flags.
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-l":
			longListing = true
		case "-a":
			showHidden = true
		case "-R":
			recursive = true
		case "-r":
			reverse = true
		case "-t":
			sortByTime = true
		default:
			// This is a file or directory argument, so break out of the loop.
			offset = i
			break
		}
	}

	// If there are no files specified, show the current directory.
	files := os.Args[offset:]
	if len(files) == 0 {
		files = []string{"."}
	}

	// If the -R flag is set, show long listing recursively.
	if recursive {
		for _, f := range files {
			showLongListingRecursive(f)
		}
	} else {
		// Otherwise, show short or long listing depending on the -l flag.
		if longListing {
			showLongListing(files)
		} else {
			showShortListing(files)
		}
	}
}

// Show short listing function.
func showShortListing(files []string) {
	var noFilesList []string // list of files that could not be listed
	var filesList []string   // list of files to be listed
	var dirListing []string  // list of directories to be listed
	for _, f := range files {
		// Check if the file or directory exists.
		fi, err := os.Stat(f) // get file info
		if nil != err {
			s := fmt.Sprintf("ls: %v: no file or directory", f) // if file not found, add error message to noFilesList
			noFilesList = append(noFilesList, s)
			continue
		}
		if !showHidden && strings.HasPrefix(fi.Name(), ".") {
			continue
		}
		if !fi.IsDir() { // if the file is not a directory, add it to filesList
			filesList = append(filesList, f)
			continue
		}

		// If it is a directory, get a short listing of its contents.
		dirListing, err = addShortDirListing(dirListing, f)
		if err != nil {
			s := fmt.Sprintf("ls: %v: %v", f, err)
			noFilesList = append(noFilesList, s)
		}
	}
	// Print out the lists.
	for _, s := range noFilesList {
		fmt.Println(s)
	}
	for _, s := range filesList {
		fmt.Println(s)
	}
	for _, s := range dirListing {
		fmt.Println(s)
	}

}

func addShortDirListing(dirListing []string, dirName string) ([]string, error) {
	f, err := os.Open(dirName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fis, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}

	for _, fi := range fis {
		if !showHidden && strings.HasPrefix(fi.Name(), ".") {
			continue
		}
		s := fi.Name()
		if fi.IsDir() {
			s += "/"
		}
		dirListing = append(dirListing, s)
	}
	return dirListing, nil
}

func addDirListing(listing []string, f string, longListing bool) []string {
	dir, err := os.Open(f) // open the directory
	if err != nil {
		log.Printf("Error opening directory: %v", err)
		return listing
	}
	defer dir.Close()

	fis, err := dir.Readdir(0) // get list of files in directory
	if err != nil {
		log.Printf("Error reading directory: %v", err)
		return listing
	}

	if sortByTime {
		sort.Slice(fis, func(i, j int) bool {
			return fis[i].ModTime().After(fis[j].ModTime())
		})
	}

	listing = append(listing, "\n"+f+":")
	for _, fi := range fis {
		if !showHidden && fi.Name()[0] == '.' {
			continue
		}

		if longListing {
			var sb strings.Builder
			sb.WriteString(getFileMode(fi.Mode()))                              // file mode
			sb.WriteString(getFilePermissions(fi.Mode()))                       // file permissions
			sb.WriteString(fmt.Sprintf(" %d ", fi.Sys().(*syscall.Stat_t).Uid)) // user ID
			sb.WriteString(fmt.Sprintf("%10d ", fi.Size()))                     // file size
			sb.WriteString(getTimeString(fi.ModTime()))                         // modification time
			sb.WriteString(fmt.Sprintf(" %s", fi.Name()))                       // file name
			listing = append(listing, sb.String())
		} else {
			listing = append(listing, fi.Name())
		}
	}

	return listing
}

func showLongListingRecursive(dir string) {
	fmt.Println(dir)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !showHidden && strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		if info.IsDir() {
			fmt.Printf("%s:\n", path)
		}
		showLongListing([]string{path})
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

func getTimeString(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func getFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func showLongListing(files []string) {
	for _, file := range files {
		fileInfo, err := os.Stat(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file info: %v\n", err)
			continue
		}
		fmt.Printf("%s\t%d\t%s\t%s\n",
			fileInfo.Mode(),
			fileInfo.Size(),
			fileInfo.ModTime().Format("Jan 02 15:04"),
			file)
	}
}

// Format time function.
func formatTime(t time.Time) string {
	layout := "Jan 2 15:04"
	if time.Now().Sub(t).Hours() > 24*365 {
		layout = "Jan 2 2006"
	}
	return t.Format(layout)
}

func getOwnerAndGroup(sys interface{}) (string, string) {
	stat := sys.(*syscall.Stat_t)
	uid := fmt.Sprint(stat.Uid)
	gid := fmt.Sprint(stat.Gid)
	u, err := user.LookupId(uid)
	if err != nil {
		return uid, gid
	}
	g, err := user.LookupGroupId(gid)
	if err != nil {
		return u.Username, gid
	}
	return u.Username, g.Name
}

// LookupGroupId is a wrapper around the user.LookupGroupId function.
// It takes a group ID string as an argument and returns a user.Group object and an error.
func LookupGroupId(gid string) (*user.Group, error) {
	return user.LookupGroupId(gid)
}

// This function takes a file mode and returns a string representation of the file's permissions
// in the form of a 9-character string, where each character represents a permission
// (r, w, or x) for the owner, group, and others.
func getFilePermissions(mode fs.FileMode) string {
	// Initialize a string to represent the permission string,
	// with all characters initially set to "-"
	perm := "---------"
	for bit, val := range filePermMap { // Loop through the filePermMap (a map of permission bit masks to permission strings)
		// If the permission bit is set in the file mode, update the permission string with the corresponding permission string from filePermMap
		if mode&bit != 0 {
			perm = perm[:9-len(val)] + val + perm[10-len(val):]
		}
	}
	// Return the permission string
	return perm
}

// This function takes a file mode and returns a string representation of the file's type,
// using one of the following characters: "-", "d", "l", "b", "c", "p", or "s".
func getFileMode(mode fs.FileMode) string {
	switch { // Use a switch statement to determine the file type based on the file mode
	case mode.IsRegular():
		return "-" // regular file
	case mode.IsDir():
		return "d" // directory
	case mode&fs.ModeSymlink != 0:
		return "l" // symbolic link
	case mode&fs.ModeDevice != 0:
		return "b" // block device
	case mode&fs.ModeCharDevice != 0:
		return "c" // character device
	case mode&fs.ModeNamedPipe != 0:
		return "p" // named pipe (FIFO)
	case mode&fs.ModeSocket != 0:
		return "s" // socket
	default:
		return "?" // unknown type
	}
}
