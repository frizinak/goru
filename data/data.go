package data

import _ "embed"

//go:embed data/db.gob
var Words []byte

//go:embed data/LobsterRegular-R7AM.otf
var FontLobster []byte

//go:embed data/open-sans.regular.ttf
var FontOpenSans []byte
