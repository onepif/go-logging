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
	verbose bool
	logLevel string
	fd *os.File
}

type TttySize struct {
	X, Y int
}

type TlogShell struct {
	Shell	string
	TTYsize TttySize
}

type TlogAlert struct {
	e error
	level string
	msg string
}

type Settinger interface {
	set()
}

type Tverb struct {
	verb bool
}

type Tverbose bool
type TlogLevel string
type Tfile os.File

func Set (v Settinger) {
	v.set()
}

func (v Tverb) set() {
	li.verbose = v.verb
}

func (v Tverbose) set() {
	li.verbose = bool(v)
}

func (l TlogLevel) set() {
	li.logLevel = string(l)
}

func (f *Tfile) set() {
	*li.fd = os.File(*f)
}

func GetVerbose() bool {
	return li.verbose
}

func GetLogLevel() string {
	return li.logLevel
}

func GetFd() *os.File {
	return li.fd
}

func (self *TlogInit) New() {
/*	verbose = self.Verbose
	logLevel = self.LogLevel
	fd = self.Fd*/
	li = self

	for ix, _ := range LOGLEVELS {
		if ix == "notset" {
			groupLogger[ix] = TlogDist {
				log.New(os.Stdout, fmt.Sprintf("[ %s..%s ] ", u.GREEN, u.RESET), log.Lmsgprefix),
				log.New(li.fd, "[ .. ] ", log.Lmsgprefix),
			}
		} else {
			groupLogger[ix] = TlogDist {
				log.New(os.Stdout, fmt.Sprintf("[ %s%s%s%s ] - ", colorlvl[ix], u.BOLD, strings.ToUpper(ix), u.RESET), log.Ltime|log.Lmsgprefix),
				log.New(li.fd, fmt.Sprintf("[ %s ] - ", strings.ToUpper(ix)), log.Ltime|log.Lmsgprefix),
			}
		}
	}
}

//func Alert(e error, level string, msg *string) {
func Alert(a *TlogAlert) {
	if a.level == "" {
		if a.e != nil { a.level = "warn" } else { a.level = "info" }
	}

	if LOGLEVELS[string(li.logLevel)] >= LOGLEVELS[a.level] {
		if a.level == "notset" {
			if li.verbose { groupLogger[a.level].Term.Printf("%s", a.msg) }
			groupLogger[a.level].File.Printf("%s", a.msg)
		} else {
			if a.e != nil {
				if li.verbose { groupLogger[a.level].Term.Printf("%s [ %s%v%s ]\n", a.msg, u.BROWN, a.e, u.RESET) }
				groupLogger[a.level].File.Printf("%s [ %v ]\n", a.msg, a.e)
			} else {
				if li.verbose { groupLogger[a.level].Term.Println(a.msg) }
				groupLogger[a.level].File.Println(a.msg)
			}
		}
	}
}

func (self *TlogShell) ShellExec(command *string) (*string, error) {
	var buf bytes.Buffer

	cmd := exec.Command(self.Shell, "-c", *command)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	if li.fd != nil {
		cmd.Stdout = io.MultiWriter(&buf, li.fd)
		if li.verbose { cmd.Stderr = io.MultiWriter(os.Stderr, li.fd) } else { cmd.Stderr = li.fd}
	} else {
		cmd.Stdout = &buf
		if li.verbose { cmd.Stderr = os.Stderr }
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

	if ! li.verbose {
		r, w = io.Pipe()
		defer r.Close()
	}

	cmd := exec.Command(self.Shell, "-c", *command)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	if li.fd != nil {
		if li.verbose {
			cmd.Stdout = io.MultiWriter(os.Stdout, li.fd)
			cmd.Stderr = io.MultiWriter(os.Stderr, li.fd)
		} else {
			cmd.Stdout = io.MultiWriter(w, li.fd)
//			cmd.Stderr = io.MultiWriter(w, li.fd) // а надо ли ?? может только в fd??
			cmd.Stderr = li.fd
		}
	} else {
		if li.verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		} else {
			cmd.Stdout = w
			cmd.Stderr = w // а надо ли ??
		}
	}

	if ! li.verbose {
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