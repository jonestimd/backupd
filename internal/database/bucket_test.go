package database

type mockBucket struct {
	keyValues map[string][]byte
}

func (b *mockBucket) Get(key []byte) []byte {
	return b.keyValues[string(key)]
}

func (b *mockBucket) Put(key []byte, value []byte) error {
	b.keyValues[string(key)] = value
	return nil
}

func (b *mockBucket) ForEach(cb func(key []byte, value []byte) error) error {
	for key, value := range b.keyValues {
		cb([]byte(key), value)
	}
	return nil
}

func makeMockBucket() *mockBucket {
	return &mockBucket{make(map[string][]byte)}
}

func makeFileBucket(files ...*RemoteFile) *mockBucket {
	b := makeMockBucket()
	for _, file := range files {
		b.keyValues[file.Name] = toBytes(file)
	}
	return b
}
