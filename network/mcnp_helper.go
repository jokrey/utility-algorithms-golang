package network

import (
	"net"
	"own_util"
	"io"
	"encoding/binary"
	"math"
	"errors"
	"os"
	"fmt"
)

/// Simple, High-Level wrapper, that adds some functionality to the most basic byte stream.
///
/// Useful for simple, standardized inter programming language communication. Where build in high level constructs are not available.
/// Inter-programming Language entails that both sides will only be able to use the most basic, low level byte-"streams".
/// Writing a number bytes is easy here. However it gets tricky when the receiver doesn't know how many bytes to except.
///       For example when transmitting a file or a complex data structur such as a string.
/// Sends and reads multiple byte arrays(chunks) of completly variable length over a single established connection.
///       Apart from that some very basic(and common) data types of fixed length are also supported by the protocol.
///       (byte, byte arrays of fixed length, int16(twos_compl), int32(twos_compl), int64(twos_compl), float32(IEEE-754), float64(IEEE-754))
///            NOTE: Booleans are not supported since their implementation greatly differs between common languages.
///
/// The idea is that after establishing a connection the client sends a "cause" byte, indicating what kind of complex "conversation" it would like to have.
/// After that both sides have to each know exactly what kind of data the other one wants.
///
///    An Example of a typical "conversation" (usefulness of the example data in the braces is debatable ;) ):
/// |           Client            |            Server            |
/// |                             |    waitForNewConnection      |
/// |     establishConnection     |  handleConnection(newThread) |
/// |                             |         waitForCause         |
/// |         sendCause           |         receiveCause         |
/// |                             |          waitForInt          |
/// |         sendInt x           |          receiveInt          |
/// |         waitForInt          |      doOperationOnX (x*x)    |
/// |         receiveInt          |         sendInt (x*x)        |
/// |                             | waitForChunkOfVariableLength |
/// |      closeConnection        | closeConnection(finishThread)|
///
/// More complex, simultaneous, two way communication (for a example a game server may need), can also be achieved using this protocol.
///    Then both sides would have 2 simultaneous, one sided(likely too slow otherwise), conversations.
///    However that may not be fast enough. Then fixed package size, with a fixed cause at byte position 0, and fixed data sizes being send in the same chunk would be preferable.
///    This protocol may then be overkill.
///
/// MCNP <=> Multi Chunk Network Protocol



//CAUSE
  //SEND
	func Send_cause(conn net.Conn, cause int32) error {
		return Send_fixed_chunk_int32(conn, cause)
	}
  //READ
	func Read_cause(conn net.Conn) (int32, error) {
		i, err := Read_fixed_chunk_int32(conn)
		return i, err
	}



//FIXED SIZE CHUNKS
//SEND
	func Send_fixed_chunk_byte(conn net.Conn, b byte) error {
		b_as_slice := make([]byte, 1)
		b_as_slice[0] = b;
		return Send_fixed_chunk_bytes(conn, b_as_slice)
	}
	func Send_fixed_chunk_int16(conn net.Conn, i int16) error {
		i_as_slice := make([]byte, 2)
		binary.BigEndian.PutUint16(i_as_slice, uint16(i))
		return Send_fixed_chunk_bytes(conn, i_as_slice)
	}
	func Send_fixed_chunk_int32(conn net.Conn, i int32) error {
		i_as_slice := make([]byte, 4)
		binary.BigEndian.PutUint32(i_as_slice, uint32(i))
		return Send_fixed_chunk_bytes(conn, i_as_slice)
	}
	func Send_fixed_chunk_int64(conn net.Conn, i int64) error {
		i_as_slice := make([]byte, 8)
		binary.BigEndian.PutUint64(i_as_slice, uint64(i))
		return Send_fixed_chunk_bytes(conn, i_as_slice)
	}
	func Send_fixed_chunk_float32(conn net.Conn, f float32) error {
		i_as_slice := make([]byte, 4)
		binary.BigEndian.PutUint32(i_as_slice, math.Float32bits(f))
		return Send_fixed_chunk_bytes(conn, i_as_slice)
	}
	func Send_fixed_chunk_float64(conn net.Conn, f float64) error {
		i_as_slice := make([]byte, 8)
		binary.BigEndian.PutUint64(i_as_slice, math.Float64bits(f))
		return Send_fixed_chunk_bytes(conn, i_as_slice)
	}
	func Send_fixed_chunk_bytes(conn net.Conn, bytes []byte) error {
		_, err := conn.Write(bytes)
		return err
	}
//READ
	func Read_fixed_chunk_byte(conn net.Conn) (byte, error) {
		bytes, err := Read_fixed_chunk_bytes(conn, 1)
		return bytes[0], err
	}
	func Read_fixed_chunk_int16(conn net.Conn) (int16, error) {
		bytes, err := Read_fixed_chunk_bytes(conn, 2)
		return int16(binary.BigEndian.Uint16(bytes)), err
	}
	func Read_fixed_chunk_int32(conn net.Conn) (int32, error) {
		bytes, err := Read_fixed_chunk_bytes(conn, 4)
		return int32(binary.BigEndian.Uint32(bytes)), err
	}
	func Read_fixed_chunk_int64(conn net.Conn) (int64, error) {
		bytes, err := Read_fixed_chunk_bytes(conn, 8)
		return int64(binary.BigEndian.Uint64(bytes)), err
	}
	func Read_fixed_chunk_float32(conn net.Conn) (float32, error) {
		bytes, err := Read_fixed_chunk_bytes(conn, 4)
		return math.Float32frombits(binary.BigEndian.Uint32(bytes)), err
	}
	func Read_fixed_chunk_float64(conn net.Conn) (float64, error) {
		bytes, err := Read_fixed_chunk_bytes(conn, 8)
		return math.Float64frombits(binary.BigEndian.Uint64(bytes)), err
	}
	func Read_fixed_chunk_bytes(conn net.Conn, bytesToRead int32) ([]byte, error) {
		read_buffer := make([]byte, bytesToRead)
		n, err := conn.Read(read_buffer)
		if n != int(bytesToRead) {
			return read_buffer, errors.New("read inexact number of bytes")
		}
		return read_buffer, err
	}






//VARIABLE SIZE CHUNKS
//SEND
	func Start_chunk(conn net.Conn, chunk_size int64) error {
		return Send_fixed_chunk_int64(conn, chunk_size);//not really semantic wise, just for code minimalism
	}
	func Send_variable_chunk_utf8(conn net.Conn, s string) error {
		return Send_variable_chunk_bytearr(conn, []byte(s))
	}
	func Send_variable_chunk_bytearr(conn net.Conn, bytes []byte) error {
		err_sc := Start_chunk(conn, int64(len(bytes)))
		if err_sc!=nil {
			return err_sc
		}
		return Send_fixed_chunk_bytes(conn, bytes)
	}
//TODO UNTESTED
	func Send_variable_chunk_from_file(conn net.Conn, filepath string) error {
		file, err := os.Open(filepath)
		if err == nil {
			defer file.Close()
			fileStats, _ := file.Stat()
			fileLength := fileStats.Size()

			Start_chunk(conn, fileLength)

			byteCounter := int64(0)
			for {
				bufferSize := 1024 * 4
				buffer := make([]byte, own_util.Min(fileLength-byteCounter, int64(bufferSize)))

				n, err := file.Read(buffer)
				if n == 0 || err != nil {
					break
				}
				err = Send_fixed_chunk_bytes(conn, buffer[:n])
				if err != nil {
					break
				}
				byteCounter += int64(n)
			}
		}
		return err
	}

//READ
	func Read_variable_chunk_utf8(conn net.Conn) (string, error) {
		bytes, err := Read_variable_chunk_bytearr(conn)
		return string(bytes[:]), err
	}
	func Read_variable_chunk_bytearr(conn net.Conn) ([]byte, error) {
		incomingChunkSize, err := Read_fixed_chunk_int64(conn)

		readinto := make([]byte, 0)

		if err == nil {
			byteCounter := 0
			for {
				if int64(byteCounter) >= incomingChunkSize {
					break
				}
				bufferSize := 1024 * 4
				readTmpBuffer := make([]byte, own_util.Min(incomingChunkSize, int64(bufferSize)))
				n, err := conn.Read(readTmpBuffer)
				if err != nil {
					if err == io.EOF {//this error is ok
						return readinto, nil //the eof is fine, so it should not be handled as such by caller
					}
					break
				}
				if int64(byteCounter+n) <= incomingChunkSize {
					readinto = append(readinto, readTmpBuffer[0:n]...) //i don't get the dots, but thats how that apparently works in golang
					byteCounter += n
				} else {
					break
				}
			}
		}

		return readinto, err
	}
	func Read_variable_chunk_in_parts(conn net.Conn, received_part_callback func([]byte)) error {
		incomingChunkSize, err := Read_fixed_chunk_int64(conn)

		if err == nil {
			byteCounter := 0
			for {
				if int64(byteCounter) >= incomingChunkSize {
					break
				}
				bufferSize := 1024 * 4
				readTmpBuffer := make([]byte, own_util.Min(incomingChunkSize, int64(bufferSize)))
				n, err := conn.Read(readTmpBuffer)
				if err != nil {
					if err == io.EOF {//this error is ok
						received_part_callback(readTmpBuffer[0:n])
						return nil //the eof is fine, so it should not be handled as such by caller
					}
					break
				}
				if int64(byteCounter+n) <= incomingChunkSize {
					received_part_callback(readTmpBuffer[0:n])
					byteCounter += n
				} else {
					break
				}
			}
		}

		return err
	}
	func Read_variable_chunk_into_file(conn net.Conn, filepath string) error {
		f, err := os.Create(filepath)
		defer f.Close()

		if err == nil {
			incomingChunkSize, err := Read_fixed_chunk_int64(conn)

			if err == nil {
				byteCounter := int64(0)
				for {
					if int64(byteCounter) >= incomingChunkSize {
						break
					}
					bufferSize := 1024/// 4
					readTmpBuffer := make([]byte, own_util.Min(incomingChunkSize-byteCounter, int64(bufferSize)))
					n, err := conn.Read(readTmpBuffer)
					if err != nil {
						if err == io.EOF {
							return nil
						}
						break
					}
					if byteCounter+int64(n) <= incomingChunkSize {
						f.Write(readTmpBuffer[0:n])
						byteCounter += int64(n)
					} else {
						break
					}
				}
			}
		}
		return err
	}