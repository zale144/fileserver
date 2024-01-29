package merkle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTree(t *testing.T) {
	type args struct {
		dataBlocks [][]byte
	}
	tests := []struct {
		name         string
		args         args
		wantRootHash string
	}{
		{
			name: "TestNewTree",
			args: args{
				dataBlocks: [][]byte{
					[]byte("test1"),
					[]byte("test2"),
					[]byte("test3"),
					[]byte("test4"),
				},
			},
			wantRootHash: "f208e011cdaae9c1bf083c2cc413880aa53441449820d477a41934d30b8a687b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTree(tt.args.dataBlocks)
			assert.NotNil(t, got)
			assert.Equal(t, tt.wantRootHash, got.RootHash())
		})
	}
}

func TestNewTreeFromStream(t *testing.T) {
	type args struct {
		dataBlocks <-chan []byte
		dataSize   int
	}
	tests := []struct {
		name         string
		args         args
		wantRootHash string
	}{
		{
			name: "TestNewTree",
			args: args{
				dataBlocks: func() <-chan []byte {
					ch := make(chan []byte)
					go func() {
						ch <- HashData([]byte("test1"))
						ch <- HashData([]byte("test2"))
						ch <- HashData([]byte("test3"))
						ch <- HashData([]byte("test4"))
						close(ch)
					}()
					return ch
				}(),
				dataSize: 4,
			},
			wantRootHash: "f208e011cdaae9c1bf083c2cc413880aa53441449820d477a41934d30b8a687b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTreeFromStream(tt.args.dataBlocks, tt.args.dataSize)
			require.NotNil(t, got)
			require.NotNil(t, got.Root)
			assert.Equal(t, tt.wantRootHash, got.RootHash())
		})
	}
}
