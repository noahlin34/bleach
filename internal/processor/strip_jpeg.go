package processor

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

var (
	jpegExifHeader = []byte("Exif\x00\x00")
	jpegXmpHeader  = []byte("http://ns.adobe.com/xap/1.0/\x00")
	jpegPhotoshop  = []byte("Photoshop 3.0\x00")
	jpegICCHeader  = []byte("ICC_PROFILE\x00")
)

func stripJPEG(r io.Reader, w io.Writer, preserveICC bool) error {
	br := bufio.NewReader(r)
	bw := bufio.NewWriter(w)

	soi := make([]byte, 2)
	if _, err := io.ReadFull(br, soi); err != nil {
		return err
	}
	if soi[0] != 0xff || soi[1] != 0xd8 {
		return fmt.Errorf("invalid JPEG SOI")
	}
	if _, err := bw.Write(soi); err != nil {
		return err
	}

	for {
		markerPrefix, err := br.ReadByte()
		if err != nil {
			return err
		}
		for markerPrefix != 0xff {
			markerPrefix, err = br.ReadByte()
			if err != nil {
				return err
			}
		}

		marker, err := br.ReadByte()
		if err != nil {
			return err
		}
		for marker == 0xff {
			marker, err = br.ReadByte()
			if err != nil {
				return err
			}
		}

		if marker == 0xd9 { // EOI
			if _, err := bw.Write([]byte{0xff, 0xd9}); err != nil {
				return err
			}
			break
		}

		if marker == 0xda { // SOS
			if _, err := bw.Write([]byte{0xff, marker}); err != nil {
				return err
			}
			if _, err := io.Copy(bw, br); err != nil {
				return err
			}
			break
		}

		if marker == 0x01 || (marker >= 0xd0 && marker <= 0xd7) {
			if _, err := bw.Write([]byte{0xff, marker}); err != nil {
				return err
			}
			continue
		}

		lenBuf := make([]byte, 2)
		if _, err := io.ReadFull(br, lenBuf); err != nil {
			return err
		}
		segLen := int(binary.BigEndian.Uint16(lenBuf))
		if segLen < 2 {
			return fmt.Errorf("invalid JPEG segment length")
		}
		payloadLen := segLen - 2
		if marker == 0xe1 || marker == 0xe2 || marker == 0xed {
			payload := make([]byte, payloadLen)
			if _, err := io.ReadFull(br, payload); err != nil {
				return err
			}

			if shouldDropJPEGSegment(marker, payload, preserveICC) {
				continue
			}

			if _, err := bw.Write([]byte{0xff, marker}); err != nil {
				return err
			}
			if _, err := bw.Write(lenBuf); err != nil {
				return err
			}
			if _, err := bw.Write(payload); err != nil {
				return err
			}
			continue
		}

		if _, err := bw.Write([]byte{0xff, marker}); err != nil {
			return err
		}
		if _, err := bw.Write(lenBuf); err != nil {
			return err
		}
		if _, err := io.CopyN(bw, br, int64(payloadLen)); err != nil {
			return err
		}
	}

	return bw.Flush()
}

func shouldDropJPEGSegment(marker byte, payload []byte, preserveICC bool) bool {
	switch marker {
	case 0xe1:
		if hasPrefix(payload, jpegExifHeader) || hasPrefix(payload, jpegXmpHeader) {
			return true
		}
	case 0xed:
		if hasPrefix(payload, jpegPhotoshop) {
			return true
		}
	case 0xe2:
		if !preserveICC && hasPrefix(payload, jpegICCHeader) {
			return true
		}
	}

	return false
}

func hasPrefix(buf, prefix []byte) bool {
	if len(buf) < len(prefix) {
		return false
	}
	for i := range prefix {
		if buf[i] != prefix[i] {
			return false
		}
	}
	return true
}
