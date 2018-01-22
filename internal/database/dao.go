package database

type remoteFile struct {
	Name         string
	Size         uint64
	Md5Checksum  *string
	ParentIds    []string
	LastModified *string
	LocalId      *string
}

type Transation interface {
	InsertFile(id string, name string, size uint64, md5checksum *string, parentIds []string, lastModified string, localId *string) error
	SetPaths() error
	ForEachPath(func(path string, fileId string) error) error
}

type Dao interface {
	IsEmpty() bool
	Update(func(Transation) error) error
	View(func(Transation) error) error
}
