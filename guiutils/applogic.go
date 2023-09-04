package guiutils

import (
	"embed"
	"fmt"
	"gocleasy/files"
	"image"
	"path/filepath"
	"time"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/dustin/go-humanize"
)

//go:embed images/gocleasy-logo.png
var imageFile embed.FS

type State string

const (
	HomeS         State = "home"          // Show the scan button
	LoadingFilesS State = "loadingFilesS" // Show the files to be selected
	SelFilesS     State = "selFileS"      // Show the files to be selected
	DelFilesS     State = "delFileS"      // Show the selected files to be deleted
)

type AppLogic struct {
	theme      *material.Theme   // Store the them of the application
	Files      *files.File       // Used to store the files with their structure
	Selfiles   []*files.File     // Used to store the files that has been selected
	Files2Show []*files.FileShow // Used to store the filest that are going to be rendered
	Appstate   State
}

type C = layout.Context
type D = layout.Dimensions

// Create an instance of AppLogic
func NewAppLogic() *AppLogic {

	return &AppLogic{
		theme:    material.NewTheme(gofont.Collection()),
		Appstate: HomeS,
	}
}

func (applogic *AppLogic) ReportProgress(win *app.Window, total *int, progress <-chan int) {

	// Controls how frequently to update the application
	const interval = 250 * time.Millisecond
	ticker := time.NewTicker(interval)

	for {
		select {
		case loadedFile, ok := <-progress:

			if ok && (applogic.Appstate == LoadingFilesS) {
				*total += loadedFile
			} else if !ok {
				applogic.Appstate = SelFilesS
				win.Invalidate()
				return
			}

		case <-ticker.C:
			win.Invalidate()
		}
	}
}

func showGocleasyLogo(gtx C, margins layout.Inset) layout.FlexChild {

	// Show logo
	return layout.Flexed(1, func(gtx C) D {
		// Open the file using the file path
		file, err := imageFile.Open("images/gocleasy-logo.png")
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

func (applogic *AppLogic) HomePage(gtx C, scanbutton *widget.Clickable, initialpathinput *widget.Editor, numfilesdeleted int64, sizeliberated int64) D {

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

func (applogic *AppLogic) FillFirstLayer2Show() {

	for _, file := range applogic.Files.Files {
		applogic.Files2Show = append(applogic.Files2Show, &files.FileShow{
			File:         file,
			IsSelected:   widget.Bool{},
			ActionButton: widget.Bool{},
		})
	}
}

func (applogic *AppLogic) ShowLoadingPage(gtx C, actualFilesRead int) D {

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
		createTextNLoading(gtx, applogic.theme, fmt.Sprintf("%d", actualFilesRead)),
		layout.Rigid(
			layout.Spacer{Height: unit.Dp(25)}.Layout,
		),
	)
}

func selectFilesTableRow(th *material.Theme, file *files.FileShow, numchildren string, filepath string) []layout.FlexChild {

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
			return material.Body1(th, humanize.Bytes(uint64(file.File.Size))).Layout(gtx)
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

func (applogic *AppLogic) ShowFiles(gtx C, nextbutton *widget.Clickable, filelist *widget.List) D {

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

// Checks if ref is inside slf
func isFileSelected(ref *files.File, slf []*files.File) bool {

	for _, file := range slf {
		if file == ref {
			return true
		}
	}
	return false
}

// Calculates how many files need to be deleted from the slice when a folder is closed
// It loops from the actual position till the end of the slice or till a file with lower
// level than the folder being closed
func getNumFiles2NotShow(pos int, level int, sl []*files.FileShow) int {

	var res int = 0

	for ; pos < len(sl); pos++ {
		file := sl[pos]
		if file.File.Level <= level {
			return res
		}
		res++
	}

	return res
}

// It loops over Files2Show and checks if there is any checkbox has been clicked to open a folder.
// It also checks if any folder/file has been selected and adds it to Selfiles
func (applogic *AppLogic) getFiles2Show() {

	var file *files.FileShow

	index := 0
	for index < len(applogic.Files2Show) {

		file = applogic.Files2Show[index]

		// Check Open/Close folders
		if file.ActionButton.Changed() && file.File.IsDir {
			if file.ActionButton.Value {
				// Add children from Files2Show (Open folder)

				// Create temporal slice to add to children (Files2Show)
				slice2add := []*files.FileShow{}
				for _, file2append := range file.File.Files {

					// Check if the file was selected before to add it selected
					slice2add = append(slice2add, &files.FileShow{
						File:         file2append,
						IsSelected:   widget.Bool{Value: isFileSelected(file2append, applogic.Selfiles)},
						ActionButton: widget.Bool{},
					})
				}

				// Insert temporal slice into files to show
				applogic.Files2Show = append(applogic.Files2Show[:index+1], append(slice2add, applogic.Files2Show[index+1:]...)...)

			} else {
				// Delete children from Files2Show (Close folder)
				numFiles2NotShow := getNumFiles2NotShow(index+1, file.File.Level, applogic.Files2Show)
				applogic.Files2Show = append(applogic.Files2Show[:index+1], applogic.Files2Show[index+1+numFiles2NotShow:]...)
			}
		}

		// Check selected files
		if file.IsSelected.Changed() {
			if file.IsSelected.Value {
				// Add file to Selfiles
				applogic.Selfiles = append(applogic.Selfiles, file.File)

			} else {
				// Delete file from Selfiles
				for id, delfile := range applogic.Selfiles {
					if delfile == file.File {
						applogic.Selfiles = append(applogic.Selfiles[:id], applogic.Selfiles[id+1:]...)
					}
				}

			}
		}

		index++

	}
}

// Contains the file Tree
func (applogic *AppLogic) fileTree(gtx C, filelist *widget.List, path string) D {

	// empty the files to show
	var numfiles int = 0
	applogic.getFiles2Show()
	numfiles = len(applogic.Files2Show)

	if numfiles == 0 {
		return D{}
	}

	return filelist.List.Layout(gtx, numfiles, func(gtx C, index int) D {

		var file *files.FileShow = applogic.Files2Show[index]
		var widgets []layout.FlexChild
		var spacers []layout.FlexChild

		spacers = append(spacers, layout.Rigid(layout.Spacer{Width: unit.Dp(file.File.Level * 25)}.Layout))

		if file.File.IsDir {
			widgets = selectFilesTableRow(applogic.theme, file, humanize.Comma(file.File.NumChildren), fmt.Sprintf("%s/", filepath.Join(path, file.File.Name)))
		} else {
			widgets = selectFilesTableRow(applogic.theme, file, "-", filepath.Join(path, file.File.Name))
		}
		widgets = append(spacers, widgets...)
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, widgets...)
	})
}

func (applogic *AppLogic) ShowDeletingPage(gtx C, comebackbutton *widget.Clickable, copy2clipboard *widget.Clickable, deletebutton *widget.Clickable, filedeletelist *widget.List) D {

	margins := layout.Inset{
		Top:    unit.Dp(15),
		Bottom: unit.Dp(15),
		Right:  unit.Dp(15),
		Left:   unit.Dp(15),
	}

	var tot_files, tot_size int64 = 0, 0
	for _, file := range applogic.Selfiles {
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
				// Show copy to clipboard button
				layout.Flexed(1, func(gtx C) D {
					return margins.Layout(gtx, func(gtx C) D {
						return material.Button(applogic.theme, copy2clipboard, "Copy to Clipboard").Layout(gtx)
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

func (applogic *AppLogic) selectedFiles(gtx C, filedeletelist *widget.List) D {
	return filedeletelist.List.Layout(gtx, len(applogic.Selfiles), func(gtx C, index int) D {
		var selfile *files.File = applogic.Selfiles[index]
		var num_children, fullpath string
		if selfile.IsDir {
			fullpath = fmt.Sprintf("%s/", selfile.FullPath)
			num_children = humanize.Comma(selfile.NumChildren)
		} else {
			num_children = "-"
			fullpath = selfile.FullPath
		}

		return deleteFilesTableRow(gtx, applogic.theme, fullpath, num_children, humanize.Bytes(uint64(selfile.Size)))
	})
}
