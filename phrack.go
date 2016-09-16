package main

import (
	"archive/tar"
	"fmt"
	"github.com/jroimartin/gocui"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

/**
 *  Primary structure for the Phrack Issue data.
 */
type Phracked struct {
  wg         sync.WaitGroup
  status     chan string
  issue      string
  url        string
  tempPrefix string
  temp       string
  pages int
  tgz        *os.File
  filePath   string
}

/**
 *  Make sure we clean up after ourselves.
 */
func (p *Phracked) cleanPhracked() {
	if p.temp != "" {
		p.tgz.Close()
		os.RemoveAll(p.temp)
	}
}

/**
 *  Initialize all the data in the struct based on the issue number.
 */
func (p *Phracked) initPhracked(issue string) {
	p.cleanPhracked()
	var err error
	p.status = make(chan string, 1)
	p.issue = issue
	p.url = "http://www.phrack.org/archives/tgz/phrack" + p.issue + ".tar.gz"
	p.tempPrefix = "issue-" + p.issue + "-"
	p.temp, err = ioutil.TempDir("", p.tempPrefix)
	if err != nil {
		log.Fatal(err)
	}
	p.filePath = p.temp + "/" + p.issue + ".tar.gz"
	p.tgz, err = os.Create(p.filePath)
	if err != nil {
		log.Fatal(err)
	}
}

/**
 * Count the pages for the current Phrack issue.
 */
func (p *Phracked) countPages(g *gocui.Gui) {
  files, err := ioutil.ReadDir(p.temp)
  if err != nil {
    log.Fatal(err)
  }
  p.pages = 0
  for _, file := range files {
    if strings.HasSuffix(file.Name(), ".txt") {
      p.pages++
    }
  }
}


/**
 *  Figure out what issue to start with.
 */
func init() {
	issue := "1"
	if len(os.Args) > 1 {
		issue = os.Args[1]
	}
	phracked.initPhracked(issue)
}

/**
 *  Creating in global scope for ease of access.
 */
var phracked = new(Phracked)
var g = gocui.NewGui()

func main() {
	
	if err := g.Init(); err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.SetLayout(layout)
	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}

	defer phracked.cleanPhracked()
	phracked.wg.Add(1)
	go grabUrl()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}

	phracked.wg.Wait()
}

func clearStatus() {
	statusView, err := g.View("status")
	if err != nil {
		panic(err)
	}
	statusView.Clear()
	statusView.SetCursor(0, 0)
	statusView.SetOrigin(0, 0)
}

func updateTitle(title string) {
	mainView, err := g.View("main")
	if err != nil {
		panic(err)
	}
	mainView.Title = title
}

func updateStatus(g *gocui.Gui, status string) {
	statusView, err := g.View("status")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(statusView, "%s", status)
	if strings.HasSuffix(status, "\n") {
		cx, cy := statusView.Cursor()
		statusView.SetCursor(cx, cy+1)
		if cy == 3 {
			ox, oy := statusView.Origin()
			statusView.SetOrigin(ox, oy+1)
		}
	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("side", 0, 0, int(math.Abs(float64(maxX-86))), maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Highlight = true
		v.Title = "Pages"
		fmt.Fprintf(v, "%s\n", "Loading...")
	}

	if v, err := g.SetView("main", maxX-86, 0, maxX-2, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Phracked Issue #1"

		v.Editable = false
		v.Wrap = true
		if err := g.SetCurrentView("main"); err != nil {
			return err
		}
	}

	if v, err := g.SetView("status", -1, maxY-5, maxX, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = false
		v.Wrap = true
	}
	return nil
}

func nextView(g *gocui.Gui, v *gocui.View) error {
	if v == nil || v.Name() == "side" {
		return g.SetCurrentView("main")
	}
	return g.SetCurrentView("side")
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		ox, oy := v.Origin()
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}
	return nil
}

func cursorSelect(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		_, cy := v.Cursor()
		l, err := v.Line(cy)
		if err != nil {
			l = ""
		}
		if l == "load" {
			maxX, maxY := g.Size()
			if msg, err := g.SetView("msg", maxX/2-30, maxY/2, maxX/2+30, maxY/2+2); err != nil {
				msg.Editable = true
				msg.Highlight = true
				msg.Title = "Issue Number To Load"
				if err != gocui.ErrUnknownView {
					return err
				}
				if err := g.SetCurrentView("msg"); err != nil {
					return err
				}
			}
			return nil
		} else {
			updateMainFile(l+".txt")
		}

	}
	return nil
}

func loadIssue(g *gocui.Gui, v *gocui.View) error {
	v.Rewind()
	vb := v.ViewBuffer()
	reg, err := regexp.Compile("[^0-9]+")
	if err != nil {
		log.Fatal(err)
	}
	safer := reg.ReplaceAllString(vb, "")
	if err := g.DeleteView("msg"); err != nil {
		return err
	}
	if err := g.SetCurrentView("main"); err != nil {
		return err
	}
	phracked.initPhracked(safer)
	phracked.wg.Add(1)
	go grabUrl()
	return nil
}

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("side", gocui.KeyTab, gocui.ModNone, nextView); err != nil {
		return err
	}
	if err := g.SetKeybinding("main", gocui.KeyTab, gocui.ModNone, nextView); err != nil {
		return err
	}
	if err := g.SetKeybinding("side", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("side", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("side", gocui.KeyEnter, gocui.ModNone, cursorSelect); err != nil {
		return err
	}
	if err := g.SetKeybinding("main", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("main", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("msg", gocui.KeyEnter, gocui.ModNone, loadIssue); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	phracked.status <- "done"
	return gocui.ErrQuit
}

func untar(tarball, target string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}

func initSide() {
	v, err := g.View("side")
	if err != nil {
		panic(err)
	}
	v.Clear()
	fmt.Fprintf(v, "%s\n", "load")
	for i := 1; i <= phracked.pages; i++ {
		fmt.Fprintf(v, "%s\n", strconv.Itoa(i))
	}
	updateMainFile("1.txt")
}

func updateMainFile(path string) {
	path = phracked.temp + "/" + path
	mainView, err := g.View("main")
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		mainView.Clear()
		fmt.Fprintf(mainView, "Can't find file...")
	} else {
		mainView.Clear()
		mainView.SetCursor(0, 0)
		mainView.SetOrigin(0, 0)
		fmt.Fprintf(mainView, "%s", b)
		if err := g.SetCurrentView("main"); err != nil {
			log.Fatal(err)
		}
	}
}

func grabUrl() {
	defer phracked.wg.Done()
	clearStatus()
	updateTitle("Phrack Issue #" + phracked.issue)
	go func() {
		phracked.status <- "Fetching " + phracked.url + "..."
		resp, err := http.Get(phracked.url)
		if err != nil {
			phracked.cleanPhracked()
			log.Fatal(err)
		}
		phracked.status <- "\nDownload Complete...\n"
		_, err = io.Copy(phracked.tgz, resp.Body)
		phracked.status <- "Wrote to " + phracked.filePath + "\n"
		if err != nil {
			phracked.cleanPhracked()
			log.Fatal(err)
		}
		phracked.status <- "Unpacking tar.gz..."
		err = untar(phracked.filePath, phracked.temp)
		if err != nil {
			phracked.cleanPhracked()
			log.Fatal(err)
		}
		phracked.status <- "Issue unpacked\n"
		phracked.status <- "Building UI\n"
		phracked.countPages(g)
		initSide()
		phracked.status <- "done"
	}()

	for {
		select {
		case status := <-phracked.status:
      if status == "done" {
        return
      }
			g.Execute(func(g *gocui.Gui) error {
				updateStatus(g, status)
				return nil
			})
		case <-time.After(1000 * time.Millisecond):
			g.Execute(func(g *gocui.Gui) error {
				updateStatus(g, ".")
				return nil
			})
		}
	}
}
