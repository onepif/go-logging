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
	LOGLEVELS = map[TlogLevel]int {
		"notset":	NOTSET,
		"error":	ERROR,
		"warn":		WARN,
		"info":		INFO,
		"debug":	DEBUG,
		"debugext":	DEBUGEXT,
		"trace":	TRACE,
		"skip":		SKIP,
	}
	GroupLogger = make(map[string]TLogDist)

	li = new(TLogInit)
/*	verbose	*bool
	logLevel string
	fd		*os.File*/
)

type TLogDist struct {
	Term, File	*log.Logger
}

type TLogInit struct {
	verbose		*Tverbose
	logLevel	TlogLevel
	fd			*TFile
}

type TttySize struct {
	X, Y int
}

type TLogShell struct {
	Shell	string
	TTYsize TttySize
}

type Settinger interface {
	Set()
}

type Tverbose bool
type TlogLevel string
type Tfile os.File

func Set (v Settinger) {
	v.set()
}

func (v *Tverbose) set() {
	li.verbose = v
}

func (l TlogLevel) set() {
	li.logLevel = l
}

func (f *Tfile) set() {
	li.fd = f
}

func GetVerbose() bool {
	return *li.verbose
}

func GetLogLevel() string {
	return li.logLevel
}

func GetFd() *os.File {
	return li.fd
}

func (self *TLogInit) New() {
/*	verbose = self.Verbose
	logLevel = self.LogLevel
	fd = self.Fd*/
	li = self

	for ix, _ := range LOGLEVELS {
		if ix == "notset" {
			GroupLogger[ix] = TLogDist {
				log.New(os.Stdout, fmt.Sprintf("[ %s..%s ] ", u.GREEN, u.RESET), log.Lmsgprefix),
				log.New(fd, "[ .. ] ", log.Lmsgprefix),
			}
		} else {
			GroupLogger[ix] = TLogDist {
				log.New(os.Stdout, fmt.Sprintf("[ %s%s%s%s ] - ", colorlvl[ix], u.BOLD, strings.ToUpper(ix), u.RESET), log.Ltime|log.Lmsgprefix),
				log.New(fd, fmt.Sprintf("[ %s ] - ", strings.ToUpper(ix)), log.Ltime|log.Lmsgprefix),
			}
		}
	}
}

func Alert(e error, level string, msg *string) {
	if level == "" {
		if e != nil { level = "warn" } else { level = "info" }
	}

	if LOGLEVELS[li.logLevel] >= LOGLEVELS[level] {
		if level == "notset" {
			if *li.verbose { GroupLogger[level].Term.Printf("%s", *msg) }
			GroupLogger[level].File.Printf("%s", *msg)
		} else {
			if e != nil {
				if *li.verbose { GroupLogger[level].Term.Printf("%s [ %s%v%s ]\n", *msg, u.BROWN, e, u.RESET) }
				GroupLogger[level].File.Printf("%s [ %v ]\n", *msg, e)
			} else {
				if *li.verbose { GroupLogger[level].Term.Println(*msg) }
				GroupLogger[level].File.Println(*msg)
			}
		}
	}
}

func (self *TLogShell) ShellExec(command *string) (*string, error) {
	var buf bytes.Buffer

	cmd := exec.Command(self.Shell, "-c", *command)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	if li.fd != nil {
		cmd.Stdout = io.MultiWriter(&buf, li.fd)
		if *li.verbose { cmd.Stderr = io.MultiWriter(os.Stderr, fd) } else { cmd.Stderr = li.fd}
	} else {
		cmd.Stdout = &buf
		if *li.verbose { cmd.Stderr = os.Stderr }
	}

	e := cmd.Run()
 
 	str := strings.TrimSpace(buf.String())

	return &str, e
}

func (self *TLogShell) Dialog(command, backTitle, title, textBox *string, typeBox string) error {
	var (
		r	*io.PipeReader
		w	*io.PipeWriter
	)

	if ! *li.verbose {
		r, w = io.Pipe()
		defer r.Close()
	}

	cmd := exec.Command(self.Shell, "-c", *command)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	if li.fd != nil {
		if *li.verbose {
			cmd.Stdout = io.MultiWriter(os.Stdout, li.fd)
			cmd.Stderr = io.MultiWriter(os.Stderr, li.fd)
		} else {
			cmd.Stdout = io.MultiWriter(w, li.fd)
//			cmd.Stderr = io.MultiWriter(w, li.fd) // а надо ли ?? может только в fd??
			cmd.Stderr = li.fd
		}
	} else {
		if *li.verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		} else {
			cmd.Stdout = w
			cmd.Stderr = w // а надо ли ??
		}
	}

	if ! *li.verbose {
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

func (self *TLogShell) DialogExec(command, backTitle, textBox *string) error {
	return self.Dialog(command, backTitle, new(string), textBox, "progressbox")
}

func (self *TLogShell) DialogInfo(backTitle, title, textBox *string) error {
	cmd := fmt.Sprintf(`dialog --stdout --backtitle "%s" --title "%s" --infobox "%s" %d %d`,
		*backTitle,
		*title,
		*textBox,
		self.TTYsize.Y, self.TTYsize.X)

	_, e := self.ShellExec(&cmd)
	return e
//	return self.Dialog(new(string), backTitle, title, textBox, "infobox")
}

func (self *TLogShell) DialogYesNo(backTitle, textBox *string) error {
	cmd := fmt.Sprintf(`dialog --stdout --backtitle "%s" --yesno "%s" %d %d`,
		*backTitle,
		*textBox,
		self.TTYsize.Y, self.TTYsize.X)

	_, e := self.ShellExec(&cmd)
	return e
//	return self.Dialog(new(string), backTitle, new(string), textBox, "yesno")
}

func (self *TLogShell) DialogMsgBox(backTitle, title, textBox *string) error {
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

func (self *TLogShell) DialogCheckList(backTitle, title, textBox, extField *string) (*string, error) {
	cmd := fmt.Sprintf(`dialog --stdout --title "%s" --backtitle "%s" --no-tags --checklist "%s" %d %d 0 %s`,
		*title,
		*backTitle,
		*textBox,
		self.TTYsize.Y, self.TTYsize.X,
		*extField)

	return self.ShellExec(&cmd)
}

func (self *TLogShell) DialogInputBox(backTitle, title, textBox, extField *string) (*string, error) {
	cmd := fmt.Sprintf(`dialog --stdout --title "%s" --backtitle "%s" --no-tags --inputbox "%s" %d %d %s`,
		*title,
		*backTitle,
		*textBox,
		self.TTYsize.Y, self.TTYsize.X,
		*extField)

	return self.ShellExec(&cmd)
}

//dialog --stdout --title title --backtitle backtitle --no-tags --checklist checklist 31 160 0 1 a a 2 b b 3 c c