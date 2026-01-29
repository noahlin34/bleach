package processor

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

var pngSignature = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

func stripPNG(r io.Reader, w io.Writer, preserveICC bool) error {
	br := bufio.NewReader(r)
	bw := bufio.NewWriter(w)

	sig := make([]byte, 8)
	if _, err := io.ReadFull(br, sig); err != nil {
		return err
	}
	if !bytesEqual(sig, pngSignature) {
		return fmt.Errorf("invalid PNG signature")
	}
	if _, err := bw.Write(sig); err != nil {
		return err
	}

	for {
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(br, lenBuf); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		length := binary.BigEndian.Uint32(lenBuf)

		typeBuf := make([]byte, 4)
		if _, err := io.ReadFull(br, typeBuf); err != nil {
			return err
		}
		chunkName := string(typeBuf)

		if shouldDropPNGChunk(chunkName, preserveICC) {
			if _, err := io.CopyN(io.Discard, br, int64(length)+4); err != nil {
				return err
			}
			if chunkName == "IEND" {
				break
			}
			continue
		}

		if _, err := bw.Write(lenBuf); err != nil {
			return err
		}
		if _, err := bw.Write(typeBuf); err != nil {
			return err
		}
		if _, err := io.CopyN(bw, br, int64(length)+4); err != nil {
			return err
		}

		if chunkName == "IEND" {
			break
		}
	}

	return bw.Flush()
}

func shouldDropPNGChunk(chunkName string, preserveICC bool) bool {
	switch chunkName {
	case "tEXt", "zTXt", "iTXt", "eXIf", "tIME":
		return true
	case "iCCP":
		return !preserveICC
	default:
		return false
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
