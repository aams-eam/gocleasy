package main

import (
	"fmt"
	"gocleasy/files"
	"gocleasy/guiutils"
	"gocleasy/ignore"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"golang.design/x/clipboard"
)

var filesFromDirsBeingLoaded = make(chan string, 10) // To send files being scanned inside a directory that has been clicked to be expanded

func calculateDirSize(basepath string) (int64, int64, error) {

	var size int64 = 0
	var numchildren int64 = 0

	err := filepath.WalkDir(basepath, func(path string, dire os.DirEntry, err error) error {

		// Get the size if not a directory
		fileinfo, err := os.Stat(path)
		if err == nil {
			size += fileinfo.Size()
			numchildren++
		}

		// Continue even if you cannot read one specific file
		return nil
	})

	numchildren--

	return size, numchildren, err
}

func DeleteFiles(selected_files []*files.File) (int64, int64) {

	var errslice []error
	var err error
	var numfiles, sizeliberated int64

	// Loop over selected files and delete them
	for _, file := range selected_files {
		if file.IsDir {
			numfiles += file.NumChildren
		} else {
			numfiles++
		}
		sizeliberated += file.Size
		log.Print("WARNING: If you are testing, you may want to comment the following lines")
		err = os.RemoveAll(file.FullPath)
		if err != nil {
			errslice = append(errslice, err)
		}
	}

	for _, er1 := range errslice {
		// If there is any error with any file/folder you can handle it here
		log.Print(er1)
	}

	return numfiles, sizeliberated
}

func getRootPath() string {
	switch runtime.GOOS {
	case "windows":
		return getWindowsRootPath()
	case "darwin":
		// macOS
		return "/"
	case "android", "linux":
		// For Android apps you will need to request permission for reading from external folders.
		// I was not able to perform that with gioui or golang, maybe you need to create a connector for JAVA
		return "/"
	case "ios":
		// iOS apps are sandboxed too, so the root path will not be directly accessible
		return getIOSRootPath()
	default:
		return "Unknown OS"
	}
}

func getWindowsRootPath() string {
	// On Windows, the root path is typically the drive where the OS is installed,
	// so we need to get the current drive and concatenate it with the path separator.
	return filepath.VolumeName(os.Getenv("SystemDrive")) + string(filepath.Separator)
}

func getIOSRootPath() string {
	// In iOS, the application's root path is restricted, but you can use other directories like the Documents directory.
	// This is just an example of how you could handle it, but it's not the actual root path.
	documentsDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(documentsDir, "Documents")
}

// Deletes a string from a slice if exists
func deleteStringFromSlice(str string, slice []string) []string {
	// Find and remove the string from the slide
	for i, v := range slice {
		if v == str {
			slice = append(slice[:i], slice[i+1:]...)
			break
		}
	}
	return slice
}

func printTree(file *files.File, indent string) {
	fmt.Printf("%s%s\n", indent, file.Name)
	for _, subFile := range file.Files {
		printTree(subFile, indent+"\t")
	}
}

func getSelectedFiles(children []*files.File, selfiles *[]*files.File) []*files.File {

	// create a pointer to every file
	var filep *files.File

	for index := range children {
		filep = children[index]

		// If it is selected add to selfiles
		if filep.IsSelected.Value {
			*selfiles = append(*selfiles, filep)
		}

		if (filep.IsDir) && // It is a directory
			((filep.Files != nil) && (len(filep.Files) > 0)) && // It contains files inside
			!filep.IsSelected.Value { // It is not a selected folder

			*selfiles = getSelectedFiles(filep.Files, selfiles) // Look for selected files inside
		}
	}

	return *selfiles
}

func copyFilesInClipboard(selfiles []*files.File) {

	var result string = ""

	for _, file := range selfiles {
		result += "\"" + file.FullPath + "\" "
	}

	clipboard.Write(clipboard.FmtText, []byte(result))
}

func Run(win *app.Window) error {

	var applogic *guiutils.AppLogic = guiutils.NewAppLogic()

	// ops are the operations from the UI
	var ops op.Ops

	// Widget declarations
	var scanButton widget.Clickable
	var deleteButton widget.Clickable
	var comeBackButton widget.Clickable
	var copy2clipboard widget.Clickable
	var nextButton widget.Clickable
	var initialPathInput widget.Editor
	var filelist widget.List = widget.List{
		List: layout.List{
			Axis: layout.Vertical,
		},
	}
	var filedeletelist widget.List = widget.List{
		List: layout.List{
			Axis: layout.Vertical,
		},
	}

	var scanfilesLoadingChann chan int = make(chan int) // Used to transmit how many files have been read
	var numfilesdeleted int64 = 0
	var sizeliberated int64 = 0

	var initialpath string

	var totalFilesReadShow int = 0 // Used to maintain a count of the files read

	// Initialize clipboard so you can set the clipboard
	err := clipboard.Init()
	if err != nil {
		panic(err)
	}

	// Listen for events in the window
	for {
		e := <-win.Events()
		log.Print(e)
		switch e := e.(type) {
		// Window closed
		case system.DestroyEvent:
			return e.Err

		// Actions in the window apart from closing
		case system.FrameEvent:

			gtx := layout.NewContext(&ops, e)

			//
			// ACTIONS TO CHANGE THE STATE OF THE APPLICATION ***
			//
			// Goes from homePage to selection of files
			if scanButton.Clicked() {

				// reset file directory
				applogic.Files = nil

				initialpath = initialPathInput.Text()
				if initialpath == "" {
					initialpath = getRootPath()
				}

				// Test the introduced path, if not good, use the root path
				_, err := os.ReadDir(initialpath)
				if err != nil {
					initialPathInput.SetText("The path: \"%s\" does not exists or cannot be read; Introduce another path or leave blank for root")
				} else {
					// If there is no problem, continue
					applogic.Appstate = guiutils.LoadingFilesS

					go applogic.ReportProgress(win, &totalFilesReadShow, scanfilesLoadingChann)
					go func() {
						applogic.Files = files.WalkFolder(initialpath, ioutil.ReadDir, ignore.IgnoreBasedOnIgnoreFile(ignore.ReadIgnoreFile()), scanfilesLoadingChann)
					}()
					applogic.ShowLoadingPage(gtx, totalFilesReadShow)
				}
			}

			// Go to confirm deleting the files
			if nextButton.Clicked() {
				// See which are the selected files everytime we click next button to go to delFileS
				applogic.Selfiles = nil
				applogic.Selfiles = getSelectedFiles(applogic.Files.Files, &applogic.Selfiles)
				applogic.Appstate = guiutils.DelFilesS
			}

			// Go back to selecting the files
			if comeBackButton.Clicked() {
				applogic.Appstate = guiutils.SelFilesS
			}

			// copy files in clipboard
			if copy2clipboard.Clicked() {
				copyFilesInClipboard(applogic.Selfiles)
			}

			// Delete the files show a message of number of files deleted and amount of memory freed
			if deleteButton.Clicked() {
				numfilesdeleted, sizeliberated = DeleteFiles(applogic.Selfiles)
				applogic.Appstate = guiutils.HomeS
			}
			// ACTIONS TO CHANGE THE STATE OF THE APPLICATION ***

			//
			// STATES OF THE APPLICATION ***
			// What template to render based on applogic state
			//
			switch applogic.Appstate {

			case guiutils.HomeS:
				applogic.HomePage(gtx, &scanButton, &initialPathInput, numfilesdeleted, sizeliberated)

			case guiutils.LoadingFilesS:
				applogic.ShowLoadingPage(gtx, totalFilesReadShow)

			case guiutils.SelFilesS:
				applogic.ShowFiles(gtx, &nextButton, &filelist)

			case guiutils.DelFilesS:
				applogic.ShowDeletingPage(gtx, &comeBackButton, &copy2clipboard, &deleteButton, &filedeletelist)

			}
			// STATES OF THE APPLICATION ***

			e.Frame(gtx.Ops)

		}
	}
}

func main() {

	go func() {

		// create window
		w := app.NewWindow(
			app.Title("Gocleasy"),
			app.Size(unit.Dp(550), unit.Dp(550)),
		)

		// Run main loop
		if err := Run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
