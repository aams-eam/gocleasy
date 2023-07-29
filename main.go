package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/dustin/go-humanize"
)

type State string

const (
	homeS         State = "home"          // Show the scan button
	loadingFilesS State = "loadingFilesS" // Show the files to be selected
	selFilesS     State = "selFileS"      // Show the files to be selected
	delFilesS     State = "delFileS"      // Show the selected files to be deleted
)

type File struct {
	Name         string      // Name of the file
	Path         string      // Path of the file
	Level        int         // Indicates in which level the file is compared with the root level
	Size         int64       // Size of the file or directory
	IsDir        bool        // To indicate if the file is a folder or not
	IsSelected   widget.Bool // See if the file is selected or not. Will be used to fill AppLogic.selfiles.
	ActionButton widget.Bool // See if the file is needs to be shown in the interface. Will be used to fill AppLogic.files2show.
	Children     []File      // Files that contain in case IsDir == true.
	NumChildren  int64       // Num of files that the directory contains
}

type AppLogic struct {
	theme      *material.Theme // Store the them of the application
	files      []File          // Used to store the files with their structure
	selfiles   []*File         // Used to store the files that has been selected
	files2show []*File         // Used to store the filest that are going to be rendered
	appstate   State
}

type C = layout.Context
type D = layout.Dimensions

var activeDirectoriesLoading = make([]string, 0) // show what directories are being loaded to not run multiple go routines on the same directory
var loadedDirFileStr string
var filesFromDirsBeingLoaded = make(chan string, 10) // To send files being scanned inside a directory that has been clicked to be expanded

// Create an instance of AppLogic
func NewAppLogic() *AppLogic {

	return &AppLogic{
		theme:    material.NewTheme(gofont.Collection()),
		appstate: homeS,
	}
}

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

// Given the path of a directory, returns the file and sizes of that directory, the level passed indicates the level of the files
func LoadFilesFromDir(path string, level int, files *[]File, loadedfilechann chan string) (int, error) {

	var added_files int = 0
	var filesize int64 = 0
	var numchildren int64 = 0
	var sortedFiles []File = []File{}

	dir, err := os.ReadDir(path)
	if err != nil {
		fmt.Println("Error:", err)
		*files = sortedFiles // Make the directory to be empty because we cannot access it
		if loadedfilechann != nil {
			loadedfilechann <- path
			loadedfilechann <- ""
		}
		return added_files, err
	}

	// iterate over the files inside the dir
	for _, info := range dir {

		fullpath := filepath.Join(path, info.Name())
		fileInfo, err := os.Stat(fullpath)

		if err == nil {

			if loadedfilechann != nil {
				loadedfilechann <- fullpath
			}

			if info.IsDir() {
				filesize, numchildren, err = calculateDirSize(fullpath)
			} else {
				filesize = fileInfo.Size()
				numchildren = 0
			}

			if err == nil {
				sortedFiles = append(sortedFiles, File{
					Name:        info.Name(),
					Path:        fullpath,
					Level:       level,
					IsDir:       info.IsDir(),
					IsSelected:  widget.Bool{},
					Size:        filesize,
					NumChildren: numchildren,
				})

				added_files++
			}
		}
	}

	if len(sortedFiles) == 0 {
		// It was an empty folder; we cannot append we have to assign to make sure it stops being nil
		*files = sortedFiles

	} else {
		// Sort files by size (bigger first)
		sort.Slice(sortedFiles, func(i, j int) bool {
			return sortedFiles[i].Size > sortedFiles[j].Size
		})
		*files = append(*files, sortedFiles...)
	}

	if loadedfilechann != nil {
		loadedfilechann <- path
		loadedfilechann <- ""
	}

	return added_files, nil
}

func DeleteFiles(selected_files []*File) (int64, int64) {

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
		err = os.RemoveAll(file.Path)
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

func Run(win *app.Window) error {

	var applogic *AppLogic = NewAppLogic()

	// ops are the operations from the UI
	var ops op.Ops

	// Widget declarations
	var scanButton widget.Clickable
	var deleteButton widget.Clickable
	var comeBackButton widget.Clickable
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

	var scanfilesLoadingChann chan string = make(chan string, 10) // Used to transmit which directory it is being read
	var fileLoadedStr string                                      // Used to store what to show with the loading page
	var numfilesdeleted int64 = 0
	var sizeliberated int64 = 0

	var initialpath string

	// Listen for events in the window
	for {
		select {
		case e := <-win.Events():
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
					applogic.files = nil

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
						applogic.appstate = loadingFilesS
						go func() {
							LoadFilesFromDir(initialpath, 0, &applogic.files, scanfilesLoadingChann)
						}()
						applogic.showLoadingPage(gtx, fileLoadedStr)
					}
				}

				// Go to confirm deleting the files
				if nextButton.Clicked() {
					// See which are the selected files everytime we click next button to go to delFileS
					applogic.selfiles = nil
					applogic.selfiles = getSelectedFiles(applogic.files, &applogic.selfiles)
					applogic.appstate = delFilesS
				}

				// Go back to selecting the files
				if comeBackButton.Clicked() {
					applogic.appstate = selFilesS
				}

				// Delete the files show a message of number of files deleted and amount of memory freed
				if deleteButton.Clicked() {
					numfilesdeleted, sizeliberated = DeleteFiles(applogic.selfiles)
					applogic.appstate = homeS
				}
				// ACTIONS TO CHANGE THE STATE OF THE APPLICATION ***

				//
				// STATES OF THE APPLICATION ***
				// What template to render based on applogic state
				//
				switch applogic.appstate {

				case homeS:
					applogic.homePage(gtx, &scanButton, &initialPathInput, numfilesdeleted, sizeliberated)

				case loadingFilesS:
					applogic.showLoadingPage(gtx, fileLoadedStr)

				case selFilesS:
					applogic.showFiles(gtx, &nextButton, &filelist)

				case delFilesS:
					applogic.showDeletingPage(gtx, &comeBackButton, &deleteButton, &filedeletelist)

				}
				// STATES OF THE APPLICATION ***

				e.Frame(gtx.Ops)
			}

		// This two cases are for the loading page, to show the files being analyzed
		case loadedFile := <-scanfilesLoadingChann:
			if (loadedFile != "") && (applogic.appstate == loadingFilesS) {
				fileLoadedStr = loadedFile
				win.Invalidate()
			} else if loadedFile == "" {
				applogic.appstate = selFilesS
			}

		// This two cases are for the loading page, to show the files being analyzed
		case loadedFileFromDir := <-filesFromDirsBeingLoaded:
			// fill the variable to show that is being loaded
			loadedDirFileStr = loadedFileFromDir
			// If the loaded file that arrives is the folder that was being loaded it would be deleted from the slice of activeDirectoriesLoading
			activeDirectoriesLoading = deleteStringFromSlice(loadedFileFromDir, activeDirectoriesLoading)
			win.Invalidate()

		}
	}
}

func showGocleasyLogo(gtx C, margins layout.Inset) layout.FlexChild {

	// Show logo
	return layout.Flexed(1, func(gtx C) D {
		// Open the file using the file path
		file, err := os.Open("images/gocleasy-logo.png")
		if err != nil {
			fmt.Println("Error:", err)
		}
		defer file.Close()

		// Pass the file to the Decode function
		img, format, err := image.Decode(file)
		if err != nil {
			fmt.Println("Format: %s; Error decoding image: %s", format, err)
		}

		return margins.Layout(gtx, func(gtx C) D {
			return widget.Image{
				Src: paint.NewImageOp(img),
				Fit: widget.Contain,
			}.Layout(gtx)
		})
	})
}

func (applogic *AppLogic) homePage(gtx C, scanbutton *widget.Clickable, initialpathinput *widget.Editor, numfilesdeleted int64, sizeliberated int64) D {

	margins := layout.Inset{
		Top:    unit.Dp(25),
		Bottom: unit.Dp(25),
		Right:  unit.Dp(25),
		Left:   unit.Dp(25),
	}

	var deletedfilesoutput string
	if numfilesdeleted == 0 || sizeliberated == 0 {
		deletedfilesoutput = ""
	} else {
		deletedfilesoutput = fmt.Sprintf("   Deleted %s files and %s", humanize.Comma(numfilesdeleted), humanize.Bytes(uint64(sizeliberated)))
	}

	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Middle,
		Spacing:   layout.SpaceEnd,
	}.Layout(gtx,
		showGocleasyLogo(gtx, margins),
		layout.Rigid(func(gtx C) D {
			return material.Body1(applogic.theme, deletedfilesoutput).Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Inset{
				Right: unit.Dp(25),
				Left:  unit.Dp(25),
			}.Layout(gtx, func(gtx C) D {
				return material.Editor(applogic.theme, initialpathinput, " Introduce Initial Path. Leave blank for root path.").Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx C) D {
			return margins.Layout(gtx, func(gtx C) D {
				return material.Button(applogic.theme, scanbutton, "Scan Files").Layout(gtx)
			})
		}),
	)
}

func createTextNLoading(gtx C, th *material.Theme, text string) layout.FlexChild {
	return layout.Rigid(func(gtx C) D {
		return layout.Flex{
			Axis: layout.Horizontal,
		}.Layout(gtx,
			layout.Rigid(
				layout.Spacer{Width: unit.Dp(25)}.Layout,
			),
			layout.Rigid(func(gtx C) D {
				return material.Body1(th, fmt.Sprintf("Loading file \"%s\"...", text)).Layout(gtx)
			}),
			layout.Rigid(
				layout.Spacer{Width: unit.Dp(25)}.Layout,
			),
			layout.Rigid(func(gtx C) D {
				return layout.Center.Layout(gtx, func(gtx C) D {
					return material.Loader(th).Layout(gtx)
				})
			}))
	})
}

func (applogic *AppLogic) showLoadingPage(gtx C, loadedFilePath string) D {

	margins := layout.Inset{
		Top:    unit.Dp(25),
		Bottom: unit.Dp(25),
		Right:  unit.Dp(35),
		Left:   unit.Dp(35),
	}

	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Middle,
		Spacing:   layout.SpaceEnd,
	}.Layout(gtx,
		// Space on the top of the window
		layout.Rigid(
			layout.Spacer{Height: unit.Dp(25)}.Layout,
		),
		showGocleasyLogo(gtx, margins),
		// Show Reading files and loading circle
		createTextNLoading(gtx, applogic.theme, loadedFilePath),
		layout.Rigid(
			layout.Spacer{Height: unit.Dp(25)}.Layout,
		),
	)
}

func selectFilesTableRow(th *material.Theme, file *File, numchildren string, filepath string) []layout.FlexChild {

	return []layout.FlexChild{
		// Name of the file
		layout.Rigid(func(gtx C) D {
			return material.CheckBox(th, &file.IsSelected, "").Layout(gtx)
		}),
		// Checkbox to see the files inisde of the folder. Represents if we are seeing the files inside or not
		layout.Rigid(func(gtx C) D {
			return material.CheckBox(th, &file.ActionButton, filepath).Layout(gtx)
		}),
		// Ocupy the space in between buttons and text (checkbox and filenames, size and numfiles)
		layout.Flexed(1, layout.Spacer{}.Layout),
		// Num of files inside the directory (0 if it is a file)
		layout.Rigid(func(gtx C) D {
			return material.Body1(th, numchildren).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(25)}.Layout),
		// Size of the file
		layout.Rigid(func(gtx C) D {
			return material.Body1(th, humanize.Bytes(uint64(file.Size))).Layout(gtx)
		}),
	}
}

func selectFilesTableHeader(gtx C, th *material.Theme) D {

	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
		Spacing:   layout.SpaceStart,
	}.Layout(gtx,
		layout.Rigid(layout.Spacer{Width: unit.Dp(75)}.Layout),
		// Name of the file
		layout.Rigid(func(gtx C) D {
			return material.Body1(th, "Path").Layout(gtx)
		}),
		// Ocupy the space in between buttons and text (checkbox and filenames, size and numfiles)
		layout.Flexed(1, layout.Spacer{}.Layout),
		// Num of files inside the directory (0 if it is a file)
		layout.Rigid(func(gtx C) D {
			return material.Body1(th, "Num Children").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(25)}.Layout),
		// Size of the file
		layout.Rigid(func(gtx C) D {
			return material.Body1(th, "Size").Layout(gtx)
		}),
	)
}

func deleteFilesTableRow(gtx C, th *material.Theme, field1 string, field2 string, field3 string) D {

	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(layout.Spacer{Width: unit.Dp(25)}.Layout),
		// Name of the file
		layout.Rigid(func(gtx C) D {
			return material.Body1(th, field1).Layout(gtx)
		}),
		// Ocupy the space in between buttons and text (checkbox and filenames, size and numfiles)
		layout.Flexed(1, layout.Spacer{}.Layout),
		// Num of files inside the directory (0 if it is a file)
		layout.Rigid(func(gtx C) D {
			return material.Body1(th, field2).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(25)}.Layout),
		// Size of the file
		layout.Rigid(func(gtx C) D {
			return material.Body1(th, field3).Layout(gtx)
		}),
	)
}

func (applogic *AppLogic) showFiles(gtx C, nextbutton *widget.Clickable, filelist *widget.List) D {

	var widgets []layout.FlexChild = []layout.FlexChild{
		// Space on the top of the window
		layout.Rigid(
			layout.Spacer{Height: unit.Dp(25)}.Layout,
		),
		layout.Rigid(func(gtx C) D {
			return selectFilesTableHeader(gtx, applogic.theme)
		}),
		// Where files are shown
		layout.Flexed(1, func(gtx C) D {
			return applogic.fileTree(gtx, filelist, "")
		}),
	}

	if len(activeDirectoriesLoading) > 0 { // We are loading new files
		widgets = append(widgets, createTextNLoading(gtx, applogic.theme, loadedDirFileStr))
	}

	widgets = append(widgets,
		// Button to confirm selected files
		layout.Rigid(func(gtx C) D {
			margins := layout.Inset{
				Top:    unit.Dp(25),
				Bottom: unit.Dp(25),
				Right:  unit.Dp(35),
				Left:   unit.Dp(35),
			}
			return margins.Layout(gtx, func(gtx C) D {
				return material.Button(applogic.theme, nextbutton, "Next").Layout(gtx)
			})
		}))

	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Middle,
		Spacing:   layout.SpaceStart,
	}.Layout(gtx, widgets...)
}

func isStringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// Loops over a file tree and fills the second argument with the files that need to be shown.
// If the file needs to be shown its ActionButton.Value will be true
func getFiles2show(files []File, files2show []*File) ([]*File, int) {
	var numfiles int = 0
	var numtmp int = 0
	var file *File

	for i := range files {

		numtmp = 0
		file = &files[i]

		files2show = append(files2show, file) // append the file to show
		numfiles++

		// Show the files inside if it is a directory and is marked to be shown
		// also consider that if the directory is selected to be deleted, do not show
		if file.IsSelected.Value {
			file.ActionButton.Value = false

		} else if file.IsDir && file.ActionButton.Value { // If the file is a dir and the value is true (show)

			if file.Children == nil { // If there are no children we load them asynchronously
				fullpath := fmt.Sprintf("%s/", file.Path)

				// Check if the directory is being processed right now
				if !isStringInSlice(fullpath, activeDirectoriesLoading) {
					// Add the string to the slice
					activeDirectoriesLoading = append(activeDirectoriesLoading, fullpath)
					// Create a go routine to scan files inside the directory
					var tmp *File = file // need variable because file variable is going to change while go routine processess
					go func() {
						LoadFilesFromDir(fullpath, file.Level+1, &tmp.Children, filesFromDirsBeingLoaded)
					}()

				}
			} else {
				files2show, numtmp = getFiles2show(file.Children, files2show)
				numfiles += numtmp
			}
		}
	}

	return files2show, numfiles
}

// Contains the file Tree
func (applogic *AppLogic) fileTree(gtx C, filelist *widget.List, path string) D {

	// empty the files to show
	applogic.files2show = nil
	var numfiles int
	applogic.files2show, numfiles = getFiles2show(applogic.files, applogic.files2show)

	if numfiles == 0 {
		return D{}
	}

	return filelist.List.Layout(gtx, numfiles, func(gtx C, index int) D {

		var file *File = applogic.files2show[index]
		var widgets []layout.FlexChild
		var spacers []layout.FlexChild
		for i := 0; i < file.Level; i++ {
			spacers = append(spacers, layout.Rigid(layout.Spacer{Width: unit.Dp(25)}.Layout))
		}

		if file.IsDir {
			widgets = selectFilesTableRow(applogic.theme, file, humanize.Comma(file.NumChildren), fmt.Sprintf("%s/", filepath.Join(path, file.Name)))
		} else {
			widgets = selectFilesTableRow(applogic.theme, file, "-", filepath.Join(path, file.Name))
		}
		widgets = append(spacers, widgets...)
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, widgets...)
	})
}

func (applogic *AppLogic) showDeletingPage(gtx C, comebackbutton *widget.Clickable, deletebutton *widget.Clickable, filedeletelist *widget.List) D {

	margins := layout.Inset{
		Top:    unit.Dp(15),
		Bottom: unit.Dp(15),
		Right:  unit.Dp(15),
		Left:   unit.Dp(15),
	}

	var tot_files, tot_size int64 = 0, 0
	for _, file := range applogic.selfiles {
		if file.IsDir {
			tot_files += file.NumChildren
		} else {
			tot_files++
		}
		tot_size += file.Size

	}

	return layout.Flex{
		Alignment: layout.Middle,
		Axis:      layout.Vertical,
	}.Layout(gtx,
		// Space on the top of the window
		layout.Rigid(
			layout.Spacer{Height: unit.Dp(25)}.Layout,
		),
		layout.Rigid(func(gtx C) D {
			return deleteFilesTableRow(gtx, applogic.theme, "Path", "Num Children", "Size")
		}),
		// Show selected files
		layout.Flexed(1, func(gtx C) D {
			return applogic.selectedFiles(gtx, filedeletelist)
		}),
		// Show total
		layout.Rigid(func(gtx C) D {
			return deleteFilesTableRow(gtx, applogic.theme, "Total", humanize.Comma(tot_files), humanize.Bytes(uint64(tot_size)))
		}),
		// Show control buttons
		layout.Rigid(func(gtx C) D {
			return layout.Flex{
				Alignment: layout.Middle,
				Axis:      layout.Horizontal,
			}.Layout(gtx,
				// Show comebackbutton
				layout.Flexed(1, func(gtx C) D {
					return margins.Layout(gtx, func(gtx C) D {
						return material.Button(applogic.theme, comebackbutton, "Back").Layout(gtx)
					})
				}),
				// Show delete button
				layout.Flexed(1, func(gtx C) D {
					return margins.Layout(gtx, func(gtx C) D {
						return material.Button(applogic.theme, deletebutton, "Delete").Layout(gtx)
					})
				}),
			)
		}),
	)
}

func getSelectedFiles(files []File, selfiles *[]*File) []*File {

	// create a pointer to every file
	var filep *File

	for index := range files {
		filep = &files[index]

		// If it is selected add to selfiles
		if filep.IsSelected.Value {
			*selfiles = append(*selfiles, filep)
		}

		if (filep.IsDir) && // It is a directory
			((filep.Children != nil) && (len(filep.Children) > 0)) && // It contains files inside
			!filep.IsSelected.Value { // It is not a selected folder

			*selfiles = getSelectedFiles(filep.Children, selfiles) // Look for selected files inside
		}
	}

	return *selfiles
}

func (applogic *AppLogic) selectedFiles(gtx C, filedeletelist *widget.List) D {
	return filedeletelist.List.Layout(gtx, len(applogic.selfiles), func(gtx C, index int) D {
		var selfile *File = applogic.selfiles[index]
		var num_children, fullpath string
		if selfile.IsDir {
			fullpath = fmt.Sprintf("%s/", selfile.Path)
			num_children = humanize.Comma(selfile.NumChildren)
		} else {
			num_children = "-"
			fullpath = selfile.Path
		}

		return deleteFilesTableRow(gtx, applogic.theme, fullpath, num_children, humanize.Bytes(uint64(selfile.Size)))
	})
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
