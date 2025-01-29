package core

type Msg struct {
	Subject string
	Data    any
	Header  map[string][]string
}
