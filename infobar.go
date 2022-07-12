package dlog

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"dlog/filters"
	"dlog/logging"
	"dlog/runes"
	"dlog/utils"

	"github.com/nsf/termbox-go"
)

const promptLength = 1

type infoBarMode uint

const (
	ibModeStatus infoBarMode = iota
	ibModeSearch
	ibModeBackSearch
	ibModeFilter
	ibModeAppend
	ibModeExclude
	ibModeSave
	ibModeMessage
	ibModeKeepCharacters
	ibModeHighlight
)

type infoBar struct {
	y              int
	width          int
	cx             int // cursor position
	editBuffer     []rune
	mode           infoBarMode
	flock          *sync.RWMutex
	totalLines     LineNo
	currentLine    *Pos
	filtersEnabled *bool
	keepChars      *int
	history        ibHistory
	searchType     filters.SearchType
	message        ibMessage
	winName        string
}

type ibMessage struct {
	str   string
	color termbox.Attribute
}

const ibHistorySize = 1000

type ibHistory struct {
	buffer       [][]rune
	wlock        sync.RWMutex
	pos          int    // position from the end of file. New records appended, so 0 is always "before" last record with ==1 being last record
	currentInput []rune // when navigating from zero position will hold input use entered and displayed once back to zero Line
	loaded       bool
}

var historyPath string

func init() {
	dir := os.Getenv("DLOG_DIR")
	if len(dir) == 0 {
		dir = filepath.Join(utils.GetHomeDir(), ".dlog")
	}

	historyPath = filepath.Join(dir, "history")
}

func (v *infoBar) moveCursor(direction int) error {
	target := v.cx + direction
	if target < 0 {
		return errors.New("reached beginning of the Line")
	}
	if target > len(v.editBuffer) {
		return errors.New("reached end of the Line")
	}
	v.moveCursorToPosition(target)
	return nil
}

func (v *infoBar) reset(mode infoBarMode) {
	v.cx = 0
	v.editBuffer = v.editBuffer[:0]
	v.mode = mode
	v.draw()
}

func (v *infoBar) clear() {
	for i := 0; i < v.width; i++ {
		termbox.SetCell(i, v.y, ' ', termbox.ColorDefault, termbox.ColorDefault)
	}
}

func (v *infoBar) statusBar() {
	v.clear()
	v.message = ibMessage{}
	v.flock.Lock()
	defer v.flock.Unlock()
	str := []rune(fmt.Sprintf("%s/%d", *v.currentLine, v.totalLines))
	for i := 0; i < len(str); i++ {
		termbox.SetCell(v.width-len(str)+i, v.y, str[i], termbox.ColorYellow, termbox.ColorDefault)
	}

	name := []rune(v.winName)
	for i := 0; i < len(name) && i+1 < v.width; i++ {
		termbox.SetCell(i, v.y, name[i], termbox.ColorYellow, termbox.ColorDefault)
	}

	if !*v.filtersEnabled {
		str := []rune("[-FILTERS]")
		for i := 0; i < len(str) && i+1 < v.width; i++ {
			termbox.SetCell(i+1, v.y, str[i], termbox.ColorMagenta, termbox.ColorDefault)
		}
	}

	logging.LogOnErr(termbox.Flush())
}

func (v *infoBar) showSearch() {
	v.moveCursorToPosition(v.cx)
	v.syncSearchString()
}

func (v *infoBar) draw() {
	switch v.mode {
	case ibModeBackSearch:
		termbox.SetCell(0, v.y, '?', termbox.ColorGreen, termbox.ColorDefault)
		v.showSearch()
	case ibModeSearch:
		termbox.SetCell(0, v.y, '/', termbox.ColorGreen, termbox.ColorDefault)
		v.showSearch()
	case ibModeFilter:
		termbox.SetCell(0, v.y, '&', termbox.ColorGreen, termbox.ColorDefault)
		v.showSearch()
	case ibModeExclude:
		termbox.SetCell(0, v.y, '-', termbox.ColorGreen, termbox.ColorDefault)
		v.showSearch()
	case ibModeHighlight:
		termbox.SetCell(0, v.y, '~', termbox.ColorGreen, termbox.ColorDefault)
		v.showSearch()
	case ibModeSave:
		termbox.SetCell(0, v.y, '>', termbox.ColorMagenta, termbox.ColorDefault)
		v.showSearch()
	case ibModeAppend:
		termbox.SetCell(0, v.y, '+', termbox.ColorGreen, termbox.ColorDefault)
		v.showSearch()
	case ibModeKeepCharacters:
		termbox.SetCell(0, v.y, 'K', termbox.ColorGreen, termbox.ColorDefault)
		v.editBuffer = []rune(strconv.Itoa(*v.keepChars))
		v.showSearch()
		v.moveCursorToPosition(len(v.editBuffer))
	case ibModeStatus:
		v.statusBar()
	case ibModeMessage:
		v.showMessage()
	default:
		panic("Not implemented")
	}
}

func (v *infoBar) setInput(str string) {
	v.editBuffer = []rune(str)
	v.showSearch()
	v.moveCursorToPosition(len(v.editBuffer))
}

func (v *infoBar) setMessage(message ibMessage) {
	logging.Debug("Setting message", message)
	v.message = message
	v.reset(ibModeMessage)
}

func (v *infoBar) showMessage() {
	v.clear()
	logging.Debug("Showing message", v.message)
	str := []rune(v.message.str)
	for i := 0; i < len(str) && i+1 < v.width; i++ {
		logging.Debug("Adding char", str[i])
		termbox.SetCell(i+1, v.y, str[i], v.message.color, termbox.ColorDefault)
	}
	logging.LogOnErr(termbox.Flush())
}

func (v *infoBar) navigateWord(forward bool) {
	v.moveCursorToPosition(v.findWord(forward))
}

func (v *infoBar) findWord(forward bool) (pos int) {
	var addittor int
	var starter int
	if forward {
		addittor = +1
		pos = len(v.editBuffer)
	} else {
		starter = -1
		addittor = -1
		pos = 0
	}
	for i := v.cx + addittor + starter; 0 < i && i < len(v.editBuffer); i += addittor {
		if v.editBuffer[i] == ' ' {
			pos = i
			if !forward {
				pos++
			}
			break
		}
	}
	return
}

func (v *infoBar) deleteWord(forward bool) {
	pos := v.findWord(forward)
	var newPos int
	if forward {
		newPos = v.cx
		if pos >= len(v.editBuffer) {
			pos--
		}
		v.editBuffer = append(v.editBuffer[:v.cx], v.editBuffer[pos+1:]...)
	} else {
		newPos = pos
		v.editBuffer = append(v.editBuffer[:pos], v.editBuffer[v.cx:]...)
	}
	v.moveCursorToPosition(newPos)
	v.syncSearchString()
}

func (v *infoBar) moveCursorToPosition(pos int) {
	v.cx = pos
	termbox.SetCursor(pos+promptLength, v.y)

	logging.LogOnErr(termbox.Flush())
}

func (v *infoBar) moveCursorToEnd() {
	v.moveCursorToPosition(len(v.editBuffer))
}

func (v *infoBar) requestSearch() {
	searchString := append([]rune(nil), v.editBuffer...) // Buffer may be modified by concurrent reset
	searchMode := v.mode
	go func() {
		go func() {
			requestSearch <- infobarRequest{searchString, searchMode}
		}()
		termbox.Interrupt()
	}()
}

func (v *infoBar) resize(width, height int) {
	v.width = width
	v.y = height
}

func (v *infoBar) processKey(ev termbox.Event) (a action) {
	if ev.Ch != 0 || ev.Key == termbox.KeySpace {
		ch := ev.Ch
		if ev.Key == termbox.KeySpace {
			ch = ' '
		}
		v.editBuffer = runes.InsertRune(v.editBuffer, ch, v.cx)

		logging.LogOnErr(v.moveCursor(+1))

		v.syncSearchString()

	} else {
		switch ev.Key {
		case termbox.KeyEsc:
			logging.Debug("processing esc key")
			switch getEscKey(ev) {
			case ALT_LEFT_ARROW:
				v.navigateWord(false)
			case ALT_RIGHT_ARROW:
				v.navigateWord(true)
			case ALT_BACKSPACE:
				v.deleteWord(false)
			case ALT_D:
				v.deleteWord(true)
			case ESC:
				v.reset(ibModeStatus)
				return ACTION_RESET_FOCUS
			}
		case termbox.KeyEnter:
			v.addToHistory()
			v.requestSearch()
			v.reset(ibModeStatus)
			return ACTION_RESET_FOCUS
		case termbox.KeyArrowLeft:
			logging.LogOnErr(v.moveCursor(-1))
		case termbox.KeyArrowRight:
			logging.LogOnErr(v.moveCursor(+1))
		case termbox.KeyArrowUp:
			v.onKeyUp()
		case termbox.KeyArrowDown:
			v.onKeyDown()
		case termbox.KeyCtrlSlash:
			v.switchSearchType()
		case termbox.KeyCtrlR:
			v.switchSearchType()
		case termbox.KeyBackspace, termbox.KeyBackspace2:
			err := v.moveCursor(-1)
			if err == nil {
				v.editBuffer = runes.DeleteRune(v.editBuffer, v.cx)
				v.syncSearchString()
			}
		}
	}
	return
}
func (v *infoBar) switchSearchType() {
	switch v.mode {
	case ibModeExclude,
		ibModeAppend,
		ibModeSearch,
		ibModeBackSearch,
		ibModeHighlight,
		ibModeFilter:
		st := v.searchType
		nextID := st.ID + 1
		if _, ok := filters.SearchTypeMap[nextID]; !ok {
			nextID = 0
		}
		nextSt := filters.SearchTypeMap[nextID]
		v.searchType = nextSt
		v.draw()
	}
}

func (history *ibHistory) load() {
	if history.loaded {
		return
	}
	history.loaded = true

	f, err := os.Open(historyPath)
	if os.IsNotExist(err) {
		return
	}
	defer logging.LogOnErr(f.Close())

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		history.buffer = append(history.buffer, []rune(scanner.Text()))
	}
}

func (v *infoBar) addToHistory() {
	switch v.mode {
	case ibModeKeepCharacters, ibModeSave:
		return
	default:
		v.history.add(v.editBuffer)
	}
}

func (history *ibHistory) add(str []rune) {
	if len(str) == 0 {
		return // no need to save empty strings
	}
	history.load()
	data := make([]rune, len(str))
	copy(data, str)
	history.wlock.Lock()
	history.buffer = append(history.buffer, data)
	history.pos = 0
	history.wlock.Unlock()
	go history.save(str)
}

func (history *ibHistory) save(str []rune) {
	history.wlock.Lock()
	defer history.wlock.Unlock()
	err := os.MkdirAll(filepath.Dir(historyPath), os.ModePerm)
	logging.LogOnErr(err)

	f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		logging.Debug(fmt.Sprintf("Could not open history file: %s", err))
		return
	}
	defer logging.LogOnErr(f.Close())

	_, err = f.Write([]byte(string(str) + "\n"))
	logging.LogOnErr(err)

	logging.Debug("len, size", len(history.buffer), ibHistorySize)
	if len(history.buffer) >= ibHistorySize {
		history.trim()
	}
}

func (history *ibHistory) trim() {
	tmpPath := historyPath + "_tmp"
	tmpFile := utils.OpenRewrite(tmpPath)
	writer := bufio.NewWriter(tmpFile)
	keptHistory := history.buffer[len(history.buffer)-ibHistorySize/100*80:]
	for _, str := range keptHistory {
		_, err := writer.WriteString(string(str) + "\n")
		logging.LogOnErr(err)
	}

	if err := writer.Flush(); err != nil {
		logging.Debug("Could not write temporary history file")
		return
	}
	if err := tmpFile.Close(); err != nil {
		logging.Debug("Could not close temporary history file")
		return
	}
	history.buffer = keptHistory
	logging.LogOnErr(os.Rename(tmpPath, historyPath))
}

func (v *infoBar) onKeyUp() {
	switch v.mode {
	case ibModeKeepCharacters:
		v.changeKeepChars(+1)
	default:
		v.navigateHistory(+1)
	}
}

func (v *infoBar) onKeyDown() {
	switch v.mode {
	case ibModeKeepCharacters:
		v.changeKeepChars(-1)
	default:
		v.navigateHistory(-1)
	}
}

func (v *infoBar) navigateHistory(i int) {
	v.history.load()
	target := v.history.pos + i
	if len(v.history.buffer) == 0 {
		target = 0
	}
	if target > len(v.history.buffer) {
		target = len(v.history.buffer)
	}
	if target < 0 {
		target = 0
	}
	onPosChange := func() {
		v.moveCursorToEnd()
		v.syncSearchString()
	}
	if target == 0 {
		if v.history.pos != 0 {
			v.history.pos = target
			v.editBuffer = v.history.currentInput
			onPosChange()
		}
		return // Does not matter where we are going, but nothing to do here.
	}
	if v.history.pos == 0 { // Moved out from zero-search to existing search string
		v.history.currentInput = v.editBuffer
	}
	v.history.pos = target
	targetString := v.history.buffer[len(v.history.buffer)-target]
	v.editBuffer = make([]rune, len(targetString))
	copy(v.editBuffer, targetString)
	onPosChange()
}

func (v *infoBar) setPromptCell(x int, ch rune, fg, bg termbox.Attribute) {
	termbox.SetCell(x+promptLength, v.y, ch, fg, bg)
}

func (v *infoBar) syncSearchString() {
	// TODO: Does not handle well very narrow screen
	// TODO: All setCelling here need to be moved to some nicer wrapper funcs
	var color termbox.Attribute
	switch v.mode {
	case ibModeKeepCharacters:
		color = termbox.ColorYellow
	default:
		color = v.searchType.Color
	}
	for i := 0; i < v.width-promptLength; i++ {
		ch := ' '
		if i < len(v.editBuffer) {
			ch = v.editBuffer[i]
		}
		v.setPromptCell(i, ch, color, termbox.ColorDefault)
	}
	runeName := []rune(v.searchType.Name)
	for i := v.width - len(runeName); i < v.width && i > promptLength; i++ {
		c := i + len(runeName) - v.width
		termbox.SetCell(i, v.y, runeName[c], v.searchType.Color, termbox.ColorDefault)
	}
	logging.LogOnErr(termbox.Flush())
}

func (v *infoBar) changeKeepChars(direction int) {
	go func() {
		go termbox.Interrupt()
		requestKeepCharsChange <- direction
	}()
}
