package oviewer

// Go to the top line.
func (root *Root) moveTop() {
	root.moveLine(0)
}

// Go to the bottom line.
func (root *Root) moveBottom() {
	root.reRead()
	root.moveLine(root.Doc.endNum + 1)
}

// Move to the specified line.
func (root *Root) moveLine(num int) {
	root.resetSelect()
	root.Doc.lineNum = num
	root.Doc.branch = 0
}

// Move up one screen.
func (root *Root) movePgUp() {
	n := root.Doc.lineNum - root.realHightNum()
	if n >= root.Doc.lineNum {
		n = root.Doc.lineNum - 1
	}
	root.moveLine(n)
}

// Moves down one screen.
func (root *Root) movePgDn() {
	n := root.bottomPos - root.Doc.Header
	if n <= root.Doc.lineNum {
		n = root.Doc.lineNum + 1
	}
	root.moveLine(n)
}

// realHightNum returns the actual number of line on the screen.
func (root *Root) realHightNum() int {
	return root.bottomPos - (root.Doc.lineNum + root.Doc.Header)
}

// Moves up half a screen.
func (root *Root) moveHfUp() {
	root.moveLine(root.Doc.lineNum - (root.realHightNum() / 2))
}

// Moves down half a screen.
func (root *Root) moveHfDn() {
	root.moveLine(root.Doc.lineNum + (root.realHightNum() / 2))
}

// Move up one line.
func (root *Root) moveUp() {
	root.resetSelect()

	if !root.Doc.WrapMode {
		root.Doc.branch = 0
		root.Doc.lineNum--
		return
	}

	// WrapMode
	lc, err := root.Doc.lineToContents(root.Doc.lineNum+root.Doc.Header, root.Doc.TabWidth)
	if err != nil {
		return
	}
	if len(lc) < (root.vWidth-root.startX) || root.Doc.branch <= 0 {
		if (root.Doc.lineNum) >= 1 {
			pre, err := root.Doc.lineToContents(root.Doc.lineNum+root.Doc.Header-1, root.Doc.TabWidth)
			if err != nil {
				return
			}
			yyLen := len(pre) / ((root.vWidth - root.startX) + 1)
			root.Doc.branch = yyLen
		}
		root.Doc.lineNum--
		return
	}
	root.Doc.branch--
}

// Move down one line.
func (root *Root) moveDown() {
	root.resetSelect()

	if !root.Doc.WrapMode {
		root.Doc.branch = 0
		root.Doc.lineNum++
		return
	}

	// WrapMode
	lc, err := root.Doc.lineToContents(root.Doc.lineNum+root.Doc.Header, root.Doc.TabWidth)
	if err != nil {
		return
	}
	branch := (len(lc) / (root.vWidth - root.startX))
	if len(lc) < (root.vWidth-root.startX) || root.Doc.branch >= branch {
		root.Doc.branch = 0
		root.Doc.lineNum++
		return
	}
	root.Doc.branch++
}

// Move to the left.
func (root *Root) moveLeft() {
	root.resetSelect()
	if root.Doc.ColumnMode {
		if root.Doc.columnNum > 0 {
			root.Doc.columnNum--
			root.Doc.x = root.columnModeX()
		}
		return
	}
	if root.Doc.WrapMode {
		return
	}
	root.Doc.x--
	if root.Doc.x < root.minStartX {
		root.Doc.x = root.minStartX
	}
}

// Move to the right.
func (root *Root) moveRight() {
	root.resetSelect()
	if root.Doc.ColumnMode {
		root.Doc.columnNum++
		root.Doc.x = root.columnModeX()
		return
	}
	if root.Doc.WrapMode {
		return
	}
	root.Doc.x++
}

// columnModeX returns the actual x from root.Doc.columnNum.
func (root *Root) columnModeX() int {
	lc, err := root.Doc.lineToContents(root.Doc.lineNum+root.Doc.Header, root.Doc.TabWidth)
	if err != nil {
		return 0
	}
	lineStr, byteMap := contentsToStr(lc)
	start, end := rangePosition(lineStr, root.Doc.ColumnDelimiter, root.Doc.columnNum)
	if start < 0 || end < 0 {
		root.Doc.columnNum = 0
		start, _ = rangePosition(lineStr, root.Doc.ColumnDelimiter, root.Doc.columnNum)
	}
	return byteMap[start]
}

// Move to the left by half a screen.
func (root *Root) moveHfLeft() {
	if root.Doc.WrapMode {
		return
	}
	root.resetSelect()
	moveSize := (root.vWidth / 2)
	if root.Doc.x > 0 && (root.Doc.x-moveSize) < 0 {
		root.Doc.x = 0
	} else {
		root.Doc.x -= moveSize
		if root.Doc.x < root.minStartX {
			root.Doc.x = root.minStartX
		}
	}
}

// Move to the right by half a screen.
func (root *Root) moveHfRight() {
	if root.Doc.WrapMode {
		return
	}
	root.resetSelect()
	if root.Doc.x < 0 {
		root.Doc.x = 0
	} else {
		root.Doc.x += (root.vWidth / 2)
	}
}
