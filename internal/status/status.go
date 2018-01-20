package status

type Key struct {
	Ino uint64
}

type Status struct {
	Destination string
	Path        string
	LastUpdate  uint64
	Size        uint64
	Checksum    string
}
