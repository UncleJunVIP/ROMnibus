package models

type DATGame struct {
	Name string
	ROM  DATROM
}

type DATROM struct {
	Name   string
	Size   string
	CRC    string
	MD5    string
	SHA1   string
	SHA256 string
}
