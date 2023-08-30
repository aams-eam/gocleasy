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
	theme      *material.Theme // Store the them of the application
	Files      *files.File     // Used to store the files with their structure
	Selfiles   []*files.File   // Used to store the files that has been selected
	files2show []*files.File   // Used to store the filest that are going to be rendered
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

func selectFilesTableRow(th *material.Theme, file *files.File, numchildren string, filepath string) []layout.FlexChild {

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

// Loops over a file tree and fills the second argument with the files that need to be shown.
// If the file needs to be shown its ActionButton.Value will be true
func getFiles2Show(children []*files.File, files2show []*files.File) ([]*files.File, int) {
	var numfiles int = 0
	var numtmp int = 0
	var file *files.File

	for i := range children {

		numtmp = 0
		file = children[i]

		files2show = append(files2show, file) // append the file to show
		numfiles++

		// Show the files inside if it is a directory and is marked to be shown
		// also consider that if the directory is selected to be deleted, do not show
		if file.IsSelected.Value {
			file.ActionButton.Value = false

		} else if file.IsDir && file.ActionButton.Value && file.Files != nil { // If the file is a dir and the value is true (show)
			files2show, numtmp = getFiles2Show(file.Files, files2show)
			numfiles += numtmp
		}
	}

	return files2show, numfiles
}

// Contains the file Tree
func (applogic *AppLogic) fileTree(gtx C, filelist *widget.List, path string) D {

	// empty the files to show
	applogic.files2show = nil
	var numfiles int
	applogic.files2show, numfiles = getFiles2Show(applogic.Files.Files, applogic.files2show)

	if numfiles == 0 {
		return D{}
	}

	return filelist.List.Layout(gtx, numfiles, func(gtx C, index int) D {

		var file *files.File = applogic.files2show[index]
		var widgets []layout.FlexChild
		var spacers []layout.FlexChild

		spacers = append(spacers, layout.Rigid(layout.Spacer{Width: unit.Dp(file.Level * 25)}.Layout))

		if file.IsDir {
			widgets = selectFilesTableRow(applogic.theme, file, humanize.Comma(file.NumChildren), fmt.Sprintf("%s/", filepath.Join(path, file.Name)))
		} else {
			widgets = selectFilesTableRow(applogic.theme, file, "-", filepath.Join(path, file.Name))
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
