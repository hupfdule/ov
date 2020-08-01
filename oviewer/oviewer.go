package oviewer

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/gdamore/tcell"
	"gitlab.com/tslocum/cbind"
)

// Root structure contains information about the drawing.
type Root struct {
	// tcell.Screen is the root screen.
	tcell.Screen
	// Config contains settings that determine the behavior of ov.
	Config

	// Doc contains the model of ov
	Doc *Document
	// help
	helpDoc *Document
	// DocList
	DocList    []*Document
	CurrentDoc int

	// input contains the input mode.
	input *Input
	// keyConfig contains the binding settings for the key.
	keyConfig *cbind.Configuration

	// message is the message to display.
	message string

	// vWidth represents the screen width.
	vWidth int
	// vHight represents the screen height.
	vHight int

	// startX is the start position of x.
	startX int

	// wrapHeaderLen is the actual header length when wrapped.
	wrapHeaderLen int
	// bottomPos is the position of the last line displayed.
	bottomPos int
	// statusPos is the position of the status line.
	statusPos int
	// minStartX is the minimum start position of x.
	minStartX int
}

// status structure contains the status of the display.
type status struct {
	// TabWidth is tab stop num.
	TabWidth int
	// HeaderLen is number of header rows to be fixed.
	Header int
	// Color to alternate rows
	AlternateRows bool
	// Column mode
	ColumnMode bool
	// Line Number
	LineNumMode bool
	// Wrap is Wrap mode.
	WrapMode bool
	// Column Delimiter
	ColumnDelimiter string
}

// Config represents the settings of ov.
type Config struct {
	// Alternating background color.
	ColorAlternate string
	// Header color.
	ColorHeader string
	// OverStrike color.
	ColorOverStrike string
	// OverLine color.
	ColorOverLine string

	Status status

	// AfterWrite writes the current screen on exit.
	AfterWrite bool
	// QuiteSmall Quit if the output fits on one screen.
	QuitSmall bool
	// CaseSensitive is case-sensitive if true
	CaseSensitive bool
	// Debug represents whether to enable the debug output.
	Debug bool
	// KeyBinding
	Keybind map[string][]string
}

var (
	// HeaderStyle represents the style of the header.
	HeaderStyle = tcell.StyleDefault.Bold(true)
	// ColorAlternate represents alternating colors.
	ColorAlternate = tcell.ColorGray
	// OverStrikeStyle represents the overstrike style.
	OverStrikeStyle = tcell.StyleDefault.Bold(true)
	// OverLineStyle represents the overline underline style.
	OverLineStyle = tcell.StyleDefault.Underline(true)
)

var (
	// ErrOutOfRange indicates that value is out of range.
	ErrOutOfRange = errors.New("out of range")
	// ErrFatalCache indicates that the cache value had a fatal error.
	ErrFatalCache = errors.New("fatal error in cache value")
	// ErrMissingFile indicates that the file does not exist.
	ErrMissingFile = errors.New("missing filename")
	// ErrNotFound indicates not found.
	ErrNotFound = errors.New("not found")
	// ErrInvalidNumber indicates an invalid number.
	ErrInvalidNumber = errors.New("invalid number")
	// ErrFailedKeyBind indicates keybinding failed.
	ErrFailedKeyBind = errors.New("failed to set keybind")
)

// NewOviewer return the structure of oviewer.
func NewOviewer(docs ...*Document) (*Root, error) {
	root := &Root{
		minStartX: -10,
	}
	root.keyConfig = cbind.NewConfiguration()
	root.DocList = append(root.DocList, docs...)
	root.Doc = root.DocList[0]
	root.input = NewInput()

	return root, nil
}

func (root *Root) screenInit() error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err = screen.Init(); err != nil {
		return err
	}
	root.Screen = screen
	return nil
}

// Open reads the file named of the argument and return the structure of oviewer.
func Open(fileNames ...string) (*Root, error) {
	if len(fileNames) == 0 {
		return openSTDIN()
	}
	return openFiles(fileNames)
}

func openSTDIN() (*Root, error) {
	doc, err := NewDocument()
	if err != nil {
		return nil, err
	}
	err = doc.ReadFile("")
	if err != nil {
		return nil, err
	}
	return NewOviewer(doc)
}

func openFiles(fileNames []string) (*Root, error) {
	docList := make([]*Document, 0)
	for _, fileName := range fileNames {
		m, err := NewDocument()
		if err != nil {
			return nil, err
		}
		err = m.ReadFile(fileName)
		if err != nil {
			return nil, err
		}
		docList = append(docList, m)
	}

	return NewOviewer(docList...)
}

// SetConfig sets config.
func (root *Root) SetConfig(config Config) {
	root.Config = config
}

func (root *Root) setKeyConfig() error {
	for _, doc := range root.DocList {
		doc.status = root.Config.Status
	}

	keyBind := GetKeyBinds(root.Config.Keybind)
	if err := root.setKeyBind(keyBind); err != nil {
		return err
	}

	help, err := NewHelp(keyBind)
	if err != nil {
		return err
	}
	root.helpDoc = help
	return nil
}

// NewHelp generates a document for help.
func NewHelp(k KeyBind) (*Document, error) {
	help, err := NewDocument()
	if err != nil {
		return nil, err
	}
	help.FileName = "Help"
	str := KeyBindString(k)
	help.lines = append(help.lines, "\t\t\tov help\n")
	help.lines = append(help.lines, strings.Split(str, "\n")...)
	help.eof = true
	help.endNum = len(help.lines)
	return help, err
}

// Run starts the terminal pager.
func (root *Root) Run() error {
	if err := root.setKeyConfig(); err != nil {
		return err
	}

	if err := root.screenInit(); err != nil {
		return err
	}
	defer root.Screen.Fini()

	root.setGlobalStyle()
	root.Screen.Clear()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
	go func() {
		<-c
		root.Screen.Fini()
		os.Exit(1)
	}()

	root.viewSync()
	// Exit if fits on screen
	if root.QuitSmall && root.contentsSmall() {
		root.AfterWrite = true
		return nil
	}

	root.main()

	return nil
}

// setDocument sets the Document.
func (root *Root) setDocument(m *Document) {
	root.Doc = m
	root.viewSync()
}

// Help is to switch between Help screen and normal screen.
func (root *Root) Help() {
	if root.input.mode == Help {
		root.toNormal()
		return
	}
	root.toHelp()
}

func (root *Root) toHelp() {
	root.setDocument(root.helpDoc)
	root.input.mode = Help
}

func (root *Root) toNormal() {
	root.setDocument(root.DocList[root.CurrentDoc])
	root.input.mode = Normal
}

// setGlobalStyle sets some styles that are determined by the settings.
func (root *Root) setGlobalStyle() {
	if root.ColorAlternate != "" {
		ColorAlternate = tcell.GetColor(root.ColorAlternate)
	}
	if root.ColorHeader != "" {
		HeaderStyle = HeaderStyle.Foreground(tcell.GetColor(root.ColorHeader))
	}
	if root.ColorOverStrike != "" {
		OverStrikeStyle = OverStrikeStyle.Foreground(tcell.GetColor(root.ColorOverStrike))
	}
	if root.ColorOverLine != "" {
		OverLineStyle = OverLineStyle.Foreground(tcell.GetColor(root.ColorOverLine))
	}
}

// prepareView prepares when the screen size is changed.
func (root *Root) prepareView() {
	screen := root.Screen
	root.vWidth, root.vHight = screen.Size()
	root.setWrapHeaderLen()
	root.statusPos = root.vHight - 1
}

// contentsSmall returns with bool whether the file to display fits on the screen.
func (root *Root) contentsSmall() bool {
	root.prepareView()
	m := root.Doc
	hight := 0
	for y := 0; y < m.BufEndNum(); y++ {
		hight += 1 + (len(m.getContents(y, root.Doc.TabWidth)) / root.vWidth)
		if hight > root.vHight {
			return false
		}
	}
	return true
}

// WriteOriginal writes to the original terminal.
func (root *Root) WriteOriginal() {
	m := root.Doc
	for i := 0; i < root.vHight-1; i++ {
		n := root.Doc.lineNum + i
		if n >= m.BufEndNum() {
			break
		}
		fmt.Println(m.GetLine(n))
	}
}

// headerLen returns the actual number of lines in the header.
func (root *Root) headerLen() int {
	if root.Doc.WrapMode {
		return root.wrapHeaderLen
	}
	return root.Doc.Header
}

// setWrapHeaderLen sets the value in wrapHeaderLen.
func (root *Root) setWrapHeaderLen() {
	m := root.Doc
	root.wrapHeaderLen = 0
	for y := 0; y < root.Doc.Header; y++ {
		root.wrapHeaderLen += 1 + (len(m.getContents(y, root.Doc.TabWidth)) / root.vWidth)
	}
}

// bottomLineNum returns the display start line
// when the last line number as an argument.
func (root *Root) bottomLineNum(num int) int {
	m := root.Doc
	if !root.Doc.WrapMode {
		if num <= root.vHight {
			return 0
		}
		return num - (root.vHight - root.Doc.Header) + 1
	}

	for y := root.vHight - root.wrapHeaderLen; y > 0; {
		y -= 1 + (len(m.getContents(num, root.Doc.TabWidth)) / root.vWidth)
		num--
	}
	num++
	return num
}

// toggleWrapMode toggles wrapMode each time it is called.
func (root *Root) toggleWrapMode() {
	root.Doc.WrapMode = !root.Doc.WrapMode
	root.Doc.x = 0
	root.setWrapHeaderLen()
}

//  toggleColumnMode toggles ColumnMode each time it is called.
func (root *Root) toggleColumnMode() {
	root.Doc.ColumnMode = !root.Doc.ColumnMode
}

// toggleAlternateRows toggles the AlternateRows each time it is called.
func (root *Root) toggleAlternateRows() {
	root.Doc.ClearCache()
	root.Doc.AlternateRows = !root.Doc.AlternateRows
}

// toggleLineNumMode toggles LineNumMode every time it is called.
func (root *Root) toggleLineNumMode() {
	root.Doc.LineNumMode = !root.Doc.LineNumMode
	root.updateEndNum()
}

// resize is a wrapper function that calls viewSync.
func (root *Root) resize() {
	root.viewSync()
}

// Sync redraws the whole thing.
func (root *Root) viewSync() {
	root.prepareStartX()
	root.prepareView()
	root.draw()
}

// prepareStartX prepares startX.
func (root *Root) prepareStartX() {
	root.startX = 0
	if root.Doc.LineNumMode {
		root.startX = len(fmt.Sprintf("%d", root.Doc.BufEndNum())) + 1
	}
}

// updateEndNum updates the last line number.
func (root *Root) updateEndNum() {
	root.prepareStartX()
	root.statusDraw()
}

// goLine will move to the specified line.
func (root *Root) goLine(input string) {
	lineNum, err := strconv.Atoi(input)
	if err != nil {
		root.message = ErrInvalidNumber.Error()
		return
	}

	root.moveLine(lineNum - root.Doc.Header - 1)
	root.message = fmt.Sprintf("Moved to line %d", lineNum)
}

// markLineNum stores the specified number of lines.
func (root *Root) markLineNum() {
	s := strconv.Itoa(root.Doc.lineNum + 1)
	root.input.GoCandidate.list = toLast(root.input.GoCandidate.list, s)
	root.input.GoCandidate.p = 0
	root.message = fmt.Sprintf("Marked to line %d", root.Doc.lineNum)
}

// setHeader sets the number of lines in the header.
func (root *Root) setHeader(input string) {
	lineNum, err := strconv.Atoi(input)
	if err != nil {
		root.message = ErrInvalidNumber.Error()
		return
	}
	if lineNum < 0 || lineNum > root.vHight-1 {
		root.message = ErrOutOfRange.Error()
		return
	}
	if root.Doc.Header == lineNum {
		return
	}

	root.Doc.Header = lineNum
	root.message = fmt.Sprintf("Set Header %d", lineNum)
	root.setWrapHeaderLen()
	root.Doc.ClearCache()
}

// setDelimiter sets the delimiter string.
func (root *Root) setDelimiter(input string) {
	root.Doc.ColumnDelimiter = input
	root.message = fmt.Sprintf("Set delimiter %s", input)
}

// setTabWidth sets the tab width.
func (root *Root) setTabWidth(input string) {
	width, err := strconv.Atoi(input)
	if err != nil {
		root.message = ErrInvalidNumber.Error()
		return
	}
	if root.Doc.TabWidth == width {
		return
	}

	root.Doc.TabWidth = width
	root.message = fmt.Sprintf("Set tab width %d", width)
	root.Doc.ClearCache()
}

func (root *Root) markNext() {
	root.goLine(newGotoInput(root.input.GoCandidate).Up(""))
}

func (root *Root) markPrev() {
	root.goLine(newGotoInput(root.input.GoCandidate).Down(""))
}

func (root *Root) nextDoc() {
	root.CurrentDoc++
	if len(root.DocList) <= root.CurrentDoc {
		root.CurrentDoc = root.CurrentDoc - 1
	}
	root.setDocument(root.DocList[root.CurrentDoc])
	root.input.mode = Normal
}

func (root *Root) previousDoc() {
	root.CurrentDoc--
	if root.CurrentDoc < 0 {
		root.CurrentDoc = 0
	}
	root.setDocument(root.DocList[root.CurrentDoc])
	root.input.mode = Normal
}