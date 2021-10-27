// +build web

package data

import _ "embed"

//go:embed data/LobsterRegular-R7AM.otf
var FontLobster []byte

//go:embed data/open-sans.regular.ttf
var FontOpenSans []byte

//go:embed data/app.js
var AppJS string

//go:embed data/n.png
var ImgN []byte

//go:embed data/f.png
var ImgF []byte

//go:embed data/m.png
var ImgM []byte

//go:embed data/fav.png
var ImgFav []byte
