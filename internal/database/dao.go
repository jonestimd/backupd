package database

type RemoteFile struct {
	Id           string
	Name         string
	Size         uint64
	Md5Checksum  *string
	ParentIds    []string
	LastModified *string
	LocalId      *string
}

type Transation interface {
	InsertFile(*RemoteFile) error
	SetPaths() error
	ForEachPath(func(path string, fileId string) error) error
}

type Dao interface {
	IsEmpty() bool
	Update(func(Transation) error) error
	View(func(Transation) error) error
}
