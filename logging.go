package logging

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	u "github.com/onepif/go-utils"
)

const (
	NOTSET	int = iota
	ERROR
	WARN
	INFO
	DEBUG
	DEBUGEXT
	TRACE

	SKIP	int = 99
)

var (
	colorlvl = map[string]string {
		"error":	u.RED,
		"warn":		u.BROWN,
		"info":		u.GREEN,
		"debug":	u.CYAN,
		"debugext":	u.PURPLE,
		"trace":	u.BLUE,
	}
	LOGLEVELS = map[string]int {
		"notset":	NOTSET,
		"error":	ERROR,
		"warn":		WARN,
		"info":		INFO,
		"debug":	DEBUG,
		"debugext":	DEBUGEXT,
		"trace":	TRACE,
		"skip":		SKIP,
	}
	groupLogger = make(map[string]TlogDist)

	li = new(TlogInit)
)

type TlogDist struct {
	Term, File *log.Logger
}

type TlogInit struct {
	Verbose bool
	LogLevel string
	Fd *os.File
}

type TttySize struct {
	X, Y int
}

type TlogShell struct {
	Shell	string
	TTYsize TttySize
}

type TlogAlert struct {
	Err error
	Level string
	Msg string
}

type Settinger interface {
	set()
}

type Tverbose bool
type TlogLevel string
type Tfile os.File

func Set (self Settinger) {
	self.set()
}

func (self Tverbose) set() {
	li.Verbose = bool(self)
}

func (self TlogLevel) set() {
	li.LogLevel = string(self)
}

func (self *Tfile) set() {
	*li.Fd = os.File(*self)
}

func GetVerbose() bool {
	return li.Verbose
}

func GetLogLevel() string {
	return li.LogLevel
}

func GetFd() *os.File {
	return li.Fd
}

func New(self *TlogInit) {
	li = self

	for ix, _ := range LOGLEVELS {
		if ix == "notset" {
			groupLogger[ix] = TlogDist {
				log.New(os.Stdout, fmt.Sprintf("[ %s..%s ] ", u.GREEN, u.RESET), log.Lmsgprefix),
				log.New(li.Fd, "[ .. ] ", log.Lmsgprefix),
			}
		} else {
			groupLogger[ix] = TlogDist {
				log.New(os.Stdout, fmt.Sprintf("[ %s%s%s%s ] - ", colorlvl[ix], u.BOLD, strings.ToUpper(ix), u.RESET), log.Ltime|log.Lmsgprefix),
				log.New(li.Fd, fmt.Sprintf("[ %s ] - ", strings.ToUpper(ix)), log.Ltime|log.Lmsgprefix),
			}
		}
	}
}

//func Alert(e error, level string, msg *string) {
func Alert(self *TlogAlert) {
	if self.Level == "" {
		if self.E != nil { self.Level = "warn" } else { self.Level = "info" }
	}

	if LOGLEVELS[string(li.LogLevel)] >= LOGLEVELS[self.Level] {
		if self.Level == "notset" {
			if li.Verbose { groupLogger[self.Level].Term.Printf("%s", self.Msg) }
			groupLogger[self.Level].File.Printf("%s", self.Msg)
		} else {
			if self.E != nil {
				if li.Verbose { groupLogger[self.Level].Term.Printf("%s [ %s%v%s ]\n", self.Msg, u.BROWN, self.E, u.RESET) }
				groupLogger[self.Level].File.Printf("%s [ %v ]\n", self.Msg, self.E)
			} else {
				if li.Verbose { groupLogger[self.Level].Term.Println(self.Msg) }
				groupLogger[self.Level].File.Println(self.Msg)
			}
		}
	}
}

func (self *TlogShell) ShellExec(command *string) (*string, error) {
	var buf bytes.Buffer

	cmd := exec.Command(self.Shell, "-c", *command)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	if li.Fd != nil {
		cmd.Stdout = io.MultiWriter(&buf, li.Fd)
		if li.Verbose { cmd.Stderr = io.MultiWriter(os.Stderr, li.Fd) } else { cmd.Stderr = li.Fd}
	} else {
		cmd.Stdout = &buf
		if li.Verbose { cmd.Stderr = os.Stderr }
	}

	e := cmd.Run()
 
 	str := strings.TrimSpace(buf.String())

	return &str, e
}

func (self *TlogShell) Dialog(command, backTitle, title, textBox *string, typeBox string) error {
	var (
		r	*io.PipeReader
		w	*io.PipeWriter
	)

	if ! li.Verbose {
		r, w = io.Pipe()
		defer r.Close()
	}

	cmd := exec.Command(self.Shell, "-c", *command)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	if li.Fd != nil {
		if li.Verbose {
			cmd.Stdout = io.MultiWriter(os.Stdout, li.Fd)
			cmd.Stderr = io.MultiWriter(os.Stderr, li.Fd)
		} else {
			cmd.Stdout = io.MultiWriter(w, li.Fd)
//			cmd.Stderr = io.MultiWriter(w, li.Fd) // а надо ли ?? может только в fd??
			cmd.Stderr = li.Fd
		}
	} else {
		if li.Verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		} else {
			cmd.Stdout = w
			cmd.Stderr = w // а надо ли ??
		}
	}

	if ! li.Verbose {
		go func() {
			//--no-tags 
			cmd := exec.Command(self.Shell, "-c", fmt.Sprintf("dialog --stdout --backtitle \"%s\" --title \"%s\" --%s \"%s\" %d %d", *backTitle, *title, typeBox, *textBox, self.TTYsize.Y, self.TTYsize.X))
			cmd.Stdin = r
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stdout // Stderr
			cmd.Run()
		}()
	}

	return cmd.Run()
}

func (self *TlogShell) DialogExec(command, backTitle, textBox *string) error {
	return self.Dialog(command, backTitle, new(string), textBox, "progressbox")
}

func (self *TlogShell) DialogInfo(backTitle, title, textBox *string) error {
	cmd := fmt.Sprintf(`dialog --stdout --backtitle "%s" --title "%s" --infobox "%s" %d %d`,
		*backTitle,
		*title,
		*textBox,
		self.TTYsize.Y, self.TTYsize.X)

	_, e := self.ShellExec(&cmd)
	return e
//	return self.Dialog(new(string), backTitle, title, textBox, "infobox")
}

func (self *TlogShell) DialogYesNo(backTitle, textBox *string) error {
	cmd := fmt.Sprintf(`dialog --stdout --backtitle "%s" --yesno "%s" %d %d`,
		*backTitle,
		*textBox,
		self.TTYsize.Y, self.TTYsize.X)

	_, e := self.ShellExec(&cmd)
	return e
//	return self.Dialog(new(string), backTitle, new(string), textBox, "yesno")
}

func (self *TlogShell) DialogMsgBox(backTitle, title, textBox *string) error {
	cmd := fmt.Sprintf(`dialog --stdout --backtitle "%s" --title "%s" --msgbox "%s" %d %d`,
		*backTitle,
		*title,
		*textBox,
		self.TTYsize.Y, self.TTYsize.X)

//	m := fmt.Sprintf("DialogMsgBox: %v", self); Alert(nil, "debug", &m)

	_, e := self.ShellExec(&cmd)
	return e
//	return self.Dialog(new(string), backTitle, title, textBox, "mbox")
}

func (self *TlogShell) DialogCheckList(backTitle, title, textBox, extField *string) (*string, error) {
	cmd := fmt.Sprintf(`dialog --stdout --title "%s" --backtitle "%s" --no-tags --checklist "%s" %d %d 0 %s`,
		*title,
		*backTitle,
		*textBox,
		self.TTYsize.Y, self.TTYsize.X,
		*extField)

	return self.ShellExec(&cmd)
}

func (self *TlogShell) DialogInputBox(backTitle, title, textBox, extField *string) (*string, error) {
	cmd := fmt.Sprintf(`dialog --stdout --title "%s" --backtitle "%s" --no-tags --inputbox "%s" %d %d %s`,
		*title,
		*backTitle,
		*textBox,
		self.TTYsize.Y, self.TTYsize.X,
		*extField)

	return self.ShellExec(&cmd)
}

//dialog --stdout --title title --backtitle backtitle --no-tags --checklist checklist 31 160 0 1 a a 2 b b 3 c c