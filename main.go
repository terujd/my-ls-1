package main

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

/*
Blue: Directory
Green: Executable or recognized data file
Cyan (Sky Blue): Symbolic link file
Yellow with black background: Device
Magenta (Pink): Graphic image file
Red: Archive file
Red with black background: Broken link
*/
var (
	showHidden  bool   // -a flag
	showAll     bool   // -A flag
	longListing bool   // -l flag
	reverse     bool   // -r flag
	sortByTime  bool   // -t flag
	recursive   bool   // -R flag
	humanize    bool   // -h flag
	dirPath     string // directory path to list

)

var (
	color   = false
	Reset   = "\033[0m"
	Blue    = "\033[34m"
	Green   = "\033[32m"
	Cyan    = "\033[36m"
	Yellow  = "\033[33m"
	Red     = "\033[31m"
	Magenta = "\033[35m"
)

//----------------------------------------------------------------
//colors and ls meaning:
/*
Blue: Directory
Green: Executable or recognized data file
Cyan (Sky Blue): Symbolic link file
Yellow with black background: Device
Magenta (Pink): Graphic image file
Red: Archive file
Red with black background: Broken link
*/
func main() {
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-a":
			showHidden = true
		case "-A":
			showAll = true
		case "-l":
			longListing = true
		case "-r":
			reverse = true
		case "-t":
			sortByTime = true
		case "-R":
			recursive = true
		case "-h":
			humanize = true
		case "-la":
			showHidden = true
			longListing = true
		case "--color=auto":
			color = true
			fmt.Println("color is on")
		default:
			dirPath = os.Args[i]
		}
	}
	if dirPath == "" {
		dirPath = "./"
	}
	dir, err := os.Open(dirPath)
	if err != nil {
		log.Fatal(err)
	}
	defer dir.Close()

	dirEntries, err := dir.ReadDir(0)
	if err != nil {
		log.Fatal(err)
	}
	if sortByTime {
		sortByModificationTime(dirEntries)
	} else if reverse {
		sortReverse(dirEntries)
	} else {
		sortByName(dirEntries)
	}

	if longListing {
		var totalSize int64
		for _, entry := range dirEntries {
			if !showAll && !showHidden && entry.Name()[0] == '.' {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				log.Println(err)
				continue
			}
			totalSize += info.Size()
		}
		if humanize {
			fmt.Printf("total %s\n", humanizeBytes(totalSize))
		} else {
			fmt.Printf("total %d\n", totalSize/1024)
		}
		printLongListing(dirEntries)
	} else {
		printShortListing(dirEntries)
	}
	if recursive {
		err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("Error walking path %q: %v\n", path, err)
				return nil
			}
			if info.IsDir() {
				if info.Name() != "." && info.Name() != ".." {
					fmt.Printf("\n%s:\n", path)
					dir, err := os.Open(path)
					if err != nil {
						log.Printf("Error opening directory %q: %v\n", path, err)
						return nil
					}
					defer dir.Close()
					dirEntries, err := dir.ReadDir(0)
					if err != nil {
						log.Printf("Error reading directory %q: %v\n", path, err)
						return nil
					}
					if sortByTime {
						sort.Slice(dirEntries, func(i, j int) bool {
							return info.ModTime().After(info.ModTime())
						})
						if reverse {
							reverseSlice(dirEntries)
						}
					} else if reverse {
						sortReverse(dirEntries)
					} else {
						sortByName(dirEntries)
					}
					/*if sortByTime {
						sortByModificationTime(dirEntries)
					} else if reverse {
						sortReverse(dirEntries)
					} else {
						sortByName(dirEntries)
					}*/
					//----------------------------------------------------------------
					if longListing {
						var totalSize int64
						for _, entry := range dirEntries {
							if !showAll && !showHidden && entry.Name()[0] == '.' {
								continue
							}
							info, err := entry.Info()
							if err != nil {
								log.Println(err)
								continue
							}
							totalSize += info.Size()
						}
						if humanize {
							fmt.Printf("total %s\n", humanizeBytes(totalSize))
						} else {
							fmt.Printf("total %d\n", totalSize/1024)
						}
						printLongListing(dirEntries)
					} else {
						printShortListing(dirEntries)
					}
				}
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func reverseSlice(entries []os.DirEntry) {
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
}

func humanizeBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func sortByName(entries []os.DirEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
}

func sortByModificationTime(entries []os.DirEntry) {
	sort.Slice(entries, func(i, j int) bool {
		time1, err := entries[i].Info()
		if err != nil {
			return false
		}
		time2, err := entries[j].Info()
		if err != nil {
			return false
		}
		return time1.ModTime().Before(time2.ModTime())
	})
}

func sortReverse(entries []os.DirEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() > entries[j].Name()
	})
}
func printShortListing(entries []os.DirEntry) {
	for _, entry := range entries {
		//if !showAll && !showHidden && entry.Name()[0] == '.' {
		if entry.Name() == "." || entry.Name() == ".." || entry.Name() == "..." || (!showHidden && entry.Name()[0] == '.') {
			continue
		}
		if longListing {
			mode := entry.Type().String()
			info, err := entry.Info()
			if err != nil {
				log.Println(err)
				continue
			}
			fmt.Printf("%s %3d %s %s %6d %s %s\n",
				mode, info.Sys().(*syscall.Stat_t).Nlink,
				strconv.Itoa(int(info.Sys().(*syscall.Stat_t).Uid)),
				strconv.Itoa(int(info.Sys().(*syscall.Stat_t).Gid)),
				info.Size(), info.ModTime().Format("%+03d:%+03d" /*"Jan _2 15:04"*/), entry.Name())

		} else {
			fmt.Println(entry.Name())
		}
	}
}
func printLongListing(entries []os.DirEntry) {
	for _, entry := range entries {
		if !showAll && !showHidden && entry.Name()[0] == '.' {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			log.Println(err)
			continue
		}
		name := entry.Name()
		if info.IsDir() {
			if color {
				name = Blue + name + Reset
			}
		} else if info.Mode()&0111 != 0 { // check if file is executable
			if color {
				name = Green + name + Reset
			}
		} else if info.Mode()&os.ModeSymlink != 0 {
			if color {
				name = Cyan + name + Reset
			}
		} else if info.Mode()&os.ModeDevice != 0 {
			if color {
				name = Yellow + name + Reset
			}
			// else if graphic image file then color it
		} else if strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") || strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".gif") || strings.HasSuffix(name, ".bmp") || strings.HasSuffix(name, ".tiff") || strings.HasSuffix(name, ".svg") {
			if color {
				name = Magenta + name + Reset
			}
			// if archive file then color it
		} else if strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".rar") || strings.HasSuffix(name, ".tar") || strings.HasSuffix(name, ".gz") || strings.HasSuffix(name, ".7z") || strings.HasSuffix(name, ".bz2") || strings.HasSuffix(name, ".xz") {
			if color {
				name = Red + name + Reset
			}
		}

		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			log.Println("Error getting system-specific file info")
			continue
		}
		mode := info.Mode().String()
		fmt.Printf("%s %3d %s %s %6d %s %s\n",
			mode, stat.Nlink, getUserName(stat.Uid), getGroupName(stat.Gid),
			info.Size(), info.ModTime().Format("Jan _2 15:04"), name)
	}
}

func getUserName(uid uint32) string {
	user, err := user.LookupId(strconv.Itoa(int(uid)))
	if err != nil {
		return fmt.Sprintf("%d", uid)
	}
	return user.Username
}

func getGroupName(gid uint32) string {
	group, err := user.LookupGroupId(strconv.Itoa(int(gid)))
	if err != nil {
		return fmt.Sprintf("%d", gid)
	}
	return group.Name
}
