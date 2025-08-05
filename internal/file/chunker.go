package file

import "io"

const chunkSize = 1024 * 1024

func Chunk(r io.Reader) ([][]byte, error) {
	var chunks [][]byte
	buf := make([]byte, chunkSize)
	for {
		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}
		chunks = append(chunks, append([]byte(nil), buf[:n]...))
	}
	return chunks, nil
}
